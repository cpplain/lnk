package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrphanSingle(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, tmpDir string, configRepo string) string
		link          string
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, tmpDir string, configRepo string, link string)
	}{
		{
			name: "orphan valid symlink",
			setupFunc: func(t *testing.T, tmpDir string, configRepo string) string {
				// Create source file in config repo
				sourceFile := filepath.Join(configRepo, "home", "testfile")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("test content"), 0644)

				// Create symlink
				linkPath := filepath.Join(tmpDir, "testlink")
				os.Symlink(sourceFile, linkPath)

				return linkPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, configRepo string, link string) {
				// Link should be replaced with actual file
				info, err := os.Lstat(link)
				if err != nil {
					t.Fatalf("Failed to stat orphaned file: %v", err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("File is still a symlink after orphaning")
				}

				// Content should be preserved
				content, _ := os.ReadFile(link)
				if string(content) != "test content" {
					t.Errorf("File content mismatch: got %q, want %q", content, "test content")
				}

				// Source file should be removed
				sourceFile := filepath.Join(configRepo, "home", "testfile")
				if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
					t.Error("Source file still exists in repository")
				}
			},
		},
		{
			name: "orphan non-symlink",
			setupFunc: func(t *testing.T, tmpDir string, configRepo string) string {
				// Create regular file
				regularFile := filepath.Join(tmpDir, "regular.txt")
				os.WriteFile(regularFile, []byte("regular"), 0644)
				return regularFile
			},
			expectError:   true,
			errorContains: "not a symlink",
		},
		{
			name: "orphan symlink not managed by repo",
			setupFunc: func(t *testing.T, tmpDir string, configRepo string) string {
				// Create external file
				externalFile := filepath.Join(tmpDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)

				// Create symlink to external file
				linkPath := filepath.Join(tmpDir, "external-link")
				os.Symlink(externalFile, linkPath)

				return linkPath
			},
			expectError:   true,
			errorContains: "not managed by this repository",
		},
		{
			name: "orphan broken symlink",
			setupFunc: func(t *testing.T, tmpDir string, configRepo string) string {
				// Create symlink to non-existent file in repo
				targetPath := filepath.Join(configRepo, "home", "nonexistent")
				linkPath := filepath.Join(tmpDir, "broken-link")
				os.Symlink(targetPath, linkPath)

				return linkPath
			},
			expectError:   true,
			errorContains: "symlink target does not exist",
		},
		{
			name: "orphan symlink with private source",
			setupFunc: func(t *testing.T, tmpDir string, configRepo string) string {
				// Create source file in private area
				sourceFile := filepath.Join(configRepo, "private", "home", "secret")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("private content"), 0600)

				// Create symlink
				linkPath := filepath.Join(tmpDir, "secret-link")
				os.Symlink(sourceFile, linkPath)

				return linkPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, tmpDir string, configRepo string, link string) {
				// Verify file is orphaned with correct permissions
				info, _ := os.Stat(link)
				if info.Mode().Perm() != 0600 {
					t.Errorf("File permissions incorrect: got %v, want %v", info.Mode().Perm(), 0600)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			tmpDir, err := os.MkdirTemp("", "lnk-orphan-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			configRepo := filepath.Join(tmpDir, "config-repo")
			os.MkdirAll(configRepo, 0755)

			// Create config
			config := &Config{
				LinkMappings: []LinkMapping{
					{Source: filepath.Join(configRepo, "home"), Target: "~/"},
					{Source: filepath.Join(configRepo, "private/home"), Target: "~/"},
				},
			}

			// Setup test environment
			link := tt.link
			if tt.setupFunc != nil {
				link = tt.setupFunc(t, tmpDir, configRepo)
			}

			// Test orphan with confirmation bypassed
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			go func() {
				defer w.Close()
				w.Write([]byte("y\n"))
			}()
			defer func() { os.Stdin = oldStdin }()

			// Run orphan
			err = Orphan(link, config, false, true)

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
				tt.validateFunc(t, tmpDir, configRepo, link)
			}
		})
	}
}

func TestOrphanDirectoryFull(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "lnk-orphan-dir-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configRepo := filepath.Join(tmpDir, "config-repo")
	os.MkdirAll(configRepo, 0755)

	// Create config
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(configRepo, "home"), Target: "~/"},
		},
	}

	// Create source files in config repo
	file1 := filepath.Join(configRepo, "home", "dir1", "file1")
	file2 := filepath.Join(configRepo, "home", "dir2", "file2")
	os.MkdirAll(filepath.Dir(file1), 0755)
	os.MkdirAll(filepath.Dir(file2), 0755)
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Create directory with symlinks
	targetDir := filepath.Join(tmpDir, "target")
	os.MkdirAll(filepath.Join(targetDir, "subdir"), 0755)

	link1 := filepath.Join(targetDir, "link1")
	link2 := filepath.Join(targetDir, "subdir", "link2")
	os.Symlink(file1, link1)
	os.Symlink(file2, link2)

	// Also add a non-managed symlink and regular file
	externalFile := filepath.Join(tmpDir, "external")
	os.WriteFile(externalFile, []byte("external"), 0644)
	os.Symlink(externalFile, filepath.Join(targetDir, "external-link"))
	os.WriteFile(filepath.Join(targetDir, "regular.txt"), []byte("regular"), 0644)

	// Mock user confirmation
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer w.Close()
		w.Write([]byte("y\n"))
	}()
	defer func() { os.Stdin = oldStdin }()

	// Test orphan directory
	err = Orphan(targetDir, config, false, true)
	if err != nil {
		t.Fatalf("Orphan directory failed: %v", err)
	}

	// Validate results
	// Managed links should be orphaned
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
	for _, src := range []string{file1, file2} {
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Errorf("Source file %s still exists", src)
		}
	}

	// External symlink should remain unchanged
	extLink := filepath.Join(targetDir, "external-link")
	info, _ := os.Lstat(extLink)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("External symlink was modified")
	}

	// Regular file should remain unchanged
	regularFile := filepath.Join(targetDir, "regular.txt")
	content, _ := os.ReadFile(regularFile)
	if string(content) != "regular" {
		t.Error("Regular file was modified")
	}
}

func TestOrphanDryRunAdditional(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "lnk-orphan-dryrun-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configRepo := filepath.Join(tmpDir, "config-repo")
	os.MkdirAll(configRepo, 0755)

	// Create config
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(configRepo, "home"), Target: "~/"},
		},
	}

	// Create source file and symlink
	sourceFile := filepath.Join(configRepo, "home", "dryrun-test")
	os.MkdirAll(filepath.Dir(sourceFile), 0755)
	os.WriteFile(sourceFile, []byte("dry run content"), 0644)

	linkPath := filepath.Join(tmpDir, "dryrun-link")
	os.Symlink(sourceFile, linkPath)

	// Test dry run
	err = Orphan(linkPath, config, true, true)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify nothing changed
	// Link should still exist
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal("Link was removed during dry run")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Link was modified during dry run")
	}

	// Source file should still exist
	if _, err := os.Stat(sourceFile); err != nil {
		t.Error("Source file was removed during dry run")
	}
}

func TestOrphanErrors(t *testing.T) {
	tests := []struct {
		name          string
		link          string
		configRepo    string
		expectError   bool
		errorContains string
	}{
		{
			name:          "non-existent path",
			link:          "/non/existent/path",
			configRepo:    "/tmp",
			expectError:   true,
			errorContains: "no such file",
		},
		{
			name:          "symlink not managed by repo",
			link:          "/tmp",
			configRepo:    "/nonexistent/repo",
			expectError:   true,
			errorContains: "not managed by this repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}

			err := Orphan(tt.link, config, false, true)

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

func TestOrphanDirectoryNoSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	_ = filepath.Join(tempDir, "repo")

	// Create directories
	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)

	// Create only regular files
	os.WriteFile(filepath.Join(homeDir, ".config", "file.txt"), []byte("test"), 0644)

	// Test orphaning - should fail
	config := &Config{}
	err := Orphan(filepath.Join(homeDir, ".config"), config, false, true)
	if err == nil {
		t.Errorf("expected error when orphaning directory with no symlinks")
	}
	if !containsString(err.Error(), "no managed symlinks found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOrphanUntrackedFile(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

	// Create an untracked file in the repo
	targetPath := filepath.Join(configRepo, "home", ".untrackedfile")
	os.WriteFile(targetPath, []byte("untracked content"), 0644)

	// Create symlink to the untracked file
	linkPath := filepath.Join(homeDir, ".untrackedfile")
	os.Symlink(targetPath, linkPath)

	// Set up test environment to bypass confirmation prompts
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write "y" to simulate user confirmation
	go func() {
		defer w.Close()
		w.Write([]byte("y\n"))
	}()

	// Run orphan
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(configRepo, "home"), Target: "~/"},
		},
	}
	err := Orphan(linkPath, config, false, true)
	if err != nil {
		t.Fatalf("orphan failed: %v", err)
	}

	// Verify symlink is removed and replaced with regular file
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("failed to stat orphaned file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("file is still a symlink after orphan")
	}

	// Verify content was copied back
	content, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("failed to read orphaned file: %v", err)
	}
	if string(content) != "untracked content" {
		t.Errorf("content mismatch: got %q, want %q", string(content), "untracked content")
	}

	// Verify original file was removed from repository
	if _, err := os.Stat(targetPath); err == nil || !os.IsNotExist(err) {
		t.Errorf("untracked file was not removed from repository")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
