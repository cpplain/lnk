package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestOrphanWithOptions(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string
		paths         []string
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string)
	}{
		{
			name: "orphan single file",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create source file in repo
				sourceFile := filepath.Join(sourceDir, ".bashrc")
				os.WriteFile(sourceFile, []byte("test content"), 0644)

				// Create symlink
				linkPath := filepath.Join(targetDir, ".bashrc")
				os.Symlink(sourceFile, linkPath)

				return []string{linkPath}
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				linkPath := paths[0]

				// Link should be replaced with actual file
				info, err := os.Lstat(linkPath)
				if err != nil {
					t.Fatalf("Failed to stat orphaned file: %v", err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("File is still a symlink after orphaning")
				}

				// Content should be preserved
				content, _ := os.ReadFile(linkPath)
				if string(content) != "test content" {
					t.Errorf("File content mismatch: got %q, want %q", content, "test content")
				}

				// Source file should be removed
				sourceFile := filepath.Join(sourceDir, ".bashrc")
				if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
					t.Error("Source file still exists in repository")
				}
			},
		},
		{
			name: "orphan multiple files",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create source files
				file1 := filepath.Join(sourceDir, ".bashrc")
				file2 := filepath.Join(sourceDir, ".vimrc")
				os.WriteFile(file1, []byte("bash"), 0644)
				os.WriteFile(file2, []byte("vim"), 0644)

				// Create symlinks
				link1 := filepath.Join(targetDir, ".bashrc")
				link2 := filepath.Join(targetDir, ".vimrc")
				os.Symlink(file1, link1)
				os.Symlink(file2, link2)

				return []string{link1, link2}
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				// Both links should be replaced with actual files
				for i, linkPath := range paths {
					info, err := os.Lstat(linkPath)
					if err != nil {
						t.Errorf("Failed to stat orphaned file %d: %v", i, err)
						continue
					}
					if info.Mode()&os.ModeSymlink != 0 {
						t.Errorf("File %d is still a symlink after orphaning", i)
					}
				}

				// Source files should be removed
				for _, filename := range []string{".bashrc", ".vimrc"} {
					sourceFile := filepath.Join(sourceDir, filename)
					if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
						t.Errorf("Source file %s still exists in repository", filename)
					}
				}
			},
		},
		{
			name: "orphan with dry-run",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create source file
				sourceFile := filepath.Join(sourceDir, ".testfile")
				os.WriteFile(sourceFile, []byte("test"), 0644)

				// Create symlink
				linkPath := filepath.Join(targetDir, ".testfile")
				os.Symlink(sourceFile, linkPath)

				return []string{linkPath}
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				linkPath := paths[0]

				// Link should still exist
				info, err := os.Lstat(linkPath)
				if err != nil {
					t.Fatal("Link was removed during dry run")
				}
				if info.Mode()&os.ModeSymlink == 0 {
					t.Error("Link was modified during dry run")
				}

				// Source file should still exist
				sourceFile := filepath.Join(sourceDir, ".testfile")
				if _, err := os.Stat(sourceFile); err != nil {
					t.Error("Source file was removed during dry run")
				}
			},
		},
		{
			name: "orphan non-symlink",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create regular file
				regularFile := filepath.Join(targetDir, "regular.txt")
				os.WriteFile(regularFile, []byte("regular"), 0644)

				return []string{regularFile}
			},
			expectError: false, // Continues processing, returns nil (graceful error handling)
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				// Regular file should not be modified
				regularFile := paths[0]
				content, _ := os.ReadFile(regularFile)
				if string(content) != "regular" {
					t.Error("Regular file was modified")
				}
			},
		},
		{
			name: "orphan unmanaged symlink",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create external file
				externalFile := filepath.Join(tmpDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)

				// Create symlink to external file
				linkPath := filepath.Join(targetDir, "external-link")
				os.Symlink(externalFile, linkPath)

				return []string{linkPath}
			},
			expectError: false, // Continues processing, returns nil (graceful error handling)
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				// External symlink should remain unchanged
				linkPath := paths[0]
				info, _ := os.Lstat(linkPath)
				if info.Mode()&os.ModeSymlink == 0 {
					t.Error("External symlink was modified")
				}
			},
		},
		{
			name: "orphan directory with managed links",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create source files
				file1 := filepath.Join(sourceDir, "file1")
				file2 := filepath.Join(sourceDir, "subdir", "file2")
				os.MkdirAll(filepath.Dir(file2), 0755)
				os.WriteFile(file1, []byte("content1"), 0644)
				os.WriteFile(file2, []byte("content2"), 0644)

				// Create symlinks in target directory
				os.MkdirAll(filepath.Join(targetDir, "orphan-dir", "subdir"), 0755)
				link1 := filepath.Join(targetDir, "orphan-dir", "file1")
				link2 := filepath.Join(targetDir, "orphan-dir", "subdir", "file2")
				os.Symlink(file1, link1)
				os.Symlink(file2, link2)

				return []string{filepath.Join(targetDir, "orphan-dir")}
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string, paths []string) {
				// Both links should be orphaned
				dirPath := paths[0]
				link1 := filepath.Join(dirPath, "file1")
				link2 := filepath.Join(dirPath, "subdir", "file2")

				for _, link := range []string{link1, link2} {
					info, err := os.Lstat(link)
					if err != nil {
						t.Errorf("Failed to stat %s: %v", link, err)
						continue
					}
					if info.Mode()&os.ModeSymlink != 0 {
						t.Errorf("%s is still a symlink", link)
					}
				}

				// Source files should be removed
				for _, file := range []string{"file1", filepath.Join("subdir", "file2")} {
					sourceFile := filepath.Join(sourceDir, file)
					if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
						t.Errorf("Source file %s still exists", file)
					}
				}
			},
		},
		{
			name: "error: no paths specified",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				return []string{} // No paths
			},
			paths:         []string{},
			expectError:   true,
			errorContains: "at least one file path is required",
		},
		{
			name: "error: source directory does not exist",
			setupFunc: func(t *testing.T, tmpDir string, sourceDir string, targetDir string) []string {
				// Create a symlink in target
				linkPath := filepath.Join(targetDir, ".bashrc")
				os.Symlink("/nonexistent/file", linkPath)

				return []string{linkPath}
			},
			expectError:   true,
			errorContains: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tmpDir := t.TempDir()
			sourceDir := filepath.Join(tmpDir, "dotfiles")
			targetDir := filepath.Join(tmpDir, "target")

			// Only create source dir for non-error tests
			if !tt.expectError || !strings.Contains(tt.errorContains, "does not exist") {
				os.MkdirAll(sourceDir, 0755)
			}
			os.MkdirAll(targetDir, 0755)

			// Setup test environment
			var paths []string
			if tt.setupFunc != nil {
				paths = tt.setupFunc(t, tmpDir, sourceDir, targetDir)
			}
			if tt.paths != nil {
				paths = tt.paths
			}

			// Determine if this is a dry-run test
			dryRun := strings.Contains(tt.name, "dry-run")

			// Run orphan
			opts := OrphanOptions{
				SourceDir: sourceDir,
				TargetDir: targetDir,
				Paths:     paths,
				DryRun:    dryRun,
			}

			// Special handling for source dir not exist test
			if tt.expectError && strings.Contains(tt.errorContains, "does not exist") {
				opts.SourceDir = "/nonexistent/dotfiles"
			}

			err := OrphanWithOptions(opts)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Error message doesn't contain %q: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Run validation
			if !tt.expectError && tt.validateFunc != nil {
				tt.validateFunc(t, tmpDir, sourceDir, targetDir, paths)
			}
		})
	}
}

func TestOrphanWithOptionsBrokenLink(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "dotfiles")
	targetDir := filepath.Join(tmpDir, "target")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create symlink to non-existent file in repo
	targetPath := filepath.Join(sourceDir, "nonexistent")
	linkPath := filepath.Join(targetDir, ".broken-link")
	os.Symlink(targetPath, linkPath)

	// Run orphan
	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
		DryRun:    false,
	}

	err := OrphanWithOptions(opts)

	// Should return nil (graceful error handling) but not orphan the broken link
	if err != nil {
		t.Errorf("Expected nil error for broken link, got: %v", err)
	}

	// Broken link should still exist (not orphaned)
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal("Broken link was removed (should have been skipped)")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Broken link was modified (should have been skipped)")
	}
}
