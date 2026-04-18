package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStatusBrokenLinksToStdout verifies broken links are printed to stdout, not stderr.
// The spec (status.md §5 step 3) requires broken links to be informational output on stdout.
func TestStatusBrokenLinksToStdout(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "dotfiles")
	targetDir := filepath.Join(tmpDir, "home")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create broken symlink (source file does not exist)
	createTestSymlink(t, filepath.Join(sourceDir, ".missing"), filepath.Join(targetDir, ".missing"))

	opts := LinkOptions{SourceDir: sourceDir, TargetDir: targetDir}

	stdout, stderr := captureOutput(t, func() {
		if err := Status(opts); err != nil {
			t.Fatalf("Status() unexpected error: %v", err)
		}
	})

	// "broken" output must appear on stdout (piped format: "broken <path>")
	if !strings.Contains(stdout, "broken") {
		t.Errorf("Status() broken link not in stdout\nstdout: %q\nstderr: %q", stdout, stderr)
	}
	if !strings.Contains(stdout, ".missing") {
		t.Errorf("Status() broken link path not in stdout\nstdout: %q", stdout)
	}

	// "broken" must NOT appear on stderr — broken links are informational, not errors
	if strings.Contains(stderr, "broken") || strings.Contains(stderr, ".missing") {
		t.Errorf("Status() broken link must not go to stderr\nstderr: %q", stderr)
	}
}

// TestStatusEmptyResultMessage verifies the exact message when no managed links exist.
func TestStatusEmptyResultMessage(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "dotfiles")
	targetDir := filepath.Join(tmpDir, "home")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	opts := LinkOptions{SourceDir: sourceDir, TargetDir: targetDir}

	stdout, _ := captureOutput(t, func() {
		if err := Status(opts); err != nil {
			t.Fatalf("Status() unexpected error: %v", err)
		}
	})

	want := "No managed links found."
	if !strings.Contains(stdout, want) {
		t.Errorf("Status() empty result = %q, want to contain %q", stdout, want)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(tmpDir string) LinkOptions
		wantError    bool
		wantContains []string
	}{
		{
			name: "single source directory with active links",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files
				os.WriteFile(filepath.Join(sourceDir, ".bashrc"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(sourceDir, ".vimrc"), []byte("test"), 0644)

				// Create symlinks
				createTestSymlink(t, filepath.Join(sourceDir, ".bashrc"), filepath.Join(targetDir, ".bashrc"))
				createTestSymlink(t, filepath.Join(sourceDir, ".vimrc"), filepath.Join(targetDir, ".vimrc"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
				}
			},
			wantError:    false,
			wantContains: []string{"active", ".bashrc", ".vimrc"},
		},
		{
			name: "nested subdirectories",
			setupFunc: func(tmpDir string) LinkOptions {
				sourceDir := filepath.Join(tmpDir, "dotfiles")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(filepath.Join(sourceDir, "subdir1"), 0755)
				os.MkdirAll(filepath.Join(sourceDir, "subdir2"), 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files in subdirectories
				os.WriteFile(filepath.Join(sourceDir, "subdir1", ".bashrc"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(sourceDir, "subdir2", ".gitconfig"), []byte("test"), 0644)

				// Create symlinks (preserving directory structure)
				createTestSymlink(t, filepath.Join(sourceDir, "subdir1", ".bashrc"), filepath.Join(targetDir, "subdir1", ".bashrc"))
				createTestSymlink(t, filepath.Join(sourceDir, "subdir2", ".gitconfig"), filepath.Join(targetDir, "subdir2", ".gitconfig"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
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
				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create source files but no symlinks
				os.WriteFile(filepath.Join(sourceDir, ".bashrc"), []byte("test"), 0644)

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
				}
			},
			wantError:    false,
			wantContains: []string{"No managed links found."},
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
				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create broken symlink (target doesn't exist)
				createTestSymlink(t, filepath.Join(sourceDir, ".missing"), filepath.Join(targetDir, ".missing"))

				return LinkOptions{
					SourceDir: sourceDir,
					TargetDir: targetDir,
				}
			},
			wantError:    false,
			wantContains: []string{"broken", ".missing"},
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
				err := Status(opts)
				if tt.wantError && err == nil {
					t.Errorf("Status() expected error but got nil")
				}
				if !tt.wantError && err != nil {
					t.Errorf("Status() unexpected error: %v", err)
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
						t.Errorf("Status() error = %v, want one of %v", err, tt.wantContains)
					}
				}
			})

			// Check output contains expected text (for non-error cases)
			if !tt.wantError {
				for _, want := range tt.wantContains {
					if !strings.Contains(output, want) {
						t.Errorf("Status() output missing %q\nGot:\n%s", want, output)
					}
				}

				// For partial status test, verify gitconfig is NOT present
				if tt.name == "partial status - only specified package" {
					if strings.Contains(output, ".gitconfig") {
						t.Errorf("Status() should not show .gitconfig for home package only")
					}
				}
			}
		})
	}
}
