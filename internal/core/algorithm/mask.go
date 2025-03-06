package algorithm

import (
	"context"
	"fmt"
	"passwordCrakerBackend/internal/core/domain"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type Mask struct {
	settings      domain.CrackingSettings
	progress      float64
	mu            sync.RWMutex
	stop          chan struct{}
	charsets      map[rune]string
	attempts      int64
	totalPatterns int64
	processed     int64
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
	}
}

func (m *Mask) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(passwords)
		defer close(errors)

		if len(m.settings.CustomRules) == 0 {
			errors <- fmt.Errorf("no mask patterns specified")
			return
		}

		m.totalPatterns = int64(len(m.settings.CustomRules))

		// Use worker pool for parallel processing
		workerCount := 4
		patternChan := make(chan string, m.totalPatterns)

		var wg sync.WaitGroup
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go m.worker(ctx, patternChan, passwords, errors, &wg)
		}

		// Feed patterns to workers
		for _, rule := range m.settings.CustomRules {
			patternChan <- rule
		}
		close(patternChan)

		wg.Wait()
	}()

	return passwords, errors
}

func (m *Mask) worker(ctx context.Context, patterns <-chan string, passwords chan<- string, errors chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for pattern := range patterns {
		masks := m.expandMaskPattern(pattern)
		for _, mask := range masks {
			if err := m.generateFromMask(ctx, mask, "", passwords); err != nil {
				select {
				case errors <- err:
				case <-ctx.Done():
					return
				}
				return
			}
		}
		atomic.AddInt64(&m.processed, 1)
		m.updateProgress()
	}
}

func (m *Mask) expandMaskPattern(pattern string) []string {
	var masks []string

	if strings.HasPrefix(pattern, "?") {
		masks = append(masks, pattern)
		return masks
	}

	re := regexp.MustCompile(`\[([^\]]+)\]\{(\d+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	if len(matches) > 0 {
		var expandedMask strings.Builder
		for _, match := range matches {
			charsetType := match[1]
			count, _ := strconv.Atoi(match[2])

			charsetSymbol := m.getCharsetSymbol(charsetType)
			expandedMask.WriteString(strings.Repeat(charsetSymbol, count))
		}
		masks = append(masks, expandedMask.String())
	}

	return masks
}

func (m *Mask) getCharsetSymbol(charsetType string) string {
	switch charsetType {
	case "lower":
		return "?l"
	case "upper":
		return "?u"
	case "digits":
		return "?d"
	case "special":
		return "?s"
	case "all":
		return "?a"
	default:
		return "?a"
	}
}

func (m *Mask) generateFromMask(ctx context.Context, mask, current string, passwords chan<- string) error {
	if len(current) > m.settings.MaxLength {
		return nil
	}

	if len(current) >= m.settings.MinLength && len(current) <= m.settings.MaxLength {
		atomic.AddInt64(&m.attempts, 1)
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

func (m *Mask) updateProgress() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progress = float64(atomic.LoadInt64(&m.processed)) / float64(m.totalPatterns) * 100
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
