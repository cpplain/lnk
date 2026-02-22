// Package e2e contains end-to-end tests that verify lnk's CLI behavior by
// building and executing the actual binary. These tests complement unit tests
// by ensuring the command-line interface works correctly from a user's perspective.
//
// Test files:
//   - e2e_test.go: Core command tests (version, help, status, create, remove, etc.)
//   - workflows_test.go: Multi-command workflow tests and edge cases
//   - helpers_test.go: Test utilities for building binary and running commands
package e2e

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
	}{
		{
			name:     "short help flag",
			args:     []string{"-h"},
			wantExit: 0,
			contains: []string{"Usage:", "Action Flags", "Examples:"},
		},
		{
			name:     "long help flag",
			args:     []string{"--help"},
			wantExit: 0,
			contains: []string{"Usage:", "Action Flags", "Examples:"},
		},
		{
			name:     "no arguments shows usage",
			args:     []string{},
			wantExit: 2,
			contains: []string{"at least one path is required"},
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
		})
	}
}

// TestInvalidFlags tests error handling for invalid flags
func TestInvalidFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantExit    int
		contains    []string
		notContains []string
	}{
		{
			name:     "unknown flag",
			args:     []string{"--invalid", "home"},
			wantExit: 2,
			contains: []string{"unknown flag: --invalid"},
		},
		{
			name:     "multiple action flags",
			args:     []string{"-C", "-R", "home"},
			wantExit: 2,
			contains: []string{"cannot use multiple action flags"},
		},
		{
			name:     "conflicting flags",
			args:     []string{"--quiet", "--verbose", "home"},
			wantExit: 2,
			contains: []string{"cannot use --quiet and --verbose together"},
		},
		{
			name:     "missing package argument",
			args:     []string{"-C"},
			wantExit: 2,
			contains: []string{"at least one path is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			assertContains(t, result.Stderr, tt.contains...)
			assertNotContains(t, result.Stderr, tt.notContains...)
		})
	}
}

// TestStatus tests the status action
func TestStatus(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	tests := []struct {
		name     string
		args     []string
		setup    func(t *testing.T)
		wantExit int
		contains []string
	}{
		{
			name:     "status with no links",
			args:     []string{"-S", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"No active links found"},
		},
		{
			name: "status with links",
			args: []string{"-S", "-t", targetDir, filepath.Join(sourceDir, "home")},
			setup: func(t *testing.T) {
				// First create some links
				homeSourceDir := filepath.Join(sourceDir, "home")
				result := runCommand(t, "-C", "-t", targetDir, homeSourceDir)
				assertExitCode(t, result, 0)
			},
			wantExit: 0,
			// Note: sandbox only allows non-dotfiles
			contains: []string{"readonly/test"},
		},
		{
			name: "status with private directory",
			args: []string{"-S", "-t", targetDir, filepath.Join(sourceDir, "private", "home")},
			setup: func(t *testing.T) {
				// Create links from private/home source directory
				privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
				result := runCommand(t, "-C", "-t", targetDir, privateHomeSourceDir)
				assertExitCode(t, result, 0)
			},
			wantExit: 0,
			contains: []string{".ssh/config"},
		},
		{
			name:     "status with verbose",
			args:     []string{"-S", "-v", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"Source directory:", "Target directory:"},
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

// TestCreate tests the create action (default)
func TestCreate(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name:     "create dry-run",
			args:     []string{"-C", "-n", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			// Dry-run shows all files that would be linked
			contains: []string{"dry-run:", "Would create"},
			verify: func(t *testing.T) {
				// Verify no actual links were created
				assertNoSymlink(t, filepath.Join(targetDir, ".config"))
			},
		},
		{
			name:     "create links from home source directory",
			args:     []string{"-C", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			// Note: sandbox only allows non-dotfiles
			contains: []string{"Created", "readonly/test"},
			verify: func(t *testing.T) {
				// Verify links were created for allowed files (non-dotfiles only)
				homeSourceDir := filepath.Join(sourceDir, "home")
				assertSymlink(t,
					filepath.Join(targetDir, "readonly", "test"),
					filepath.Join(homeSourceDir, "readonly", "test"))
			},
		},
		{
			name:     "create with quiet mode",
			args:     []string{"-C", "-q", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{}, // Should have no output
		},
		{
			name:     "create from private source directory",
			args:     []string{"-C", "-t", targetDir, filepath.Join(sourceDir, "private", "home")},
			wantExit: 0,
			// Note: sandbox allows .ssh/config but blocks top-level dotfiles
			contains: []string{".ssh/config"},
			verify: func(t *testing.T) {
				// Verify links were created (where allowed)
				privateSourceDir := filepath.Join(sourceDir, "private", "home")
				assertSymlink(t,
					filepath.Join(targetDir, ".ssh", "config"),
					filepath.Join(privateSourceDir, ".ssh", "config"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't cleanup between subtests to test progression
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if len(tt.contains) > 0 {
				assertContains(t, result.Stdout, tt.contains...)
			} else if slices.Contains(tt.args, "-q") {
				// In quiet mode, should have minimal output
				if len(result.Stdout) > 0 && result.Stdout != "\n" {
					t.Errorf("Expected no output in quiet mode, got: %s", result.Stdout)
				}
			}

			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestRemove tests the remove action
func TestRemove(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// First create some links
	homeSourceDir := filepath.Join(sourceDir, "home")
	result := runCommand(t, "-C", "-t", targetDir, homeSourceDir)
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
			args:     []string{"-R", "-n", "-t", targetDir, homeSourceDir},
			wantExit: 0,
			contains: []string{"dry-run:", "Would remove"},
			verify: func(t *testing.T) {
				// Verify links still exist for allowed files (non-dotfiles only)
				assertSymlink(t,
					filepath.Join(targetDir, "readonly", "test"),
					filepath.Join(homeSourceDir, "readonly", "test"))
			},
		},
		{
			name:     "remove links",
			args:     []string{"-R", "-t", targetDir, homeSourceDir},
			wantExit: 0,
			contains: []string{"Removed"},
			verify: func(t *testing.T) {
				// Verify allowed links are gone (non-dotfiles only)
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

// TestAdopt tests the adopt action
func TestAdopt(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

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
				// Create a file to adopt with unique name
				testFile := filepath.Join(targetDir, ".adopt-test")
				if err := os.WriteFile(testFile, []byte("# Test file for adopt\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"-A", "-s", filepath.Join(sourceDir, "home"), "-t", targetDir, filepath.Join(targetDir, ".adopt-test")},
			wantExit: 0,
			contains: []string{"Adopted", ".adopt-test"},
			verify: func(t *testing.T) {
				// Verify file was moved and linked
				homeSourceDir := filepath.Join(sourceDir, "home")
				assertSymlink(t,
					filepath.Join(targetDir, ".adopt-test"),
					filepath.Join(homeSourceDir, ".adopt-test"))
			},
		},
		{
			name:     "adopt missing paths",
			args:     []string{"-A", "-s", sourceDir, "-t", targetDir},
			wantExit: 2,
			contains: []string{"at least one path is required"},
		},
		{
			name: "adopt non-existent file",
			args: []string{"-A", "-s", filepath.Join(sourceDir, "home"), "-t", targetDir,
				filepath.Join(targetDir, ".doesnotexist")},
			wantExit: 0, // Continues processing (graceful error handling)
			contains: []string{"No files were adopted"},
		},
		{
			name: "adopt dry-run",
			setup: func(t *testing.T) {
				// Create another file to adopt
				testFile := filepath.Join(targetDir, ".dryruntest")
				if err := os.WriteFile(testFile, []byte("# Dry run test\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"-A", "-n", "-s", filepath.Join(sourceDir, "home"), "-t", targetDir,
				filepath.Join(targetDir, ".dryruntest")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would adopt"},
			verify: func(t *testing.T) {
				// Verify file was NOT moved
				assertNoSymlink(t, filepath.Join(targetDir, ".dryruntest"))
			},
		},
		{
			name: "adopt multiple files",
			setup: func(t *testing.T) {
				// Create multiple files to adopt
				testFile1 := filepath.Join(targetDir, ".multi1")
				testFile2 := filepath.Join(targetDir, ".multi2")
				if err := os.WriteFile(testFile1, []byte("# Test 1\n"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(testFile2, []byte("# Test 2\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"-A", "-s", filepath.Join(sourceDir, "home"), "-t", targetDir,
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

// TestOrphan tests the orphan action
func TestOrphan(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Create links from home source directory (has readonly/test)
	homeSourceDir := filepath.Join(sourceDir, "home")
	result := runCommand(t, "-C", "-t", targetDir, homeSourceDir)
	assertExitCode(t, result, 0)

	// Also create links from private/home (has .ssh/config)
	privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
	result2 := runCommand(t, "-C", "-t", targetDir, privateHomeSourceDir)
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
			args: []string{"-O", "-s", homeSourceDir, "-t", targetDir,
				filepath.Join(targetDir, "readonly", "test")},
			wantExit: 0,
			contains: []string{"Orphaned", "test"},
			verify: func(t *testing.T) {
				// Verify file exists but is not a symlink
				assertNoSymlink(t, filepath.Join(targetDir, "readonly", "test"))
			},
		},
		{
			name:     "orphan missing path",
			args:     []string{"-O", "-s", sourceDir, "-t", targetDir},
			wantExit: 2,
			contains: []string{"at least one path is required"},
		},
		{
			name: "orphan dry-run",
			args: []string{"-O", "-n", "-s", privateHomeSourceDir, "-t", targetDir,
				filepath.Join(targetDir, ".ssh", "config")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would orphan"},
			verify: func(t *testing.T) {
				// Verify link still exists
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

// TestPrune tests the prune action
func TestPrune(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

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
			args:     []string{"-s", sourceDir, "-t", targetDir, "-P", "-n"},
			wantExit: 0,
			contains: []string{"dry-run:", "Would prune", ".broken"},
			verify: func(t *testing.T) {
				// Verify broken link still exists
				assertSymlink(t, brokenLink, nonExistentSource)
			},
		},
		{
			name:     "prune broken links",
			args:     []string{"-s", sourceDir, "-t", targetDir, "-P"},
			wantExit: 0,
			contains: []string{"Pruned", ".broken"},
			verify: func(t *testing.T) {
				// Verify broken link is gone
				assertNoSymlink(t, brokenLink)
			},
		},
		{
			name:     "prune with no broken links",
			args:     []string{"-s", sourceDir, "-t", targetDir, "-P"},
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
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	tests := []struct {
		name        string
		args        []string
		wantExit    int
		contains    []string
		notContains []string
	}{
		{
			name:        "quiet and verbose conflict",
			args:        []string{"-q", "-v", "home"},
			wantExit:    2,
			contains:    []string{"cannot use --quiet and --verbose together"},
			notContains: []string{},
		},
		{
			name:        "quiet mode suppresses output",
			args:        []string{"-q", "-S", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit:    0,
			contains:    []string{},
			notContains: []string{"No active links found"},
		},
		{
			name:     "verbose mode shows extra info",
			args:     []string{"-v", "-S", "-t", targetDir, filepath.Join(sourceDir, "home")},
			wantExit: 0,
			contains: []string{"Source directory:", "Target directory:"},
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
