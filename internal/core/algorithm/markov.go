package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/utils/wordlist"
	"sort"
	"sync"
)

type MarkovChain struct {
	transitions map[string]map[string]int
	initial     map[string]int
}

type Markov struct {
	settings domain.CrackingSettings
	progress float64
	mu       sync.RWMutex
	stop     chan struct{}
	chain    *MarkovChain
	order    int
}

func NewMarkov() *Markov {
	return &Markov{
		stop: make(chan struct{}),
		chain: &MarkovChain{
			transitions: make(map[string]map[string]int),
			initial:     make(map[string]int),
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

		// Train the Markov model using wordlists
		if err := m.trainModel(ctx); err != nil {
			select {
			case errors <- err:
			case <-ctx.Done():
			}
			return
		}

		// Generate passwords using the trained model
		for candidate := range m.generatePasswords(ctx) {
			select {
			case passwords <- candidate:
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
	for _, path := range m.settings.WordlistPaths {
		words, err := wordlist.LoadWordlist(path)
		if err != nil {
			return err
		}

		for _, word := range words {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-m.stop:
				return nil
			default:
				m.updateChain(word)
			}
		}
	}
	return nil
}

func (m *Markov) updateChain(word string) {
	if len(word) < m.order {
		return
	}

	// Update initial probabilities
	prefix := word[:m.order]
	m.chain.initial[prefix]++

	// Update transition probabilities
	for i := 0; i < len(word)-m.order; i++ {
		current := word[i : i+m.order]
		next := string(word[i+m.order])

		if m.chain.transitions[current] == nil {
			m.chain.transitions[current] = make(map[string]int)
		}
		m.chain.transitions[current][next]++
	}
}

func (m *Markov) generatePasswords(ctx context.Context) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		initialStates := make([]string, 0, len(m.chain.initial))
		for state := range m.chain.initial {
			initialStates = append(initialStates, state)
		}

		sort.Slice(initialStates, func(i, j int) bool {
			return m.chain.initial[initialStates[i]] > m.chain.initial[initialStates[j]]
		})

		total := float64(len(initialStates))
		for i, initial := range initialStates {
			password := initial
			for len(password) < m.settings.MaxLength {
				current := password[len(password)-m.order:]
				if next := m.predictNextChar(current); next != "" {
					password += next
				} else {
					break
				}
			}

			m.mu.Lock()
			m.progress = float64(i) / total
			m.mu.Unlock()

			if len(password) >= m.settings.MinLength {
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

func (m *Markov) predictNextChar(current string) string {
	if transitions, exists := m.chain.transitions[current]; exists {
		var maxCount int
		var nextChar string
		for char, count := range transitions {
			if count > maxCount {
				maxCount = count
				nextChar = char
			}
		}
		return nextChar
	}
	return ""
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
