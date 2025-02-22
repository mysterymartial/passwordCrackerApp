package metrics

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"passwordCrakerBackend/internal/core/domain"
	"runtime"
	"sync"
	"time"
)

type Collector struct {
	mu             sync.RWMutex
	metrics        map[string]*domain.ResourceMetrics
	updateInterval time.Duration
}

func NewCollector(interval time.Duration) *Collector {
	return &Collector{
		metrics:        make(map[string]*domain.ResourceMetrics),
		updateInterval: interval,
	}
}

func (c *Collector) StartCollection(jobID string) {
	c.mu.Lock()
	c.metrics[jobID] = &domain.ResourceMetrics{
		LastUpdated: time.Now(),
	}
	c.mu.Unlock()

	go c.collect(jobID)
}

func (c *Collector) StopCollection(jobID string) {
	c.mu.Lock()
	delete(c.metrics, jobID)
	c.mu.Unlock()
}

func (c *Collector) GetMetrics(jobID string) *domain.ResourceMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if metrics, exists := c.metrics[jobID]; exists {
		return metrics
	}
	return nil
}

func (c *Collector) collect(jobID string) {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()

	for {
		c.mu.RLock()
		if _, exists := c.metrics[jobID]; !exists {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		cpuUsage, _ := cpu.Percent(time.Second, false)
		_, _ = mem.VirtualMemory()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		c.mu.Lock()
		c.metrics[jobID].CPUUsage = cpuUsage[0]
		c.metrics[jobID].MemoryUsageMB = int64(m.Alloc / 1024 / 1024)
		c.metrics[jobID].LastUpdated = time.Now()
		c.mu.Unlock()

		<-ticker.C
	}
}

func (c *Collector) UpdateAttempts(jobID string, attempts, activeThreads int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if metrics, exists := c.metrics[jobID]; exists {
		metrics.TotalAttempts = attempts
		metrics.ActiveThreads = int(activeThreads)
		metrics.AttemptsPerSec = attempts / int64(time.Since(metrics.LastUpdated).Seconds())
	}
}
