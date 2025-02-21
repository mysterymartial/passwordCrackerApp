package algorithm

import (
	"context"
	"fmt"
	"passwordCrakerBackend/internal/core/domain"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Mask struct {
	job       *domain.CrackingJob
	settings  domain.CrackingSettings
	progress  float64
	mu        sync.RWMutex
	stop      chan struct{}
	charsets  map[rune]string
	attempts  int64
	startTime time.Time
}

func NewMask() *Mask {
	return &Mask{
		stop: make(chan struct{}),
		charsets: map[rune]string{
			'l': domain.CharsetLower,
			'u': domain.CharsetUpper,
			'd': domain.CharsetDigits,
			's': domain.CharsetSpecial,
			'a': domain.CharsetAll,
		},
		startTime: time.Now(),
	}
}

func (m *Mask) SetJob(job *domain.CrackingJob) {
	m.job = job
	m.settings = job.Settings
}

func (m *Mask) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)
		defer m.updateMetrics()

		if len(m.settings.CustomRules) == 0 {
			errors <- fmt.Errorf("no mask patterns specified")
			return
		}

		totalPatterns := len(m.settings.CustomRules)
		for i, rule := range m.settings.CustomRules {
			masks := m.expandMaskPattern(rule)
			for _, mask := range masks {
				if err := m.generateFromMask(ctx, mask, "", passwords); err != nil {
					errors <- err
					return
				}
			}
			m.mu.Lock()
			m.progress = float64(i+1) / float64(totalPatterns)
			m.mu.Unlock()
			m.updateMetrics()
		}
	}()

	return passwords, errors
}

func (m *Mask) expandMaskPattern(pattern string) []string {
	var masks []string

	// Handle direct mask patterns (?l?d?d?d)
	if strings.HasPrefix(pattern, "?") {
		masks = append(masks, pattern)
		return masks
	}

	// Handle custom pattern syntax [lower]{1}[digits]{3}
	re := regexp.MustCompile(`\[([^\]]+)\]\{(\d+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	if len(matches) > 0 {
		var expandedMask strings.Builder
		for _, match := range matches {
			charsetType := match[1]
			count, _ := strconv.Atoi(match[2])

			var charsetSymbol string
			switch charsetType {
			case "lower":
				charsetSymbol = "?l"
			case "upper":
				charsetSymbol = "?u"
			case "digits":
				charsetSymbol = "?d"
			case "special":
				charsetSymbol = "?s"
			case "all":
				charsetSymbol = "?a"
			}

			expandedMask.WriteString(strings.Repeat(charsetSymbol, count))
		}
		masks = append(masks, expandedMask.String())
	}

	return masks
}

func (m *Mask) generateFromMask(ctx context.Context, mask, current string, passwords chan<- string) error {
	if len(current) > m.settings.MaxLength {
		return nil
	}

	if len(current) >= m.settings.MinLength && len(current) <= m.settings.MaxLength {
		m.attempts++
		select {
		case passwords <- current:
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stop:
			return nil
		}
	}

	if len(mask) < 2 || mask[0] != '?' {
		return nil
	}

	charset := m.charsets[rune(mask[1])]
	for _, char := range charset {
		if err := m.generateFromMask(ctx, mask[2:], current+string(char), passwords); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mask) updateMetrics() {
	if m.job != nil {
		duration := time.Since(m.startTime).Seconds()
		m.job.ResourceMetrics.AttemptsPerSec = int64(float64(m.attempts) / duration)
		m.job.ResourceMetrics.TotalAttempts = m.attempts
		m.job.ResourceMetrics.LastUpdated = time.Now()
		m.job.AttemptCount = m.attempts
		m.job.LastAttempt = time.Now()
		m.job.Progress = m.progress
	}
}

func (m *Mask) Stop() {
	close(m.stop)
}

func (m *Mask) Progress() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.progress
}

func (m *Mask) Name() domain.CrackingAlgorithm {
	return domain.AlgoMask
}

func (m *Mask) SetSettings(settings domain.CrackingSettings) {
	m.settings = settings
}
