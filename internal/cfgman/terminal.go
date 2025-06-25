package cfgman

import (
	"os"
)

// isTerminal returns true if the given file descriptor is a terminal.
// This implementation uses a simple and portable approach that works
// across Unix-like systems without relying on platform-specific syscalls.
func isTerminal(fd uintptr) bool {
	// Directly check stdout's file info without creating a new file descriptor
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return (fi.Mode() & os.ModeCharDevice) != 0
}
