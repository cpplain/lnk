package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusWithLinkMappings(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	configRepo := filepath.Join(tmpDir, "dotfiles")
	homeDir := filepath.Join(tmpDir, "home")

	// Create directory structure
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)
	os.MkdirAll(filepath.Join(configRepo, "work"), 0755)
	os.MkdirAll(homeDir, 0755)

	// Create test files in different mappings
	homeFile := filepath.Join(configRepo, "home", ".bashrc")
	workFile := filepath.Join(configRepo, "work", ".gitconfig")
	os.WriteFile(homeFile, []byte("# bashrc"), 0644)
	os.WriteFile(workFile, []byte("# gitconfig"), 0644)

	// Create symlinks
	os.Symlink(homeFile, filepath.Join(homeDir, ".bashrc"))
	os.Symlink(workFile, filepath.Join(homeDir, ".gitconfig"))

	// Set test env to use our test home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	// Create config with mappings using absolute paths
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source: filepath.Join(configRepo, "home"),
				Target: "~/",
			},
			{
				Source: filepath.Join(configRepo, "work"),
				Target: "~/",
			},
		},
	}

	// Capture output
	output := CaptureOutput(t, func() {
		err := Status(config)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
	})

	// Debug: print the actual output
	t.Logf("Status output:\n%s", output)

	// Verify the output shows the active links (in simplified format when piped)
	if !strings.Contains(output, "active ~/.bashrc") {
		t.Errorf("Output should show active bashrc link")
	}
	if !strings.Contains(output, "active ~/.gitconfig") {
		t.Errorf("Output should show active gitconfig link")
	}

	// Verify output contains the files and paths
	// We no longer show source mappings in brackets since the full path shows the source
	if !strings.Contains(output, ".bashrc") {
		t.Errorf("Output should contain .bashrc")
	}
	if !strings.Contains(output, ".gitconfig") {
		t.Errorf("Output should contain .gitconfig")
	}

	// Removed directories linked as units section - no longer supported
}

func TestDetermineSourceMapping(t *testing.T) {
	configRepo := "/tmp/dotfiles"
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(configRepo, "home"), Target: "~/"},
			{Source: filepath.Join(configRepo, "work"), Target: "~/"},
			{Source: filepath.Join(configRepo, "private/home"), Target: "~/"},
		},
	}

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "home mapping",
			target:   "/tmp/dotfiles/home/.bashrc",
			expected: filepath.Join(configRepo, "home"),
		},
		{
			name:     "work mapping",
			target:   "/tmp/dotfiles/work/.gitconfig",
			expected: filepath.Join(configRepo, "work"),
		},
		{
			name:     "private/home mapping",
			target:   "/tmp/dotfiles/private/home/.ssh/config",
			expected: filepath.Join(configRepo, "private/home"),
		},
		{
			name:     "unknown mapping",
			target:   "/tmp/dotfiles/other/file",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineSourceMapping(tt.target, config)
			if result != tt.expected {
				t.Errorf("DetermineSourceMapping(%s) = %s; want %s", tt.target, result, tt.expected)
			}
		})
	}
}

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
