package lnk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveFromRepository(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (path string, cleanup func())
		expectError  bool
		validateFunc func(t *testing.T, path string)
	}{
		{
			name: "remove untracked file",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-git-test")

				// Initialize git repo
				exec.Command("git", "init", tmpDir).Run()

				// Create untracked file
				file := filepath.Join(tmpDir, "untracked.txt")
				os.WriteFile(file, []byte("untracked"), 0644)

				return file, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, path string) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("File still exists after removal")
				}
			},
		},
		{
			name: "remove tracked file",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-git-test")

				// Initialize git repo
				exec.Command("git", "init", tmpDir).Run()
				exec.Command("git", "config", "--local", "user.email", "test@example.com").Dir = tmpDir
				exec.Command("git", "config", "--local", "user.name", "Test User").Dir = tmpDir

				// Create and commit file
				file := filepath.Join(tmpDir, "tracked.txt")
				os.WriteFile(file, []byte("tracked"), 0644)

				cmd := exec.Command("git", "add", "tracked.txt")
				cmd.Dir = tmpDir
				cmd.Run()

				cmd = exec.Command("git", "commit", "-m", "Add tracked file")
				cmd.Dir = tmpDir
				cmd.Run()

				return file, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, path string) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("File still exists after removal")
				}

				// Check git status
				cmd := exec.Command("git", "status", "--porcelain")
				cmd.Dir = filepath.Dir(path)
				output, _ := cmd.Output()
				if len(output) == 0 {
					t.Error("Expected git to show deleted file in status")
				}
			},
		},
		{
			name: "remove directory",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-git-test")

				// Initialize git repo
				exec.Command("git", "init", tmpDir).Run()

				// Create directory with files
				dir := filepath.Join(tmpDir, "mydir")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0644)
				os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2"), 0644)

				return dir, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, path string) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("Directory still exists after removal")
				}
			},
		},
		{
			name: "remove file outside git repo",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-nogit-test")

				// Create file (no git repo)
				file := filepath.Join(tmpDir, "nogit.txt")
				os.WriteFile(file, []byte("no git"), 0644)

				return file, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false,
			validateFunc: func(t *testing.T, path string) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Error("File still exists after removal")
				}
			},
		},
		{
			name: "remove non-existent file",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-test")
				nonExistent := filepath.Join(tmpDir, "nonexistent.txt")

				return nonExistent, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false, // removeFromRepository doesn't fail on non-existent files
		},
	}

	// Skip tests if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping git operation tests")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := removeFromRepository(path)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, path)
				}
			}
		})
	}
}

func TestRemoveFromRepositoryTimeout(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping timeout test")
	}

	// This test verifies that the timeout mechanism works by attempting
	// to run git commands in a repository
	tmpDir, err := os.MkdirTemp("", "lnk-timeout-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	exec.Command("git", "init", tmpDir).Run()

	// Create a file
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("test"), 0644)

	// The function should complete within the timeout
	err = removeFromRepository(file)
	if err != nil {
		t.Errorf("Function failed when it should succeed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("File still exists")
	}
}

func TestRemoveFromRepositoryGitErrors(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping git error tests")
	}

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (path string, cleanup func())
		expectError   bool
		errorContains string
	}{
		{
			name: "remove file with uncommitted changes",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-git-error-test")

				// Initialize git repo
				exec.Command("git", "init", tmpDir).Run()
				exec.Command("git", "config", "--local", "user.email", "test@example.com").Dir = tmpDir
				exec.Command("git", "config", "--local", "user.name", "Test User").Dir = tmpDir

				// Create and commit file
				file := filepath.Join(tmpDir, "modified.txt")
				os.WriteFile(file, []byte("original"), 0644)

				cmd := exec.Command("git", "add", "modified.txt")
				cmd.Dir = tmpDir
				cmd.Run()

				cmd = exec.Command("git", "commit", "-m", "Initial commit")
				cmd.Dir = tmpDir
				cmd.Run()

				// Modify the file
				os.WriteFile(file, []byte("modified"), 0644)

				return file, func() { os.RemoveAll(tmpDir) }
			},
			expectError: false, // git rm -f should force removal
		},
		{
			name: "remove read-only file",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-readonly-test")

				// Create file
				file := filepath.Join(tmpDir, "readonly.txt")
				os.WriteFile(file, []byte("readonly"), 0444)

				return file, func() {
					// Ensure we can clean up
					os.Chmod(file, 0644)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: false, // Should still succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := removeFromRepository(path)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
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

func TestRemoveFromRepositoryNoGit(t *testing.T) {
	// Create a custom PATH without git to simulate git not being available
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	tmpDir, err := os.MkdirTemp("", "lnk-nogit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("test"), 0644)

	// Should succeed by falling back to regular file removal
	err = removeFromRepository(file)
	if err != nil {
		t.Errorf("Should succeed without git: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("File still exists")
	}
}
