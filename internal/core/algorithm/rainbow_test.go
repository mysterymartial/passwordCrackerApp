package algorithm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"passwordCrakerBackend/internal/core/domain"
	"testing"
	"time"
)

func TestRainbow_Start(t *testing.T) {
	tests := []struct {
		name      string
		settings  domain.CrackingSettings
		wantFound bool
	}{
		{
			name: "MD5 hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashMD5),
				HashType:     domain.HashMD5,
			},
			wantFound: true,
		},
		{
			name: "SHA1 hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashSHA1),
				HashType:     domain.HashSHA1,
			},
			wantFound: true,
		},
		{
			name: "Empty hash",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   "",
				HashType:     domain.HashMD5,
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRainbow()
			r.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			passwords, errors := r.Start(ctx)

			var found bool
			var err error

			select {
			case password := <-passwords:
				found = r.hash(password) == tt.settings.TargetHash
			case err = <-errors:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			if tt.wantFound && !found {
				t.Errorf("Expected to find password but didn't")
			}
			if !tt.wantFound && err == nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestRainbow_Progress(t *testing.T) {
	r := NewRainbow()
	settings := domain.CrackingSettings{
		MinLength:    3,
		MaxLength:    3,
		CharacterSet: "abc123",
		TargetHash:   createHash("abc", domain.HashMD5),
		HashType:     domain.HashMD5,
	}
	r.SetSettings(settings)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _ = r.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	progress := r.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestRainbow_Stop(t *testing.T) {
	r := NewRainbow()
	settings := domain.CrackingSettings{
		MinLength:    3,
		MaxLength:    3,
		CharacterSet: "abc123",
		TargetHash:   createHash("abc", domain.HashMD5),
		HashType:     domain.HashMD5,
	}
	r.SetSettings(settings)

	ctx := context.Background()
	passwords, _ := r.Start(ctx)

	done := make(chan struct{})
	go func() {
		for range passwords {
		}
		close(done)
	}()

	r.Stop()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}

func TestRainbow_Name(t *testing.T) {
	r := NewRainbow()
	if r.Name() != domain.AlgoRainbow {
		t.Errorf("Name() = %v, want %v", r.Name(), domain.AlgoRainbow)
	}
}

func createHash(input string, hashType domain.HashType) string {
	switch hashType {
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
