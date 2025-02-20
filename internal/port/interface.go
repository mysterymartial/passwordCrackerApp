package port

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
)

type CrackingService interface {
	StartCracking(ctx context.Context, hash string, settings domain.CrackingSettings) (*domain.CrackingJob, error)
	StopCracking(ctx context.Context, jobID string) error
	PauseCracking(ctx context.Context, jobID string) error
	ResumeCracking(ctx context.Context, jobID string) error
	GetJobStatus(ctx context.Context, jobID string) (*domain.JobStatus, error)
	ListJobs(ctx context.Context, filter JobFilter) ([]domain.CrackingJob, error)
	ImportWordlist(ctx context.Context, path string, category string) error
	GetStatistics(ctx context.Context, jobID string) (*domain.ResourceMetrics, error)
}

type Repository interface {
	SaveJob(ctx context.Context, job *domain.CrackingJob) error
	UpdateJob(ctx context.Context, job *domain.CrackingJob) error
	DeleteJob(ctx context.Context, jobID string) error
	GetJob(ctx context.Context, jobID string) (*domain.CrackingJob, error)
	ListJobs(ctx context.Context, filter JobFilter) ([]domain.CrackingJob, error)
	SaveWordlist(ctx context.Context, wordlist *domain.WordlistEntry) error
	GetWordlists(ctx context.Context) ([]domain.WordlistEntry, error)
	SaveMetrics(ctx context.Context, jobID string, metrics *domain.ResourceMetrics) error
}

type HashService interface {
	Identify(hash string) domain.HashType
	verify(password, hash string, hashType domain.HashType) bool
	Generate(password string, hashType domain.HashType) (string, error)
}
type PasswordGenerator interface {
	Generate(ctx context.Context) <-chan string
	SetPattern(pattern string)
	SetCharset(charset string)
	Reset()
}

type JobFilter struct {
	Status    domain.JobStatus
	HashType  domain.HashType
	Algorithm domain.CrackingAlgorithm
	StartDate int64
	EndDate   int64
	Limit     int
	Offset    int
}
