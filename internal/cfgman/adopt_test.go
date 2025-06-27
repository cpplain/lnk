package cfgman

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdopt tests the Adopt function
func TestAdopt(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)
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
			errorContains: "does not exist",
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
					{Source: "home", Target: "~/", LinkAsDirectory: []string{}},
					{Source: "private/home", Target: "~/", LinkAsDirectory: []string{}},
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
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify symlink was created
			linkInfo, err := os.Lstat(testPath)
			if err != nil {
				t.Fatalf("failed to stat adopted path: %v", err)
			}
			if linkInfo.Mode()&os.ModeSymlink == 0 {
				t.Errorf("expected symlink, got regular file/directory")
			}

			// Verify target exists in repo
			repoSubdir := "home"
			if tt.isPrivate {
				repoSubdir = filepath.Join("private", "home")
			}
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

			// For directories, verify contents are accessible through symlink
			if tt.createDir {
				content, err := os.ReadFile(filepath.Join(testPath, "file.txt"))
				if err != nil {
					t.Errorf("failed to read file through symlinked directory: %v", err)
				}
				if string(content) != "test content" {
					t.Errorf("wrong content through symlink: got %s, want 'test content'", string(content))
				}
			}
		})
	}
}

// TestAdoptDryRun tests the dry-run functionality
func TestAdoptDryRun(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(configRepo, 0755)

	testFile := filepath.Join(homeDir, ".testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/", LinkAsDirectory: []string{}},
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

// TestAdoptFileInsideLinkedDirectory tests adopting a file inside an already-linked directory
func TestAdoptFileInsideLinkedDirectory(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

	// First create a regular directory with a file
	dirInHome := filepath.Join(homeDir, ".config")
	os.MkdirAll(dirInHome, 0755)
	fileInDir := filepath.Join(dirInHome, "testfile.txt")
	os.WriteFile(fileInDir, []byte("test content"), 0644)

	// Now adopt the directory (not the file)
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/", LinkAsDirectory: []string{}},
		},
	}
	err := Adopt(dirInHome, configRepo, config, "home", false)
	if err != nil {
		t.Fatalf("failed to adopt directory: %v", err)
	}

	// Now try to adopt the file inside the linked directory
	err = Adopt(fileInDir, configRepo, config, "home", false)

	if err == nil {
		t.Fatalf("expected error when adopting file inside linked directory")
	}
	if !strings.Contains(err.Error(), "inside a directory that's already linked") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestAdoptConfigUpdate tests that directories are added to link_as_directory config
func TestAdoptConfigUpdate(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(configRepo, 0755)

	// Create test directory
	testDir := filepath.Join(homeDir, ".config", "testapp")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "config.txt"), []byte("test"), 0644)

	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/", LinkAsDirectory: []string{}},
		},
	}

	// Mock user input to say "yes" to link_as_directory prompt
	// Note: In real implementation, we'd need to inject the confirm function
	// For now, we'll test that the config can be updated

	// Manually test the AddDirectoryLinkToMapping functionality
	relPath := ".config/testapp"
	err := config.AddDirectoryLinkToMapping("home", relPath)
	if err != nil {
		t.Fatalf("failed to add directory link: %v", err)
	}

	mapping := config.GetMapping("home")
	if mapping == nil {
		t.Fatalf("home mapping not found")
	}
	if len(mapping.LinkAsDirectory) != 1 {
		t.Errorf("expected 1 directory in mapping, got %d", len(mapping.LinkAsDirectory))
	}
	if mapping.LinkAsDirectory[0] != relPath {
		t.Errorf("expected %s in mapping, got %s", relPath, mapping.LinkAsDirectory[0])
	}

	// Test saving config
	err = config.Save(configRepo)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify config file was created - should be .cfgman.json for new format
	configPath := filepath.Join(configRepo, ".cfgman.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var savedConfig Config
	if err := json.Unmarshal(data, &savedConfig); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	savedMapping := savedConfig.GetMapping("home")
	if savedMapping == nil || len(savedMapping.LinkAsDirectory) != 1 || savedMapping.LinkAsDirectory[0] != relPath {
		t.Errorf("saved config doesn't match: %+v", savedConfig)
	}
}

// TestAdoptDirectoryConfigSelection tests that directories are added to the correct config list
func TestAdoptDirectoryConfigSelection(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)

	// Override the confirmFunc to always return true for this test
	oldConfirmFunc := confirmFunc
	confirmFunc = func(prompt string) bool { return true }
	defer func() { confirmFunc = oldConfirmFunc }()

	tests := []struct {
		name      string
		isPrivate bool
	}{
		{
			name:      "home directory adopt adds to LinkAsDirectory",
			isPrivate: false,
		},
		{
			name:      "private_home directory adopt adds to PrivateLinkAsDirectory",
			isPrivate: true,
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

			// Create test directory
			testDir := filepath.Join(homeDir, ".config", "testapp")
			os.MkdirAll(testDir, 0755)
			os.WriteFile(filepath.Join(testDir, "config.txt"), []byte("test"), 0644)

			// Create test config
			config := &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: "~/", LinkAsDirectory: []string{}},
					{Source: "private/home", Target: "~/", LinkAsDirectory: []string{}},
				},
			}

			// Set HOME environment
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			// Run adopt
			sourceDir := "home"
			if tt.isPrivate {
				sourceDir = "private/home"
			}
			err := Adopt(testDir, configRepo, config, sourceDir, false)
			if err != nil {
				t.Fatalf("adopt failed: %v", err)
			}

			// Check that the directory was added to the correct mapping
			relPath := ".config/testapp"

			mappingSource := "home"
			if tt.isPrivate {
				mappingSource = "private/home"
			}

			mapping := config.GetMapping(mappingSource)
			if mapping == nil {
				t.Fatalf("%s mapping not found", mappingSource)
			}

			if len(mapping.LinkAsDirectory) != 1 {
				t.Errorf("expected 1 directory in %s mapping, got %d", mappingSource, len(mapping.LinkAsDirectory))
			}
			if len(mapping.LinkAsDirectory) > 0 && mapping.LinkAsDirectory[0] != relPath {
				t.Errorf("expected %s in %s mapping, got %s", relPath, mappingSource, mapping.LinkAsDirectory[0])
			}

			// Check that other mappings weren't affected
			otherSource := "private/home"
			if tt.isPrivate {
				otherSource = "home"
			}
			otherMapping := config.GetMapping(otherSource)
			if otherMapping != nil && len(otherMapping.LinkAsDirectory) != 0 {
				t.Errorf("expected 0 directories in %s mapping, got %d", otherSource, len(otherMapping.LinkAsDirectory))
			}

			// Verify the config file was saved correctly - should be .cfgman.json for new format
			configPath := filepath.Join(configRepo, ".cfgman.json")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config file: %v", err)
			}

			var savedConfig Config
			if err := json.Unmarshal(data, &savedConfig); err != nil {
				t.Fatalf("failed to parse saved config: %v", err)
			}

			// Verify saved config matches in-memory config
			savedMapping := savedConfig.GetMapping(mappingSource)
			if savedMapping == nil {
				t.Fatalf("saved %s mapping not found", mappingSource)
			}
			if len(savedMapping.LinkAsDirectory) != 1 || savedMapping.LinkAsDirectory[0] != relPath {
				t.Errorf("saved %s mapping doesn't match: %+v", mappingSource, savedConfig)
			}

			// Verify other mapping is empty in saved config
			savedOtherMapping := savedConfig.GetMapping(otherSource)
			if savedOtherMapping != nil && len(savedOtherMapping.LinkAsDirectory) != 0 {
				t.Errorf("saved %s mapping should be empty: %+v", otherSource, savedConfig)
			}
		})
	}
}

// TestOrphan tests the Orphan function
func TestOrphan(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)
	tests := []struct {
		name          string
		createFile    bool
		createDir     bool
		notSymlink    bool
		brokenLink    bool
		expectError   bool
		errorContains string
	}{
		{
			name:       "orphan regular file",
			createFile: true,
		},
		{
			name:      "orphan directory",
			createDir: true,
		},
		{
			name:          "orphan non-existent",
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name:          "orphan non-symlink",
			createFile:    true,
			notSymlink:    true,
			expectError:   true,
			errorContains: "not a symlink",
		},
		{
			name:          "orphan broken symlink",
			brokenLink:    true,
			expectError:   true,
			errorContains: "target does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			homeDir := filepath.Join(tempDir, "home")
			configRepo := filepath.Join(tempDir, "repo")

			os.MkdirAll(homeDir, 0755)
			os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

			testPath := filepath.Join(homeDir, ".testfile")
			targetPath := filepath.Join(configRepo, "home", ".testfile")

			if tt.createDir {
				testPath = filepath.Join(homeDir, ".testdir")
				targetPath = filepath.Join(configRepo, "home", ".testdir")
				os.MkdirAll(targetPath, 0755)
				os.WriteFile(filepath.Join(targetPath, "file.txt"), []byte("test content"), 0644)
				if !tt.notSymlink {
					os.Symlink(targetPath, testPath)
				}
			} else if tt.createFile {
				if tt.notSymlink {
					os.WriteFile(testPath, []byte("test"), 0644)
				} else {
					os.WriteFile(targetPath, []byte("test content"), 0644)
					os.Symlink(targetPath, testPath)
				}
			} else if tt.brokenLink {
				// Create a broken symlink
				os.Symlink(targetPath, testPath)
			}

			// Change to home directory
			oldDir, _ := os.Getwd()
			os.Chdir(homeDir)
			defer os.Chdir(oldDir)

			// Run orphan
			config := &Config{}
			err := Orphan(testPath, configRepo, config, false)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// For successful tests, we can't fully test because confirm() reads from stdin
			// In a real test, we'd need to mock the confirm function
		})
	}
}

// TestOrphanDirectory tests orphaning a directory with multiple symlinks
func TestOrphanDirectory(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)

	// Override confirmFunc to always return true
	oldConfirmFunc := confirmFunc
	confirmFunc = func(prompt string) bool { return true }
	defer func() { confirmFunc = oldConfirmFunc }()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	// Create directories
	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home", ".config", "app1"), 0755)
	os.MkdirAll(filepath.Join(configRepo, "home", ".config", "app2"), 0755)
	os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)

	// Create files in repo
	file1Target := filepath.Join(configRepo, "home", ".config", "app1", "config.txt")
	file2Target := filepath.Join(configRepo, "home", ".config", "app2", "settings.json")
	os.WriteFile(file1Target, []byte("app1 config"), 0644)
	os.WriteFile(file2Target, []byte("app2 settings"), 0644)

	// Create symlinks
	link1 := filepath.Join(homeDir, ".config", "app1")
	link2 := filepath.Join(homeDir, ".config", "app2")
	os.Symlink(filepath.Join(configRepo, "home", ".config", "app1"), link1)
	os.Symlink(filepath.Join(configRepo, "home", ".config", "app2"), link2)

	// Also create a regular file (not a symlink) in the directory
	regularFile := filepath.Join(homeDir, ".config", "regular.txt")
	os.WriteFile(regularFile, []byte("regular file"), 0644)

	// Change to home directory
	oldDir, _ := os.Getwd()
	os.Chdir(homeDir)
	defer os.Chdir(oldDir)

	// Test orphaning the .config directory
	config := &Config{}
	err := Orphan(filepath.Join(homeDir, ".config"), configRepo, config, false)
	if err != nil {
		t.Fatalf("failed to orphan directory: %v", err)
	}

	// Verify symlinks were removed and content was copied
	if _, err := os.Lstat(link1); err == nil {
		linkInfo, _ := os.Lstat(link1)
		if linkInfo.Mode()&os.ModeSymlink != 0 {
			t.Errorf("symlink1 still exists after orphaning")
		}
	}

	if _, err := os.Lstat(link2); err == nil {
		linkInfo, _ := os.Lstat(link2)
		if linkInfo.Mode()&os.ModeSymlink != 0 {
			t.Errorf("symlink2 still exists after orphaning")
		}
	}

	// Verify content was copied correctly
	content1, err := os.ReadFile(filepath.Join(link1, "config.txt"))
	if err != nil || string(content1) != "app1 config" {
		t.Errorf("app1 content not copied correctly: %v", err)
	}

	content2, err := os.ReadFile(filepath.Join(link2, "settings.json"))
	if err != nil || string(content2) != "app2 settings" {
		t.Errorf("app2 content not copied correctly: %v", err)
	}

	// Verify regular file was not affected
	regularContent, err := os.ReadFile(regularFile)
	if err != nil || string(regularContent) != "regular file" {
		t.Errorf("regular file was affected: %v", err)
	}
}

// TestOrphanDirectoryDryRun tests the dry-run mode for directory orphaning
func TestOrphanDirectoryDryRun(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	// Create directories
	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home", ".config"), 0755)

	// Create a symlink
	target := filepath.Join(configRepo, "home", ".config", "app")
	link := filepath.Join(homeDir, ".config", "app")
	os.MkdirAll(filepath.Dir(link), 0755)
	os.MkdirAll(target, 0755)
	os.WriteFile(filepath.Join(target, "config.txt"), []byte("test"), 0644)
	os.Symlink(target, link)

	// Test dry-run
	config := &Config{}
	err := Orphan(filepath.Join(homeDir, ".config"), configRepo, config, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Verify symlink still exists
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("symlink was removed in dry-run mode")
	}
}

// TestOrphanDirectoryNoSymlinks tests orphaning a directory with no symlinks
func TestOrphanDirectoryNoSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	// Create directories
	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)

	// Create only regular files
	os.WriteFile(filepath.Join(homeDir, ".config", "file.txt"), []byte("test"), 0644)

	// Test orphaning - should fail
	config := &Config{}
	err := Orphan(filepath.Join(homeDir, ".config"), configRepo, config, false)
	if err == nil {
		t.Errorf("expected error when orphaning directory with no symlinks")
	}
	if !strings.Contains(err.Error(), "no managed symlinks found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestOrphanDryRun tests the orphan dry-run functionality
func TestOrphanDryRun(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)

	// Create a managed symlink
	targetPath := filepath.Join(configRepo, "home", ".testfile")
	os.WriteFile(targetPath, []byte("test"), 0644)

	linkPath := filepath.Join(homeDir, ".testfile")
	os.Symlink(targetPath, linkPath)

	// Run orphan in dry-run mode
	config := &Config{}
	err := Orphan(linkPath, configRepo, config, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Verify symlink still exists
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("failed to stat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("symlink was removed in dry-run mode")
	}

	// Verify target still exists
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("target was removed in dry-run mode")
	}
}

// TestOrphanUntrackedFile tests orphaning untracked files that are not in git
func TestOrphanUntrackedFile(t *testing.T) {
	// Set test environment
	oldTestEnv := os.Getenv("CFGMAN_TEST")
	os.Setenv("CFGMAN_TEST", "1")
	defer os.Setenv("CFGMAN_TEST", oldTestEnv)

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

	// Set confirmFunc to always return true
	oldConfirmFunc := confirmFunc
	confirmFunc = func(prompt string) bool { return true }
	defer func() { confirmFunc = oldConfirmFunc }()

	// Run orphan
	config := &Config{}
	err := Orphan(linkPath, configRepo, config, false)
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

// TestCopyPath tests the copyPath function
func TestCopyPath(t *testing.T) {
	t.Run("copy file", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")

		// Create source file with specific permissions
		os.WriteFile(srcFile, []byte("test content"), 0755)

		// Copy file
		err := copyPath(srcFile, dstFile)
		if err != nil {
			t.Fatalf("failed to copy file: %v", err)
		}

		// Verify content
		content, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("failed to read destination: %v", err)
		}
		if string(content) != "test content" {
			t.Errorf("wrong content: got %s, want 'test content'", string(content))
		}

		// Verify permissions
		srcInfo, _ := os.Stat(srcFile)
		dstInfo, _ := os.Stat(dstFile)
		if srcInfo.Mode() != dstInfo.Mode() {
			t.Errorf("permissions not preserved: got %v, want %v", dstInfo.Mode(), srcInfo.Mode())
		}
	})

	t.Run("copy directory", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "source")
		dstDir := filepath.Join(tempDir, "dest")

		// Create source directory structure
		os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)

		// Copy directory
		err := copyPath(srcDir, dstDir)
		if err != nil {
			t.Fatalf("failed to copy directory: %v", err)
		}

		// Verify structure
		verifyFiles := []struct {
			path    string
			content string
		}{
			{filepath.Join(dstDir, "file1.txt"), "content1"},
			{filepath.Join(dstDir, "subdir", "file2.txt"), "content2"},
		}

		for _, vf := range verifyFiles {
			content, err := os.ReadFile(vf.path)
			if err != nil {
				t.Errorf("failed to read %s: %v", vf.path, err)
			}
			if string(content) != vf.content {
				t.Errorf("wrong content in %s: got %s, want %s", vf.path, string(content), vf.content)
			}
		}
	})
}

// TestIsInsideLinkedDirectory tests the isInsideLinkedDirectory function
func TestIsInsideLinkedDirectory(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configRepo := filepath.Join(tempDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(filepath.Join(configRepo, "home", ".config"), 0755)

	// Create a linked directory
	linkedDirInRepo := filepath.Join(configRepo, "home", ".config")
	linkedDirInHome := filepath.Join(homeDir, ".config")
	os.Symlink(linkedDirInRepo, linkedDirInHome)

	// Create subdirectory inside the linked directory (through the symlink)
	os.MkdirAll(filepath.Join(linkedDirInHome, "subdir"), 0755)

	// Create a regular directory
	regularDir := filepath.Join(homeDir, ".regular")
	os.MkdirAll(regularDir, 0755)

	tests := []struct {
		name       string
		path       string
		wantInside bool
		wantLink   string
	}{
		{
			name:       "file inside linked directory",
			path:       filepath.Join(linkedDirInHome, "subdir", "file.txt"),
			wantInside: true,
			wantLink:   linkedDirInHome,
		},
		{
			name:       "file in regular directory",
			path:       filepath.Join(regularDir, "file.txt"),
			wantInside: false,
		},
		{
			name:       "file in home directory",
			path:       filepath.Join(homeDir, "file.txt"),
			wantInside: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inside, linkPath := isInsideLinkedDirectory(tt.path, homeDir, configRepo)
			if inside != tt.wantInside {
				t.Errorf("got inside=%v, want %v", inside, tt.wantInside)
			}
			if inside && linkPath != tt.wantLink {
				t.Errorf("got linkPath=%s, want %s", linkPath, tt.wantLink)
			}
		})
	}
}
