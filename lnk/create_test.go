package lnk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (configRepo string, opts LinkOptions)
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "single source directory",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				linkPath := filepath.Join(tmpDir, "home", ".bashrc")
				assertSymlink(t, linkPath, filepath.Join(configRepo, ".bashrc"))
			},
		},
		{
			name: "multiple files",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
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
			name: "package with dot (current directory)",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
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
			name: "nested directory structure",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".ssh", "config"), "# ssh config")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".ssh", "config"), filepath.Join(configRepo, ".ssh", "config"))
			},
		},
		{
			name: "ignore patterns",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "README.md"), "# readme")
				createTestFile(t, filepath.Join(configRepo, ".vimrc"), "# vimrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					IgnorePatterns: []string{"README.md"},
					DryRun:         false,
				}
			},
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				assertSymlink(t, filepath.Join(tmpDir, "home", ".bashrc"), filepath.Join(configRepo, ".bashrc"))
				assertSymlink(t, filepath.Join(tmpDir, "home", ".vimrc"), filepath.Join(configRepo, ".vimrc"))
				assertNotExists(t, filepath.Join(tmpDir, "home", "README.md"))
			},
		},
		{
			name: "dry run mode",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, ".bashrc"), "# bashrc")
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
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
			name: "empty source directory",
			setup: func(t *testing.T, tmpDir string) (string, LinkOptions) {
				configRepo := filepath.Join(tmpDir, "repo")
				os.MkdirAll(configRepo, 0755)
				return configRepo, LinkOptions{
					SourceDir:      configRepo,
					TargetDir:      filepath.Join(tmpDir, "home"),
					IgnorePatterns: []string{},
					DryRun:         false,
				}
			},
			wantErr: false, // Gracefully handles empty directory
		},
		{
			name: "source directory does not exist",
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
