package lnk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (string, LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "remove links from source directory",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source files
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")
				createTestFile(t, filepath.Join(configRepo, ".vimrc"), "# vimrc content")

				// Create symlinks
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(configRepo, ".vimrc"), filepath.Join(homeDir, ".vimrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
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
			name: "remove links with subdirectories",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source files in subdirectories
				createTestFile(t, filepath.Join(configRepo, "subdir1", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "subdir2", ".vimrc"), "# vimrc")

				// Create symlinks (preserving directory structure)
				createTestSymlink(t, filepath.Join(configRepo, "subdir1", ".bashrc"), filepath.Join(homeDir, "subdir1", ".bashrc"))
				createTestSymlink(t, filepath.Join(configRepo, "subdir2", ".vimrc"), filepath.Join(homeDir, "subdir2", ".vimrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Both links should be removed
				assertNotExists(t, filepath.Join(homeDir, "subdir1", ".bashrc"))
				assertNotExists(t, filepath.Join(homeDir, "subdir2", ".vimrc"))
			},
		},
		{
			name: "dry run mode",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")

				// Create symlink
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         true,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Link should still exist (dry-run)
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"), filepath.Join(configRepo, ".bashrc"))
			},
		},
		{
			name: "no matching links",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file but no symlinks
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc content")

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				// Nothing to verify - just shouldn't error
			},
		},
		{
			name: "error: source directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				return "", LinkOptions{
					SourceDir:      filepath.Join(tmpDir, "nonexistent"),
					TargetDir:      filepath.Join(tmpDir, "home"),
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: true,
		},
		{
			name: "skips non-managed symlinks in target",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				otherRepo := filepath.Join(tmpDir, "other")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file in our repo
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")

				// Create managed symlink
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				// Create symlink from a different source (not managed by configRepo)
				createTestFile(t, filepath.Join(otherRepo, ".vimrc"), "# vimrc from other")
				createTestSymlink(t, filepath.Join(otherRepo, ".vimrc"), filepath.Join(homeDir, ".vimrc"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				otherRepo := filepath.Join(tmpDir, "other")
				// Managed link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				// Unmanaged link should still exist
				assertSymlink(t, filepath.Join(homeDir, ".vimrc"), filepath.Join(otherRepo, ".vimrc"))
			},
		},
		{
			name: "skips target path that is a regular file",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")

				// Create a regular file at the target path instead of a symlink
				createTestFile(t, filepath.Join(homeDir, ".bashrc"), "# regular file")

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Regular file should be untouched
				info, err := os.Lstat(filepath.Join(homeDir, ".bashrc"))
				if err != nil {
					t.Fatalf("Expected file to still exist: %v", err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("Expected regular file, got symlink")
				}
			},
		},
		{
			name: "broken symlinks from deleted source files are not found by source walk",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create a source file and its managed symlink
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				createTestSymlink(t, filepath.Join(configRepo, ".bashrc"), filepath.Join(homeDir, ".bashrc"))

				// Create a broken symlink: source file was deleted but symlink remains
				brokenTarget := filepath.Join(configRepo, ".deleted_config")
				createTestSymlink(t, brokenTarget, filepath.Join(homeDir, ".deleted_config"))
				// Note: brokenTarget doesn't exist, so this symlink is broken
				// Source-walk won't find it because the source file doesn't exist

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Managed link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				// Broken symlink should still exist (source-walk doesn't find it)
				_, err := os.Lstat(filepath.Join(homeDir, ".deleted_config"))
				if err != nil {
					t.Error("Broken symlink should still exist — source-walk only finds current source files")
				}
			},
		},
		{
			name: "empty parent directories cleaned after removal",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file in nested directory
				createTestFile(t, filepath.Join(configRepo, ".config", "app", "settings.conf"), "# settings")

				// Create symlink in matching nested structure
				createTestSymlink(t,
					filepath.Join(configRepo, ".config", "app", "settings.conf"),
					filepath.Join(homeDir, ".config", "app", "settings.conf"))

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Symlink should be removed
				assertNotExists(t, filepath.Join(homeDir, ".config", "app", "settings.conf"))
				// Empty parent directories should be cleaned up
				assertNotExists(t, filepath.Join(homeDir, ".config", "app"))
				assertNotExists(t, filepath.Join(homeDir, ".config"))
				// But the target dir itself (homeDir) should still exist
				assertDirExists(t, homeDir)
			},
		},
		{
			name: "non-empty parent directories preserved after removal",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create source file in nested directory
				createTestFile(t, filepath.Join(configRepo, ".config", "app", "managed.conf"), "# managed")

				// Create symlink
				createTestSymlink(t,
					filepath.Join(configRepo, ".config", "app", "managed.conf"),
					filepath.Join(homeDir, ".config", "app", "managed.conf"))

				// Create an unmanaged file in the same parent directory
				createTestFile(t, filepath.Join(homeDir, ".config", "app", "local.conf"), "# local")

				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      homeDir,
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Symlink should be removed
				assertNotExists(t, filepath.Join(homeDir, ".config", "app", "managed.conf"))
				// Parent directory should still exist because it has other files
				assertDirExists(t, filepath.Join(homeDir, ".config", "app"))
			},
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
