package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"sync"
)

type Hybrid struct {
	settings   domain.CrackingSettings
	progress   float64
	mu         sync.RWMutex
	stop       chan struct{}
	algorithms []Algorithm
}

func NewHybrid() *Hybrid {
	return &Hybrid{
		stop: make(chan struct{}),
		algorithms: []Algorithm{
			NewDictionary(),
			NewMask(),
			NewMarkov(),
		},
	}
}

func (h *Hybrid) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		// Create a channel for each algorithm
		results := make([]<-chan string, len(h.algorithms))
		errs := make([]<-chan error, len(h.algorithms))

		// Start all algorithms
		for i, alg := range h.algorithms {
			alg.SetSettings(h.settings)
			results[i], errs[i] = alg.Start(ctx)
		}

		// Merge results using fan-in pattern
		merged := h.fanIn(ctx, results...)

		// Process results
		for password := range merged {
			select {
			case passwords <- password:
			case <-ctx.Done():
				return
			case <-h.stop:
				return
			}
		}
	}()

	return passwords, errors
}

func (h *Hybrid) fanIn(ctx context.Context, channels ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	multiplexed := make(chan string)

	// Start a goroutine for each input channel
	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan string) {
			defer wg.Done()
			for password := range c {
				select {
				case multiplexed <- password:
				case <-ctx.Done():
					return
				case <-h.stop:
					return
				}
			}
		}(ch)
	}

	// Close multiplexed channel when all input channels are done
	go func() {
		wg.Wait()
		close(multiplexed)
	}()

	return multiplexed
}

// Required interface methods
func (h *Hybrid) Stop() {
	close(h.stop)
	for _, alg := range h.algorithms {
		alg.Stop()
	}
}

func (h *Hybrid) Progress() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.progress
}

func (h *Hybrid) Name() domain.CrackingAlgorithm {
	return domain.AlgoHybrid
}

func (h *Hybrid) SetSettings(settings domain.CrackingSettings) {
	h.settings = settings
	for _, alg := range h.algorithms {
		alg.SetSettings(settings)
	}
}
