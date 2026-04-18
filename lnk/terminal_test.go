package lnk

import (
	"os"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// Save original stdout
	originalStdout := os.Stdout

	tests := []struct {
		name     string
		setup    func() func()
		expected bool
	}{
		{
			name: "pipe is not terminal",
			setup: func() func() {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = w

				return func() {
					os.Stdout = originalStdout
					w.Close()
					r.Close()
				}
			},
			expected: false,
		},
		{
			name: "file is not terminal",
			setup: func() func() {
				tmpFile, err := os.CreateTemp("", "terminal-test")
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = tmpFile

				return func() {
					os.Stdout = originalStdout
					tmpFile.Close()
					os.Remove(tmpFile.Name())
				}
			},
			expected: false,
		},
		{
			name: "dev null behavior",
			setup: func() func() {
				devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = devNull

				return func() {
					os.Stdout = originalStdout
					devNull.Close()
				}
			},
			expected: true, // On macOS, /dev/null is treated as a character device
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			result := isTerminal()
			if result != tt.expected {
				t.Errorf("isTerminal() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTerminalErrorHandling(t *testing.T) {
	// Test with closed file descriptor
	originalStdout := os.Stdout
	defer func() { os.Stdout = originalStdout }()

	// Create a pipe and close it immediately
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	r.Close()

	// Set stdout to the closed pipe
	os.Stdout = w

	// Should return false when stat fails
	result := isTerminal()
	if result {
		t.Error("isTerminal() should return false when stat fails")
	}
}

func TestIsTerminalActualTerminal(t *testing.T) {
	// This test only makes sense when run in an actual terminal
	// When run in CI or as part of automated tests, stdout is usually not a terminal

	// Save the result when using the actual stdout
	result := isTerminal()

	// Just verify the function runs without panicking
	// The actual result depends on the test environment
	t.Logf("isTerminal() returned %v in current environment", result)

	// If we're in a real terminal (like when running go test manually),
	// we expect true. In CI/automated environments, we expect false.
	// We can't assert a specific value here.
}
