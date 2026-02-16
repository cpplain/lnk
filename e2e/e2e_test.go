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

// TestVersion tests the version command
func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "version command",
			args:     []string{"version"},
			wantExit: 0,
			contains: []string{"lnk "},
		},
		{
			name:     "version flag",
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
			name:     "help flag",
			args:     []string{"--help"},
			wantExit: 0,
			contains: []string{"Usage:", "Commands:", "Options:"},
		},
		{
			name:     "help command",
			args:     []string{"help"},
			wantExit: 0,
			contains: []string{"Usage:", "Commands:", "Options:"},
		},
		{
			name:     "command help",
			args:     []string{"help", "create"},
			wantExit: 0,
			contains: []string{"lnk create", "Create symlinks"},
		},
		{
			name:     "command --help",
			args:     []string{"create", "--help"},
			wantExit: 0,
			contains: []string{"lnk create", "Create symlinks"},
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

// TestInvalidCommands tests error handling for invalid commands
func TestInvalidCommands(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantExit    int
		contains    []string
		notContains []string
	}{
		{
			name:     "unknown command",
			args:     []string{"invalid"},
			wantExit: 2, // ExitUsage
			contains: []string{"unknown command"},
		},
		{
			name:     "typo suggestion",
			args:     []string{"crate"}, // typo of "create"
			wantExit: 2,
			contains: []string{"Did you mean 'create'"},
		},
		{
			name:     "no command",
			args:     []string{},
			wantExit: 2,
			contains: []string{"Usage:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)
			if tt.name == "no command" {
				// Usage goes to stdout when there's no command
				assertContains(t, result.Stdout, tt.contains...)
			} else {
				assertContains(t, result.Stderr, tt.contains...)
			}
			assertNotContains(t, result.Stderr, tt.notContains...)
		})
	}
}

// TestStatus tests the status command
func TestStatus(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)

	tests := []struct {
		name     string
		args     []string
		setup    func(t *testing.T)
		wantExit int
		contains []string
	}{
		{
			name:     "status with no links",
			args:     []string{"--config", configPath, "status"},
			wantExit: 0,
			contains: []string{"No active links found"},
		},
		{
			name: "status with links",
			args: []string{"--config", configPath, "status"},
			setup: func(t *testing.T) {
				// First create some links
				result := runCommand(t, "--config", configPath, "create")
				assertExitCode(t, result, 0)
			},
			wantExit: 0,
			contains: []string{".bashrc", ".gitconfig", ".config/nvim/init.vim", ".ssh/config"},
		},
		{
			name:     "status with JSON output",
			args:     []string{"--config", configPath, "--output", "json", "status"},
			wantExit: 0,
			contains: []string{"{", "}", "links"},
		},
		{
			name:     "status with verbose",
			args:     []string{"--config", configPath, "--verbose", "status"},
			wantExit: 0,
			contains: []string{"Using configuration from:"},
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

			// Validate JSON output
			if slices.Contains(tt.args, "json") {
				var data map[string]interface{}
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

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name:     "create dry-run",
			args:     []string{"--config", configPath, "create", "--dry-run"},
			wantExit: 0,
			contains: []string{"dry-run:", ".bashrc", ".gitconfig"},
			verify: func(t *testing.T) {
				// Verify no actual links were created
				targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
				assertNoSymlink(t, filepath.Join(targetDir, ".bashrc"))
			},
		},
		{
			name:     "create links",
			args:     []string{"--config", configPath, "create"},
			wantExit: 0,
			contains: []string{"Creating", ".bashrc", ".gitconfig", ".config/nvim/init.vim"},
			verify: func(t *testing.T) {
				// Verify links were created
				targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
				sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")

				assertSymlink(t,
					filepath.Join(targetDir, ".bashrc"),
					filepath.Join(sourceDir, ".bashrc"))
				assertSymlink(t,
					filepath.Join(targetDir, ".gitconfig"),
					filepath.Join(sourceDir, ".gitconfig"))
				assertSymlink(t,
					filepath.Join(targetDir, ".config", "nvim", "init.vim"),
					filepath.Join(sourceDir, ".config", "nvim", "init.vim"))
			},
		},
		{
			name:     "create with existing links",
			args:     []string{"--config", configPath, "create"},
			wantExit: 0,
			contains: []string{"already exist"},
		},
		{
			name:     "create with quiet mode",
			args:     []string{"--config", configPath, "--quiet", "create"},
			wantExit: 0,
			contains: []string{}, // Should have no output
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't cleanup between subtests in this group to test progression
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if len(tt.contains) > 0 {
				assertContains(t, result.Stdout, tt.contains...)
			} else if slices.Contains(tt.args, "--quiet") {
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

// TestRemove tests the remove command
func TestRemove(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)

	// First create some links
	result := runCommand(t, "--config", configPath, "create")
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
			args:     []string{"--config", configPath, "remove", "--dry-run"},
			wantExit: 0,
			contains: []string{"dry-run:", "Would remove"},
			verify: func(t *testing.T) {
				// Verify links still exist
				targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
				sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")
				assertSymlink(t, filepath.Join(targetDir, ".bashrc"), filepath.Join(sourceDir, ".bashrc"))
			},
		},
		{
			name:     "remove with --yes flag",
			args:     []string{"--config", configPath, "--yes", "remove"},
			wantExit: 0,
			contains: []string{"Removed", ".bashrc"},
			verify: func(t *testing.T) {
				// Verify links are gone
				targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
				assertNoSymlink(t, filepath.Join(targetDir, ".bashrc"))
				assertNoSymlink(t, filepath.Join(targetDir, ".gitconfig"))
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

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")

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
			args: []string{"--config", configPath, "adopt",
				"--path", filepath.Join(targetDir, ".adopt-test"),
				"--source-dir", sourceDir},
			wantExit: 0,
			contains: []string{"Adopted", ".adopt-test"},
			verify: func(t *testing.T) {
				// Verify file was moved and linked
				assertSymlink(t,
					filepath.Join(targetDir, ".adopt-test"),
					filepath.Join(sourceDir, ".adopt-test"))
			},
		},
		{
			name:     "adopt missing required flags",
			args:     []string{"--config", configPath, "adopt", "--path", "/tmp/test"},
			wantExit: 2,
			contains: []string{"both --path and --source-dir are required"},
		},
		{
			name: "adopt non-existent file",
			args: []string{"--config", configPath, "adopt",
				"--path", filepath.Join(targetDir, ".doesnotexist"),
				"--source-dir", sourceDir},
			wantExit: 1,
			contains: []string{"no such file"},
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
			args: []string{"--config", configPath, "adopt", "--dry-run",
				"--path", filepath.Join(targetDir, ".dryruntest"),
				"--source-dir", sourceDir},
			wantExit: 0,
			contains: []string{"dry-run:", "Would adopt"},
			verify: func(t *testing.T) {
				// Verify file was NOT moved
				assertNoSymlink(t, filepath.Join(targetDir, ".dryruntest"))
			},
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

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Create links first
	result := runCommand(t, "--config", configPath, "create")
	assertExitCode(t, result, 0)

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
		verify   func(t *testing.T)
	}{
		{
			name: "orphan a file with --yes",
			args: []string{"--config", configPath, "--yes", "orphan",
				"--path", filepath.Join(targetDir, ".bashrc")},
			wantExit: 0,
			contains: []string{"Orphaned", ".bashrc"},
			verify: func(t *testing.T) {
				// Verify file exists but is not a symlink
				assertNoSymlink(t, filepath.Join(targetDir, ".bashrc"))
			},
		},
		{
			name:     "orphan missing path",
			args:     []string{"--config", configPath, "orphan"},
			wantExit: 2,
			contains: []string{"--path is required"},
		},
		{
			name: "orphan dry-run",
			args: []string{"--config", configPath, "orphan", "--dry-run",
				"--path", filepath.Join(targetDir, ".gitconfig")},
			wantExit: 0,
			contains: []string{"dry-run:", "Would orphan"},
			verify: func(t *testing.T) {
				// Verify link still exists
				sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")
				assertSymlink(t, filepath.Join(targetDir, ".gitconfig"), filepath.Join(sourceDir, ".gitconfig"))
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

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Create a broken symlink that points to a file within the configured source directory
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")
	nonExistentSource := filepath.Join(sourceDir, ".nonexistent")
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
			args:     []string{"--config", configPath, "prune", "--dry-run"},
			wantExit: 0,
			contains: []string{"dry-run:", "Would prune", ".broken"},
			verify: func(t *testing.T) {
				// Verify broken link still exists
				assertSymlink(t, brokenLink, nonExistentSource)
			},
		},
		{
			name:     "prune with --yes",
			args:     []string{"--config", configPath, "--yes", "prune"},
			wantExit: 0,
			contains: []string{"Pruned", ".broken"},
			verify: func(t *testing.T) {
				// Verify broken link is gone
				assertNoSymlink(t, brokenLink)
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

// TestGlobalFlags tests global flag behavior
func TestGlobalFlags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)

	tests := []struct {
		name        string
		args        []string
		wantExit    int
		contains    []string
		notContains []string
	}{
		{
			name:        "quiet and verbose conflict",
			args:        []string{"--quiet", "--verbose", "status"},
			wantExit:    2,
			contains:    []string{"cannot use --quiet and --verbose together"},
			notContains: []string{},
		},
		{
			name:        "invalid output format",
			args:        []string{"--output", "xml", "status"},
			wantExit:    2,
			contains:    []string{"invalid output format", "Valid formats are: text, json"},
			notContains: []string{},
		},
		{
			name:        "quiet mode suppresses output",
			args:        []string{"--config", configPath, "--quiet", "status"},
			wantExit:    0,
			contains:    []string{},
			notContains: []string{"No symlinks found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCommand(t, tt.args...)
			assertExitCode(t, result, tt.wantExit)

			if len(tt.contains) > 0 {
				assertContains(t, result.Stderr, tt.contains...)
			}
			if len(tt.notContains) > 0 {
				assertNotContains(t, result.Stdout, tt.notContains...)
			}
		})
	}
}
