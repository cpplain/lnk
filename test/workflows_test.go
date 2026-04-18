package test

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
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	// Step 1: Initial status - should have no links
	t.Run("initial status", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "status", homeSourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})

	// Step 2: Create links
	t.Run("create links", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "create", homeSourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Created")
	})

	// Step 3: Verify status shows links
	t.Run("status after create", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "status", homeSourceDir)
		assertExitCode(t, result, 0)
		assertNotContains(t, result.Stdout, "No active links found")
	})

	// Step 4: Adopt a new file
	t.Run("adopt new file", func(t *testing.T) {
		newFile := filepath.Join(targetDir, ".workflow-adoptrc")
		if err := os.WriteFile(newFile, []byte("# Workflow adopt test file\n"), 0644); err != nil {
			t.Fatal(err)
		}

		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "adopt", homeSourceDir, newFile)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Adopted", ".workflow-adoptrc")

		assertSymlink(t, newFile, filepath.Join(homeSourceDir, ".workflow-adoptrc"))
	})

	// Step 5: Orphan a file
	t.Run("orphan a file", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "orphan", homeSourceDir,
			filepath.Join(targetDir, ".workflow-adoptrc"))
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Orphaned", ".workflow-adoptrc")

		assertNoSymlink(t, filepath.Join(targetDir, ".workflow-adoptrc"))
	})

	// Step 6: Remove all links
	t.Run("remove all links", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "remove", homeSourceDir)
		assertExitCode(t, result, 0)
	})

	// Step 7: Final status - should have no links again
	t.Run("final status", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "status", homeSourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "No active links found")
	})
}

// TestFlatRepositoryWorkflow tests using a source directory directly
func TestFlatRepositoryWorkflow(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles", "private", "home")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	// Step 1: Create links from source directory
	t.Run("create from source directory", func(t *testing.T) {
		result := runCommand(t, "create", sourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, "Created", ".ssh/config")

		assertSymlink(t,
			filepath.Join(targetDir, ".ssh", "config"),
			filepath.Join(sourceDir, ".ssh", "config"))
	})

	// Step 2: Status of source directory
	t.Run("status from source directory", func(t *testing.T) {
		result := runCommand(t, "status", sourceDir)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, ".ssh/config")
	})

	// Step 3: Remove links from source directory
	t.Run("remove from source directory", func(t *testing.T) {
		result := runCommand(t, "remove", sourceDir)
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
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	tests := []struct {
		name     string
		setup    func(t *testing.T)
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "non-existent source directory",
			args:     []string{"create", "/nonexistent"},
			wantExit: 1,
			contains: []string{"does not exist"},
		},
		{
			name: "create with existing non-symlink file",
			setup: func(t *testing.T) {
				regularFile := filepath.Join(targetDir, ".regularfile")
				if err := os.WriteFile(regularFile, []byte("regular file"), 0644); err != nil {
					t.Fatal(err)
				}

				homeSourceDir := filepath.Join(sourceDir, "home")
				sourceFile := filepath.Join(homeSourceDir, ".regularfile")
				if err := os.WriteFile(sourceFile, []byte("source file"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args:     []string{"create", filepath.Join(sourceDir, "home")},
			wantExit: 1,
			contains: []string{"Failed to create 1 symlink(s)"},
		},
		{
			name: "orphan non-symlink",
			setup: func(t *testing.T) {
				regularFile := filepath.Join(targetDir, ".regular")
				if err := os.WriteFile(regularFile, []byte("regular"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			args: []string{"orphan", filepath.Join(sourceDir, "home"),
				filepath.Join(targetDir, ".regular")},
			wantExit: 0,
			contains: []string{"not a symlink"},
		},
		{
			name: "adopt already managed file",
			setup: func(t *testing.T) {
				privateHomeSourceDir := filepath.Join(sourceDir, "private", "home")
				result := runCommand(t, "create", privateHomeSourceDir)
				assertExitCode(t, result, 0)
			},
			args: []string{"adopt", filepath.Join(sourceDir, "private", "home"),
				filepath.Join(targetDir, ".ssh", "config")},
			wantExit: 1,
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
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission tests on Windows")
	}

	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	t.Run("create in read-only directory", func(t *testing.T) {
		readOnlyDir := filepath.Join(targetDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(readOnlyDir, 0755)

		homeSourceDir := filepath.Join(sourceDir, "home")
		sourceFile := filepath.Join(homeSourceDir, "readonly", "test")
		if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		result := runCommand(t, "create", homeSourceDir)
		assertExitCode(t, result, 1)
		assertContains(t, result.Stderr, "failed to create 1 symlink(s)")
	})
}

// TestIgnorePatterns tests ignore pattern functionality
func TestIgnorePatterns(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	projectRoot := getProjectRoot(t)
	sourceDir := filepath.Join(projectRoot, "test", "testdata", "dotfiles")
	targetDir := filepath.Join(projectRoot, "test", "testdata", "target")

	t.Run("ignore pattern via CLI flag", func(t *testing.T) {
		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "create",
			"--ignore", "readonly/*", homeSourceDir)
		assertExitCode(t, result, 0)

		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})

	t.Run("multiple ignore patterns", func(t *testing.T) {
		cleanup()

		homeSourceDir := filepath.Join(sourceDir, "home")
		result := runCommand(t, "create",
			"--ignore", ".config/*",
			"--ignore", "readonly/*",
			homeSourceDir)
		assertExitCode(t, result, 0)

		assertNoSymlink(t, filepath.Join(targetDir, ".config"))
		assertNoSymlink(t, filepath.Join(targetDir, "readonly"))
	})
}
