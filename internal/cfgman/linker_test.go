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
			name: "link files in nested directories",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "nvim", "init.vim"), "# nvim config")
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "nvim", "lua", "config.lua"), "-- lua config")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// Verify that nvim directory exists but is NOT a symlink
				nvimDir := filepath.Join(homeDir, ".config", "nvim")
				info, err := os.Lstat(nvimDir)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("Expected .config/nvim to be a directory, not a symlink")
				}

				// Verify individual files are linked
				assertSymlink(t, filepath.Join(homeDir, ".config", "nvim", "init.vim"),
					filepath.Join(configRepo, "home", ".config", "nvim", "init.vim"))
				assertSymlink(t, filepath.Join(homeDir, ".config", "nvim", "lua", "config.lua"),
					filepath.Join(configRepo, "home", ".config", "nvim", "lua", "config.lua"))
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
			name: "private files linked recursively",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create private work config
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".config", "work", "settings.json"), "{ \"private\": true }")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".config", "work", "secrets", "api.key"), "secret-key")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "private/home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// Work directory should exist but not be a symlink
				workDir := filepath.Join(homeDir, ".config", "work")
				info, err := os.Lstat(workDir)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode()&os.ModeSymlink != 0 {
					t.Error("Expected .config/work to be a directory, not a symlink")
				}

				// Individual files should be linked
				assertSymlink(t, filepath.Join(homeDir, ".config", "work", "settings.json"),
					filepath.Join(configRepo, "private", "home", ".config", "work", "settings.json"))
				assertSymlink(t, filepath.Join(homeDir, ".config", "work", "secrets", "api.key"),
					filepath.Join(configRepo, "private", "home", ".config", "work", "secrets", "api.key"))
			},
		},
		{
			name: "public files linked individually",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create files in directories
				createTestFile(t, filepath.Join(configRepo, "home", ".myapp", "config.json"), "{ \"test\": true }")
				createTestFile(t, filepath.Join(configRepo, "home", ".myapp", "data.db"), "data")
				createTestFile(t, filepath.Join(configRepo, "home", ".otherapp", "settings.ini"), "[settings]")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// All files should be linked individually
				assertSymlink(t, filepath.Join(homeDir, ".myapp", "config.json"),
					filepath.Join(configRepo, "home", ".myapp", "config.json"))
				assertSymlink(t, filepath.Join(homeDir, ".myapp", "data.db"),
					filepath.Join(configRepo, "home", ".myapp", "data.db"))
				assertSymlink(t, filepath.Join(homeDir, ".otherapp", "settings.ini"),
					filepath.Join(configRepo, "home", ".otherapp", "settings.ini"))
			},
		},
		{
			name: "private files linked individually",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create files in private directories
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".work", "config.json"), "{ \"private\": true }")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".work", "secrets.env"), "SECRET=value")
				createTestFile(t, filepath.Join(configRepo, "private", "home", ".personal", "notes.txt"), "notes")
				return configRepo, &Config{
					LinkMappings: []LinkMapping{
						{Source: "private/home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")

				// All files should be linked individually
				assertSymlink(t, filepath.Join(homeDir, ".work", "config.json"),
					filepath.Join(configRepo, "private", "home", ".work", "config.json"))
				assertSymlink(t, filepath.Join(homeDir, ".work", "secrets.env"),
					filepath.Join(configRepo, "private", "home", ".work", "secrets.env"))
				assertSymlink(t, filepath.Join(homeDir, ".personal", "notes.txt"),
					filepath.Join(configRepo, "private", "home", ".personal", "notes.txt"))
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
							Source: "home",
							Target: "~/",
						},
						{
							Source: "work",
							Target: "~/",
						},
						{
							Source: "dotfiles",
							Target: "~/",
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
				assertSymlink(t, filepath.Join(homeDir, ".config", "git", "config"),
					filepath.Join(configRepo, "home", ".config", "git", "config"))

				// Check work mapping
				assertSymlink(t, filepath.Join(homeDir, ".config", "work", "settings.json"),
					filepath.Join(configRepo, "work", ".config", "work", "settings.json"))
				assertSymlink(t, filepath.Join(homeDir, ".ssh", "config"),
					filepath.Join(configRepo, "work", ".ssh", "config"))

				// Check dotfiles mapping
				assertSymlink(t, filepath.Join(homeDir, ".vim", "vimrc"),
					filepath.Join(configRepo, "dotfiles", ".vim", "vimrc"))
				assertSymlink(t, filepath.Join(homeDir, ".vim", "plugins.vim"),
					filepath.Join(configRepo, "dotfiles", ".vim", "plugins.vim"))
			},
		},
		{
			name: "no empty directories created",
			setup: func(t *testing.T, tmpDir string) (string, *Config) {
				configRepo := filepath.Join(tmpDir, "repo")
				// Create files in some directories but not others
				createTestFile(t, filepath.Join(configRepo, "home", ".config", "app1", "config.txt"), "config")
				// Create a directory with only ignored files
				createTestFile(t, filepath.Join(configRepo, "home", ".cache", ".DS_Store"), "ignored")
				createTestFile(t, filepath.Join(configRepo, "home", ".cache", "temp.swp"), "ignored")
				// Create a directory with only subdirectories (no files)
				os.MkdirAll(filepath.Join(configRepo, "home", ".empty", "subdir"), 0755)

				return configRepo, &Config{
					IgnorePatterns: []string{".DS_Store", "*.swp"},
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}
			},
			dryRun: false,
			checkResult: func(t *testing.T, tmpDir, configRepo string) {
				homeDir := filepath.Join(tmpDir, "home")
				// Should create directory for app1 since it has a file
				assertDirExists(t, filepath.Join(homeDir, ".config", "app1"))
				assertSymlink(t, filepath.Join(homeDir, ".config", "app1", "config.txt"),
					filepath.Join(configRepo, "home", ".config", "app1", "config.txt"))

				// Should NOT create .cache directory (only ignored files)
				if _, err := os.Stat(filepath.Join(homeDir, ".cache")); err == nil {
					t.Errorf(".cache directory should not exist (contains only ignored files)")
				}

				// Should NOT create .empty directory (no files at all)
				if _, err := os.Stat(filepath.Join(homeDir, ".empty")); err == nil {
					t.Errorf(".empty directory should not exist (contains no files)")
				}
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

// assertDirExists verifies that a directory exists
func assertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", path)
		} else {
			t.Errorf("Error checking directory %s: %v", path, err)
		}
	} else if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", path)
	}
}
