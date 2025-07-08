package cfgman

import (
	"os"
)

// isTerminal returns true if stdout is a terminal.
// This implementation uses a simple and portable approach that works
// across Unix-like systems without relying on platform-specific syscalls.
func isTerminal() bool {
	// Check stdout's file info
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ShouldSimplifyOutput returns true if output should be simplified for piping.
// This is true when stdout is not a terminal and JSON format is not requested.
func ShouldSimplifyOutput() bool {
	return !isTerminal() && !IsJSONFormat()
}
