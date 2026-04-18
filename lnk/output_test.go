package lnk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPrintWarningWithHint verifies the function writes to stderr with correct format.
// Tests run in piped mode (stdout redirected to pipe), so ShouldSimplifyOutput() is true.
func TestPrintWarningWithHint(t *testing.T) {
	t.Run("writes to stderr not stdout", func(t *testing.T) {
		err := errors.New("something failed")
		stdout, _ := captureOutput(t, func() {
			PrintWarningWithHint(err)
		})
		if stdout != "" {
			t.Errorf("PrintWarningWithHint() must not write to stdout, got: %q", stdout)
		}
	})

	t.Run("piped mode: formats as warning prefix", func(t *testing.T) {
		err := errors.New("something failed")
		_, stderr := captureOutput(t, func() {
			PrintWarningWithHint(err)
		})
		if !strings.Contains(stderr, "warning: something failed") {
			t.Errorf("PrintWarningWithHint() stderr = %q, want to contain %q", stderr, "warning: something failed")
		}
	})

	t.Run("piped mode: no hint line when error has no hint", func(t *testing.T) {
		err := errors.New("something failed")
		_, stderr := captureOutput(t, func() {
			PrintWarningWithHint(err)
		})
		if strings.Contains(stderr, "hint:") {
			t.Errorf("PrintWarningWithHint() should not emit hint line when error has no hint: %q", stderr)
		}
	})

	t.Run("piped mode: emits hint line when error has hint", func(t *testing.T) {
		err := WithHint(errors.New("something failed"), "try this instead")
		_, stderr := captureOutput(t, func() {
			PrintWarningWithHint(err)
		})
		if !strings.Contains(stderr, "warning: something failed") {
			t.Errorf("PrintWarningWithHint() stderr = %q, want to contain %q", stderr, "warning: something failed")
		}
		if !strings.Contains(stderr, "hint: try this instead") {
			t.Errorf("PrintWarningWithHint() stderr = %q, want to contain %q", stderr, "hint: try this instead")
		}
	})

	t.Run("piped mode: wraps fmt.Errorf error with hint", func(t *testing.T) {
		inner := WithHint(errors.New("permission denied"), "check file permissions")
		err := fmt.Errorf("Failed to remove %s: %w", "~/.bashrc", inner)
		_, stderr := captureOutput(t, func() {
			PrintWarningWithHint(err)
		})
		if !strings.Contains(stderr, "warning:") {
			t.Errorf("PrintWarningWithHint() stderr = %q, want warning prefix", stderr)
		}
		if !strings.Contains(stderr, "hint: check file permissions") {
			t.Errorf("PrintWarningWithHint() stderr = %q, want hint line", stderr)
		}
	})
}

// TestPrintNextStep verifies the function accepts three arguments and includes sourceDir.
func TestPrintNextStep(t *testing.T) {
	t.Run("includes command in output", func(t *testing.T) {
		stdout, _ := captureOutput(t, func() {
			PrintNextStep("status", "/tmp/dotfiles", "verify links")
		})
		if !strings.Contains(stdout, "status") {
			t.Errorf("PrintNextStep() output %q does not contain command %q", stdout, "status")
		}
	})

	t.Run("includes description in output", func(t *testing.T) {
		stdout, _ := captureOutput(t, func() {
			PrintNextStep("status", "/tmp/dotfiles", "verify links")
		})
		if !strings.Contains(stdout, "verify links") {
			t.Errorf("PrintNextStep() output %q does not contain description %q", stdout, "verify links")
		}
	})

	t.Run("includes sourceDir in output", func(t *testing.T) {
		stdout, _ := captureOutput(t, func() {
			PrintNextStep("status", "/tmp/dotfiles", "verify links")
		})
		if !strings.Contains(stdout, "dotfiles") {
			t.Errorf("PrintNextStep() output %q does not contain sourceDir %q", stdout, "dotfiles")
		}
	})

	t.Run("contracts home directory in sourceDir", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		sourceDir := filepath.Join(homeDir, "git", "dotfiles")
		stdout, _ := captureOutput(t, func() {
			PrintNextStep("status", sourceDir, "verify links")
		})
		if !strings.Contains(stdout, "~/git/dotfiles") {
			t.Errorf("PrintNextStep() output %q does not contract home dir; want ~/git/dotfiles", stdout)
		}
		if strings.Contains(stdout, homeDir+"/") {
			t.Errorf("PrintNextStep() output %q should contract home dir but contains raw path", stdout)
		}
	})

	t.Run("formats as Next: Run 'lnk <command> <sourceDir>' to <description>", func(t *testing.T) {
		stdout, _ := captureOutput(t, func() {
			PrintNextStep("status", "/tmp/dotfiles", "verify links")
		})
		want := "Next: Run 'lnk status /tmp/dotfiles' to verify links"
		if !strings.Contains(stdout, want) {
			t.Errorf("PrintNextStep() output %q does not contain expected format %q", stdout, want)
		}
	})
}
