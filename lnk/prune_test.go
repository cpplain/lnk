package lnk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrune(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (string, LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "prune broken links from source directory",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file that exists
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")

				// Create symlinks (one active, one broken)
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				// Broken link - points to non-existent file
				createTestSymlink(t, filepath.Join(configRepo, ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
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
			name: "prune broken links in subdirectories",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create subdirectories
				os.MkdirAll(filepath.Join(configRepo, "subdir1"), 0755)
				os.MkdirAll(filepath.Join(configRepo, "subdir2"), 0755)

				// Create broken links in different subdirectories
				createTestSymlink(t, filepath.Join(configRepo, "subdir1", ".missing1"), filepath.Join(homeDir, "subdir1", ".missing1"))
				createTestSymlink(t, filepath.Join(configRepo, "subdir2", ".missing2"), filepath.Join(homeDir, "subdir2", ".missing2"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Both broken links should be removed
				if _, err := os.Lstat(filepath.Join(homeDir, "subdir1", ".missing1")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing1 should be removed")
				}
				if _, err := os.Lstat(filepath.Join(homeDir, "subdir2", ".missing2")); !os.IsNotExist(err) {
					t.Errorf("Broken link .missing2 should be removed")
				}
			},
		},
		{
			name: "dry-run mode preserves broken links",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create repo directory
				os.MkdirAll(configRepo, 0755)

				// Create broken link
				createTestSymlink(t, filepath.Join(configRepo, ".missing"), filepath.Join(homeDir, ".missing"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
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
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
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

			err := Prune(opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Prune() error = %v, wantErr %v", err, tt.wantErr)
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
