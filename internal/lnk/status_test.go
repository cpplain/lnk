package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusWithOptions(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(tmpDir string) LinkOptions
		wantError    bool
		wantContains []string
	}{
		{
			name: "single package with active links",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "home"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files
				os.WriteFile(filepath.Join(sourceDir, "home", ".bashrc"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(sourceDir, "home", ".vimrc"), []byte("test"), 0644)

				// Create symlinks
				createTestSymlink(t, filepath.Join(sourceDir, "home", ".bashrc"), filepath.Join(targetDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(sourceDir, "home", ".vimrc"), filepath.Join(targetDir, ".vimrc"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home"},
				}
			},
			wantError:    false,
			wantContains: []string{"active", ".bashrc", ".vimrc"},
		},
		{
			name: "multiple packages",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "home"), 0755)
				os.MkdirAll(filepath.Join(sourceDir, "work"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files
				os.WriteFile(filepath.Join(sourceDir, "home", ".bashrc"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(sourceDir, "work", ".gitconfig"), []byte("test"), 0644)

				// Create symlinks
				createTestSymlink(t, filepath.Join(sourceDir, "home", ".bashrc"), filepath.Join(targetDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(sourceDir, "work", ".gitconfig"), filepath.Join(targetDir, ".gitconfig"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home", "work"},
				}
			},
			wantError:    false,
			wantContains: []string{"active", ".bashrc", ".gitconfig"},
		},
		{
			name: "no matching links",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "home"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files but no symlinks
				os.WriteFile(filepath.Join(sourceDir, "home", ".bashrc"), []byte("test"), 0644)

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home"},
				}
			},
			wantError:    false,
			wantContains: []string{"No active links found"},
		},
		{
			name: "package with . (current directory)",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files directly in source dir (flat repo)
				os.WriteFile(filepath.Join(sourceDir, ".bashrc"), []byte("test"), 0644)

				// Create symlink
				createTestSymlink(t, filepath.Join(sourceDir, ".bashrc"), filepath.Join(targetDir, ".bashrc"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"."},
				}
			},
			wantError:    false,
			wantContains: []string{"active", ".bashrc"},
		},
		{
			name: "broken links",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "home"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create broken symlink (target doesn't exist)
				createTestSymlink(t, filepath.Join(sourceDir, "home", ".missing"), filepath.Join(targetDir, ".missing"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home"},
				}
			},
			wantError:    false,
			wantContains: []string{"broken", ".missing"},
		},
		{
			name: "partial status - only specified package",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "home"), 0755)
				os.MkdirAll(filepath.Join(sourceDir, "work"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files
				os.WriteFile(filepath.Join(sourceDir, "home", ".bashrc"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(sourceDir, "work", ".gitconfig"), []byte("test"), 0644)

				// Create symlinks for both packages
				createTestSymlink(t, filepath.Join(sourceDir, "home", ".bashrc"), filepath.Join(targetDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(sourceDir, "work", ".gitconfig"), filepath.Join(targetDir, ".gitconfig"))

				// Only ask for status of "home" package
				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home"},
				}
			},
			wantError: false,
			// Should contain bashrc but NOT gitconfig
			wantContains: []string{"active", ".bashrc"},
		},
		{
			name: "error - no packages specified",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{},
				}
			},
			wantError:    true,
			wantContains: []string{"no packages specified"},
		},
		{
			name: "error - source directory does not exist",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "nonexistent")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(targetDir, 0755)

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
					Packages:  []string{"home"},
				}
			},
			wantError:    true,
			wantContains: []string{"source directory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			opts := tt.setupFunc(tmpDir)

			// Capture output
			output := CaptureOutput(t, func() {
				err := StatusWithOptions(opts)
				if tt.wantError && err == nil {
					t.Errorf("StatusWithOptions() expected error but got nil")
				}
				if !tt.wantError && err != nil {
					t.Errorf("StatusWithOptions() unexpected error: %v", err)
				}

				// Check error message contains expected text
				if tt.wantError && err != nil {
					found := false
					for _, want := range tt.wantContains {
						if strings.Contains(err.Error(), want) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("StatusWithOptions() error = %v, want one of %v", err, tt.wantContains)
					}
				}
			})

			// Check output contains expected text (for non-error cases)
			if !tt.wantError {
				for _, want := range tt.wantContains {
					if !strings.Contains(output, want) {
						t.Errorf("StatusWithOptions() output missing %q\nGot:\n%s", want, output)
					}
				}

				// For partial status test, verify gitconfig is NOT present
				if tt.name == "partial status - only specified package" {
					if strings.Contains(output, ".gitconfig") {
						t.Errorf("StatusWithOptions() should not show .gitconfig for home package only")
					}
				}
			}
		})
	}
}
