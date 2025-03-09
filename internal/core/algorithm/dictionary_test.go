package algorithm

import (
	"context"
	"os"
	"passwordCrakerBackend/internal/core/domain"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestDictionary_Start(t *testing.T) {
	tempDir := t.TempDir()
	wordlist1 := createTempWordlist(t, tempDir, "wordlist1.txt", []string{"password", "admin", "root"})
	wordlist2 := createTempWordlist(t, tempDir, "wordlist2.txt", []string{"test123", "letmein"})

	tests := []struct {
		name     string
		settings domain.CrackingSettings
		want     []string
		wantErr  bool
	}{
		{
			name: "Basic wordlist",
			settings: domain.CrackingSettings{
				MinLength:     3,
				MaxLength:     8,
				WordlistPaths: []string{wordlist1},
			},
			want:    []string{"password", "admin", "root"},
			wantErr: false,
		},
		{
			name: "Multiple wordlists",
			settings: domain.CrackingSettings{
				MinLength:     3,
				MaxLength:     8,
				WordlistPaths: []string{wordlist1, wordlist2},
			},
			want:    []string{"password", "admin", "root", "test123", "letmein"},
			wantErr: false,
		},
		{
			name: "With rules",
			settings: domain.CrackingSettings{
				MinLength:     3,
				MaxLength:     8,
				WordlistPaths: []string{wordlist1},
				CustomRules:   []string{"uppercase", "reverse"},
			},
			want:    []string{"password", "PASSWORD", "drowssap", "admin", "ADMIN", "nimda", "root", "ROOT", "toor"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDictionary()
			d.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			passwords, errors := d.Start(ctx)

			var results []string
			done := make(chan struct{})

			go func() {
				defer close(done)
				for password := range passwords {
					results = append(results, password)
				}
			}()

			select {
			case err := <-errors:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			case <-done:
				// Success case
			case <-ctx.Done():
				t.Fatal("timeout waiting for password generation")
			}

			sort.Strings(results)
			sort.Strings(tt.want)

			if !reflect.DeepEqual(results, tt.want) {
				t.Errorf("got passwords = %v, want %v", results, tt.want)
			}
		})
	}
}

func TestDictionary_Rules(t *testing.T) {
	tests := []struct {
		name     string
		word     string
		rule     string
		expected string
	}{
		{"Uppercase", "password", "uppercase", "PASSWORD"},
		{"Capitalize", "password", "capitalize", "Password"},
		{"Reverse", "password", "reverse", "drowssap"},
		{"Leet", "password", "leet", "p455w0rd"},
		{"Append Numbers", "pass", "append_numbers", "pass0123456789"},
		{"Unknown Rule", "password", "unknown", "password"},
	}

	d := NewDictionary()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.applyRule(tt.word, tt.rule)
			if result != tt.expected {
				t.Errorf("applyRule(%q, %q) = %q, want %q", tt.word, tt.rule, result, tt.expected)
			}
		})
	}
}

func TestDictionary_Progress(t *testing.T) {
	tempDir := t.TempDir()
	wordlist := createTempWordlist(t, tempDir, "progress.txt", []string{"one", "two", "three"})

	d := NewDictionary()
	d.SetSettings(domain.CrackingSettings{
		WordlistPaths: []string{wordlist},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _ = d.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	progress := d.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestDictionary_Stop(t *testing.T) {
	tempDir := t.TempDir()
	wordlist := createTempWordlist(t, tempDir, "stop.txt", []string{"password1", "password2"})

	d := NewDictionary()
	d.SetSettings(domain.CrackingSettings{
		WordlistPaths: []string{wordlist},
	})

	ctx := context.Background()
	passwords, _ := d.Start(ctx)

	done := make(chan struct{})
	go func() {
		for range passwords {
		}
		close(done)
	}()

	d.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}

func TestDictionary_Name(t *testing.T) {
	d := NewDictionary()
	if d.Name() != domain.AlgoDictionary {
		t.Errorf("Name() = %v, want %v", d.Name(), domain.AlgoDictionary)
	}
}

// Helper functions
func createTempWordlist(t *testing.T, dir, name string, words []string) string {
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

func checkResults(t *testing.T, got, want []string) {
	found := make(map[string]bool)
	for _, w := range want {
		found[w] = false
	}

	for _, g := range got {
		if _, ok := found[g]; !ok {
			t.Errorf("Unexpected password generated: %s", g)
		}
		found[g] = true
	}

	for w, f := range found {
		if !f {
			t.Errorf("Expected password not generated: %s", w)
		}
	}
}
