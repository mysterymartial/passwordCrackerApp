package algorithm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/pbkdf2" // For WPA
	"log"
	"passwordCrakerBackend/internal/core/domain"
	"sync"
	"sync/atomic"
	"unicode/utf16"
)

type Rainbow struct {
	settings      domain.CrackingSettings
	progress      float64
	mu            sync.RWMutex
	stop          chan struct{}
	totalTried    uint64
	totalPossible uint64
}

func NewRainbow() *Rainbow {
	return &Rainbow{
		stop: make(chan struct{}),
	}
}

func (r *Rainbow) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string, 1)
	errors := make(chan error, 1)

	go func() {
		defer close(passwords)
		defer close(errors)

		if r.settings.TargetHash == "" {
			errors <- fmt.Errorf("empty target hash")
			return
		}

		if r.settings.CharacterSet == "" || r.settings.MinLength == 0 {
			errors <- fmt.Errorf("invalid settings: empty charset or zero length")
			return
		}

		charsetLen := uint64(len(r.settings.CharacterSet))
		r.totalPossible = 0
		for l := r.settings.MinLength; l <= r.settings.MaxLength; l++ {
			r.totalPossible += pow(charsetLen, uint64(l))
		}

		for length := r.settings.MinLength; length <= r.settings.MaxLength; length++ {
			if err := r.generateCombinations(ctx, length, []byte{}, passwords); err != nil {
				errors <- err
				return
			}
		}
	}()

	return passwords, errors
}

func (r *Rainbow) generateCombinations(ctx context.Context, length int, prefix []byte, passwords chan<- string) error {
	if len(prefix) == length {
		password := string(prefix)
		atomic.AddUint64(&r.totalTried, 1)
		r.updateProgress()

		// Handle hash-specific matching
		switch r.settings.HashType {
		case domain.HashBCRYPT:
			if err := bcrypt.CompareHashAndPassword([]byte(r.settings.TargetHash), []byte(password)); err == nil {
				log.Printf("Found password: %s, bcrypt match", password)
				return r.sendPassword(ctx, passwords, password)
			}
		case domain.HashArgon2:
			// Assume target hash is raw Argon2 output; simplistic comparison
			hash := argon2.IDKey([]byte(password), []byte("somesalt"), 1, 64*1024, 4, 32)
			if hex.EncodeToString(hash) == r.settings.TargetHash {
				log.Printf("Found password: %s, argon2 match", password)
				return r.sendPassword(ctx, passwords, password)
			}
		case domain.HashNTLM:
			if r.ntlmHash(password) == r.settings.TargetHash {
				log.Printf("Found password: %s, ntlm match", password)
				return r.sendPassword(ctx, passwords, password)
			}
		case domain.HashWPA:
			// Simplified WPA (PBKDF2-SHA1); assumes SSID "testssid" and target is raw hash
			hash := pbkdf2.Key([]byte(password), []byte("testssid"), 4096, 32, sha1.New)
			if hex.EncodeToString(hash) == r.settings.TargetHash {
				log.Printf("Found password: %s, wpa match", password)
				return r.sendPassword(ctx, passwords, password)
			}
		default:
			// MD5, SHA1, SHA256, SHA512
			if r.hash(password) == r.settings.TargetHash {
				log.Printf("Found password: %s, hash: %s", password, r.hash(password))
				return r.sendPassword(ctx, passwords, password)
			}
		}
		return nil
	}

	for _, char := range r.settings.CharacterSet {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stop:
			return nil
		default:
			if err := r.generateCombinations(ctx, length, append(prefix, byte(char)), passwords); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Rainbow) sendPassword(ctx context.Context, passwords chan<- string, password string) error {
	select {
	case passwords <- password:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.stop:
		return nil
	}
}

func (r *Rainbow) hash(input string) string {
	switch r.settings.HashType {
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
		return input // Fallback for unsupported hashes
	}
}

func (r *Rainbow) ntlmHash(input string) string {
	// NTLM uses MD4 over UTF-16LE encoded password
	utf16le := utf16.Encode([]rune(input))
	bytes := make([]byte, len(utf16le)*2)
	for i, c := range utf16le {
		bytes[i*2] = byte(c)
		bytes[i*2+1] = byte(c >> 8)
	}
	hash := md4.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil))
}

func (r *Rainbow) updateProgress() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress = float64(r.totalTried) / float64(r.totalPossible) * 100
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
}

func pow(base, exp uint64) uint64 {
	result := uint64(1)
	for i := uint64(0); i < exp; i++ {
		result *= base
	}
	return result
}
