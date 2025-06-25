package cfgman

import (
	"fmt"
	"os"
	"sync"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[0;33m"
	ColorBlue   = "\033[0;34m"
	ColorCyan   = "\033[0;36m"
	ColorBold   = "\033[1m"
)

// colorEnabled caches the result of whether colors should be enabled
var (
	colorEnabled     bool
	colorEnabledOnce sync.Once
)

// ShouldEnableColor determines if color output should be enabled based on:
// 1. NO_COLOR environment variable (https://no-color.org/)
// 2. Whether stdout is a terminal (TTY)
func ShouldEnableColor() bool {
	colorEnabledOnce.Do(func() {
		// Check NO_COLOR environment variable
		// According to https://no-color.org/, any non-empty value disables color
		if os.Getenv("NO_COLOR") != "" {
			colorEnabled = false
			return
		}

		// Check if stdout is a terminal
		colorEnabled = isTerminal(os.Stdout.Fd())
	})
	return colorEnabled
}

// Colored output helpers
func Red(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorRed, s, ColorReset)
}

func Green(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorGreen, s, ColorReset)
}

func Yellow(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorYellow, s, ColorReset)
}

func Blue(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorBlue, s, ColorReset)
}

func Cyan(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorCyan, s, ColorReset)
}

func Bold(s string) string {
	if !ShouldEnableColor() {
		return s
	}
	return fmt.Sprintf("%s%s%s", ColorBold, s, ColorReset)
}
