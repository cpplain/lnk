package cfgman

import (
	"os"
	"path/filepath"
	"testing"
)

// ==========================================
// Core Functionality Tests
// ==========================================

// TestCreateLinks tests the CreateLinks function
func TestCreateLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (configRepo string, config *Config)
		dryRun      bool
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "basic file linking",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				linkPath := filepath.Join(homeDir, ".bashrc")
				assertSymlink(t, linkPath, filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
		{
			name: "nested directory structure",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "git", "config"), "# git config")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "nvim", "init.vim"), "# nvim config")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				assertSymlink(t, filepath.Join(homeDir, ".config", "git", "config"),
					filepath.Join(configRepo, "home", ".config", "git", "config"))
				assertSymlink(t, filepath.Join(homeDir, ".config", "nvim", "init.vim"),
					filepath.Join(configRepo, "home", ".config", "nvim", "init.vim"))
			},
		},
		{
			name: "link as directory",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "nvim", "init.vim"), "# nvim config")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "nvim", "lua", "config.lua"), "-- lua config")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/", LinkAsDirectory: []string{".config/nvim"}},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				nvimLink := filepath.Join(homeDir, ".config", "nvim")
				assertSymlink(t, nvimLink, filepath.Join(configRepo, "home", ".config", "nvim"))

				// Verify that nvim is a symlink, not a directory with file symlinks inside
				info, err := os.Lstat(nvimLink)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode()&os.ModeSymlink == 0 {
					t.Error("Expected .config/nvim to be a symlink to the directory")
				}
			},
		},
		{
			name: "dry run mode",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc content")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: true,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// In dry run, no actual links should be created
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
			},
		},
		{
			name: "skip existing non-symlink files",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create an existing file that's not a symlink
				createTestFile(t, filepath.Join(homeDir, ".bashrc"), "# existing bashrc")

				// Create repo file
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# repo bashrc")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				bashrcPath := filepath.Join(homeDir, ".bashrc")

				// File should still exist but not be a symlink
				info, err := os.Lstat(bashrcPath)
				if err != nil {
					t.Fatalf("Expected file to exist: %v", err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("Expected file to remain non-symlink")
				}
			},
		},
		{
			name: "private repository files",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# public bashrc")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".ssh", "config"), "# ssh config")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
						{Source: "private/home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"),
					filepath.Join(configRepo, "home", ".bashrc"))
				assertSymlink(t, filepath.Join(homeDir, ".ssh", "config"),
					filepath.Join(configRepo, "private", "home", ".ssh", "config"))
			},
		},
		{
			name: "private directory linking",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create private work config
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".config", "work", "settings.json"), "{ \"private\": true }")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".config", "work", "secrets", "api.key"), "secret-key")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "private/home", Target: "~/", LinkAsDirectory: []string{".config/work"}},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// Private work should be linked as directory
				workLink := filepath.Join(homeDir, ".config", "work")
				assertSymlink(t, workLink, filepath.Join(configRepo, "private", "home", ".config", "work"))

				// Verify work is a symlink to directory
				info, err := os.Lstat(workLink)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode()&os.ModeSymlink == 0 {
					t.Error("Expected .config/work to be a symlink to the directory")
				}
			},
		},
		{
			name: "public uses LinkAsDirectory config",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create directory that should be linked as directory
				createTestFile(t, filepath.Join(configRepo, "home", ".myapp", "config.json"), "{ \"test\": true }")
				createTestFile(t, filepath.Join(configRepo, "home", ".myapp", "data.db"), "data")
				// Create directory that should be linked file-by-file
				createTestFile(t, filepath.Join(configRepo, "home", ".otherapp", "settings.ini"), "[settings]")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/", LinkAsDirectory: []string{".myapp"}},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// .myapp should be linked as directory
				myappLink := filepath.Join(homeDir, ".myapp")
				assertSymlink(t, myappLink, filepath.Join(configRepo, "home", ".myapp"))

				// .otherapp should have file links inside
				otherappFile := filepath.Join(homeDir, ".otherapp", "settings.ini")
				assertSymlink(t, otherappFile, filepath.Join(configRepo, "home", ".otherapp", "settings.ini"))
			},
		},
		{
			name: "private uses PrivateLinkAsDirectory config",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create directory that should be linked as directory
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".work", "config.json"), "{ \"private\": true }")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".work", "secrets.env"), "SECRET=value")
				// Create directory that should be linked file-by-file
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".personal", "notes.txt"), "notes")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "private/home", Target: "~/", LinkAsDirectory: []string{".work"}},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// .work should be linked as directory
				workLink := filepath.Join(homeDir, ".work")
				assertSymlink(t, workLink, filepath.Join(configRepo, "private", "home", ".work"))

				// .personal should have file links inside
				personalFile := filepath.Join(homeDir, ".personal", "notes.txt")
				assertSymlink(t, personalFile, filepath.Join(configRepo, "private", "home", ".personal", "notes.txt"))
			},
		},
		{
			name: "link mappings with multiple sources",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create files in home mapping
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "git", "config"), "# git config")
				// Create files in work mapping
				createTestFile(t, filepath.Join(configRepo, "work", ".config", "work", "settings.json"), "{ \"work\": true }")
				createTestFile(t, filepath.Join(configRepo, "work", ".ssh", "config"), "# work ssh config")
				// Create a dotfiles mapping with directory linking
				createTestFile(t, filepath.Join(configRepo, "dotfiles", ".vim", "vimrc"), "\" vim config")
				createTestFile(t, filepath.Join(configRepo, "dotfiles", ".vim", "plugins.vim"), "\" plugins")

				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{
							Source:          "home",
							Target:          "~/",
							LinkAsDirectory: []string{".config/git"},
						},
						{
							Source:          "work",
							Target:          "~/",
							LinkAsDirectory: []string{".config/work"},
						},
						{
							Source:          "dotfiles",
							Target:          "~/",
							LinkAsDirectory: []string{".vim"},
						},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// Check home mapping
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"),
					filepath.Join(configRepo, "home", ".bashrc"))
				assertSymlink(t, filepath.Join(homeDir, ".config", "git"),
					filepath.Join(configRepo, "home", ".config", "git"))

				// Check work mapping
				assertSymlink(t, filepath.Join(homeDir, ".config", "work"),
					filepath.Join(configRepo, "work", ".config", "work"))
				assertSymlink(t, filepath.Join(homeDir, ".ssh", "config"),
					filepath.Join(configRepo, "work", ".ssh", "config"))

				// Check dotfiles mapping
				assertSymlink(t, filepath.Join(homeDir, ".vim"),
					filepath.Join(configRepo, "dotfiles", ".vim"))
			},
		},
		{
			name: "link mappings with non-existent source",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Only create home directory
				createTestFile(t, filepath.Join(configRepo, "home", ".bashrc"), "# bashrc")

				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{
							Source: "home",
							Target: "~/",
						},
						{
							Source: "missing", // This directory doesn't exist
							Target: "~/",
						},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Should still link files from existing mapping
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"),
					filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			homeDir := filepath.Join(tmpDir, "home")
			if err := os.MkdirAll(homeDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Set HOME to our temp directory
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			configRepo, config := tt.setup(t, tmpDir)

			err := CreateLinks(configRepo, config, tt.dryRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

// TestRemoveLinks tests the RemoveLinks function
func TestRemoveLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (configRepo string)
		dryRun      bool
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "remove single link",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create a symlink
				source := filepath.Join(configRepo, "home", ".bashrc")
				target := filepath.Join(homeDir, ".bashrc")
				createTestFile(t, source, "# bashrc")
				os.Symlink(source, target)

				return configRepo
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
			},
		},
		{
			name: "remove multiple links",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create multiple symlinks
				files := []string{".bashrc", ".zshrc", ".vimrc"}
				for _, file := range files {
					source := filepath.Join(configRepo, "home", file)
					target := filepath.Join(homeDir, file)
					createTestFile(t, source, "# "+file)
					os.Symlink(source, target)
				}

				return configRepo
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				assertNotExists(t, filepath.Join(homeDir, ".zshrc"))
				assertNotExists(t, filepath.Join(homeDir, ".vimrc"))
			},
		},
		{
			name: "dry run remove",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				source := filepath.Join(configRepo, "home", ".bashrc")
				target := filepath.Join(homeDir, ".bashrc")
				createTestFile(t, source, "# bashrc")
				os.Symlink(source, target)

				return configRepo
			},
			dryRun: true,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Link should still exist in dry run
				assertSymlink(t, filepath.Join(homeDir, ".bashrc"),
					filepath.Join(configRepo, "home", ".bashrc"))
			},
		},
		{
			name: "skip internal links",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create external link
				externalSource := filepath.Join(configRepo, "home", ".bashrc")
				externalTarget := filepath.Join(homeDir, ".bashrc")
				createTestFile(t, externalSource, "# bashrc")
				os.Symlink(externalSource, externalTarget)

				// Create internal link (within repo)
				internalSource := filepath.Join(configRepo, "private", "secret")
				internalTarget := filepath.Join(configRepo, "link-to-secret")
				createTestFile(t, internalSource, "# secret")
				os.Symlink(internalSource, internalTarget)

				return configRepo
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// External link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
				// Internal link should remain
				assertSymlink(t, filepath.Join(configRepo, "link-to-secret"),
					filepath.Join(configRepo, "private", "secret"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			homeDir := filepath.Join(tmpDir, "home")
			if err := os.MkdirAll(homeDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Set HOME to our temp directory
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			configRepo := tt.setup(t, tmpDir)

			// Use the internal function that skips confirmation for testing
			config := &Config{}
			err := removeLinks(configRepo, config, tt.dryRun, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

// TestPruneLinks tests the PruneLinks function
func TestPruneLinks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, tmpDir string) (configRepo string)
		dryRun      bool
		wantErr     bool
		checkResult func(t *testing.T, tmpDir, configRepo string)
	}{
		{
			name: "remove broken link",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create a broken symlink
				source := filepath.Join(configRepo, "home", ".bashrc")
				target := filepath.Join(homeDir, ".bashrc")
				os.Symlink(source, target) // Create link to non-existent file

				return configRepo
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
			},
		},
		{
			name: "keep valid links",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create a valid symlink
				validSource := filepath.Join(configRepo, "home", ".vimrc")
				validTarget := filepath.Join(homeDir, ".vimrc")
				createTestFile(t, validSource, "# vimrc")
				os.Symlink(validSource, validTarget)

				// Create a broken symlink
				brokenSource := filepath.Join(configRepo, "home", ".bashrc")
				brokenTarget := filepath.Join(homeDir, ".bashrc")
				os.Symlink(brokenSource, brokenTarget)

				return configRepo
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Valid link should remain
				assertSymlink(t, filepath.Join(homeDir, ".vimrc"),
					filepath.Join(configRepo, "home", ".vimrc"))
				// Broken link should be removed
				assertNotExists(t, filepath.Join(homeDir, ".bashrc"))
			},
		},
		{
			name: "dry run prune",
			setup: func(t *testing.T, tmpDir string) string {
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				// Create a broken symlink
				source := filepath.Join(configRepo, "home", ".bashrc")
				target := filepath.Join(homeDir, ".bashrc")
				os.Symlink(source, target)

				return configRepo
			},
			dryRun: true,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Broken link should still exist in dry run
				_, err := os.Lstat(filepath.Join(homeDir, ".bashrc"))
				if err != nil {
					t.Error("Expected broken link to still exist in dry run")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			homeDir := filepath.Join(tmpDir, "home")
			if err := os.MkdirAll(homeDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Set HOME to our temp directory
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			// Set test mode to skip confirmation prompts
			oldTestMode := os.Getenv("CFGMAN_TEST")
			os.Setenv("CFGMAN_TEST", "1")
			defer os.Setenv("CFGMAN_TEST", oldTestMode)

			configRepo := tt.setup(t, tmpDir)

			config := &Config{}
			err := PruneLinks(configRepo, config, tt.dryRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("PruneLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, tmpDir, configRepo)
			}
		})
	}
}

// TestGitIgnoredWithGlobal tests git ignore functionality with global gitignore
// REMOVED: Git functionality has been removed from cfgman
/*
func TestGitIgnoredWithGlobal(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary directory for our test
	tmpDir := t.TempDir()

	// Create a temporary global gitignore file
	globalIgnoreFile := filepath.Join(tmpDir, "global-gitignore")
	globalIgnoreContent := `*.global
.globalignore
global-pattern/
`
	if err := os.WriteFile(globalIgnoreFile, []byte(globalIgnoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a test repository
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure the global gitignore for this test repository
	cmd = exec.Command("git", "config", "core.excludesFile", globalIgnoreFile)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set global gitignore: %v", err)
	}

	// Create a local .gitignore file
	localGitignorePath := filepath.Join(repoDir, ".gitignore")
	localGitignoreContent := `*.local
.localignore
`
	if err := os.WriteFile(localGitignorePath, []byte(localGitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Enable debug mode for this test
	oldDebug := os.Getenv("CFGMAN_DEBUG")
	os.Setenv("CFGMAN_DEBUG", "1")
	defer os.Setenv("CFGMAN_DEBUG", oldDebug)

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		// Local gitignore patterns
		{"test.local", true, "local gitignore pattern"},
		{".localignore", true, "local gitignore file pattern"},

		// Global gitignore patterns
		{"test.global", true, "global gitignore pattern"},
		{".globalignore", true, "global gitignore file pattern"},
		{"global-pattern/file.txt", true, "global gitignore directory pattern"},

		// Should not be ignored
		{"regular.txt", false, "regular file"},
		{".bashrc", false, "dotfile not in gitignore"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isGitIgnored(tt.path, repoDir)
			if result != tt.expected {
				t.Errorf("isGitIgnored(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
*/

// TestGitIgnoredNoGit tests gitignore functionality when git is not available
// REMOVED: Git functionality has been removed from cfgman
/*
func TestGitIgnoredNoGit(t *testing.T) {
	// Create a mock environment where git command fails
	// We'll use PATH manipulation to ensure git is not found
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	tmpDir := t.TempDir()

	// Create some test files
	testFiles := []string{
		"test.log",
		".gitignore",
		"regular.txt",
	}

	// Create a .gitignore file (which should be irrelevant without git)
	gitignoreContent := `*.log
.DS_Store
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Enable debug mode to verify warning message
	oldDebug := os.Getenv("CFGMAN_DEBUG")
	os.Setenv("CFGMAN_DEBUG", "1")
	defer os.Setenv("CFGMAN_DEBUG", oldDebug)

	// All files should return false when git is not available
	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			result := isGitIgnored(filepath.Join(tmpDir, file), tmpDir)
			if result {
				t.Errorf("Expected isGitIgnored(%s) to return false when git is not available", file)
			}
		})
	}
}
*/

// TestGitIgnoredNotInRepo tests gitignore functionality when not in a git repository
// REMOVED: Git functionality has been removed from cfgman
/*
func TestGitIgnoredNotInRepo(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a directory that is NOT a git repository
	tmpDir := t.TempDir()

	// Create some test files
	testFiles := []string{
		"test.log",
		".gitignore",
		"regular.txt",
	}

	// Create a .gitignore file (which should be irrelevant outside a git repo)
	gitignoreContent := `*.log
.DS_Store
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Enable debug mode to verify debug message
	oldDebug := os.Getenv("CFGMAN_DEBUG")
	os.Setenv("CFGMAN_DEBUG", "1")
	defer os.Setenv("CFGMAN_DEBUG", oldDebug)

	// All files should return false when not in a git repository
	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			result := isGitIgnored(filepath.Join(tmpDir, file), tmpDir)
			if result {
				t.Errorf("Expected isGitIgnored(%s) to return false when not in a git repository", file)
			}
		})
	}
}
*/

// TestGitIgnoredComplexPatterns tests more complex gitignore patterns
// REMOVED: Git functionality has been removed from cfgman
/*
func TestGitIgnoredComplexPatterns(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a more complex .gitignore file
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := `# Comments should be handled
*.log
!important.log
temp/
!temp/keep/
.DS_Store
*.swp
*.swo
*~
build/
dist/
node_modules/
__pycache__/
*.pyc
.env
.env.*
!.env.example
/root-only.txt
deep/** /nested.txt
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		// Basic patterns
		{"test.log", true, "basic wildcard pattern"},
		{"important.log", false, "negated pattern"},
		{"temp/file.txt", true, "directory pattern"},
		{"temp/keep/file.txt", true, "negated directory pattern"}, // Git still considers this ignored because parent is ignored
		{".DS_Store", true, "exact match pattern"},
		{"file.swp", true, "vim swap file"},
		{"file.swo", true, "vim swap file variant"},
		{"backup~", true, "backup file pattern"},

		// Build directories
		{"build/output.js", true, "build directory"},
		{"dist/app.min.js", true, "dist directory"},
		{"node_modules/package/index.js", true, "node_modules directory"},

		// Python patterns
		{"__pycache__/module.pyc", true, "Python cache directory"},
		{"script.pyc", true, "Python compiled file"},

		// Environment files
		{".env", true, "environment file"},
		{".env.production", true, "environment variant"},
		{".env.example", false, "negated env example"},

		// Root-only pattern
		{"root-only.txt", true, "root-only file"},
		{"subdir/root-only.txt", false, "root-only pattern in subdirectory"},

		// Deep nested pattern
		{"deep/a/b/nested.txt", true, "deep nested pattern"},
		{"deep/nested.txt", true, "deep nested pattern at different level"},

		// Should not be ignored
		{"regular.txt", false, "regular file"},
		{".bashrc", false, "dotfile not in gitignore"},
		{"important-file.txt", false, "file not matching any pattern"},
	}

	// Create necessary directories for testing
	for _, dir := range []string{"temp", "temp/keep", "build", "dist", "node_modules/package",
		"__pycache__", "subdir", "deep/a/b", "deep"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isGitIgnored(filepath.Join(tmpDir, tt.path), tmpDir)
			if result != tt.expected {
				t.Errorf("isGitIgnored(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
*/

// TestGitIgnoredWithActualFiles tests gitignore with actual files created
// REMOVED: Git functionality has been removed from cfgman
/*
func TestGitIgnoredWithActualFiles(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create .gitignore
	gitignoreContent := `*.log
temp/
.cache/
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create actual files and directories
	files := []struct {
		path    string
		content string
		isDir   bool
	}{
		{"test.log", "log content", false},
		{"debug.log", "debug content", false},
		{"temp", "", true},
		{"temp/file.txt", "temp file", false},
		{".cache", "", true},
		{".cache/data.db", "cache data", false},
		{"regular.txt", "regular content", false},
		{".bashrc", "bashrc content", false},
	}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f.path)
		if f.isDir {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatal(err)
			}
		} else {
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fullPath, []byte(f.content), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test with actual files
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.log", true},
		{"debug.log", true},
		{"temp/file.txt", true},
		{".cache/data.db", true},
		{"regular.txt", false},
		{".bashrc", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isGitIgnored(filepath.Join(tmpDir, tt.path), tmpDir)
			if result != tt.expected {
				t.Errorf("isGitIgnored(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
*/

// TestCreateLinksWithGitIgnore tests that CreateLinks properly skips gitignored files
// REMOVED: Git functionality has been removed from cfgman
/*
func TestCreateLinksWithGitIgnore(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configRepo := filepath.Join(tmpDir, "repo")

	os.MkdirAll(homeDir, 0755)
	os.MkdirAll(configRepo, 0755)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = configRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create .gitignore in repo
	gitignoreContent := `*.log
.DS_Store
temp/
*.swp
`
	if err := os.WriteFile(filepath.Join(configRepo, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create various files in the repo
	testFiles := []struct {
		path      string
		content   string
		shouldLink bool
	}{
		{"home/.bashrc", "# bashrc", true},
		{"home/.vimrc", "\" vimrc", true},
		{"home/debug.log", "log file", false}, // ignored by *.log
		{"home/.DS_Store", "mac file", false}, // ignored
		{"home/temp/config.json", "temp config", false}, // ignored by temp/
		{"home/.vim/backup.swp", "swap file", false}, // ignored by *.swp
		{"home/.config/app/settings.json", "app settings", true},
	}

	for _, tf := range testFiles {
		createTestFile(t, filepath.Join(configRepo, tf.path), tf.content)
	}

	// Set HOME to our temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	// Create links
	err := CreateLinks(configRepo, &Config{}, false)
	if err != nil {
		t.Fatalf("CreateLinks failed: %v", err)
	}

	// Verify correct files were linked
	for _, tf := range testFiles {
		targetPath := filepath.Join(homeDir, strings.TrimPrefix(tf.path, "home/"))
		_, err := os.Lstat(targetPath)

		if tf.shouldLink {
			if err != nil {
				t.Errorf("Expected %s to be linked, but it wasn't: %v", tf.path, err)
			} else {
				// Verify it's actually a symlink
				assertSymlink(t, targetPath, filepath.Join(configRepo, tf.path))
			}
		} else {
			if err == nil {
				t.Errorf("Expected %s to NOT be linked (gitignored), but it was", tf.path)
			}
		}
	}
}
*/

// ==========================================
// Edge Cases and Error Scenarios
// ==========================================

// TestLinkerEdgeCases tests edge cases for the linker functions
func TestLinkerEdgeCases(t *testing.T) {
	t.Run("empty config repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)
		os.MkdirAll(configRepo, 0755)

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		// Should not error on empty repo with no mappings
		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{},
		}, false)
		if err == nil {
			t.Errorf("Expected error for empty mappings, got nil")
		}
	})

	t.Run("deeply nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)

		// Create deeply nested structure
		deepPath := filepath.Join(configRepo, "home", ".config", "app", "nested", "deep", "very", "config.json")
		createTestFile(t, deepPath, "{ \"test\": true }")

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{
				{Source: "home", Target: "~/"},
			},
		}, false)
		if err != nil {
			t.Errorf("Failed to create links for deeply nested structure: %v", err)
		}

		// Check that the deep link was created
		expectedLink := filepath.Join(homeDir, ".config", "app", "nested", "deep", "very", "config.json")
		assertSymlink(t, expectedLink, deepPath)
	})

	t.Run("symlink to symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)

		// Create a file and a symlink to it in the repo
		originalFile := filepath.Join(configRepo, "home", ".bashrc")
		symlinkInRepo := filepath.Join(configRepo, "home", ".bash_profile")
		createTestFile(t, originalFile, "# bashrc")
		os.Symlink(originalFile, symlinkInRepo)

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{
				{Source: "home", Target: "~/"},
			},
		}, false)
		if err != nil {
			t.Errorf("Failed to create links: %v", err)
		}

		// Both should be linked
		assertSymlink(t, filepath.Join(homeDir, ".bashrc"), originalFile)
		assertSymlink(t, filepath.Join(homeDir, ".bash_profile"), symlinkInRepo)
	})

	t.Run("special characters in filenames", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)

		// Create files with special characters
		specialFiles := []string{
			"file with spaces.txt",
			"file-with-dashes.conf",
			"file_with_underscores.ini",
			"file.multiple.dots.ext",
		}

		for _, filename := range specialFiles {
			createTestFile(t, filepath.Join(configRepo, "home", filename), "content")
		}

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{
				{Source: "home", Target: "~/"},
			},
		}, false)
		if err != nil {
			t.Errorf("Failed to create links: %v", err)
		}

		// Check all files were linked
		for _, filename := range specialFiles {
			assertSymlink(t,
				filepath.Join(homeDir, filename),
				filepath.Join(configRepo, "home", filename))
		}
	})

	t.Run("mixed file and directory with same prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)

		// Create a file and a directory with similar names
		createTestFile(t, filepath.Join(configRepo, "home", ".vim"), "vim config")
		createTestFile(t, filepath.Join(configRepo, "home", ".vimrc"), "vimrc")
		createTestFile(t, filepath.Join(configRepo, "home", ".vim.d", "plugin.vim"), "plugin")

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{
				{Source: "home", Target: "~/"},
			},
		}, false)
		if err != nil {
			t.Errorf("Failed to create links: %v", err)
		}

		// Check all were linked correctly
		assertSymlink(t, filepath.Join(homeDir, ".vim"),
			filepath.Join(configRepo, "home", ".vim"))
		assertSymlink(t, filepath.Join(homeDir, ".vimrc"),
			filepath.Join(configRepo, "home", ".vimrc"))
		assertSymlink(t, filepath.Join(homeDir, ".vim.d", "plugin.vim"),
			filepath.Join(configRepo, "home", ".vim.d", "plugin.vim"))
	})

	t.Run("no home directory in repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)
		os.MkdirAll(configRepo, 0755)

		// Don't create home directory in repo

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		// Should skip non-existent source directories
		err := CreateLinks(configRepo, &Config{
			LinkMappings: []LinkMapping{
				{Source: "home", Target: "~/"},
			},
		}, false)
		if err != nil {
			t.Errorf("Expected no error when home directory doesn't exist, got: %v", err)
		}
	})

	t.Run("permission denied on target", func(t *testing.T) {
		// Skip on CI or if not running as regular user
		if os.Getenv("CI") != "" {
			t.Skip("Skipping permission test in CI environment")
		}

		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		configRepo := filepath.Join(tmpDir, "repo")

		os.MkdirAll(homeDir, 0755)

		// Create a directory with no write permission
		restrictedDir := filepath.Join(homeDir, ".config")
		os.MkdirAll(restrictedDir, 0555) // read+execute only

		createTestFile(t, filepath.Join(configRepo, "home", ".config", "test.conf"), "config")

		// Set HOME to our temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		// Should handle permission error gracefully
		err := CreateLinks(configRepo, &Config{}, false)
		if err == nil {
			// If no error, it might have succeeded somehow, check if link exists
			if _, err := os.Lstat(filepath.Join(restrictedDir, "test.conf")); err == nil {
				t.Log("Link was created despite restricted permissions")
			}
		}

		// Restore permissions for cleanup
		os.Chmod(restrictedDir, 0755)
	})
}

// ==========================================
// Test Helper Functions
// ==========================================

// createTestFile creates a test file with the given content
func createTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
}

// assertSymlink verifies that a symlink exists and points to the expected target
func assertSymlink(t *testing.T, link, expectedTarget string) {
	t.Helper()

	info, err := os.Lstat(link)
	if err != nil {
		t.Errorf("Expected symlink %s to exist: %v", link, err)
		return
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %s to be a symlink", link)
		return
	}

	target, err := os.Readlink(link)
	if err != nil {
		t.Errorf("Failed to read symlink %s: %v", link, err)
		return
	}

	if target != expectedTarget {
		t.Errorf("Symlink %s points to %s, expected %s", link, target, expectedTarget)
	}
}

// assertNotExists verifies that a file or directory does not exist
func assertNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Lstat(path)
	if err == nil {
		t.Errorf("Expected %s to not exist", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("Unexpected error checking %s: %v", path, err)
	}
}
