package cfgman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdopt tests the Adopt function
func TestAdopt(t *testing.T) {
	tests := []struct {
		name          string
		isPrivate     bool
		createFile    bool
		createDir     bool
		alreadyLink   bool
		expectError   bool
		errorContains string
	}{
		{
			name:       "adopt regular file to home",
			createFile: true,
			isPrivate:  false,
		},
		{
			name:       "adopt regular file to private_home",
			createFile: true,
			isPrivate:  true,
		},
		{
			name:      "adopt directory to home",
			createDir: true,
			isPrivate: false,
		},
		{
			name:      "adopt directory to private_home",
			createDir: true,
			isPrivate: true,
		},
		{
			name:          "adopt non-existent file",
			expectError:   true,
			errorContains: "no such file",
		},
		{
			name:          "adopt already managed file",
			createFile:    true,
			alreadyLink:   true,
			expectError:   true,
			errorContains: "already adopted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tempDir := t.TempDir()
			homeDir := filepath.Join(tempDir, "home")
			configRepo := filepath.Join(tempDir, "repo")

			// Create directories
			os.MkdirAll(homeDir, 0755)
			os.MkdirAll(filepath.Join(configRepo, "home"), 0755)
			os.MkdirAll(filepath.Join(configRepo, "private", "home"), 0755)

			// Create test config with default mappings
			config := &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: "~/"},
					{Source: "private/home", Target: "~/"},
				},
			}

			// Setup test file/directory
			testPath := filepath.Join(homeDir, ".testfile")
			if tt.createDir {
				testPath = filepath.Join(homeDir, ".testdir")
				os.MkdirAll(testPath, 0755)
				// Create a file inside the directory
				os.WriteFile(filepath.Join(testPath, "file.txt"), []byte("test content"), 0644)
			} else if tt.createFile {
				os.WriteFile(testPath, []byte("test content"), 0644)
			}

			// If already linked, set it up
			if tt.alreadyLink && tt.createFile {
				targetPath := filepath.Join(configRepo, "home", ".testfile")
				os.MkdirAll(filepath.Dir(targetPath), 0755)
				os.Rename(testPath, targetPath)
				os.Symlink(targetPath, testPath)
			}

			// Change to home directory for testing
			oldDir, _ := os.Getwd()
			os.Chdir(homeDir)
			defer os.Chdir(oldDir)

			// Run adopt (set HOME to our test home dir)
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			// Determine source directory based on isPrivate flag
			sourceDir := "home"
			if tt.isPrivate {
				sourceDir = "private/home"
			}
			err := Adopt(testPath, configRepo, config, sourceDir, false)

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify results based on whether it was a file or directory
			repoSubdir := "home"
			if tt.isPrivate {
				repoSubdir = filepath.Join("private", "home")
			}

			if tt.createDir {
				// For directories, verify the directory itself is NOT a symlink
				dirInfo, err := os.Lstat(testPath)
				if err != nil {
					t.Fatalf("failed to stat adopted directory: %v", err)
				}
				if dirInfo.Mode()&os.ModeSymlink != 0 {
					t.Errorf("expected regular directory, got symlink")
				}

				// Verify the file inside is a symlink
				filePath := filepath.Join(testPath, "file.txt")
				fileInfo, err := os.Lstat(filePath)
				if err != nil {
					t.Fatalf("failed to stat file in adopted directory: %v", err)
				}
				if fileInfo.Mode()&os.ModeSymlink == 0 {
					t.Errorf("expected file to be symlink, got regular file")
				}

				// Verify symlink points to correct location in repo
				targetPath := filepath.Join(configRepo, repoSubdir, filepath.Base(testPath), "file.txt")
				target, err := os.Readlink(filePath)
				if err != nil {
					t.Fatalf("failed to read file symlink: %v", err)
				}
				if target != targetPath {
					t.Errorf("file symlink points to wrong location: got %s, want %s", target, targetPath)
				}

				// Verify content is accessible through symlink
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("failed to read file through symlink: %v", err)
				}
				if string(content) != "test content" {
					t.Errorf("file content mismatch: got %s, want 'test content'", string(content))
				}
			} else {
				// For files, verify symlink was created
				linkInfo, err := os.Lstat(testPath)
				if err != nil {
					t.Fatalf("failed to stat adopted file: %v", err)
				}
				if linkInfo.Mode()&os.ModeSymlink == 0 {
					t.Errorf("expected symlink, got regular file")
				}

				// Verify target exists in repo
				targetPath := filepath.Join(configRepo, repoSubdir, filepath.Base(testPath))
				if _, err := os.Stat(targetPath); err != nil {
					t.Errorf("target not found in repo: %v", err)
				}

				// Verify symlink points to correct location
				target, err := os.Readlink(testPath)
				if err != nil {
					t.Fatalf("failed to read symlink: %v", err)
				}
				if target != targetPath {
					t.Errorf("symlink points to wrong location: got %s, want %s", target, targetPath)
				}
			}
		})
	}
}

// TestAdoptDryRun tests the dry-run functionality
func TestAdoptDryRun(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(configRepo, 0755)

	testFile := filepath.Join(homeDir, ".testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/"},
		},
	}

	// Run adopt in dry-run mode
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	err := Adopt(testFile, configRepo, config, "home", true)
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

	// Verify file wasn't moved to repo
	targetPath := filepath.Join(configRepo, "home", ".testfile")
	if _, err := os.Stat(targetPath); err == nil {
		t.Errorf("file was moved to repo in dry-run mode")
	}
}

// TestAdoptComplexDirectory tests adopting a directory with subdirectories and multiple files
func TestAdoptComplexDirectory(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	// Create directories
	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

	// Create test config
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/"},
		},
	}

	// Create a complex directory structure
	testDir := filepath.Join(homeDir, ".config", "myapp")
	os.MkdirAll(filepath.Join(testDir, "subdir1"), 0755)
	os.MkdirAll(filepath.Join(testDir, "subdir2", "nested"), 0755)

	// Create various files
	files := map[string]string{
		"config.toml":                  "main config",
		"settings.json":                "settings",
		"subdir1/file1.txt":            "file1 content",
		"subdir1/file2.txt":            "file2 content",
		"subdir2/data.xml":             "xml data",
		"subdir2/nested/deep_file.txt": "deep content",
	}

	for path, content := range files {
		fullPath := filepath.Join(testDir, path)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Set HOME environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	// Adopt the directory
	err := Adopt(testDir, configRepo, config, "home", false)
	if err != nil {
		t.Fatalf("failed to adopt complex directory: %v", err)
	}

	// Verify the directory structure
	// 1. Original directory should exist and be a regular directory
	dirInfo, err := os.Lstat(testDir)
	if err != nil {
		t.Fatalf("failed to stat adopted directory: %v", err)
	}
	if dirInfo.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected regular directory, got symlink")
	}

	// 2. Subdirectories should exist and be regular directories
	subdir1Info, err := os.Lstat(filepath.Join(testDir, "subdir1"))
	if err != nil {
		t.Fatalf("failed to stat subdir1: %v", err)
	}
	if subdir1Info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected subdir1 to be regular directory, got symlink")
	}

	// 3. Each file should be a symlink pointing to the correct location
	for path, expectedContent := range files {
		filePath := filepath.Join(testDir, path)

		// Check if it's a symlink
		fileInfo, err := os.Lstat(filePath)
		if err != nil {
			t.Errorf("failed to stat %s: %v", path, err)
			continue
		}
		if fileInfo.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s to be symlink, got regular file", path)
			continue
		}

		// Verify symlink target
		expectedTarget := filepath.Join(configRepo, "home", ".config", "myapp", path)
		target, err := os.Readlink(filePath)
		if err != nil {
			t.Errorf("failed to read symlink %s: %v", path, err)
			continue
		}
		if target != expectedTarget {
			t.Errorf("symlink %s points to wrong location: got %s, want %s", path, target, expectedTarget)
		}

		// Verify content is accessible through symlink
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read %s through symlink: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("content mismatch for %s: got %s, want %s", path, string(content), expectedContent)
		}
	}

	// 4. Verify all files exist in the repository
	for path := range files {
		repoPath := filepath.Join(configRepo, "home", ".config", "myapp", path)
		if _, err := os.Stat(repoPath); err != nil {
			t.Errorf("file %s not found in repository: %v", path, err)
		}
	}
}
