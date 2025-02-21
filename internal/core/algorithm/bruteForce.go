package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"sync"
)

type BruteForce struct {
	settings     domain.CrackingSettings
	progress     float64
	currentLen   int
	combinations int64
	mu           sync.RWMutex
	stop         chan struct{}
}

func NewBruteForce() *BruteForce {
	return &BruteForce{
		stop: make(chan struct{}),
	}
}

func (b *BruteForce) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		charset := b.settings.CharacterSet
		if charset == "" {
			charset = domain.CharsetAll
		}

		for length := b.settings.MinLength; length <= b.settings.MaxLength; length++ {
			b.currentLen = length
			b.generatePasswords(ctx, charset, "", length, passwords)

			select {
			case <-ctx.Done():
				return
			case <-b.stop:
				return
			default:
				continue
			}
		}
	}()

	return passwords, errors
}

func (b *BruteForce) generatePasswords(ctx context.Context, charset string, current string, length int, passwords chan<- string) {
	if length == 0 {
		select {
		case passwords <- current:
		case <-ctx.Done():
			return
		case <-b.stop:
			return
		}
		return
	}

	for _, char := range charset {
		b.generatePasswords(ctx, charset, current+string(char), length-1, passwords)
	}
}

func (b *BruteForce) Stop() {
	close(b.stop)
}

func (b *BruteForce) Progress() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.progress
}

func (b *BruteForce) Name() domain.CrackingAlgorithm {
	return domain.AlgoBruteForce
}

func (b *BruteForce) SetSettings(settings domain.CrackingSettings) {
	b.settings = settings
}
