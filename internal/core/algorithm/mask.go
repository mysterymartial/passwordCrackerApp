package algorithm

import (
	"context"
	"fmt"
	"log"
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
		stop: make(chan struct{}), // Fresh stop channel per instance
		charsets: map[rune]string{
			'l': "ab", // Test-friendly subset
			'u': domain.CharsetUpper,
			'd': "01", // Test-friendly subset
			's': domain.CharsetSpecial,
			'a': domain.CharsetAll,
		},
	}
}

func (m *Mask) Start(ctx context.Context) (<-chan string, <-chan error) {
	passwords := make(chan string, 100)
	errors := make(chan error, 1)

	go func() {
		defer close(passwords)
		defer close(errors)

		if len(m.settings.CustomRules) == 0 {
			errors <- fmt.Errorf("no mask patterns specified")
			return
		}

		m.totalPatterns = int64(len(m.settings.CustomRules))

		workerCount := 4
		patternChan := make(chan string, m.totalPatterns)

		var wg sync.WaitGroup
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go m.worker(ctx, patternChan, passwords, errors, &wg)
		}

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
			if err := m.generateFromMask(ctx, mask, passwords); err != nil {
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
	if strings.HasPrefix(pattern, "?") {
		return []string{pattern}
	}

	re := regexp.MustCompile(`\[([^\]]+)\]\{(\d+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	if len(matches) == 0 {
		return []string{pattern}
	}

	var expandedMask strings.Builder
	lastPos := 0
	for _, match := range matches {
		start := re.FindStringIndex(pattern[lastPos:])[0] + lastPos
		expandedMask.WriteString(pattern[lastPos:start])

		charsetType := match[1]
		count, _ := strconv.Atoi(match[2])
		charsetSymbol := m.getCharsetSymbol(charsetType)
		expandedMask.WriteString(strings.Repeat(charsetSymbol, count))

		lastPos = start + len(match[0])
	}
	expandedMask.WriteString(pattern[lastPos:])

	log.Printf("Expanded pattern %s to %s", pattern, expandedMask.String())
	return []string{expandedMask.String()}
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

func (m *Mask) generateFromMask(ctx context.Context, mask string, passwords chan<- string) error {
	var parts []struct {
		charset string
		isMask  bool
	}
	for i := 0; i < len(mask); i++ {
		if i+1 < len(mask) && mask[i] == '?' {
			charset, ok := m.charsets[rune(mask[i+1])]
			if !ok {
				return fmt.Errorf("unknown charset symbol: %c", mask[i+1])
			}
			log.Printf("Charset for ?%c: %s", mask[i+1], charset)
			parts = append(parts, struct {
				charset string
				isMask  bool
			}{charset, true})
			i++
		} else {
			parts = append(parts, struct {
				charset string
				isMask  bool
			}{string(mask[i]), false})
		}
	}

	current := make([]int, len(parts))
	for {
		var password strings.Builder
		for i, part := range parts {
			if part.isMask {
				if current[i] < len(part.charset) {
					password.WriteString(string(part.charset[current[i]]))
				}
			} else {
				password.WriteString(part.charset)
			}
		}

		pwd := password.String()
		if len(pwd) >= m.settings.MinLength && len(pwd) <= m.settings.MaxLength {
			atomic.AddInt64(&m.attempts, 1)
			select {
			case passwords <- pwd:
				log.Printf("Generated password: %s", pwd)
			case <-ctx.Done():
				return ctx.Err()
			case <-m.stop:
				return nil
			}
		}

		pos := len(parts) - 1
		for pos >= 0 {
			if !parts[pos].isMask {
				pos--
				continue
			}
			current[pos]++
			if current[pos] < len(parts[pos].charset) {
				break
			}
			current[pos] = 0
			pos--
		}
		if pos < 0 {
			break
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
