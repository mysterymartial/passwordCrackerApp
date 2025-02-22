package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Reporter struct {
	mu      sync.Mutex
	logFile *os.File
	metrics map[string][]interface{}
}

func NewReporter(logPath string) (*Reporter, error) {
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		logFile: file,
		metrics: make(map[string][]interface{}),
	}, nil
}

func (r *Reporter) Record(category string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now(),
		"data":      data,
	}

	r.metrics[category] = append(r.metrics[category], entry)
}

func (r *Reporter) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(r.metrics, "", "  ")
	if err != nil {
		return err
	}

	if _, err := r.logFile.Write(append(data, '\n')); err != nil {
		return err
	}

	r.metrics = make(map[string][]interface{})
	return nil
}

func (r *Reporter) Close() error {
	if err := r.Flush(); err != nil {
		return fmt.Errorf("failed to flush metrics: %w", err)
	}
	return r.logFile.Close()
}
