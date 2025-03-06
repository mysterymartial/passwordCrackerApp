package service

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
)

type CrackingServiceInterface interface {
	StartCracking(ctx context.Context, hash string, settings domain.CrackingSettings) (*domain.CrackingJob, error)
	GetProgress(ctx context.Context, jobID string) (*domain.JobProgress, error)
	StopCracking(ctx context.Context, jobID string) error
	GetResults(ctx context.Context, jobID string) (*domain.CrackingResult, error)
}
