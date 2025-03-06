package algorithm

import (
	"context"
	"os"
	"passwordCrakerBackend/internal/core/domain"
	"sort"
	"testing"
	"time"
)

func TestHybrid_Start(t *testing.T) {
	tests := []struct {
		name     string
		settings domain.CrackingSettings
		want     []string
	}{
		{
			name: "Basic hybrid test",
			settings: domain.CrackingSettings{
				MinLength:    2,
				MaxLength:    3,
				CharacterSet: "ab",
				WordlistPaths: []string{
					createTestWordlist(t, []string{"aa", "ab", "ba"}),
				},
			},
			want: []string{"aa", "ab", "ba", "bb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHybrid()
			h.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			passwords, errors := h.Start(ctx)

			var results []string
			resultsChan := make(chan struct{})

			go func() {
				for password := range passwords {
					results = append(results, password)
				}
				close(resultsChan)
			}()

			select {
			case err := <-errors:
				t.Fatalf("Unexpected error: %v", err)
			case <-resultsChan:
				// Success case
			case <-ctx.Done():
				t.Fatal("Test timed out")
			}

			// Remove duplicates and sort results
			uniqueResults := removeDuplicates(results)
			sort.Strings(uniqueResults)
			sort.Strings(tt.want)

			if !compareSlices(uniqueResults, tt.want) {
				t.Errorf("Got %v, want %v", uniqueResults, tt.want)
			}
		})
	}
}

func TestHybrid_Progress(t *testing.T) {
	h := NewHybrid()
	h.SetSettings(domain.CrackingSettings{
		MinLength:    2,
		MaxLength:    2,
		CharacterSet: "ab",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _ = h.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	progress := h.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestHybrid_Stop(t *testing.T) {
	h := NewHybrid()
	h.SetSettings(domain.CrackingSettings{
		MinLength:    2,
		MaxLength:    3,
		CharacterSet: "ab",
	})

	ctx := context.Background()
	passwords, _ := h.Start(ctx)

	done := make(chan struct{})
	go func() {
		for range passwords {
		}
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	h.Stop()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}

func TestHybrid_Name(t *testing.T) {
	h := NewHybrid()
	if h.Name() != domain.AlgoHybrid {
		t.Errorf("Name() = %v, want %v", h.Name(), domain.AlgoHybrid)
	}
}

func TestHybrid_SetSettings(t *testing.T) {
	h := NewHybrid()
	settings := domain.CrackingSettings{
		MinLength:    2,
		MaxLength:    3,
		CharacterSet: "ab",
	}
	h.SetSettings(settings)

	for _, alg := range h.algorithms {
		switch a := alg.(type) {
		case *BruteForce:
			if a.settings.MinLength != settings.MinLength ||
				a.settings.MaxLength != settings.MaxLength ||
				a.settings.CharacterSet != settings.CharacterSet {
				t.Error("Settings not properly propagated to BruteForce algorithm")
			}
		}
	}
}

// Helper functions
func createTestWordlist(t *testing.T, words []string) string {
	file := t.TempDir() + "/wordlist.txt"
	writeWordlistToFile(t, file, words)
	return file
}

func writeWordlistToFile(t *testing.T, path string, words []string) {
	content := ""
	for _, word := range words {
		content += word + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
