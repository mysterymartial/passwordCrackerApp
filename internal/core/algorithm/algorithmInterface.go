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
