package service

import (
	"context"
	"github.com/google/uuid"
	"passwordCrakerBackend/internal/algorithm"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/metrics"
	"passwordCrakerBackend/internal/port"
	"runtime"
	"sync"
	"time"
)

type CrackingService struct {
	repo             port.Repository
	hashService      port.HashService
	metricsCollector *metrics.Collector
	algorithms       map[domain.CrackingAlgorithm]algorithm.Algorithm
	batchSize        int
	workerPool       chan struct{}
}

func NewCrackingService(repo port.Repository, hashSvc port.HashService) *CrackingService {
	return &CrackingService{
		repo:             repo,
		hashService:      hashSvc,
		metricsCollector: metrics.NewCollector(),
		algorithms: map[domain.CrackingAlgorithm]algorithm.Algorithm{
			domain.AlgoBruteForce: algorithm.NewBruteForce(),
			domain.AlgoDictionary: algorithm.NewDictionary(),
			domain.AlgoMarkov:     algorithm.NewMarkov(),
			domain.AlgoHybrid:     algorithm.NewHybrid(),
			domain.AlgoMask:       algorithm.NewMask(),
			domain.AlgoRainbow:    algorithm.NewRainbow(),
		},
		batchSize:  1000,
		workerPool: make(chan struct{}, runtime.NumCPU()),
	}
}

func (s *CrackingService) StartCracking(ctx context.Context, hash string, settings domain.CrackingSettings) (*domain.CrackingJob, error) {
	job := &domain.CrackingJob{
		ID:         uuid.New().String(),
		TargetHash: hash,
		Status:     domain.StatusRunning,
		StartTime:  time.Now(),
		Settings:   settings,
	}

	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	s.metricsCollector.StartJob(job.ID)

	resultChan := make(chan domain.CrackResult)
	errorChan := make(chan error)

	var wg sync.WaitGroup
	for algo := range s.algorithms {
		wg.Add(1)
		go func(alg algorithm.Algorithm) {
			defer wg.Done()
			s.runAlgorithm(ctx, job, alg, resultChan, errorChan)
		}(s.algorithms[algo])
	}

	go s.monitorProgress(ctx, job, resultChan, errorChan, &wg)

	return job, nil
}

func (s *CrackingService) runAlgorithm(
	ctx context.Context,
	job *domain.CrackingJob,
	alg algorithm.Algorithm,
	resultChan chan<- domain.CrackResult,
	errorChan chan<- error,
) {
	s.workerPool <- struct{}{}        // Acquire worker
	defer func() { <-s.workerPool }() // Release worker

	passwords, errors := alg.Start(ctx)
	batch := make([]string, 0, s.batchSize)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errors:
			errorChan <- err
			return
		case pwd, ok := <-passwords:
			if !ok {
				return
			}

			batch = append(batch, pwd)
			if len(batch) >= s.batchSize {
				if found := s.verifyPasswordBatch(batch, job); found != "" {
					resultChan <- domain.CrackResult{
						Password:  found,
						Algorithm: alg.Name(),
						TimeTaken: time.Since(job.StartTime),
					}
					return
				}
				batch = batch[:0]
			}
		}
	}
}

func (s *CrackingService) verifyPasswordBatch(batch []string, job *domain.CrackingJob) string {
	for _, pwd := range batch {
		if s.hashService.Verify(pwd, job.TargetHash) {
			return pwd
		}
	}
	return ""
}

func (s *CrackingService) monitorProgress(
	ctx context.Context,
	job *domain.CrackingJob,
	resultChan <-chan domain.CrackResult,
	errorChan <-chan error,
	wg *sync.WaitGroup,
) {
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				continue
			}
			s.handleSuccess(ctx, job, result)
			return

		case err, ok := <-errorChan:
			if !ok {
				continue
			}
			s.handleError(ctx, job, err)

		case <-ctx.Done():
			s.handleCancellation(ctx, job)
			return
		}
	}
}

func (s *CrackingService) handleSuccess(ctx context.Context, job *domain.CrackingJob, result domain.CrackResult) {
	job.Status = domain.StatusComplete
	job.FoundPassword = result.Password
	job.EndTime = time.Now()
	job.SuccessfulAlgorithm = result.Algorithm
	s.repo.UpdateJob(ctx, job)
	s.stopAllAlgorithms()
}

func (s *CrackingService) handleError(ctx context.Context, job *domain.CrackingJob, err error) {
	job.Status = domain.StatusFailed
	job.ErrorMessage = err.Error()
	s.repo.UpdateJob(ctx, job)
}

func (s *CrackingService) handleCancellation(ctx context.Context, job *domain.CrackingJob) {
	job.Status = domain.StatusCancelled
	job.EndTime = time.Now()
	s.repo.UpdateJob(ctx, job)
	s.stopAllAlgorithms()
}

func (s *CrackingService) stopAllAlgorithms() {
	for _, alg := range s.algorithms {
		alg.Stop()
	}
}

func (s *CrackingService) GetProgress(ctx context.Context, jobID string) (*domain.JobProgress, error) {
	metrics := s.metricsCollector.GetMetrics(jobID)
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &domain.JobProgress{
		JobID:         jobID,
		Status:        job.Status,
		Progress:      metrics.Progress,
		Speed:         metrics.Speed,
		TimeRemaining: metrics.EstimatedTimeRemaining,
		ActiveThreads: len(s.algorithms),
	}, nil
}
func (s *CrackingService) StopCracking(ctx context.Context, jobID string) error {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	job.Status = domain.StatusCancelled
	return s.repo.UpdateJob(ctx, job)
}

func (s *CrackingService) GetResults(ctx context.Context, jobID string) (*domain.CrackingResult, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &domain.CrackingResult{
		JobID:         job.ID,
		Hash:          job.TargetHash,
		FoundPassword: job.FoundPassword,
		TimeTaken:     job.EndTime.Sub(job.StartTime).Seconds(),
		Algorithm:     string(job.SuccessfulAlgorithm),
	}, nil
}
