package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/utils/wordlist"
	"sort"
	"sync"
	"sync/atomic"
)

type MarkovChain struct {
	transitions map[string]map[string]float64
	initial     map[string]float64
	totalStates int64
}

type Markov struct {
	settings        domain.CrackingSettings
	progress        float64
	mu              sync.RWMutex
	stop            chan struct{}
	chain           *MarkovChain
	order           int
	processedStates int64
	totalPasswords  int64
}

func NewMarkov() *Markov {
	return &Markov{
		stop: make(chan struct{}),
		chain: &MarkovChain{
			transitions: make(map[string]map[string]float64),
			initial:     make(map[string]float64),
		},
		order: 3,
	}
}

func (m *Markov) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		if err := m.trainModel(ctx); err != nil {
			select {
			case errors <- err:
			case <-ctx.Done():
			}
			return
		}

		m.normalizeChain()

		workerCount := 4
		results := make([]<-chan string, workerCount)

		for i := 0; i < workerCount; i++ {
			results[i] = m.generatePasswordsWorker(ctx, i, workerCount)
		}

		merged := m.mergeChannels(ctx, results...)
		for password := range merged {
			select {
			case passwords <- password:
				atomic.AddInt64(&m.processedStates, 1)
				m.updateProgress()
			case <-ctx.Done():
				return
			case <-m.stop:
				return
			}
		}
	}()

	return passwords, errors
}

func (m *Markov) trainModel(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(m.settings.WordlistPaths))

	for _, path := range m.settings.WordlistPaths {
		wg.Add(1)
		go func(wordlistPath string) {
			defer wg.Done()

			words, err := wordlist.LoadWordlist(wordlistPath)
			if err != nil {
				errChan <- err
				return
			}

			for _, word := range words {
				select {
				case <-ctx.Done():
					return
				case <-m.stop:
					return
				default:
					m.updateChain(word)
				}
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

func (m *Markov) updateChain(word string) {
	if len(word) < m.order {
		return
	}

	prefix := word[:m.order]
	m.chain.initial[prefix]++

	for i := 0; i < len(word)-m.order; i++ {
		current := word[i : i+m.order]
		next := string(word[i+m.order])

		if m.chain.transitions[current] == nil {
			m.chain.transitions[current] = make(map[string]float64)
		}
		m.chain.transitions[current][next]++
	}
}

func (m *Markov) normalizeChain() {
	total := 0.0
	for _, count := range m.chain.initial {
		total += count
	}

	for state := range m.chain.initial {
		m.chain.initial[state] /= total
	}

	for state, transitions := range m.chain.transitions {
		total := 0.0
		for _, count := range transitions {
			total += count
		}
		for next := range transitions {
			m.chain.transitions[state][next] /= total
		}
	}
}

func (m *Markov) generatePasswordsWorker(ctx context.Context, workerID, totalWorkers int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		initialStates := m.getInitialStates()
		workerStates := m.partitionStates(initialStates, workerID, totalWorkers)

		for _, state := range workerStates {
			if password := m.generatePassword(state); password != "" {
				select {
				case out <- password:
				case <-ctx.Done():
					return
				case <-m.stop:
					return
				}
			}
		}
	}()

	return out
}

func (m *Markov) getInitialStates() []string {
	states := make([]string, 0, len(m.chain.initial))
	for state := range m.chain.initial {
		states = append(states, state)
	}

	sort.Slice(states, func(i, j int) bool {
		return m.chain.initial[states[i]] > m.chain.initial[states[j]]
	})

	return states
}

func (m *Markov) partitionStates(states []string, workerID, totalWorkers int) []string {
	statesPerWorker := len(states) / totalWorkers
	start := workerID * statesPerWorker
	end := start + statesPerWorker

	if workerID == totalWorkers-1 {
		end = len(states)
	}

	return states[start:end]
}

func (m *Markov) generatePassword(initial string) string {
	// Start with the initial state regardless of its length
	password := initial

	// Generate until we reach MaxLength or can't predict further
	for len(password) < m.settings.MaxLength {
		current := password[len(password)-m.order:]
		if next := m.predictNextChar(current); next != "" {
			password += next
		} else {
			break
		}
	}

	// Only return if within bounds
	if len(password) >= m.settings.MinLength && len(password) <= m.settings.MaxLength {
		return password
	}
	return ""
}

func (m *Markov) predictNextChar(current string) string {
	transitions := m.chain.transitions[current]
	if len(transitions) == 0 {
		return ""
	}

	var maxProb float64
	var nextChar string
	for char, prob := range transitions {
		if prob > maxProb {
			maxProb = prob
			nextChar = char
		}
	}
	return nextChar
}

func (m *Markov) mergeChannels(ctx context.Context, channels ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan string) {
			defer wg.Done()
			for password := range c {
				select {
				case out <- password:
				case <-ctx.Done():
					return
				case <-m.stop:
					return
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (m *Markov) updateProgress() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progress = float64(atomic.LoadInt64(&m.processedStates)) / float64(m.chain.totalStates) * 100
}

func (m *Markov) Stop() {
	close(m.stop)
}

func (m *Markov) Progress() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.progress
}

func (m *Markov) Name() domain.CrackingAlgorithm {
	return domain.AlgoMarkov
}

func (m *Markov) SetSettings(settings domain.CrackingSettings) {
	m.settings = settings
}
