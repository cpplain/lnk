package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdoptWithOptions tests the new AdoptWithOptions function
func TestAdoptWithOptions(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    map[string]string // files to create in target dir
		package_      string            // package to adopt into
		paths         []string          // paths to adopt (relative to target)
		expectError   bool
		errorContains string
	}{
		{
			name: "adopt single file to package",
			setupFiles: map[string]string{
				".bashrc": "bash config",
			},
			package_: "home",
			paths:    []string{".bashrc"},
		},
		{
			name: "adopt multiple files to package",
			setupFiles: map[string]string{
				".bashrc": "bash config",
				".vimrc":  "vim config",
			},
			package_: "home",
			paths:    []string{".bashrc", ".vimrc"},
		},
		{
			name: "adopt file to flat repository (.)",
			setupFiles: map[string]string{
				".zshrc": "zsh config",
			},
			package_: ".",
			paths:    []string{".zshrc"},
		},
		{
			name: "adopt nested file",
			setupFiles: map[string]string{
				".config/nvim/init.vim": "nvim config",
			},
			package_: "home",
			paths:    []string{".config/nvim/init.vim"},
		},
		{
			name:          "error: no package specified",
			package_:      "",
			paths:         []string{".bashrc"},
			expectError:   true,
			errorContains: "package argument is required",
		},
		{
			name:          "error: no paths specified",
			package_:      "home",
			paths:         []string{},
			expectError:   true,
			errorContains: "at least one file path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tempDir := t.TempDir()
			sourceDir := filepath.Join(tempDir, "dotfiles")
			targetDir := filepath.Join(tempDir, "target")

			os.MkdirAll(sourceDir, 0755)
			os.MkdirAll(targetDir, 0755)

			// Setup test files
			var absPaths []string
			for relPath, content := range tt.setupFiles {
				fullPath := filepath.Join(targetDir, relPath)
				os.MkdirAll(filepath.Dir(fullPath), 0755)
				os.WriteFile(fullPath, []byte(content), 0644)
				absPaths = append(absPaths, fullPath)
			}

			// Build absolute paths for adoption
			adoptPaths := make([]string, len(tt.paths))
			for i, relPath := range tt.paths {
				adoptPaths[i] = filepath.Join(targetDir, relPath)
			}

			// Run AdoptWithOptions
			opts := AdoptOptions{
				SourceDir: sourceDir,
				TargetDir: targetDir,
				Package:   tt.package_,
				Paths:     adoptPaths,
				DryRun:    false,
			}
			err := AdoptWithOptions(opts)

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify results
			for relPath := range tt.setupFiles {
				filePath := filepath.Join(targetDir, relPath)

				// Verify file is now a symlink
				linkInfo, err := os.Lstat(filePath)
				if err != nil {
					t.Errorf("failed to stat adopted file %s: %v", relPath, err)
					continue
				}
				if linkInfo.Mode()&os.ModeSymlink == 0 {
					t.Errorf("expected %s to be symlink, got regular file", relPath)
					continue
				}

				// Verify target exists in package
				var expectedTarget string
				if tt.package_ == "." {
					expectedTarget = filepath.Join(sourceDir, relPath)
				} else {
					expectedTarget = filepath.Join(sourceDir, tt.package_, relPath)
				}

				if _, err := os.Stat(expectedTarget); err != nil {
					t.Errorf("target not found in package for %s: %v", relPath, err)
				}

				// Verify symlink points to correct location
				target, err := os.Readlink(filePath)
				if err != nil {
					t.Errorf("failed to read symlink %s: %v", relPath, err)
					continue
				}
				if target != expectedTarget {
					t.Errorf("symlink %s points to wrong location: got %s, want %s", relPath, target, expectedTarget)
				}
			}
		})
	}
}

// TestAdoptWithOptionsDryRun tests dry-run mode
func TestAdoptWithOptionsDryRun(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	testFile := filepath.Join(targetDir, ".testfile")
	os.WriteFile(testFile, []byte("test content"), 0644)

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Package:   "home",
		Paths:     []string{testFile},
		DryRun:    true,
	}

	err := AdoptWithOptions(opts)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Verify nothing was changed
	info, err := os.Lstat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("file was converted to symlink in dry-run mode")
	}

	// Verify file wasn't moved to package
	targetPath := filepath.Join(sourceDir, "home", ".testfile")
	if _, err := os.Stat(targetPath); err == nil {
		t.Errorf("file was moved to package in dry-run mode")
	}
}

// TestAdoptWithOptionsSourceDirNotExist tests error when source dir doesn't exist
func TestAdoptWithOptionsSourceDirNotExist(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(targetDir, 0755)

	testFile := filepath.Join(targetDir, ".testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	opts := AdoptOptions{
		SourceDir: filepath.Join(tempDir, "nonexistent"),
		TargetDir: targetDir,
		Package:   "home",
		Paths:     []string{testFile},
		DryRun:    false,
	}

	err := AdoptWithOptions(opts)
	if err == nil {
		t.Errorf("expected error for nonexistent source directory")
	} else if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error about nonexistent directory, got: %v", err)
	}
}

