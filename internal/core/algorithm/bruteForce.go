package algorithm

import (
	"context"
	"math"
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

		// Calculate total combinations for progress tracking
		b.calculateTotalCombinations(charset)

		for length := b.settings.MinLength; length <= b.settings.MaxLength; length++ {
			b.currentLen = length
			b.generatePasswords(ctx, charset, "", length, passwords)

			select {
			case <-ctx.Done():
				return
			case <-b.stop:
				return
			default:
				b.updateProgress(length)
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

func (b *BruteForce) calculateTotalCombinations(charset string) {
	charsetLen := float64(len(charset))
	total := int64(0)

	for length := b.settings.MinLength; length <= b.settings.MaxLength; length++ {
		total += int64(math.Pow(charsetLen, float64(length)))
	}

	b.combinations = total
}

func (b *BruteForce) updateProgress(currentLength int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	charsetLen := float64(len(b.settings.CharacterSet))
	currentCombinations := int64(math.Pow(charsetLen, float64(currentLength)))

	b.progress = (float64(currentCombinations) / float64(b.combinations)) * 100
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
