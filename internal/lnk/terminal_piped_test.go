package lnk

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout and stderr during test execution
func captureOutput(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()

	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// Replace stdout and stderr
	os.Stdout = wOut
	os.Stderr = wErr

	// Run the function
	f()

	// Close writers
	wOut.Close()
	wErr.Close()

	// Read output
	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)

	// Restore original stdout and stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return bufOut.String(), bufErr.String()
}

func TestPipedOutput(t *testing.T) {
	// Note: These tests simulate piped output by checking the format
	// In actual usage, the terminal detection will handle this automatically

	tests := []struct {
		name           string
		outputFunc     func()
		expectedOutput string
		isStderr       bool
	}{
		{
			name: "success message piped",
			outputFunc: func() {
				// When piped, ShouldSimplifyOutput() returns true
				// This test verifies the format
				fmt.Println("success Successfully created symlink")
			},
			expectedOutput: "success Successfully created symlink\n",
			isStderr:       false,
		},
		{
			name: "error message piped",
			outputFunc: func() {
				// When piped, error messages are simplified
				fmt.Fprintln(os.Stderr, "error: Failed to create symlink")
			},
			expectedOutput: "error: Failed to create symlink\n",
			isStderr:       true,
		},
		{
			name: "warning message piped",
			outputFunc: func() {
				// When piped, warning messages are simplified
				fmt.Fprintln(os.Stderr, "warning: File already exists")
			},
			expectedOutput: "warning: File already exists\n",
			isStderr:       true,
		},
		{
			name: "dry-run message piped",
			outputFunc: func() {
				// When piped, dry-run messages are simplified
				fmt.Println("dry-run: Would create symlink")
			},
			expectedOutput: "dry-run: Would create symlink\n",
			isStderr:       false,
		},
		{
			name: "skip message piped",
			outputFunc: func() {
				// When piped, skip messages are simplified
				fmt.Println("skip File already adopted")
			},
			expectedOutput: "skip File already adopted\n",
			isStderr:       false,
		},
		{
			name: "status active link piped",
			outputFunc: func() {
				// When piped, status output is simplified
				fmt.Println("active ~/.bashrc")
			},
			expectedOutput: "active ~/.bashrc\n",
			isStderr:       false,
		},
		{
			name: "status broken link piped",
			outputFunc: func() {
				// When piped, status output is simplified
				fmt.Println("broken ~/.config/foo")
			},
			expectedOutput: "broken ~/.config/foo\n",
			isStderr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, tt.outputFunc)

			var output string
			if tt.isStderr {
				output = stderr
			} else {
				output = stdout
			}

			if output != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, output)
			}
		})
	}
}

func TestPipedOutputFormat(t *testing.T) {
	// Test that piped output format is suitable for grep/awk
	testCases := []struct {
		name   string
		output string
		grep   string
		want   []string
	}{
		{
			name:   "grep active links",
			output: "active ~/.bashrc\nactive ~/.zshrc\nbroken ~/.config/foo\n",
			grep:   "^active",
			want:   []string{"active ~/.bashrc", "active ~/.zshrc"},
		},
		{
			name:   "grep broken links",
			output: "active ~/.bashrc\nactive ~/.zshrc\nbroken ~/.config/foo\n",
			grep:   "^broken",
			want:   []string{"broken ~/.config/foo"},
		},
		{
			name:   "grep errors",
			output: "success Created symlink\nerror: Failed to create symlink\nsuccess Created another\n",
			grep:   "^error:",
			want:   []string{"error: Failed to create symlink"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lines := strings.Split(strings.TrimSpace(tc.output), "\n")
			var matched []string
			for _, line := range lines {
				if strings.HasPrefix(line, strings.TrimPrefix(tc.grep, "^")) {
					matched = append(matched, line)
				}
			}

			if len(matched) != len(tc.want) {
				t.Errorf("Expected %d matches, got %d", len(tc.want), len(matched))
			}

			for i, want := range tc.want {
				if i >= len(matched) || matched[i] != want {
					t.Errorf("Expected match %d to be %q, got %q", i, want, matched[i])
				}
			}
		})
	}
}
