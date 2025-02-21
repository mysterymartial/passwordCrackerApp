package algorithm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/utils/random"
	"sync"
	"time"
)

type Rainbow struct {
	job         *domain.CrackingJob
	settings    domain.CrackingSettings
	progress    float64
	chainLength int
	tableSize   int
	mu          sync.RWMutex
	stop        chan struct{}
	chainTable  map[string]string
	startTime   time.Time
	attempts    int64
}

func NewRainbow() *Rainbow {
	return &Rainbow{
		stop:        make(chan struct{}),
		chainTable:  make(map[string]string),
		chainLength: 1000,
		tableSize:   10000,
	}
}

func (r *Rainbow) SetJob(job *domain.CrackingJob) {
	r.job = job
	r.settings = job.Settings
	r.startTime = time.Now()
}

func (r *Rainbow) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)
		defer r.updateMetrics()

		if r.job == nil || r.job.TargetHash == "" {
			errors <- domain.ErrInvalidHash
			return
		}

		r.generateTable(ctx)

		endpoint := r.reduce(r.job.TargetHash, r.chainLength-1)
		for i := r.chainLength - 1; i >= 0; i-- {
			r.attempts++
			if startPoint, exists := r.chainTable[endpoint]; exists {
				candidate := r.reconstructChain(startPoint, i)
				hash := r.hash(candidate)
				if hash == r.job.TargetHash {
					r.job.FoundPassword = candidate
					r.job.EndTime = time.Now()
					select {
					case passwords <- candidate:
					case <-ctx.Done():
						return
					case <-r.stop:
						return
					}
				}
			}
			endpoint = r.reduce(endpoint, i-1)
		}
	}()

	return passwords, errors
}

func (r *Rainbow) generateTable(ctx context.Context) {
	charset := r.settings.CharacterSet
	if charset == "" {
		charset = domain.CharsetAll
	}

	for i := 0; i < r.tableSize; i++ {
		startPoint := random.GenerateRandomString(charset, r.settings.MinLength)
		endpoint := r.generateChain(startPoint)
		r.chainTable[endpoint] = startPoint

		r.mu.Lock()
		r.progress = float64(i) / float64(r.tableSize)
		r.attempts++
		r.mu.Unlock()

		r.updateMetrics()

		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		default:
		}
	}
}

func (r *Rainbow) generateChain(startPoint string) string {
	current := startPoint
	for i := 0; i < r.chainLength; i++ {
		hash := r.hash(current)
		current = r.reduce(hash, i)
	}
	return current
}

func (r *Rainbow) hash(input string) string {
	switch r.job.HashType {
	case domain.HashMD5:
		hash := md5.Sum([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashSHA1:
		hash := sha1.Sum([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashSHA256:
		hash := sha256.Sum256([]byte(input))
		return hex.EncodeToString(hash[:])
	default:
		return input
	}
}

func (r *Rainbow) reduce(hash string, position int) string {
	result := []byte(hash)
	length := r.settings.MinLength
	if length > len(result) {
		length = len(result)
	}

	charset := r.settings.CharacterSet
	if charset == "" {
		charset = domain.CharsetAll
	}

	for i := 0; i < length; i++ {
		index := (int(result[i]) + position) % len(charset)
		result[i] = charset[index]
	}

	return string(result[:length])
}

func (r *Rainbow) reconstructChain(startPoint string, position int) string {
	current := startPoint
	for i := 0; i < position; i++ {
		hash := r.hash(current)
		current = r.reduce(hash, i)
	}
	return current
}

func (r *Rainbow) updateMetrics() {
	if r.job != nil {
		duration := time.Since(r.startTime).Seconds()
		r.job.ResourceMetrics.AttemptsPerSec = int64(float64(r.attempts) / duration)
		r.job.ResourceMetrics.TotalAttempts = r.attempts
		r.job.ResourceMetrics.LastUpdated = time.Now()
		r.job.AttemptCount = r.attempts
		r.job.LastAttempt = time.Now()
		r.job.Progress = r.progress
	}
}

func (r *Rainbow) Stop() {
	close(r.stop)
}

func (r *Rainbow) Progress() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.progress
}

func (r *Rainbow) Name() domain.CrackingAlgorithm {
	return domain.AlgoRainbow
}
