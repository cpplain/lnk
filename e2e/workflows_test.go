package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCompleteWorkflow tests a complete workflow from setup to teardown
func TestCompleteWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home")

	// Step 1: Initial status - should have no links
	t.Run("initial status", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "status")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})

	// Step 2: Create links
	t.Run("create links", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "create")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Creating", ".bashrc", ".gitconfig")
	})

	// Step 3: Verify status shows links
	t.Run("status after create", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "status")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, ".bashrc", ".gitconfig", ".config/nvim/init.vim")
		assertNotContains(t, result.Stdout, "No symlinks found")
	})

	// Step 4: Adopt a new file
	t.Run("adopt new file", func(t *testing.T) {

		// Create a new file that doesn't exist in source
		newFile := filepath.Join(targetDir, ".workflow-adoptrc")
		if err := os.WriteFile(newFile, []byte("# Workflow adopt test file\n"), 0644); err != nil {
			t.Fatal(err)
		}

		result := runCommand(t, "--config", configPath, "adopt",
			"--path", newFile,
			"--source-dir", sourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Adopted", ".workflow-adoptrc")

		// Verify it's now a symlink
		assertSymlink(t, newFile, filepath.Join(sourceDir, ".workflow-adoptrc"))
	})

	// Step 5: Orphan a file
	t.Run("orphan a file", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "--yes", "orphan",
			"--path", filepath.Join(targetDir, ".bashrc"))
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Orphaned", ".bashrc")

		// Verify it's no longer a symlink
		assertNoSymlink(t, filepath.Join(targetDir, ".bashrc"))
	})

	// Step 6: Remove all links
	t.Run("remove all links", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "--yes", "remove")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Removed")
	})

	// Step 7: Final status - should have no links again
	t.Run("final status", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "status")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})
}

// TestJSONOutputWorkflow tests JSON output mode across commands
func TestJSONOutputWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)

	// Create links first
	result := runCommand(t, "--config", configPath, "create")
	assertExitCode(t, result, 0)

	// Test JSON output for status
	t.Run("status JSON output", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "--output", "json", "status")
		assertExitCode(t, result, 0)

		// Should be valid JSON
		assertContains(t, result.Stdout, "{", "}", "\"links\"")

		// Should not contain human-readable output
		assertNotContains(t, result.Stdout, "✓", "→")
	})

	// Test that JSON mode affects verbosity
	t.Run("JSON mode quiets non-data output", func(t *testing.T) {
		result := runCommand(t, "--config", configPath, "--output", "json", "create")
		assertExitCode(t, result, 0)

		// Should have minimal output since links already exist
		// But should still be valid JSON if any output
		if len(result.Stdout) > 1 {
			assertContains(t, result.Stdout, "{")
		}
	})
}

// TestEdgeCases tests various edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)
	invalidConfigPath := getInvalidConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	tests := []struct {
		name     string
		setup    func(t *testing.T)
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "invalid config file",
			args:     []string{"--config", invalidConfigPath, "status"},
			wantExit: 1,
			contains: []string{"must be an absolute path"},
		},
		{
			name:     "non-existent config file",
			args:     []string{"--config", "/nonexistent/config.json", "status"},
			wantExit: 1,
			contains: []string{"does not exist"},
		},
		{
			name: "create with existing non-symlink file",
			setup: func(t *testing.T) {
				// Create a regular file where we expect a symlink
				regularFile := filepath.Join(targetDir, ".regularfile")
				if err := os.WriteFile(regularFile, []byte("regular file"), 0644); err != nil {
					t.Fatal(err)
				}

				// Also create it in source so lnk tries to link it
				sourceFile := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home", ".regularfile")
				if err := os.WriteFile(sourceFile, []byte("source file"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"--config", configPath, "create"},
			wantExit: 0,
			contains: []string{"Failed to link", ".regularfile"},
		},
		{
			name: "orphan non-symlink",
			setup: func(t *testing.T) {
				// Create a regular file
				regularFile := filepath.Join(targetDir, ".regular")
				if err := os.WriteFile(regularFile, []byte("regular"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"--config", configPath, "--yes", "orphan",
				"--path", filepath.Join(targetDir, ".regular")},
			wantExit: 1,
			contains: []string{"not a symlink"},
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
				// Check both stdout and stderr for successful commands
				combined := result.Stdout + result.Stderr
				assertContains(t, combined, tt.contains...)
			} else {
				assertContains(t, result.Stderr, tt.contains...)
			}
		})
	}
}

// TestPermissionHandling tests handling of permission-related scenarios
func TestPermissionHandling(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission tests on Windows")
	}

	cleanup := setupTestEnv(t)
	defer cleanup()

	configPath := getConfigPath(t)
	projectRoot := getProjectRoot(t)
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	t.Run("create in read-only directory", func(t *testing.T) {
		// Create a read-only subdirectory
		readOnlyDir := filepath.Join(targetDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Make it read-only
		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		// Create a source file that would be linked there
		sourceFile := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "home", "readonly", "test")
		if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		result := runCommand(t, "--config", configPath, "create")
		// Should handle permission error gracefully
		assertExitCode(t, result, 0) // Other links should still be created
		// Check both stdout and stderr for permission error
		combined := result.Stdout + result.Stderr
		assertContains(t, combined, "permission denied")
	})
}

// TestConfigDiscovery tests configuration file discovery
func TestConfigDiscovery(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)

	t.Run("config from environment variable", func(t *testing.T) {
		// Set LNK_CONFIG environment variable
		configPath := filepath.Join(projectRoot, "e2e", "testdata", "config.json")

		// Build binary once (will use cache if already built)
		binary := buildBinary(t)
		cmd := exec.Command(binary, "status")

		// Use minimal environment like runCommand does
		testHome := filepath.Join(projectRoot, "e2e", "testdata", "target")
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + testHome,
			"LNK_CONFIG=" + configPath,
			"NO_COLOR=1",
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
				t.Fatalf("Command failed with exit code %d: %s", exitErr.ExitCode(), output)
			}
		}

		// Should work without --config flag
		assertContains(t, string(output), "No active links found")
	})
}
