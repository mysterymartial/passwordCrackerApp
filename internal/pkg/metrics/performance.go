package metrics

import (
	"runtime"
	"time"
)

type PerformanceMetrics struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	MemoryUsage  uint64
	AllocObjects uint64
	GCCycles     uint32
}

func CapturePerformance(fn func()) *PerformanceMetrics {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	startAlloc := stats.TotalAlloc
	startGC := stats.NumGC

	metrics := &PerformanceMetrics{
		StartTime: time.Now(),
	}

	fn()

	runtime.ReadMemStats(&stats)
	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	metrics.MemoryUsage = stats.TotalAlloc - startAlloc
	metrics.AllocObjects = stats.Mallocs - stats.Frees
	metrics.GCCycles = stats.NumGC - startGC

	return metrics
}
