package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"sync"
	"sync/atomic"
)

type Hybrid struct {
	settings      domain.CrackingSettings
	progress      float64
	mu            sync.RWMutex
	stop          chan struct{}
	algorithms    []Algorithm
	totalTried    uint64
	totalPossible uint64
}

func NewHybrid() *Hybrid {
	return &Hybrid{
		stop: make(chan struct{}),
		algorithms: []Algorithm{
			NewDictionary(),
			NewBruteForce(),
			NewMask(),
		},
	}
}

func (h *Hybrid) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		results := make([]<-chan string, len(h.algorithms))
		errs := make([]<-chan error, len(h.algorithms))

		// Start all algorithms concurrently
		for i, alg := range h.algorithms {
			results[i], errs[i] = alg.Start(ctx)
		}

		// Monitor errors from all algorithms
		errChan := h.mergeErrors(ctx, errs...)
		go func() {
			for err := range errChan {
				errors <- err
			}
		}()

		// Process passwords using fan-in pattern
		merged := h.fanIn(ctx, results...)
		for password := range merged {
			atomic.AddUint64(&h.totalTried, 1)
			h.updateProgress()

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

	multiplex := func(c <-chan string) {
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
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	go func() {
		wg.Wait()
		close(multiplexed)
	}()

	return multiplexed
}

func (h *Hybrid) mergeErrors(ctx context.Context, channels ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	merged := make(chan error)

	merge := func(c <-chan error) {
		defer wg.Done()
		for err := range c {
			select {
			case merged <- err:
			case <-ctx.Done():
				return
			case <-h.stop:
				return
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go merge(ch)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

func (h *Hybrid) updateProgress() {
	h.mu.Lock()
	defer h.mu.Unlock()

	var totalProgress float64
	for _, alg := range h.algorithms {
		totalProgress += alg.Progress()
	}
	h.progress = totalProgress / float64(len(h.algorithms))
}

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
