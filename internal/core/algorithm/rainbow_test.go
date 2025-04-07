package algorithm

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/pbkdf2"
	"passwordCrakerBackend/internal/core/domain"
	"testing"
	"time"
	"unicode/utf16"
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
			name: "SHA256 hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashSHA256),
				HashType:     domain.HashSHA256,
			},
			wantFound: true,
		},
		{
			name: "SHA512 hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashSHA512),
				HashType:     domain.HashSHA512,
			},
			wantFound: true,
		},
		{
			name: "BCRYPT hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashBCRYPT),
				HashType:     domain.HashBCRYPT,
			},
			wantFound: true,
		},
		{
			name: "ARGON2 hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashArgon2),
				HashType:     domain.HashArgon2,
			},
			wantFound: true,
		},
		{
			name: "NTLM hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashNTLM),
				HashType:     domain.HashNTLM,
			},
			wantFound: true,
		},
		{
			name: "WPA hash crack",
			settings: domain.CrackingSettings{
				MinLength:    3,
				MaxLength:    3,
				CharacterSet: "abc123",
				TargetHash:   createHash("abc", domain.HashWPA),
				HashType:     domain.HashWPA,
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
				switch tt.settings.HashType {
				case domain.HashBCRYPT:
					found = bcrypt.CompareHashAndPassword([]byte(tt.settings.TargetHash), []byte(password)) == nil
				case domain.HashArgon2:
					hash := argon2.IDKey([]byte(password), []byte("somesalt"), 1, 64*1024, 4, 32)
					found = hex.EncodeToString(hash) == tt.settings.TargetHash
				case domain.HashNTLM:
					found = r.ntlmHash(password) == tt.settings.TargetHash
				case domain.HashWPA:
					hash := pbkdf2.Key([]byte(password), []byte("testssid"), 4096, 32, sha1.New)
					found = hex.EncodeToString(hash) == tt.settings.TargetHash
				default:
					found = r.hash(password) == tt.settings.TargetHash
				}
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
	case domain.HashSHA512:
		hash := sha512.Sum512([]byte(input))
		return hex.EncodeToString(hash[:])
	case domain.HashBCRYPT:
		hash, _ := bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
		return string(hash)
	case domain.HashArgon2:
		hash := argon2.IDKey([]byte(input), []byte("somesalt"), 1, 64*1024, 4, 32)
		return hex.EncodeToString(hash)
	case domain.HashNTLM:
		utf16le := utf16.Encode([]rune(input))
		bytes := make([]byte, len(utf16le)*2)
		for i, c := range utf16le {
			bytes[i*2] = byte(c)
			bytes[i*2+1] = byte(c >> 8)
		}
		hash := md4.New()
		hash.Write(bytes)
		return hex.EncodeToString(hash.Sum(nil))
	case domain.HashWPA:
		hash := pbkdf2.Key([]byte(input), []byte("testssid"), 4096, 32, sha1.New)
		return hex.EncodeToString(hash)
	default:
		return input
	}
}
