// helpers_test.go provides test utilities for E2E tests.
//
// Available helper functions:
//   - runCommand(): Execute lnk with arguments and capture output
//   - assertExitCode(): Verify command exit codes
//   - assertContains(): Check output contains expected text
//   - assertNotContains(): Check output does not contain text
//   - assertSymlink(): Verify symlink exists and points correctly
//   - assertNoSymlink(): Verify path is not a symlink
//   - setupTestEnv(): Create test environment and return cleanup function
package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	// Cache the binary path to avoid rebuilding for every test
	cachedBinary   string
	cachedBinaryMu sync.Mutex
)

// buildBinary builds the lnk binary for testing, caching the result
func buildBinary(t *testing.T) string {
	t.Helper()

	cachedBinaryMu.Lock()
	defer cachedBinaryMu.Unlock()

	// Return cached binary if already built
	if cachedBinary != "" {
		// Check if binary still exists
		if _, err := os.Stat(cachedBinary); err == nil {
			return cachedBinary
		}
		// Binary was deleted, need to rebuild
		cachedBinary = ""
	}

	// Build in a fixed location that all tests can share
	projectRoot := getProjectRoot(t)
	testdataDir := filepath.Join(projectRoot, "e2e", "testdata")
	binary := filepath.Join(testdataDir, "lnk-test")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	// Ensure testdata directory exists
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binary, filepath.Join(projectRoot, "cmd", "lnk"))
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Cache the binary path
	cachedBinary = binary
	return binary
}

// commandResult holds the output and exit code of a command
type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCommand runs lnk with the given arguments and returns the result
func runCommand(t *testing.T, args ...string) commandResult {
	t.Helper()

	binary := buildBinary(t)
	cmd := exec.Command(binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set a minimal, predictable environment for testing
	// Only include what's necessary for lnk to function
	testHome := filepath.Join(getProjectRoot(t), "e2e", "testdata", "target")
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), // Need PATH to find external commands if any
		"HOME=" + testHome,          // Set HOME to our test directory
		"NO_COLOR=1",                // Disable colors for easier output testing
	}

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	return commandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// getProjectRoot returns the project root directory
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Get the directory of this test file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get current file path")
	}

	// Go up one level from e2e/ to get project root
	return filepath.Dir(filepath.Dir(filename))
}

var (
	// Track if we've already set up the test environment
	testEnvSetup   bool
	testEnvSetupMu sync.Mutex
)

// setupTestEnv sets up the test environment and returns a cleanup function
func setupTestEnv(t *testing.T) func() {
	t.Helper()

	projectRoot := getProjectRoot(t)
	setupScript := filepath.Join(projectRoot, "scripts", "setup-testdata.sh")

	testEnvSetupMu.Lock()
	// Only run setup script if not already done
	if !testEnvSetup {
		// Run setup script
		cmd := exec.Command("bash", setupScript)
		if output, err := cmd.CombinedOutput(); err != nil {
			testEnvSetupMu.Unlock()
			t.Fatalf("Failed to run setup script: %v\nOutput: %s", err, output)
		}
		testEnvSetup = true
	}
	testEnvSetupMu.Unlock()

	// Return cleanup function
	return func() {
		// Clean up only the target directory (where links are created)
		// This is much faster than recreating everything
		targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

		// Remove all contents except .gitkeep
		if entries, err := os.ReadDir(targetDir); err == nil {
			for _, entry := range entries {
				if entry.Name() != ".gitkeep" {
					os.RemoveAll(filepath.Join(targetDir, entry.Name()))
				}
			}
		}

		// Clean up any test-created files in source directories
		// Only remove files that aren't part of the original setup
		sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")

		// These are files/dirs created by setup script that should be preserved
		setupFiles := map[string]bool{
			".bashrc":    true,
			".gitconfig": true,
			".config":    true, // directory
			".ssh":       true, // directory created by script but not in home
			"readonly":   true, // directory
		}

		if entries, err := os.ReadDir(sourceDir); err == nil {
			for _, entry := range entries {
				if !setupFiles[entry.Name()] {
					// This is a file created by tests, remove it
					os.RemoveAll(filepath.Join(sourceDir, entry.Name()))
				}
			}
		}

		// If source files are missing, we need to re-run setup next time
		if _, err := os.Stat(filepath.Join(sourceDir, ".bashrc")); os.IsNotExist(err) {
			testEnvSetupMu.Lock()
			testEnvSetup = false
			testEnvSetupMu.Unlock()
		}
	}
}

// assertContains checks if the output contains all expected strings
func assertContains(t *testing.T, output string, expected ...string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Output missing expected string: %q\nFull output:\n%s", exp, output)
		}
	}
}

// assertNotContains checks if the output does not contain any of the strings
func assertNotContains(t *testing.T, output string, notExpected ...string) {
	t.Helper()

	for _, notExp := range notExpected {
		if strings.Contains(output, notExp) {
			t.Errorf("Output contains unexpected string: %q\nFull output:\n%s", notExp, output)
		}
	}
}

// assertExitCode checks if the exit code matches the expected value
func assertExitCode(t *testing.T, result commandResult, expected int) {
	t.Helper()

	if result.ExitCode != expected {
		t.Errorf("Expected exit code %d, got %d\nStdout: %s\nStderr: %s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}
}

// assertSymlink verifies that a symlink exists and points to the expected target
func assertSymlink(t *testing.T, linkPath, expectedTarget string) {
	t.Helper()

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Errorf("Failed to stat %s: %v", linkPath, err)
		return
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink, but it's not", linkPath)
		return
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Errorf("Failed to read symlink %s: %v", linkPath, err)
		return
	}

	if target != expectedTarget {
		t.Errorf("Symlink %s points to %s, expected %s", linkPath, target, expectedTarget)
	}
}

// assertNoSymlink verifies that a path is not a symlink
func assertNoSymlink(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return // File doesn't exist, which is fine
		}
		t.Errorf("Failed to stat %s: %v", path, err)
		return
	}

	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("Expected %s to not be a symlink, but it is", path)
	}
}

// getConfigPath returns the path to the test config file
func getConfigPath(t *testing.T) string {
	return filepath.Join(getProjectRoot(t), "e2e", "testdata", "config.json")
}

// getInvalidConfigPath returns the path to the invalid test config file
func getInvalidConfigPath(t *testing.T) string {
	return filepath.Join(getProjectRoot(t), "e2e", "testdata", "invalid.json")
}
