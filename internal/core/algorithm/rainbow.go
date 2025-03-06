package algorithm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/utils/random"
	"sync"
	"sync/atomic"
)

type Rainbow struct {
	settings    domain.CrackingSettings
	progress    float64
	chainLength int
	tableSize   int
	mu          sync.RWMutex
	stop        chan struct{}
	chainTable  map[string]string
	attempts    int64
	hashType    domain.HashType
	targetHash  string
}

func NewRainbow() *Rainbow {
	return &Rainbow{
		stop:        make(chan struct{}),
		chainTable:  make(map[string]string),
		chainLength: 1000,
		tableSize:   10000,
	}
}

func (r *Rainbow) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		if r.targetHash == "" {
			errors <- domain.ErrInvalidHash
			return
		}

		if err := r.generateTableParallel(ctx); err != nil {
			errors <- err
			return
		}

		if found, password := r.searchPassword(ctx); found {
			select {
			case passwords <- password:
			case <-ctx.Done():
			case <-r.stop:
			}
		}
	}()

	return passwords, errors
}

func (r *Rainbow) generateTableParallel(ctx context.Context) error {
	workerCount := 4
	workChan := make(chan int, r.tableSize)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go r.tableWorker(ctx, &wg, workChan)
	}

	go func() {
		for i := 0; i < r.tableSize; i++ {
			select {
			case workChan <- i:
			case <-ctx.Done():
				return
			case <-r.stop:
				return
			}
		}
		close(workChan)
	}()

	wg.Wait()
	return nil
}

func (r *Rainbow) tableWorker(ctx context.Context, wg *sync.WaitGroup, work <-chan int) {
	defer wg.Done()

	charset := r.settings.CharacterSet
	if charset == "" {
		charset = domain.CharsetAll
	}

	for range work {
		startPoint := random.GenerateRandomString(charset, r.settings.MinLength)
		endpoint := r.generateChain(startPoint)

		r.mu.Lock()
		r.chainTable[endpoint] = startPoint
		r.mu.Unlock()

		atomic.AddInt64(&r.attempts, 1)
		r.updateProgress()

		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		default:
		}
	}
}

func (r *Rainbow) hash(input string) string {
	switch r.hashType {
	case domain.HashMD5:
		hash := md5.Sum([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashSHA1:
		hash := sha1.Sum([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashSHA256:
		hash := sha256.Sum256([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashSHA512:
		hash := sha512.Sum512([]byte(input))
		return hex.EncodeToString(hash[:])
	default:
		return input
	}
}

func (r *Rainbow) searchPassword(ctx context.Context) (bool, string) {
	endpoint := r.reduce(r.targetHash, r.chainLength-1)

	for i := r.chainLength - 1; i >= 0; i-- {
		atomic.AddInt64(&r.attempts, 1)

		if startPoint, exists := r.chainTable[endpoint]; exists {
			if candidate := r.reconstructChain(startPoint, i); candidate != "" {
				if r.hash(candidate) == r.targetHash {
					return true, candidate
				}
			}
		}
		endpoint = r.reduce(endpoint, i-1)
	}
	return false, ""
}

func (r *Rainbow) generateChain(startPoint string) string {
	current := startPoint
	for i := 0; i < r.chainLength; i++ {
		hash := r.hash(current)
		current = r.reduce(hash, i)
	}
	return current
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

func (r *Rainbow) updateProgress() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress = float64(atomic.LoadInt64(&r.attempts)) / float64(r.tableSize*r.chainLength) * 100
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

func (r *Rainbow) SetSettings(settings domain.CrackingSettings) {
	r.settings = settings
	r.hashType = settings.HashType
	r.targetHash = settings.TargetHash
}
