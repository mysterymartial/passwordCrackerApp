package algorithm

import (
	"bufio"
	"context"
	"log"
	"os"
	"passwordCrakerBackend/internal/core/domain"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Dictionary struct {
	settings       domain.CrackingSettings
	progress       float64
	mu             sync.RWMutex
	stop           chan struct{}
	attempts       int64
	startTime      time.Time
	totalWords     int64
	processedWords int64
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		stop:      make(chan struct{}),
		startTime: time.Now(),
	}
}

func (d *Dictionary) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		for _, path := range d.settings.WordlistPaths {
			if err := d.processWordlist(ctx, path, passwords); err != nil {
				errors <- err
				return
			}
		}
	}()

	return passwords, errors
}

func (d *Dictionary) processWordlist(ctx context.Context, wordlistPath string, passwords chan<- string) error {
	file, err := os.Open(wordlistPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" {
			continue
		}

		// Debug log
		log.Printf("Processing word: %s", word)

		// Send original word
		if err := d.sendWord(ctx, word, passwords); err != nil {
			return err
		}

		// Process rules
		for _, rule := range d.settings.CustomRules {
			modified := d.applyRule(word, rule)
			if modified != word { // Only send if the word was modified
				// Debug log
				log.Printf("Applying rule '%s' to word '%s': %s", rule, word, modified)

				if err := d.sendWord(ctx, modified, passwords); err != nil {
					return err
				}
			}
		}
	}

	return scanner.Err()
}

func (d *Dictionary) sendWord(ctx context.Context, word string, passwords chan<- string) error {
	select {
	case passwords <- word:
		atomic.AddInt64(&d.attempts, 1)
		log.Printf("Sent word: %s", word) // Debug log
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

func (d *Dictionary) updateProgress() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.progress = float64(d.processedWords) / float64(d.totalWords) * 100
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
	d.countTotalWords()
}

func (d *Dictionary) countTotalWords() {
	d.totalWords = 0
	for _, path := range d.settings.WordlistPaths {
		if file, err := os.Open(path); err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				d.totalWords++
			}
			file.Close()
		}
	}
}
