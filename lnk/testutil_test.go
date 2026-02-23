package lnk

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ==========================================
// Output Capture Helpers
// ==========================================

// CaptureOutput captures stdout during function execution
func CaptureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w

	outChan := make(chan string)
	go func() {
		out, _ := io.ReadAll(r)
		outChan <- string(out)
	}()

	fn()

	w.Close()
	os.Stdout = oldStdout

	return <-outChan
}

// ContainsOutput checks if the output contains all expected strings
func ContainsOutput(t *testing.T, output string, expected ...string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Output missing expected string: %q\nFull output:\n%s", exp, output)
		}
	}
}

// NotContainsOutput checks if the output does not contain any of the strings
func NotContainsOutput(t *testing.T, output string, notExpected ...string) {
	t.Helper()

	for _, notExp := range notExpected {
		if strings.Contains(output, notExp) {
			t.Errorf("Output contains unexpected string: %q\nFull output:\n%s", notExp, output)
		}
	}
}

// ==========================================
// File System Helpers
// ==========================================

// createTestFile creates a test file with the given content
func createTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
}

// assertSymlink verifies that a symlink exists and points to the expected target
func assertSymlink(t *testing.T, link, expectedTarget string) {
	t.Helper()

	info, err := os.Lstat(link)
	if err != nil {
		t.Errorf("Expected symlink %s to exist: %v", link, err)
		return
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink", link)
		return
	}

	target, err := os.Readlink(link)
	if err != nil {
		t.Errorf("Failed to read symlink %s: %v", link, err)
		return
	}

	if target != expectedTarget {
		t.Errorf("Symlink %s points to %s, expected %s", link, target, expectedTarget)
	}
}

// assertNotExists verifies that a file or directory does not exist
func assertNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Lstat(path)
	if err == nil {
		t.Errorf("Expected %s to not exist", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("Unexpected error checking %s: %v", path, err)
	}
}

// assertDirExists verifies that a directory exists
func assertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", path)
		} else {
			t.Errorf("Error checking directory %s: %v", path, err)
		}
	} else if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", path)
	}
}
