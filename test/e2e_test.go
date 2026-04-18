// Package test contains end-to-end tests that verify lnk's CLI behavior by
// building and executing the actual binary. These tests complement unit tests
// by ensuring the command-line interface works correctly from a user's perspective.
//
// Test files:
//   - e2e_test.go: Core command tests (version, help, status, create, remove, etc.)
//   - workflows_test.go: Multi-command workflow tests and edge cases
//   - helpers_test.go: Test utilities for building binary and running commands
package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestVersion tests the version flag
func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "short version flag",
			args:     []string{"-V"},
			wantExit: 0,
			contains: []string{"lnk "},
		},
		{
			name:     "long version flag",
			args:     []string{"--version"},
			wantExit: 0,
			contains: []string{"lnk "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stdout, tt.contains...)
		})
	}
}

// TestHelp tests help output
func TestHelp(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		stream   string // "stdout" or "stderr"
	}{
		{
			name:     "short help flag",
			args:     []string{"-h"},
			wantExit: 0,
			contains: []string{"Usage:", "Commands:", "Examples:"},
			stream:   "stdout",
		},
		{
			name:     "long help flag",
			args:     []string{"--help"},
			wantExit: 0,
			contains: []string{"Usage:", "Commands:", "Examples:"},
			stream:   "stdout",
		},
		{
			name:     "no arguments shows usage error",
			args:     []string{},
			wantExit: 2,
			contains: []string{"Usage:"},
			stream:   "stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			if tt.stream == "stdout" {
				assertContains(t, result.Stdout, tt.contains...)
			} else {
				assertContains(t, result.Stderr, tt.contains...)
			}
		})
	}
}

// TestCommandHelp tests per-command help output
func TestCommandHelp(t *testing.T) {
	commands := []struct {
		name     string
		contains []string
	}{
		{"create", []string{"Usage: lnk create", "source-dir"}},
		{"remove", []string{"Usage: lnk remove", "source-dir"}},
		{"status", []string{"Usage: lnk status", "source-dir"}},
		{"prune", []string{"Usage: lnk prune", "source-dir"}},
		{"adopt", []string{"Usage: lnk adopt", "source-dir", "path"}},
		{"orphan", []string{"Usage: lnk orphan", "source-dir", "path"}},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			result := runCommand(t, cmd.name, "--help")
			assertExitCode(t, result, 0)
			assertContains(t, result.Stdout, cmd.contains...)
		})
	}
}

// TestUnknownCommand tests error handling for unknown commands
func TestUnknownCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "typo suggests closest match",
			args:     []string{"statsu"},
			wantExit: 2,
			contains: []string{"unknown command", "status"},
		},
		{
			name:     "completely wrong command",
			args:     []string{"xyzzy"},
			wantExit: 2,
			contains: []string{"unknown command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stderr, tt.contains...)
		})
	}
}

// TestInvalidFlags tests error handling for invalid flags
func TestInvalidFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "unknown flag",
			args:     []string{"create", "--invalid", "."},
			wantExit: 2,
			contains: []string{"unknown flag: --invalid"},
		},
		{
			name:     "missing source-dir",
			args:     []string{"create"},
			wantExit: 2,
			contains: []string{"missing required argument"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stderr, tt.contains...)
		})
	}
}

// TestStatus tests the status command
func TestStatus(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")

	tests := []struct {
		name     string
		args     []string
		setup    func(t *testing.T)
		wantExit int
		contains []string
	}{
		{
			name:     "status with no links",
			args:     []string{"status", filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"No active links found"},
		},
		{
			name: "status with links",
			args: []string{"status", filepath.Join(sourceDir, "home")},
			setup: func(t *testing.T) {
				homeSourceDir := filepath.Join(sourceDir, "home")
				result := runCommand(t, "create", homeSourceDir)
				assertExitCode(t, result, 0)
			},
			wantExit: 0,
			contains: []string{"readonly/test"},
		},
		{
			name: "status with private directory",
			args: []string{"status", filepath.Join(sourceDir, "private", "home")},
			setup: func(t *testing.T) {
				privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
				result := runCommand(t, "create", privateHomeSourceDir)
				assertExitCode(t, result, 0)
			},
			wantExit: 0,
			contains: []string{".ssh/config"},
		},
		{
			name:     "status with verbose",
			args:     []string{"status", "-v", filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"Source directory:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup() // Clean between tests
			if tt.setup != nil {
				tt.setup(t)
			}

			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stdout, tt.contains...)

			// Validate JSON output if requested
			if slices.Contains(tt.args, "json") {
				var data map[string]any
				if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
					t.Errorf("Invalid JSON output: %v\nOutput: %s", err, result.Stdout)
				}
			}
		})
	}
}

// TestCreate tests the create command
func TestCreate(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name:     "create dry-run",
			args:     []string{"create", "-n", filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would create"},
			verify: func(t *testing.T) {
				assertNoSymlink(t, filepath.Join(targetDir, ".config"))
			},
		},
		{
			name:     "create links from home source directory",
			args:     []string{"create", filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"Created", "readonly/test"},
			verify: func(t *testing.T) {
				homeSourceDir := filepath.Join(sourceDir, "home")
				assertSymlink(t,
					filepath.Join(targetDir, "readonly", "test"),
					filepath.Join(homeSourceDir, "readonly", "test"))
			},
		},
		{
			name:     "create from private source directory",
			args:     []string{"create", filepath.Join(sourceDir, "private", "home")},
			wantExit: 0,
			contains: []string{".ssh/config"},
			verify: func(t *testing.T) {
				privateSourceDir := filepath.Join(sourceDir, "private", "home")
				assertSymlink(t,
					filepath.Join(targetDir, ".ssh", "config"),
					filepath.Join(privateSourceDir, ".ssh", "config"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if len(tt.contains) > 0 {
				assertContains(t, result.Stdout, tt.contains...)
			}

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestRemove tests the remove command
func TestRemove(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	// First create some links
	homeSourceDir := filepath.Join(sourceDir, "home")
	result := runCommand(t, "create", homeSourceDir)
	assertExitCode(t, result, 0)

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name:     "remove dry-run",
			args:     []string{"remove", "-n", homeSourceDir},
			wantExit: 0,
			contains: []string{"dry-run:", "Would remove"},
			verify: func(t *testing.T) {
				assertSymlink(t,
					filepath.Join(targetDir, "readonly", "test"),
					filepath.Join(homeSourceDir, "readonly", "test"))
			},
		},
		{
			name:     "remove links",
			args:     []string{"remove", homeSourceDir},
			wantExit: 0,
			contains: []string{"Removed"},
			verify: func(t *testing.T) {
				assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stdout, tt.contains...)

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestAdopt tests the adopt command
func TestAdopt(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	tests := []struct {
		name     string
		setup    func(t *testing.T)
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name: "adopt a file",
			setup: func(t *testing.T) {
				testFile := filepath.Join(targetDir, ".adopt-test")
				if err := os.WriteFile(testFile, []byte("# Test file for adopt\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"adopt", filepath.Join(sourceDir, "home"), filepath.Join(targetDir, ".adopt-test")},
			wantExit: 0,
			contains: []string{"Adopted", ".adopt-test"},
			verify: func(t *testing.T) {
				homeSourceDir := filepath.Join(sourceDir, "home")
				assertSymlink(t,
					filepath.Join(targetDir, ".adopt-test"),
					filepath.Join(homeSourceDir, ".adopt-test"))
			},
		},
		{
			name:     "adopt missing paths",
			args:     []string{"adopt", filepath.Join(sourceDir, "home")},
			wantExit: 2,
			contains: []string{"adopt requires at least one file path"},
		},
		{
			name: "adopt non-existent file",
			args: []string{"adopt", filepath.Join(sourceDir, "home"),
				filepath.Join(targetDir, ".doesnotexist")},
			wantExit: 1,
			contains: []string{"no such file or directory"},
		},
		{
			name: "adopt dry-run",
			setup: func(t *testing.T) {
				testFile := filepath.Join(targetDir, ".dryruntest")
				if err := os.WriteFile(testFile, []byte("# Dry run test\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"adopt", "-n", filepath.Join(sourceDir, "home"),
				filepath.Join(targetDir, ".dryruntest")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would adopt"},
			verify: func(t *testing.T) {
				assertNoSymlink(t, filepath.Join(targetDir, ".dryruntest"))
			},
		},
		{
			name: "adopt multiple files",
			setup: func(t *testing.T) {
				testFile1 := filepath.Join(targetDir, ".multi1")
				testFile2 := filepath.Join(targetDir, ".multi2")
				if err := os.WriteFile(testFile1, []byte("# Test 1\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(testFile2, []byte("# Test 2\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"adopt", filepath.Join(sourceDir, "home"),
				filepath.Join(targetDir, ".multi1"),
				filepath.Join(targetDir, ".multi2")},
			wantExit: 0,
			contains: []string{"Adopted", ".multi1", ".multi2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if tt.wantExit == 0 {
				assertContains(t, result.Stdout, tt.contains...)
			} else {
				assertContains(t, result.Stderr, tt.contains...)
			}

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestOrphan tests the orphan command
func TestOrphan(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	// Create links from home source directory (has readonly/test)
	homeSourceDir := filepath.Join(sourceDir, "home")
	result := runCommand(t, "create", homeSourceDir)
	assertExitCode(t, result, 0)

	// Also create links from private/home (has .ssh/config)
	privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
	result2 := runCommand(t, "create", privateHomeSourceDir)
	assertExitCode(t, result2, 0)

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name: "orphan a file",
			args: []string{"orphan", homeSourceDir,
				filepath.Join(targetDir, "readonly", "test")},
			wantExit: 0,
			contains: []string{"Orphaned", "test"},
			verify: func(t *testing.T) {
				assertNoSymlink(t, filepath.Join(targetDir, "readonly", "test"))
			},
		},
		{
			name:     "orphan missing path",
			args:     []string{"orphan", homeSourceDir},
			wantExit: 2,
			contains: []string{"orphan requires at least one path"},
		},
		{
			name: "orphan dry-run",
			args: []string{"orphan", "-n", privateHomeSourceDir,
				filepath.Join(targetDir, ".ssh", "config")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would orphan"},
			verify: func(t *testing.T) {
				assertSymlink(t, filepath.Join(targetDir, ".ssh", "config"), filepath.Join(privateHomeSourceDir, ".ssh", "config"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if tt.wantExit == 0 {
				assertContains(t, result.Stdout, tt.contains...)
			} else {
				assertContains(t, result.Stderr, tt.contains...)
			}

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestPrune tests the prune command
func TestPrune(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	// Create a broken symlink that points to a file within the configured source directory
	homeSourceDir := filepath.Join(sourceDir, "home")
	nonExistentSource := filepath.Join(homeSourceDir, ".nonexistent")
	brokenLink := filepath.Join(targetDir, ".broken")
	if err := os.Symlink(nonExistentSource, brokenLink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name:     "prune dry-run",
			args:     []string{"prune", "-n", homeSourceDir},
			wantExit: 0,
			contains: []string{"dry-run:", "Would prune", ".broken"},
			verify: func(t *testing.T) {
				assertSymlink(t, brokenLink, nonExistentSource)
			},
		},
		{
			name:     "prune broken links",
			args:     []string{"prune", homeSourceDir},
			wantExit: 0,
			contains: []string{"Pruned", ".broken"},
			verify: func(t *testing.T) {
				assertNoSymlink(t, brokenLink)
			},
		},
		{
			name:     "prune with no broken links",
			args:     []string{"prune", homeSourceDir},
			wantExit: 0,
			contains: []string{"No broken symlinks found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stdout, tt.contains...)

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestGlobalFlags tests global flag behavior
func TestGlobalFlags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")

	tests := []struct {
		name        string
		args        []string
		wantExit    int
		contains    []string
		notContains []string
	}{
		{
			name:     "verbose mode shows extra info",
			args:     []string{"status", "-v", filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"Source directory:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if tt.wantExit != 0 {
				assertContains(t, result.Stderr, tt.contains...)
			} else if len(tt.contains) > 0 {
				assertContains(t, result.Stdout, tt.contains...)
			}
			if len(tt.notContains) > 0 {
				assertNotContains(t, result.Stdout, tt.notContains...)
			}
		})
	}
}
