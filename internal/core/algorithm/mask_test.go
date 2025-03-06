package algorithm

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"testing"
	"time"
)

func TestMask_Start(t *testing.T) {
	tests := []struct {
		name     string
		settings domain.CrackingSettings
		want     []string
		wantErr  bool
	}{
		{
			name: "Simple digit mask",
			settings: domain.CrackingSettings{
				MinLength:   3,
				MaxLength:   3,
				CustomRules: []string{"?d?d?d"},
			},
			want:    []string{"000", "001", "002"},
			wantErr: false,
		},
		{
			name: "Mixed charset mask",
			settings: domain.CrackingSettings{
				MinLength:   2,
				MaxLength:   2,
				CustomRules: []string{"?l?d"},
			},
			want:    []string{"a0", "a1", "b0", "b1"},
			wantErr: false,
		},
		{
			name: "Custom pattern syntax",
			settings: domain.CrackingSettings{
				MinLength:   3,
				MaxLength:   3,
				CustomRules: []string{"[lower]{1}[digits]{2}"},
			},
			want:    []string{"a00", "a01", "b00", "b01"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMask()
			m.SetSettings(tt.settings)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			passwords, errors := m.Start(ctx)

			// Collect results
			var results []string
			var err error

			done := make(chan struct{})
			go func() {
				defer close(done)
				for pass := range passwords {
					results = append(results, pass)
					if len(results) >= len(tt.want) {
						break
					}
				}
			}()

			select {
			case err = <-errors:
			case <-done:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Mask.Start() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify results contain expected passwords
			for _, want := range tt.want {
				found := false
				for _, got := range results {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Mask.Start() missing expected password %v", want)
				}
			}
		})
	}
}

func TestMask_Progress(t *testing.T) {
	m := NewMask()
	m.SetSettings(domain.CrackingSettings{
		MinLength:   3,
		MaxLength:   3,
		CustomRules: []string{"?d?d?d", "?l?l?l"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, _ = m.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	progress := m.Progress()
	if progress < 0 || progress > 100 {
		t.Errorf("Progress() = %v, want between 0 and 100", progress)
	}
}

func TestMask_Stop(t *testing.T) {
	m := NewMask()
	m.SetSettings(domain.CrackingSettings{
		MinLength:   3,
		MaxLength:   3,
		CustomRules: []string{"?d?d?d"},
	})

	ctx := context.Background()
	passwords, _ := m.Start(ctx)

	// Start consuming passwords
	done := make(chan struct{})
	go func() {
		for range passwords {
			// Consume passwords
		}
		close(done)
	}()

	// Stop the algorithm
	m.Stop()

	select {
	case <-done:
		// Success - channel closed
	case <-time.After(time.Second):
		t.Error("Stop() didn't terminate password generation")
	}
}
