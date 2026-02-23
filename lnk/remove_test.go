package lnk

import (
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
