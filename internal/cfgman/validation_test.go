package cfgman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateNoCircularSymlink(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (source, target string, cleanup func())
		expectError   bool
		errorContains string
	}{
		{
			name: "no circular reference - target doesn't exist",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "target.txt")

				os.WriteFile(source, []byte("content"), 0644)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "no circular reference - target is regular file",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "target.txt")

				os.WriteFile(source, []byte("source"), 0644)
				os.WriteFile(target, []byte("target"), 0644)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "circular reference detected",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "target-link")

				os.WriteFile(source, []byte("content"), 0644)
				// Create symlink that points back to source
				os.Symlink(source, target)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "would create circular symlink",
		},
		{
			name: "source inside target directory",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				targetDir := filepath.Join(tmpDir, "target-dir")
				source := filepath.Join(targetDir, "subdir", "source.txt")

				os.MkdirAll(filepath.Dir(source), 0755)
				os.WriteFile(source, []byte("content"), 0644)

				return source, targetDir, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "source is inside target directory",
		},
		{
			name: "target symlink points elsewhere",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "target-link")
				other := filepath.Join(tmpDir, "other.txt")

				os.WriteFile(source, []byte("source"), 0644)
				os.WriteFile(other, []byte("other"), 0644)
				// Target symlink points to other, not source
				os.Symlink(other, target)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "relative symlink circular reference",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validation-test")
				subdir := filepath.Join(tmpDir, "subdir")
				os.MkdirAll(subdir, 0755)

				source := filepath.Join(subdir, "source.txt")
				target := filepath.Join(tmpDir, "target-link")

				os.WriteFile(source, []byte("content"), 0644)
				// Create relative symlink back to source
				os.Symlink("subdir/source.txt", target)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "would create circular symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, target, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := ValidateNoCircularSymlink(source, target)

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

func TestValidateNoOverlappingPaths(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (source, target string, cleanup func())
		expectError   bool
		errorContains string
	}{
		{
			name: "no overlap - different paths",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")
				source := filepath.Join(tmpDir, "source", "file.txt")
				target := filepath.Join(tmpDir, "target", "file.txt")

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "same path",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")
				path := filepath.Join(tmpDir, "file.txt")

				return path, path, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "source and target are the same path",
		},
		{
			name: "source inside target",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")
				target := filepath.Join(tmpDir, "target-dir")
				source := filepath.Join(target, "subdir", "file.txt")

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "source path is inside target path",
		},
		{
			name: "target inside source",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")
				source := filepath.Join(tmpDir, "source-dir")
				target := filepath.Join(source, "subdir", "file.txt")

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "target path is inside source path",
		},
		{
			name: "sibling paths",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")
				source := filepath.Join(tmpDir, "dir1", "file.txt")
				target := filepath.Join(tmpDir, "dir2", "file.txt")

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "relative paths",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-overlap-test")

				// Change to temp directory
				oldWd, _ := os.Getwd()
				os.Chdir(tmpDir)

				return "source/file.txt", "target/file.txt", func() {
					os.Chdir(oldWd)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, target, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := ValidateNoOverlappingPaths(source, target)

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

func TestValidateSymlinkCreation(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (source, target string, cleanup func())
		expectError   bool
		errorContains string
	}{
		{
			name: "valid symlink creation",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validate-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "target.txt")

				os.WriteFile(source, []byte("content"), 0644)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
		{
			name: "circular symlink error",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validate-test")
				source := filepath.Join(tmpDir, "source.txt")
				target := filepath.Join(tmpDir, "circular-link")

				os.WriteFile(source, []byte("content"), 0644)
				os.Symlink(source, target)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "circular",
		},
		{
			name: "overlapping paths error",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-validate-test")
				path := filepath.Join(tmpDir, "same.txt")

				return path, path, func() { os.RemoveAll(tmpDir) }
			},
			expectError:   true,
			errorContains: "same path",
		},
		{
			name: "complex validation scenario",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-complex-test")

				// Create a directory structure
				sourceDir := filepath.Join(tmpDir, "configs")
				targetDir := filepath.Join(tmpDir, "home", ".config")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(filepath.Dir(targetDir), 0755)

				source := filepath.Join(sourceDir, "app.conf")
				target := filepath.Join(targetDir, "app.conf")

				os.WriteFile(source, []byte("config"), 0644)

				return source, target, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, target, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := ValidateSymlinkCreation(source, target)

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

func TestValidationErrorHandling(t *testing.T) {
	// Test error handling when paths cannot be resolved
	t.Run("stat error on target", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "cfgman-validation-error-test")
		defer os.RemoveAll(tmpDir)

		source := filepath.Join(tmpDir, "source")
		os.WriteFile(source, []byte("content"), 0644)

		// Use a path that will fail to stat (directory with no read permission)
		noAccessDir := filepath.Join(tmpDir, "noaccess")
		os.Mkdir(noAccessDir, 0000)
		defer os.Chmod(noAccessDir, 0755) // Ensure cleanup can remove it

		target := filepath.Join(noAccessDir, "target")

		// Should handle stat error gracefully
		err := ValidateNoCircularSymlink(source, target)
		// The function returns an error when it can't stat the target
		if err == nil || !containsString(err.Error(), "checking target") {
			t.Errorf("Expected checking target error, got: %v", err)
		}
	})

	t.Run("validation with relative paths", func(t *testing.T) {
		// ValidateNoOverlappingPaths should handle relative paths by converting to absolute
		err := ValidateNoOverlappingPaths("./source", "./source")
		if err == nil || !containsString(err.Error(), "same path") {
			t.Errorf("Expected same path error, got: %v", err)
		}
	})
}
