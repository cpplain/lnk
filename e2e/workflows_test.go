package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompleteWorkflow tests a complete workflow from setup to teardown
func TestCompleteWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Step 1: Initial status - should have no links
	t.Run("initial status", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-S", "-t", targetDir, homeSourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})

	// Step 2: Create links
	t.Run("create links", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-C", "-t", targetDir, homeSourceDir)
		assertExitCode(t, result, 0)
		// Note: sandbox allows non-dotfiles and .ssh/
		assertContains(t, result.Stdout, "Created")
	})

	// Step 3: Verify status shows links
	t.Run("status after create", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-S", "-t", targetDir, homeSourceDir)
		assertExitCode(t, result, 0)
		// Note: sandbox allows non-dotfiles and .ssh/
		// Should have at least readonly/test or .ssh/config
		assertNotContains(t, result.Stdout, "No active links found")
	})

	// Step 4: Adopt a new file
	t.Run("adopt new file", func(t *testing.T) {
		// Create a new file that doesn't exist in source
		newFile := filepath.Join(targetDir, ".workflow-adoptrc")
		if err := os.WriteFile(newFile, []byte("# Workflow adopt test file\n"), 0644); err != nil {
			t.Fatal(err)
		}

		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-A", "-s", homeSourceDir, "-t", targetDir, newFile)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Adopted", ".workflow-adoptrc")

		// Verify it's now a symlink
		assertSymlink(t, newFile, filepath.Join(homeSourceDir, ".workflow-adoptrc"))
	})

	// Step 5: Orphan a file
	t.Run("orphan a file", func(t *testing.T) {
		// Orphan the adopted file from step 4
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-O", "-s", homeSourceDir, "-t", targetDir,
			filepath.Join(targetDir, ".workflow-adoptrc"))
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Orphaned", ".workflow-adoptrc")

		// Verify it's no longer a symlink
		assertNoSymlink(t, filepath.Join(targetDir, ".workflow-adoptrc"))
	})

	// Step 6: Remove all links
	t.Run("remove all links", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-R", "-t", targetDir, homeSourceDir)
		assertExitCode(t, result, 0)
		// May say "Removed" or "No symlinks to remove" depending on what was created
		// Just verify command succeeded
	})

	// Step 7: Final status - should have no links again
	t.Run("final status", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-S", "-t", targetDir, homeSourceDir)
		assertExitCode(t, result, 0)
		// Should show no links after removal
		assertContains(t, result.Stdout, "No active links found")
	})
}

// TestFlatRepositoryWorkflow tests using a source directory directly
func TestFlatRepositoryWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	// Use private/home directory as source (has .ssh/ which works in sandbox)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "private", "home")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Step 1: Create links from source directory
	t.Run("create from source directory", func(t *testing.T) {
		result := runCommand(t, "-C", "-t", targetDir, sourceDir)
		assertExitCode(t, result, 0)
		// .ssh/config is allowed in sandbox
		assertContains(t, result.Stdout, "Created", ".ssh/config")

		// Verify links point to source directory
		assertSymlink(t,
			filepath.Join(targetDir, ".ssh", "config"),
			filepath.Join(sourceDir, ".ssh", "config"))
	})

	// Step 2: Status of source directory
	t.Run("status from source directory", func(t *testing.T) {
		result := runCommand(t, "-S", "-t", targetDir, sourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, ".ssh/config")
	})

	// Step 3: Remove links from source directory
	t.Run("remove from source directory", func(t *testing.T) {
		result := runCommand(t, "-R", "-t", targetDir, sourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Removed")
		assertNoSymlink(t, filepath.Join(targetDir, ".ssh"))
	})
}

// TestEdgeCases tests various edge cases and error conditions
func TestEdgeCases(t *testing.T) {
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
	}{
		{
			name:     "non-existent source directory",
			args:     []string{"-C", "-t", targetDir, "/nonexistent"},
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
				homeSourceDir := filepath.Join(sourceDir, "home")
				sourceFile := filepath.Join(homeSourceDir, ".regularfile")
				if err := os.WriteFile(sourceFile, []byte("source file"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"-C", "-t", targetDir, filepath.Join(sourceDir, "home")},
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
			args: []string{"-O", "-s", sourceDir, "-t", targetDir,
				filepath.Join(targetDir, ".regular")},
			wantExit: 0, // Graceful error handling
			contains: []string{"not a symlink"},
		},
		{
			name: "adopt already managed file",
			setup: func(t *testing.T) {
				// Create a link first (using .ssh/config which works in sandbox)
				// Create link from private/home source directory
				privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
				result := runCommand(t, "-C", "-t", targetDir, privateHomeSourceDir)
				assertExitCode(t, result, 0)
			},
			args: []string{"-A", "-s", filepath.Join(sourceDir, "private", "home"), "-t", targetDir,
				filepath.Join(targetDir, ".ssh", "config")},
			wantExit: 0, // Graceful error handling
			contains: []string{"already adopted"},
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

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
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
		homeSourceDir := filepath.Join(sourceDir, "home")
		sourceFile := filepath.Join(homeSourceDir, "readonly", "test")
		if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		result := runCommand(t, "-C", "-t", targetDir, homeSourceDir)
		// Should handle permission error gracefully
		assertExitCode(t, result, 0) // Other links should still be created
		// Check both stdout and stderr for permission error
		combined := result.Stdout + result.Stderr
		assertContains(t, combined, "permission denied")
	})
}

// TestIgnorePatterns tests ignore pattern functionality
func TestIgnorePatterns(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	t.Run("ignore pattern via CLI flag", func(t *testing.T) {
		// Use home source directory and ignore readonly
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-C", "-t", targetDir,
			"--ignore", "readonly/*", homeSourceDir)
		assertExitCode(t, result, 0)

		// readonly should be ignored (no files created since all others are dotfiles)
		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})

	t.Run("multiple ignore patterns", func(t *testing.T) {
		cleanup()

		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "-C", "-t", targetDir,
			"--ignore", ".config/*",
			"--ignore", "readonly/*",
			homeSourceDir)
		assertExitCode(t, result, 0)

		// Should not create .config or readonly (both ignored)
		assertNoSymlink(t, filepath.Join(targetDir, ".config"))
		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})
}
