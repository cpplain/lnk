package cfgman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (src, dst string, cleanup func())
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, src, dst string)
	}{
		{
			name: "copy regular file",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "dest.txt")

				os.WriteFile(src, []byte("test content"), 0644)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, src, dst string) {
				// Check content
				srcContent, _ := os.ReadFile(src)
				dstContent, _ := os.ReadFile(dst)
				if string(dstContent) != string(srcContent) {
					t.Error("File content doesn't match")
				}

				// Check permissions
				srcInfo, _ := os.Stat(src)
				dstInfo, _ := os.Stat(dst)
				if dstInfo.Mode() != srcInfo.Mode() {
					t.Errorf("File mode doesn't match: got %v, want %v", dstInfo.Mode(), srcInfo.Mode())
				}
			},
		},
		{
			name: "copy file with special permissions",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "executable")
				dst := filepath.Join(tmpDir, "executable-copy")

				os.WriteFile(src, []byte("#!/bin/bash\necho test"), 0755)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, src, dst string) {
				info, _ := os.Stat(dst)
				if info.Mode().Perm() != 0755 {
					t.Errorf("Executable permission not preserved: got %v", info.Mode().Perm())
				}
			},
		},
		{
			name: "copy non-existent file",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "nonexistent")
				dst := filepath.Join(tmpDir, "dest")

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "failed to open source file",
		},
		{
			name: "copy to invalid destination",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "nonexistent", "dest.txt")

				os.WriteFile(src, []byte("content"), 0644)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "failed to create destination file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := copyFile(src, dst)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Error doesn't contain %q: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, src, dst)
				}
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (src, dst string, cleanup func())
		expectError  bool
		validateFunc func(t *testing.T, src, dst string)
	}{
		{
			name: "copy directory with files and subdirs",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "source-dir")
				dst := filepath.Join(tmpDir, "dest-dir")

				// Create directory structure
				os.MkdirAll(filepath.Join(src, "subdir"), 0755)
				os.WriteFile(filepath.Join(src, "file1.txt"), []byte("file1"), 0644)
				os.WriteFile(filepath.Join(src, "file2.txt"), []byte("file2"), 0600)
				os.WriteFile(filepath.Join(src, "subdir", "file3.txt"), []byte("file3"), 0644)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, src, dst string) {
				// Check all files exist
				files := []string{"file1.txt", "file2.txt", "subdir/file3.txt"}
				for _, file := range files {
					srcPath := filepath.Join(src, file)
					dstPath := filepath.Join(dst, file)

					srcContent, err := os.ReadFile(srcPath)
					if err != nil {
						t.Errorf("Failed to read source %s: %v", srcPath, err)
						continue
					}

					dstContent, err := os.ReadFile(dstPath)
					if err != nil {
						t.Errorf("Failed to read dest %s: %v", dstPath, err)
						continue
					}

					if string(srcContent) != string(dstContent) {
						t.Errorf("Content mismatch for %s", file)
					}

					// Check permissions
					srcInfo, _ := os.Stat(srcPath)
					dstInfo, _ := os.Stat(dstPath)
					if srcInfo.Mode() != dstInfo.Mode() {
						t.Errorf("Mode mismatch for %s: got %v, want %v", file, dstInfo.Mode(), srcInfo.Mode())
					}
				}
			},
		},
		{
			name: "copy empty directory",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "empty-src")
				dst := filepath.Join(tmpDir, "empty-dst")

				os.MkdirAll(src, 0755)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, src, dst string) {
				info, err := os.Stat(dst)
				if err != nil {
					t.Fatalf("Destination directory not created: %v", err)
				}
				if !info.IsDir() {
					t.Error("Destination is not a directory")
				}
			},
		},
		{
			name: "copy non-existent directory",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "nonexistent")
				dst := filepath.Join(tmpDir, "dest")

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := copyDir(src, dst)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, src, dst)
				}
			}
		})
	}
}

func TestCopyPathAdditional(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (src, dst string, cleanup func())
		expectError   bool
		errorContains string
	}{
		{
			name: "copy file via copyPath",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "file.txt")
				dst := filepath.Join(tmpDir, "copy.txt")

				os.WriteFile(src, []byte("content"), 0644)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "copy directory via copyPath",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "dir")
				dst := filepath.Join(tmpDir, "dir-copy")

				os.MkdirAll(src, 0755)
				os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0644)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "prevent copying directory into itself",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")
				src := filepath.Join(tmpDir, "dir")
				dst := filepath.Join(tmpDir, "dir", "subdir")

				os.MkdirAll(src, 0755)

				return src, dst, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "cannot copy directory into itself",
		},
		{
			name: "copy with relative paths",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-test")

				// Change to temp directory
				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)

				os.WriteFile("source.txt", []byte("relative"), 0644)

				return "source.txt", "dest.txt", func() {
					os.Chdir(oldWd)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := copyPath(src, dst)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Error doesn't contain %q: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
