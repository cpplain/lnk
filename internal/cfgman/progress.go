package cfgman

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressIndicator represents a simple progress indicator
type ProgressIndicator struct {
	message    string
	total      int
	current    int
	startTime  time.Time
	lastUpdate time.Time
	mu         sync.Mutex
	active     bool
	spinner    int
}

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(message string) *ProgressIndicator {
	return &ProgressIndicator{
		message:   message,
		startTime: time.Now(),
		active:    true,
	}
}

// Start starts the progress indicator with an indeterminate spinner
func (p *ProgressIndicator) Start() {
	if IsQuiet() || IsJSONFormat() || !isTerminal() {
		return
	}

	p.mu.Lock()
	p.active = true
	p.mu.Unlock()

	go func() {
		for {
			p.mu.Lock()
			if !p.active {
				p.mu.Unlock()
				break
			}
			spinner := spinnerChars[p.spinner%len(spinnerChars)]
			p.spinner++
			p.mu.Unlock()

			// Clear line and print spinner
			fmt.Printf("\r%s %s %s", spinner, p.message, strings.Repeat(" ", 20))
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// Stop stops the progress indicator and clears the line
func (p *ProgressIndicator) Stop() {
	if IsQuiet() || IsJSONFormat() || !isTerminal() {
		return
	}

	p.mu.Lock()
	p.active = false
	p.mu.Unlock()

	// Clear the line
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
}

// SetTotal sets the total number of items for determinate progress
func (p *ProgressIndicator) SetTotal(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

// Update updates the progress with current count
func (p *ProgressIndicator) Update(current int) {
	if IsQuiet() || IsJSONFormat() || !isTerminal() {
		return
	}

	p.mu.Lock()
	p.current = current
	now := time.Now()

	// Only update display every 100ms to avoid flicker
	if now.Sub(p.lastUpdate) < 100*time.Millisecond {
		p.mu.Unlock()
		return
	}
	p.lastUpdate = now

	// Calculate progress
	var progressStr string
	if p.total > 0 {
		percentage := float64(p.current) * 100 / float64(p.total)
		progressStr = fmt.Sprintf(" (%d/%d, %.0f%%)", p.current, p.total, percentage)
	} else if p.current > 0 {
		progressStr = fmt.Sprintf(" (%d)", p.current)
	}

	spinner := spinnerChars[p.spinner%len(spinnerChars)]
	p.spinner++
	p.mu.Unlock()

	// Clear line and print progress
	fmt.Printf("\r%s %s%s%s", spinner, p.message, progressStr, strings.Repeat(" ", 20))
}

// ShowProgress runs a function with a progress indicator
func ShowProgress(message string, fn func() error) error {
	// Skip progress in quiet mode, JSON mode, or non-terminal
	if IsQuiet() || IsJSONFormat() || !isTerminal() {
		return fn()
	}

	// Only show progress for operations that might take time
	done := make(chan error, 1)
	var result error

	// Start the operation
	go func() {
		done <- fn()
	}()

	// Wait up to 1 second before showing progress
	select {
	case result = <-done:
		// Operation completed quickly, no need for progress
		return result
	case <-time.After(1 * time.Second):
		// Operation is taking time, show progress
	}

	progress := NewProgressIndicator(message)
	progress.Start()
	defer progress.Stop()

	// Wait for operation to complete
	result = <-done
	return result
}
