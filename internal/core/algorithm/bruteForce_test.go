package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"sort"
	"testing"
	"time"
)

func TestBruteForce_Start(t *testing.T) {
	tests := []struct {
		name     string
		settings domain.CrackingSettings
		want     []string
	}{
		{
			name: "Single character lowercase",
			settings: domain.CrackingSettings{
				MinLength:    1,
				MaxLength:    1,
				CharacterSet: "ab",
			},
			want: []string{"a", "b"},
		},
		{
			name: "Two character digits",
			settings: domain.CrackingSettings{
				MinLength:    2,
				MaxLength:    2,
				CharacterSet: "12",
			},
			want: []string{"11", "12", "21", "22"},
		},
		{
			name: "Variable length passwords",
			settings: domain.CrackingSettings{
				MinLength:    1,
				MaxLength:    2,
				CharacterSet: "ab",
			},
			want: []string{"a", "b", "aa", "ab", "ba", "bb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBruteForce()
			b.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			passwords, errors := b.Start(ctx)

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
				t.Fatalf("Unexpected error: %v", err)
			case <-done:
			case <-ctx.Done():
				t.Fatal("Test timed out")
			}

			sort.Strings(results)
			sort.Strings(tt.want)

			if len(results) != len(tt.want) {
				t.Errorf("Got %d passwords, want %d", len(results), len(tt.want))
			}

			for i := range results {
				if results[i] != tt.want[i] {
					t.Errorf("Password at position %d: got %s, want %s", i, results[i], tt.want[i])
				}
			}
		})
	}
}

func TestBruteForce_Progress(t *testing.T) {
	b := NewBruteForce()
	b.SetSettings(domain.CrackingSettings{
		MinLength:    1,
		MaxLength:    3,
		CharacterSet: "abc",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _ = b.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	progress := b.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestBruteForce_Stop(t *testing.T) {
	b := NewBruteForce()
	b.SetSettings(domain.CrackingSettings{
		MinLength:    1,
		MaxLength:    3,
		CharacterSet: "abc",
	})

	ctx := context.Background()
	passwords, _ := b.Start(ctx)

	done := make(chan struct{})
	go func() {
		for range passwords {
		}
		close(done)
	}()

	b.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}

func TestBruteForce_Name(t *testing.T) {
	b := NewBruteForce()
	if b.Name() != domain.AlgoBruteForce {
		t.Errorf("Name() = %v, want %v", b.Name(), domain.AlgoBruteForce)
	}
}

func TestBruteForce_DefaultCharset(t *testing.T) {
	b := NewBruteForce()
	b.SetSettings(domain.CrackingSettings{
		MinLength: 1,
		MaxLength: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	passwords, _ := b.Start(ctx)

	var results []string
	for password := range passwords {
		results = append(results, password)
	}

	expectedLen := len(domain.CharsetAll)
	if len(results) != expectedLen {
		t.Errorf("Got %d passwords with default charset, want %d", len(results), expectedLen)
	}
}

func TestBruteForce_CombinationsCalculation(t *testing.T) {
	b := NewBruteForce()
	settings := domain.CrackingSettings{
		MinLength:    1,
		MaxLength:    2,
		CharacterSet: "ab",
	}
	b.SetSettings(settings)

	b.calculateTotalCombinations(settings.CharacterSet)
	expectedCombinations := int64(6) // a,b + aa,ab,ba,bb = 6

	if b.combinations != expectedCombinations {
		t.Errorf("calculateTotalCombinations() = %v, want %v", b.combinations, expectedCombinations)
	}
}
