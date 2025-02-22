package services

import (
	"context"
	"passwordCrakerBackend/internal/core/algorithm"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/pkg/concurrency"
	"passwordCrakerBackend/internal/pkg/metrics"
	"passwordCrakerBackend/internal/port"
	"runtime"
	"sync"
	"time"
)

const (
	MaxConcurrentAlgorithms = 6
	WorkerPoolBufferSize    = 100
	MetricsUpdateInterval   = time.Second
	DefaultTimeout          = 30 * time.Minute
	MaxAttempts             = 1000000000 // 1 billion attempts limit
	ResultBufferSize        = 10
)

type CrackingService struct {
	repo        port.Repository
	hashService port.HashService
	algorithms  map[domain.CrackingAlgorithm]algorithm.Algorithm
	activeJobs  sync.Map
	metrics     *metrics.Collector
	reporter    *metrics.Reporter
	workerPool  *concurrency.WorkerPool
	resultCache *sync.Map
}

func NewCrackingService(
	repo port.Repository,
	hashService port.HashService,
	algorithms map[domain.CrackingAlgorithm]algorithm.Algorithm,
) port.CrackingService {
	reporter, _ := metrics.NewReporter("cracking_metrics.log")

	// Optimize worker pool size based on available CPU cores
	maxWorkers := runtime.NumCPU() * 2
	if maxWorkers > MaxConcurrentAlgorithms {
		maxWorkers = MaxConcurrentAlgorithms
	}

	return &CrackingService{
		repo:        repo,
		hashService: hashService,
		algorithms:  algorithms,
		activeJobs:  sync.Map{},
		metrics:     metrics.NewCollector(MetricsUpdateInterval),
		reporter:    reporter,
		workerPool:  concurrency.NewWorkerPool(maxWorkers, WorkerPoolBufferSize),
		resultCache: &sync.Map{},
	}
}

func (s *CrackingService) StartCracking(ctx context.Context, hash string, settings domain.CrackingSettings) (*domain.CrackingJob, error) {
	// Validate hash
	hashType := s.hashService.Identify(hash)
	if hashType == "" {
		return nil, domain.ErrInvalidHash
	}

	// Create job with unique ID
	job := &domain.CrackingJob{
		ID:           random.GenerateUUID(),
		TargetHash:   hash,
		HashType:     hashType,
		Status:       domain.StatusRunning,
		StartTime:    time.Now(),
		Settings:     settings,
		AttemptCount: 0,
		LastAttempt:  time.Now(),
		Progress:     0.0,
	}

	// Save job to repository
	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}

	// Initialize metrics and caching
	s.activeJobs.Store(job.ID, job)
	s.metrics.StartCollection(job.ID)

	// Start cracking process
	go s.orchestrateCracking(ctx, job)

	return job, nil
}

func (s *CrackingService) orchestrateCracking(ctx context.Context, job *domain.CrackingJob) {
	resultChan := make(chan domain.CrackResult, ResultBufferSize)
	errorChan := make(chan error, MaxConcurrentAlgorithms)

	// Set timeout
	timeout := time.Duration(job.Settings.TimeoutMinutes) * time.Minute
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start all algorithms
	wg := &sync.WaitGroup{}
	for algType, alg := range s.algorithms {
		wg.Add(1)
		go func(at domain.CrackingAlgorithm, a algorithm.Algorithm) {
			defer wg.Done()
			s.executeAlgorithm(ctxWithTimeout, job, a, at, resultChan, errorChan)
		}(algType, alg)
	}

	// Monitor results in separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	s.handleResults(ctxWithTimeout, job, resultChan, errorChan)
}

func (s *CrackingService) executeAlgorithm(
	ctx context.Context,
	job *domain.CrackingJob,
	alg algorithm.Algorithm,
	algType domain.CrackingAlgorithm,
	resultChan chan<- domain.CrackResult,
	errorChan chan<- error,
) {
	defer s.metrics.StopCollection(job.ID)

	passwordChan, errChan := alg.Start(ctx, job.Settings)
	startTime := time.Now()
	attempts := int64(0)

	for {
		select {
		case password, ok := <-passwordChan:
			if !ok {
				return
			}

			attempts++
			if attempts > MaxAttempts {
				errorChan <- domain.ErrMaxAttemptsReached
				return
			}

			s.updateJobMetrics(job.ID, attempts)

			if s.hashService.Verify(password, job.TargetHash, job.HashType) {
				result := domain.CrackResult{
					Password:     password,
					TimeTaken:    time.Since(startTime),
					AttemptsUsed: attempts,
					Algorithm:    algType,
					Pattern:      alg.GetPattern(),
					Complexity:   s.analyzePassword(password, startTime),
				}
				resultChan <- result
				return
			}

		case err := <-errChan:
			errorChan <- err
			return

		case <-ctx.Done():
			return
		}
	}
}

func (s *CrackingService) handleResults(
	ctx context.Context,
	job *domain.CrackingJob,
	resultChan <-chan domain.CrackResult,
	errorChan <-chan error,
) {
	var result *domain.CrackResult
	var errors []error

	for {
		select {
		case r, ok := <-resultChan:
			if !ok {
				resultChan = nil
				continue
			}
			if result == nil || r.TimeTaken < result.TimeTaken {
				result = &r
			}

		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
				continue
			}
			errors = append(errors, err)

		case <-ctx.Done():
			s.finalizeJob(ctx, job, result, errors)
			return
		}

		if resultChan == nil && errorChan == nil {
			s.finalizeJob(ctx, job, result, errors)
			return
		}
	}
}

func (s *CrackingService) finalizeJob(
	ctx context.Context,
	job *domain.CrackingJob,
	result *domain.CrackResult,
	errors []error,
) {
	if result != nil {
		job.Status = domain.StatusSuccess
		job.Result = result
	} else {
		job.Status = domain.StatusFailed
		job.Errors = errors
	}

	job.EndTime = time.Now()
	job.TimeTaken = job.EndTime.Sub(job.StartTime)

	// Update repository
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		s.reporter.Error("Failed to update job", err)
	}

	// Cleanup
	s.activeJobs.Delete(job.ID)
	s.metrics.StopCollection(job.ID)
}

func (s *CrackingService) GetJobStatus(ctx context.Context, jobID string) (*domain.CrackingJob, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *CrackingService) StopJob(ctx context.Context, jobID string) error {
	if _, exists := s.activeJobs.Load(jobID); !exists {
		return domain.ErrJobNotFound
	}

	s.activeJobs.Delete(jobID)
	s.metrics.StopCollection(jobID)

	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = domain.StatusStopped
	job.EndTime = time.Now()
	job.TimeTaken = job.EndTime.Sub(job.StartTime)

	return s.repo.UpdateJob(ctx, job)
}

func (s *CrackingService) updateJobMetrics(jobID string, attempts int64) {
	s.metrics.Update(jobID, metrics.MetricData{
		Attempts:    attempts,
		Timestamp:   time.Now(),
		CPUUsage:    metrics.GetCPUUsage(),
		MemoryUsage: metrics.GetMemoryUsage(),
	})
}

func (s *CrackingService) analyzePassword(password string, startTime time.Time) domain.PasswordComplexity {
	return domain.PasswordComplexity{
		Score:       calculatePasswordScore(password),
		Entropy:     calculateEntropy(password),
		TimeToBreak: time.Since(startTime),
		Strength:    determineStrengthLevel(password),
	}
}
