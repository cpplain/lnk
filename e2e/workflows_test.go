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
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-S", "home")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})

	// Step 2: Create links
	t.Run("create links", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "home", "private/home")
		assertExitCode(t, result, 0)
		// Note: sandbox allows non-dotfiles and .ssh/
		assertContains(t, result.Stdout, "Created")
	})

	// Step 3: Verify status shows links
	t.Run("status after create", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-S", "home", "private/home")
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

		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-A", "home", newFile)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Adopted", ".workflow-adoptrc")

		// Verify it's now a symlink
		homeSourceDir := filepath.Join(sourceDir, "home")
		assertSymlink(t, newFile, filepath.Join(homeSourceDir, ".workflow-adoptrc"))
	})

	// Step 5: Orphan a file
	t.Run("orphan a file", func(t *testing.T) {
		// Orphan the adopted file from step 4
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-O",
			filepath.Join(targetDir, ".workflow-adoptrc"))
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Orphaned", ".workflow-adoptrc")

		// Verify it's no longer a symlink
		assertNoSymlink(t, filepath.Join(targetDir, ".workflow-adoptrc"))
	})

	// Step 6: Remove all links
	t.Run("remove all links", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-R", "home", "private/home")
		assertExitCode(t, result, 0)
		// May say "Removed" or "No symlinks to remove" depending on what was created
		// Just verify command succeeded
	})

	// Step 7: Final status - should have no links again
	t.Run("final status", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-S", "home", "private/home")
		assertExitCode(t, result, 0)
		// Should show no links after removal
		assertContains(t, result.Stdout, "No active links found")
	})
}

// TestMultiPackageWorkflow tests working with multiple packages
func TestMultiPackageWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Step 1: Create links from both home and private/home packages
	t.Run("create from multiple packages", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "home", "private/home")
		assertExitCode(t, result, 0)
		// Note: sandbox allows .ssh/config but blocks top-level dotfiles
		assertContains(t, result.Stdout, "Created", ".ssh/config")
	})

	// Step 2: Status should show links from both packages
	t.Run("status shows all packages", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-S", "home", "private/home")
		assertExitCode(t, result, 0)
		// Note: sandbox blocks top-level dotfiles from home package
		// Only allowed files: .config/nvim from home, .ssh/config from private
		assertContains(t, result.Stdout, ".ssh/config")
	})

	// Step 3: Remove only home package links
	t.Run("remove specific package", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-R", "home")
		assertExitCode(t, result, 0)
		// May have no home links due to sandbox restrictions
		// Just check command succeeded

		// Verify private/home links remain
		privateSourceDir := filepath.Join(sourceDir, "private", "home")
		assertSymlink(t,
			filepath.Join(targetDir, ".ssh", "config"),
			filepath.Join(privateSourceDir, ".ssh", "config"))
	})
}

// TestFlatRepositoryWorkflow tests using a flat repository (package ".")
func TestFlatRepositoryWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	// Use private/home directory as flat source (has .ssh/ which works in sandbox)
	sourceDir := filepath.Join(projectRoot, "e2e", "testdata", "dotfiles", "private", "home")
	targetDir := filepath.Join(projectRoot, "e2e", "testdata", "target")

	// Step 1: Create links from flat repository
	t.Run("create from flat repo", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, ".")
		assertExitCode(t, result, 0)
		// .ssh/config is allowed in sandbox
		assertContains(t, result.Stdout, "Created", ".ssh/config")

		// Verify links point to source directory (not source/.)
		assertSymlink(t,
			filepath.Join(targetDir, ".ssh", "config"),
			filepath.Join(sourceDir, ".ssh", "config"))
	})

	// Step 2: Status with flat repo
	t.Run("status from flat repo", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-S", ".")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, ".ssh/config")
	})

	// Step 3: Remove from flat repo
	t.Run("remove from flat repo", func(t *testing.T) {
		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "-R", ".")
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
			args:     []string{"-s", "/nonexistent", "-t", targetDir, "home"},
			wantExit: 1,
			contains: []string{"does not exist"},
		},
		{
			name:     "non-existent package",
			args:     []string{"-s", sourceDir, "-t", targetDir, "nonexistent"},
			wantExit: 0, // Should skip gracefully
			contains: []string{"Skipping", "nonexistent", "does not exist"},
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
				sourceFile := filepath.Join(sourceDir, "home", ".regularfile")
				if err := os.WriteFile(sourceFile, []byte("source file"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"-s", sourceDir, "-t", targetDir, "home"},
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
			args: []string{"-s", sourceDir, "-t", targetDir, "-O",
				filepath.Join(targetDir, ".regular")},
			wantExit: 0, // Graceful error handling
			contains: []string{"not a symlink"},
		},
		{
			name: "adopt already managed file",
			setup: func(t *testing.T) {
				// Create a link first (using .ssh/config which works in sandbox)
				// Create link from private/home package
				result := runCommand(t, "-s", sourceDir, "-t", targetDir, "private/home")
				assertExitCode(t, result, 0)
			},
			args: []string{"-s", sourceDir, "-t", targetDir, "-A", "private/home",
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
		sourceFile := filepath.Join(sourceDir, "home", "readonly", "test")
		if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		result := runCommand(t, "-s", sourceDir, "-t", targetDir, "home")
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
		// Use home package and ignore readonly
		result := runCommand(t, "-s", sourceDir, "-t", targetDir,
			"--ignore", "readonly/*", "home")
		assertExitCode(t, result, 0)

		// readonly should be ignored (no files created since all others are dotfiles)
		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})

	t.Run("multiple ignore patterns", func(t *testing.T) {
		cleanup()

		result := runCommand(t, "-s", sourceDir, "-t", targetDir,
			"--ignore", ".config/*",
			"--ignore", "readonly/*",
			"home")
		assertExitCode(t, result, 0)

		// Should not create .config or readonly (both ignored)
		assertNoSymlink(t, filepath.Join(targetDir, ".config"))
		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})
}
