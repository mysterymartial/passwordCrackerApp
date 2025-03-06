package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
)

type Algorithm interface {
	Start(ctx context.Context) (<-chan string, <-chan error)
	Stop()
	Progress() float64
	Name() domain.CrackingAlgorithm
	SetSettings(settings domain.CrackingSettings)
}

type BaseAlgorithm struct {
	settings domain.CrackingSettings
	progress float64
	stop     chan struct{}
}

func (b *BaseAlgorithm) SetSettings(settings domain.CrackingSettings) {
	b.settings = settings
}

func (b *BaseAlgorithm) Progress() float64 {
	return b.progress
}

func (b *BaseAlgorithm) Stop() {
	close(b.stop)
}
