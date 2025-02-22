package algorithm

import (
	"bufio"
	"context"
	"os"
	"passwordCrakerBackend/internal/core/domain"
	"strings"
	"sync"
	"time"
)

type Dictionary struct {
	job       *domain.CrackingJob
	settings  domain.CrackingSettings
	progress  float64
	mu        sync.RWMutex
	stop      chan struct{}
	attempts  int64
	startTime time.Time
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		stop:      make(chan struct{}),
		startTime: time.Now(),
	}
}

func (d *Dictionary) SetJob(job *domain.CrackingJob) {
	d.job = job
	d.settings = job.Settings
}

func (d *Dictionary) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)
		defer d.updateMetrics()

		totalFiles := len(d.settings.WordlistPaths)
		for fileIndex, wordlistPath := range d.settings.WordlistPaths {
			file, err := os.Open(wordlistPath)
			if err != nil {
				errors <- err
				return
			}

			fileInfo, err := file.Stat()
			if err != nil {
				err := file.Close()
				if err != nil {
					return
				}
				errors <- err
				return
			}

			scanner := bufio.NewScanner(file)
			var processedBytes int64

			for scanner.Scan() {
				word := scanner.Text()
				processedBytes += int64(len(word) + 1)

				d.attempts++
				if err := d.processWord(ctx, word, passwords); err != nil {
					err := file.Close()
					if err != nil {
						return
					}
					errors <- err
					return
				}

				if len(d.settings.CustomRules) > 0 {
					for _, rule := range d.settings.CustomRules {
						d.attempts++
						modified := d.applyRule(word, rule)
						if err := d.processWord(ctx, modified, passwords); err != nil {
							err := file.Close()
							if err != nil {
								return
							}
							errors <- err
							return
						}
					}
				}

				fileProgress := float64(processedBytes) / float64(fileInfo.Size())
				d.mu.Lock()
				d.progress = (float64(fileIndex) + fileProgress) / float64(totalFiles)
				d.mu.Unlock()
				d.updateMetrics()
			}

			err = file.Close()
			if err != nil {
				return
			}
			if err := scanner.Err(); err != nil {
				errors <- err
				return
			}
		}
	}()

	return passwords, errors
}

func (d *Dictionary) processWord(ctx context.Context, word string, passwords chan<- string) error {
	if len(word) < d.settings.MinLength || len(word) > d.settings.MaxLength {
		return nil
	}

	select {
	case passwords <- word:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-d.stop:
		return nil
	}
}

func (d *Dictionary) applyRule(word, rule string) string {
	switch rule {
	case "uppercase":
		return strings.ToUpper(word)
	case "capitalize":
		return strings.Title(word)
	case "reverse":
		runes := []rune(word)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	case "append_numbers":
		var result strings.Builder
		result.WriteString(word)
		for i := 0; i <= 9; i++ {
			result.WriteString(string(rune('0' + i)))
		}
		return result.String()
	case "leet":
		replacements := map[string]string{
			"a": "4", "e": "3", "i": "1",
			"o": "0", "s": "5", "t": "7",
		}
		result := word
		for from, to := range replacements {
			result = strings.ReplaceAll(result, from, to)
		}
		return result
	default:
		return word
	}
}

func (d *Dictionary) updateMetrics() {
	if d.job != nil {
		duration := time.Since(d.startTime).Seconds()
		d.job.ResourceMetrics.AttemptsPerSec = int64(float64(d.attempts) / duration)
		d.job.ResourceMetrics.TotalAttempts = d.attempts
		d.job.ResourceMetrics.LastUpdated = time.Now()
		d.job.AttemptCount = d.attempts
		d.job.LastAttempt = time.Now()
		d.job.Progress = d.progress
	}
}

func (d *Dictionary) Stop() {
	close(d.stop)
}

func (d *Dictionary) Progress() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.progress
}

func (d *Dictionary) Name() domain.CrackingAlgorithm {
	return domain.AlgoDictionary
}

func (d *Dictionary) SetSettings(settings domain.CrackingSettings) {
	d.settings = settings
}
