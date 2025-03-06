package algorithm

import (
	"context"
	"os"
	"passwordCrakerBackend/internal/core/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestMarkov_Start(t *testing.T) {
	// Create test wordlists
	tempDir := t.TempDir()
	wordlist1 := createTestWordlist2(t, tempDir, "wordlist1.txt", []string{
		"password123",
		"admin123",
		"letmein123",
		"welcome123",
	})

	tests := []struct {
		name     string
		settings domain.CrackingSettings
		wantLen  int
	}{
		{
			name: "Basic Markov chain generation",
			settings: domain.CrackingSettings{
				MinLength:     6,
				MaxLength:     10,
				WordlistPaths: []string{wordlist1},
			},
			wantLen: 4, // Expect at least some generated passwords
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMarkov()
			m.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			passwords, errors := m.Start(ctx)

			var results []string
			var err error

			done := make(chan struct{})
			go func() {
				defer close(done)
				for password := range passwords {
					results = append(results, password)
				}
			}()

			select {
			case err = <-errors:
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			case <-done:
			case <-ctx.Done():
				t.Fatal("Test timed out")
			}

			validatePasswords(t, results, tt.settings.MinLength, tt.settings.MaxLength)
		})
	}
}

func TestMarkov_TrainModel(t *testing.T) {
	tempDir := t.TempDir()
	wordlist := createTestWordlist2(t, tempDir, "train.txt", []string{
		"password",
		"passw0rd",
		"p@ssword",
	})

	m := NewMarkov()
	m.SetSettings(domain.CrackingSettings{
		WordlistPaths: []string{wordlist},
	})

	ctx := context.Background()
	err := m.trainModel(ctx)

	if err != nil {
		t.Fatalf("trainModel() error = %v", err)
	}

	if len(m.chain.transitions) == 0 {
		t.Error("Expected non-empty transitions map after training")
	}

	if len(m.chain.initial) == 0 {
		t.Error("Expected non-empty initial states map after training")
	}
}

func TestMarkov_GeneratePassword(t *testing.T) {
	m := NewMarkov()
	m.SetSettings(domain.CrackingSettings{
		MinLength: 6,
		MaxLength: 12,
	})

	// Setup a simple chain for testing
	m.chain.transitions = map[string]map[string]float64{
		"pas": {"s": 1.0},
		"ass": {"w": 1.0},
		"ssw": {"o": 1.0},
		"swo": {"r": 1.0},
		"wor": {"d": 1.0},
	}

	password := m.generatePassword("pas")
	if len(password) < m.settings.MinLength || len(password) > m.settings.MaxLength {
		t.Errorf("Generated password length %d not within bounds [%d, %d]",
			len(password), m.settings.MinLength, m.settings.MaxLength)
	}
}

func TestMarkov_Stop(t *testing.T) {
	tempDir := t.TempDir()
	wordlist := createTestWordlist2(t, tempDir, "stop.txt", []string{
		"password123",
		"admin123",
	})

	m := NewMarkov()
	m.SetSettings(domain.CrackingSettings{
		WordlistPaths: []string{wordlist},
	})

	ctx := context.Background()
	passwords, _ := m.Start(ctx)

	done := make(chan struct{})
	go func() {
		for range passwords {
		}
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	m.Stop()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}

func TestMarkov_Progress(t *testing.T) {
	m := NewMarkov()
	m.SetSettings(domain.CrackingSettings{
		MinLength: 6,
		MaxLength: 10,
	})

	progress := m.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestMarkov_Name(t *testing.T) {
	m := NewMarkov()
	if m.Name() != domain.AlgoMarkov {
		t.Errorf("Name() = %v, want %v", m.Name(), domain.AlgoMarkov)
	}
}

// Helper functions
func createTestWordlist2(t *testing.T, dir, name string, words []string) string {
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	for _, word := range words {
		if _, err := file.WriteString(word + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func validatePasswords(t *testing.T, passwords []string, minLen, maxLen int) {
	for _, password := range passwords {
		if len(password) < minLen || len(password) > maxLen {
			t.Errorf("Password %q length %d not within bounds [%d, %d]",
				password, len(password), minLen, maxLen)
		}
	}
}
