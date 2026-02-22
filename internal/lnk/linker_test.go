package lnk

import (
	"os"
	"path/filepath"
	"testing"
)

// ==========================================
// New Options-Based Tests
// ==========================================

// ==========================================
// Test Helper Functions
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

func TestCreateLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (configRepo string, opts LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "single package",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				linkPath := filepath.Join(tmpDir, "home", ".bashrc")
				assertSymlink(t, linkPath, filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
		{
			name: "multiple packages",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "config", ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home", "config"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".bashrc"), filepath.Join(configRepo, "home", ".bashrc"))
				assertSymlink(t, filepath.Join(tmpDir, "home", ".vimrc"), filepath.Join(configRepo, "config", ".vimrc"))
			},
		},
		{
			name: "package with dot (current directory)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"."},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".bashrc"), filepath.Join(configRepo, ".bashrc"))
				assertSymlink(t, filepath.Join(tmpDir, "home", ".vimrc"), filepath.Join(configRepo, ".vimrc"))
			},
		},
		{
			name: "nested package path",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".ssh", "config"), "# ssh config")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"private/home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".ssh", "config"), filepath.Join(configRepo, "private", "home", ".ssh", "config"))
			},
		},
		{
			name: "ignore patterns",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "home", "README.md"), "# readme")
				createTestFile(t, filepath.Join(configRepo, "home", ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home"},
					IgnorePatterns: []string{"README.md"},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".bashrc"), filepath.Join(configRepo, "home", ".bashrc"))
				assertSymlink(t, filepath.Join(tmpDir, "home", ".vimrc"), filepath.Join(configRepo, "home", ".vimrc"))
				assertNotExists(t, filepath.Join(tmpDir, "home", "README.md"))
			},
		},
		{
			name: "dry run mode",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         true,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				// Verify symlink was NOT created in dry-run mode
				assertNotExists(t, filepath.Join(tmpDir, "home", ".bashrc"))
			},
		},
		{
			name: "non-existent package skipped",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home", "nonexistent"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				// Should create link from existing package
				assertSymlink(t, filepath.Join(tmpDir, "home", ".bashrc"), filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
		{
			name: "no packages specified",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
		{
			name: "source directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				return "", LinkOptions{
					SourceDir:      filepath.Join(tmpDir, "nonexistent"),
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			configRepo, opts := tt.setup(t, tmpDir)

			err := CreateLinks(opts)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateLinks() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateLinks() error = %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

// TestRemoveLinks tests the RemoveLinks function
func TestRemoveLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (string, LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "remove links from single package",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source files
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")
				createTestFile(t, filepath.Join(configRepo, "home", ".vimrc"), "# vimrc content")

				// Create symlinks
				createTestSymlink(t, filepath.Join(configRepo, "home", ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(configRepo, "home", ".vimrc"), filepath.Join(homeDir, ".vimrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Links should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				assertNotExists(t, filepath.Join(homeDir, ".vimrc"))
			},
		},
		{
			name: "remove links from multiple packages",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source files
				createTestFile(t, filepath.Join(configRepo, "package1", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "package2", ".vimrc"), "# vimrc")

				// Create symlinks
				createTestSymlink(t, filepath.Join(configRepo, "package1", ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(configRepo, "package2", ".vimrc"), filepath.Join(homeDir, ".vimrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"package1", "package2"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Both links should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				assertNotExists(t, filepath.Join(homeDir, ".vimrc"))
			},
		},
		{
			name: "dry run mode",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")

				// Create symlink
				createTestSymlink(t, filepath.Join(configRepo, "home", ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         true,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Link should still exist (dry-run)
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"), filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
		{
			name: "no matching links",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file but no symlinks
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				// Nothing to verify - just shouldn't error
			},
		},
		{
			name: "package with dot (current directory)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file directly in repo root
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")

				// Create symlink
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"."},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
			},
		},
		{
			name: "partial removal - only specified package",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source files in different packages
				createTestFile(t, filepath.Join(configRepo, "package1", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "package2", ".vimrc"), "# vimrc")

				// Create symlinks for both
				createTestSymlink(t, filepath.Join(configRepo, "package1", ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(configRepo, "package2", ".vimrc"), filepath.Join(homeDir, ".vimrc"))

				// Only remove package1
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"package1"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// package1 link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				// package2 link should still exist
				assertSymlink(t, filepath.Join(homeDir, ".vimrc"), filepath.Join(configRepo, "package2", ".vimrc"))
			},
		},
		{
			name: "error: no packages specified",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				return "", LinkOptions{
					SourceDir:      tmpDir,
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
		{
			name: "error: source directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				return "", LinkOptions{
					SourceDir:      filepath.Join(tmpDir, "nonexistent"),
					TargetDir:      filepath.Join(tmpDir, "home"),
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			configRepo, opts := tt.setup(t, tmpDir)

			err := RemoveLinks(opts)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveLinks() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("RemoveLinks() error = %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

func TestPruneWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (string, LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "prune broken links from single package",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file that exists
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")

				// Create symlinks (one active, one broken)
				createTestSymlink(t, filepath.Join(configRepo, "home", ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				// Broken link - points to non-existent file
				createTestSymlink(t, filepath.Join(configRepo, "home", ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Active link should still exist
				if _, err := os.Lstat(filepath.Join(homeDir, ".bashrc")); err != nil {
					t.Errorf("Active link .bashrc should still exist: %v", err)
				}
				// Broken link should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing should be removed")
				}
			},
		},
		{
			name: "prune broken links from multiple packages",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create package directories
				os.MkdirAll(filepath.Join(configRepo, "home"), 0755)
				os.MkdirAll(filepath.Join(configRepo, "work"), 0755)

				// Create broken links in different packages
				createTestSymlink(t, filepath.Join(configRepo, "home", ".missing1"), filepath.Join(homeDir, ".missing1"))
				createTestSymlink(t, filepath.Join(configRepo, "work", ".missing2"), filepath.Join(homeDir, ".missing2"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home", "work"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Both broken links should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing1")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing1 should be removed")
				}
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing2")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing2 should be removed")
				}
			},
		},
		{
			name: "dry-run mode preserves broken links",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create package directory
				os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

				// Create broken link
				createTestSymlink(t, filepath.Join(configRepo, "home", ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         true,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Broken link should still exist in dry-run mode
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing")); err != nil {
					t.Errorf("Broken link .missing should still exist in dry-run mode: %v", err)
				}
			},
		},
		{
			name: "no broken links (graceful handling)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create active link only (no broken links)
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")
				createTestSymlink(t, filepath.Join(configRepo, "home", ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Active link should still exist
				if _, err := os.Lstat(filepath.Join(homeDir, ".bashrc")); err != nil {
					t.Errorf("Active link .bashrc should still exist: %v", err)
				}
			},
		},
		{
			name: "package with . (current directory)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create repo directory
				os.MkdirAll(configRepo, 0755)

				// Create broken link in root of repo
				createTestSymlink(t, filepath.Join(configRepo, ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"."},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Broken link should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing should be removed")
				}
			},
		},
		{
			name: "partial pruning (only specified packages)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create package directories
				os.MkdirAll(filepath.Join(configRepo, "home"), 0755)
				os.MkdirAll(filepath.Join(configRepo, "work"), 0755)

				// Create broken links in different packages
				createTestSymlink(t, filepath.Join(configRepo, "home", ".missing1"), filepath.Join(homeDir, ".missing1"))
				createTestSymlink(t, filepath.Join(configRepo, "work", ".missing2"), filepath.Join(homeDir, ".missing2"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"}, // Only prune home package
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Only home package broken link should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing1")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing1 from home package should be removed")
				}
				// Work package broken link should still exist
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing2")); err != nil {
					t.Errorf("Broken link .missing2 from work package should still exist: %v", err)
				}
			},
		},
		{
			name: "no packages specified (defaults to .)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create repo directory
				os.MkdirAll(configRepo, 0755)

				// Create broken link in root of repo
				createTestSymlink(t, filepath.Join(configRepo, ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{}, // No packages - should default to "."
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Broken link should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, ".missing")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing should be removed")
				}
			},
		},
		{
			name: "error: source directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "nonexistent")
				homeDir := filepath.Join(tmpDir, "home")

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					Packages:       []string{"home"},
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			configRepo, opts := tt.setup(t, tmpDir)

			err := PruneWithOptions(opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("PruneWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

// createTestSymlink creates a symlink for testing
func createTestSymlink(t *testing.T, source, target string) {
	t.Helper()

	// Ensure target directory exists
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("Failed to create symlink %s -> %s: %v", target, source, err)
	}
}
