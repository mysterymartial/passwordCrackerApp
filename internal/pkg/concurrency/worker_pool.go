package concurrency

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"sync"
	"time"
)

type WorkerPool struct {
	workers    []*Worker
	tasks      chan Task
	results    chan Result
	numWorkers int
	metrics    *PoolMetrics
	wg         sync.WaitGroup
	mu         sync.RWMutex
	stop       chan struct{}
}

type Worker struct {
	id        int
	tasks     chan Task
	results   chan Result
	metrics   *WorkerMetrics
	isWorking bool
	mu        sync.RWMutex
}

type Task struct {
	ID       string
	JobID    string
	Function func() (string, error)
	Timeout  time.Duration
}

type Result struct {
	TaskID   string
	JobID    string
	Value    string
	Error    error
	Duration time.Duration
	WorkerID int
}

type PoolMetrics struct {
	ActiveWorkers  int
	CompletedTasks int64
	FailedTasks    int64
	TotalDuration  time.Duration
	AverageLatency time.Duration
	mu             sync.RWMutex
}

type WorkerMetrics struct {
	TasksCompleted int64
	TasksFailed    int64
	TotalDuration  time.Duration
	LastActive     time.Time
	mu             sync.RWMutex
}

func NewWorkerPool(numWorkers int, queueSize int) *WorkerPool {
	pool := &WorkerPool{
		workers:    make([]*Worker, numWorkers),
		tasks:      make(chan Task, queueSize),
		results:    make(chan Result, queueSize),
		numWorkers: numWorkers,
		metrics:    &PoolMetrics{},
		stop:       make(chan struct{}),
	}

	for i := 0; i < numWorkers; i++ {
		pool.workers[i] = &Worker{
			id:      i,
			tasks:   pool.tasks,
			results: pool.results,
			metrics: &WorkerMetrics{
				LastActive: time.Now(),
			},
		}
	}

	return pool
}

func (p *WorkerPool) Start(ctx context.Context) {
	for _, worker := range p.workers {
		p.wg.Add(1)
		go worker.start(ctx, &p.wg)
	}

	go p.collectMetrics(ctx)
}

func (p *WorkerPool) Submit(task Task) {
	p.tasks <- task
}

func (p *WorkerPool) Results() <-chan Result {
	return p.results
}

func (p *WorkerPool) Stop() {
	close(p.stop)
	close(p.tasks)
	p.wg.Wait()
	close(p.results)
}

func (p *WorkerPool) GetMetrics() domain.ResourceMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return domain.ResourceMetrics{
		ActiveThreads:  p.metrics.ActiveWorkers,
		AttemptsPerSec: p.calculateAttemptsPerSecond(),
		TotalAttempts:  p.metrics.CompletedTasks,
		LastUpdated:    time.Now(),
	}
}

func (w *Worker) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}

			w.mu.Lock()
			w.isWorking = true
			w.mu.Unlock()

			startTime := time.Now()

			taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
			value, err := w.executeTask(taskCtx, task)
			cancel()

			duration := time.Since(startTime)

			w.updateMetrics(err == nil, duration)

			w.results <- Result{
				TaskID:   task.ID,
				JobID:    task.JobID,
				Value:    value,
				Error:    err,
				Duration: duration,
				WorkerID: w.id,
			}

			w.mu.Lock()
			w.isWorking = false
			w.mu.Unlock()
		}
	}
}

func (w *Worker) executeTask(ctx context.Context, task Task) (string, error) {
	resultCh := make(chan string)
	errCh := make(chan error)

	go func() {
		value, err := task.Function()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- value
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errCh:
		return "", err
	case result := <-resultCh:
		return result, nil
	}
}

func (w *Worker) updateMetrics(success bool, duration time.Duration) {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()

	if success {
		w.metrics.TasksCompleted++
	} else {
		w.metrics.TasksFailed++
	}
	w.metrics.TotalDuration += duration
	w.metrics.LastActive = time.Now()
}

func (p *WorkerPool) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		case <-ticker.C:
			p.updatePoolMetrics()
		}
	}
}

func (p *WorkerPool) updatePoolMetrics() {
	activeWorkers := 0
	var totalCompleted, totalFailed int64
	var totalDuration time.Duration

	for _, worker := range p.workers {
		worker.metrics.mu.RLock()
		totalCompleted += worker.metrics.TasksCompleted
		totalFailed += worker.metrics.TasksFailed
		totalDuration += worker.metrics.TotalDuration
		worker.metrics.mu.RUnlock()

		worker.mu.RLock()
		if worker.isWorking {
			activeWorkers++
		}
		worker.mu.RUnlock()
	}

	p.metrics.mu.Lock()
	p.metrics.ActiveWorkers = activeWorkers
	p.metrics.CompletedTasks = totalCompleted
	p.metrics.FailedTasks = totalFailed
	p.metrics.TotalDuration = totalDuration
	if totalCompleted > 0 {
		p.metrics.AverageLatency = totalDuration / time.Duration(totalCompleted)
	}
	p.metrics.mu.Unlock()
}

func (p *WorkerPool) calculateAttemptsPerSecond() int64 {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	if p.metrics.TotalDuration == 0 {
		return 0
	}

	return int64(float64(p.metrics.CompletedTasks) / p.metrics.TotalDuration.Seconds())
}
