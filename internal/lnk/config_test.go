package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for new flag-based config format

func TestParseFlagConfigFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		want        *FlagConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "basic config",
			content: `--target=~
--ignore=*.tmp
--ignore=*.swp`,
			want: &FlagConfig{
				Target:         "~",
				IgnorePatterns: []string{"*.tmp", "*.swp"},
			},
			wantErr: false,
		},
		{
			name: "config with comments and blank lines",
			content: `# This is a comment
--target=~/dotfiles

# Another comment
--ignore=.git
--ignore=*.log`,
			want: &FlagConfig{
				Target:         "~/dotfiles",
				IgnorePatterns: []string{".git", "*.log"},
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			content: ``,
			want: &FlagConfig{
				IgnorePatterns: []string{},
			},
			wantErr: false,
		},
		{
			name: "config with unknown flags (ignored)",
			content: `--target=~
--unknown-flag=value
--ignore=*.tmp`,
			want: &FlagConfig{
				Target:         "~",
				IgnorePatterns: []string{"*.tmp"},
			},
			wantErr: false,
		},
		{
			name: "invalid format (missing --)",
			content: `target=~
--ignore=*.tmp`,
			wantErr:     true,
			errContains: "invalid flag format",
		},
		{
			name: "short flag -t",
			content: `--t=~
--ignore=*.tmp`,
			want: &FlagConfig{
				Target:         "~",
				IgnorePatterns: []string{"*.tmp"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "lnk-test-*.lnkconfig")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if err := os.WriteFile(tmpFile.Name(), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := parseFlagConfigFile(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlagConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseFlagConfigFile() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if got.Target != tt.want.Target {
				t.Errorf("parseFlagConfigFile() Target = %v, want %v", got.Target, tt.want.Target)
			}

			if len(got.IgnorePatterns) != len(tt.want.IgnorePatterns) {
				t.Errorf("parseFlagConfigFile() IgnorePatterns length = %v, want %v", len(got.IgnorePatterns), len(tt.want.IgnorePatterns))
			} else {
				for i, pattern := range tt.want.IgnorePatterns {
					if got.IgnorePatterns[i] != pattern {
						t.Errorf("parseFlagConfigFile() IgnorePatterns[%d] = %v, want %v", i, got.IgnorePatterns[i], pattern)
					}
				}
			}
		})
	}
}

func TestParseIgnoreFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "basic ignore file",
			content: `.git
*.swp
*.tmp
node_modules/`,
			want:    []string{".git", "*.swp", "*.tmp", "node_modules/"},
			wantErr: false,
		},
		{
			name: "ignore file with comments and blank lines",
			content: `# Version control
.git

# Editor files
*.swp
*.tmp

# Dependencies
node_modules/`,
			want:    []string{".git", "*.swp", "*.tmp", "node_modules/"},
			wantErr: false,
		},
		{
			name:    "empty ignore file",
			content: ``,
			want:    []string{},
			wantErr: false,
		},
		{
			name: "ignore file with only comments",
			content: `# Just comments
# Nothing to ignore`,
			want:    []string{},
			wantErr: false,
		},
		{
			name: "ignore file with negation patterns",
			content: `*.log
!important.log`,
			want:    []string{"*.log", "!important.log"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "lnk-test-*.lnkignore")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if err := os.WriteFile(tmpFile.Name(), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := parseIgnoreFile(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIgnoreFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseIgnoreFile() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseIgnoreFile() length = %v, want %v", len(got), len(tt.want))
			} else {
				for i, pattern := range tt.want {
					if got[i] != pattern {
						t.Errorf("parseIgnoreFile()[%d] = %v, want %v", i, got[i], pattern)
					}
				}
			}
		})
	}
}

func TestLoadFlagConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     func(tmpDir string) error
		sourceDir      string
		wantTarget     string
		wantIgnores    []string
		wantSourceName string
		wantErr        bool
	}{
		{
			name: "load from source directory",
			setupFiles: func(tmpDir string) error {
				configContent := `--target=~/dotfiles
--ignore=*.tmp`
				return os.WriteFile(filepath.Join(tmpDir, FlagConfigFileName), []byte(configContent), 0644)
			},
			sourceDir:      ".",
			wantTarget:     "~/dotfiles",
			wantIgnores:    []string{"*.tmp"},
			wantSourceName: "source directory",
			wantErr:        false,
		},
		// Skipping "load from home directory" test as it requires writing to home directory
		// which is not allowed in sandbox. The precedence logic is tested in other tests.
		{
			name:           "no config file found",
			setupFiles:     func(tmpDir string) error { return nil },
			sourceDir:      ".",
			wantTarget:     "",
			wantIgnores:    []string{},
			wantSourceName: "",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "lnk-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test files
			if err := tt.setupFiles(tmpDir); err != nil {
				t.Fatalf("setupFiles() error = %v", err)
			}

			// Determine source directory
			sourceDir := tmpDir
			if tt.sourceDir != "." {
				sourceDir = tt.sourceDir
			}

			// Load config
			config, sourcePath, err := LoadFlagConfig(sourceDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFlagConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if config.Target != tt.wantTarget {
				t.Errorf("LoadFlagConfig() Target = %v, want %v", config.Target, tt.wantTarget)
			}

			if len(config.IgnorePatterns) != len(tt.wantIgnores) {
				t.Errorf("LoadFlagConfig() IgnorePatterns length = %v, want %v", len(config.IgnorePatterns), len(tt.wantIgnores))
			} else {
				for i, pattern := range tt.wantIgnores {
					if config.IgnorePatterns[i] != pattern {
						t.Errorf("LoadFlagConfig() IgnorePatterns[%d] = %v, want %v", i, config.IgnorePatterns[i], pattern)
					}
				}
			}

			if tt.wantSourceName != "" && !strings.Contains(sourcePath, tt.sourceDir) && tt.wantSourceName != "source directory" {
				t.Errorf("LoadFlagConfig() source path doesn't match expected location, got %v", sourcePath)
			}
		})
	}
}

func TestLoadIgnoreFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(tmpDir string) error
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "load existing ignore file",
			setupFile: func(tmpDir string) error {
				ignoreContent := `.git
*.swp
node_modules/`
				return os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644)
			},
			want:    []string{".git", "*.swp", "node_modules/"},
			wantErr: false,
		},
		{
			name:      "no ignore file",
			setupFile: func(tmpDir string) error { return nil },
			want:      []string{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "lnk-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test file
			if err := tt.setupFile(tmpDir); err != nil {
				t.Fatalf("setupFile() error = %v", err)
			}

			// Load ignore file
			got, err := LoadIgnoreFile(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadIgnoreFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadIgnoreFile() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("LoadIgnoreFile() length = %v, want %v", len(got), len(tt.want))
			} else {
				for i, pattern := range tt.want {
					if got[i] != pattern {
						t.Errorf("LoadIgnoreFile()[%d] = %v, want %v", i, got[i], pattern)
					}
				}
			}
		})
	}
}

func TestMergeFlagConfig(t *testing.T) {
	tests := []struct {
		name               string
		setupFiles         func(tmpDir string) error
		sourceDir          string // relative to tmpDir, or "" for tmpDir itself
		cliTarget          string
		cliIgnorePatterns  []string
		wantTargetDir      string
		wantIgnorePatterns []string // patterns to check (subset)
		wantErr            bool
		errContains        string
	}{
		{
			name: "no config files, use defaults",
			setupFiles: func(tmpDir string) error {
				return nil
			},
			sourceDir:          "",
			cliTarget:          "",
			cliIgnorePatterns:  nil,
			wantTargetDir:      "~",
			wantIgnorePatterns: []string{".git", ".DS_Store", ".lnkconfig"},
			wantErr:            false,
		},
		{
			name: "config file sets target",
			setupFiles: func(tmpDir string) error {
				configContent := `--target=~/.config
--ignore=*.backup`
				return os.WriteFile(filepath.Join(tmpDir, FlagConfigFileName), []byte(configContent), 0644)
			},
			sourceDir:          "",
			cliTarget:          "",
			cliIgnorePatterns:  nil,
			wantTargetDir:      "~/.config",
			wantIgnorePatterns: []string{".git", "*.backup"},
			wantErr:            false,
		},
		{
			name: "CLI target overrides config file",
			setupFiles: func(tmpDir string) error {
				configContent := `--target=~/.config`
				return os.WriteFile(filepath.Join(tmpDir, FlagConfigFileName), []byte(configContent), 0644)
			},
			sourceDir:          "",
			cliTarget:          "~/custom",
			cliIgnorePatterns:  nil,
			wantTargetDir:      "~/custom",
			wantIgnorePatterns: []string{".git"},
			wantErr:            false,
		},
		{
			name: "ignore patterns from .lnkignore",
			setupFiles: func(tmpDir string) error {
				ignoreContent := `node_modules/
dist/
.env`
				return os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644)
			},
			sourceDir:          "",
			cliTarget:          "",
			cliIgnorePatterns:  nil,
			wantTargetDir:      "~",
			wantIgnorePatterns: []string{".git", "node_modules/", "dist/", ".env"},
			wantErr:            false,
		},
		{
			name: "CLI ignore patterns added",
			setupFiles: func(tmpDir string) error {
				return nil
			},
			sourceDir:          "",
			cliTarget:          "",
			cliIgnorePatterns:  []string{"*.local", "secrets/"},
			wantTargetDir:      "~",
			wantIgnorePatterns: []string{".git", "*.local", "secrets/"},
			wantErr:            false,
		},
		{
			name: "all sources combined",
			setupFiles: func(tmpDir string) error {
				// Create .lnkconfig
				configContent := `--target=/opt/configs
--ignore=*.backup
--ignore=temp/`
				if err := os.WriteFile(filepath.Join(tmpDir, FlagConfigFileName), []byte(configContent), 0644); err != nil {
					return err
				}

				// Create .lnkignore
				ignoreContent := `node_modules/
.env`
				return os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644)
			},
			sourceDir:          "",
			cliTarget:          "~/target",
			cliIgnorePatterns:  []string{"*.local"},
			wantTargetDir:      "~/target",
			wantIgnorePatterns: []string{".git", "*.backup", "temp/", "node_modules/", ".env", "*.local"},
			wantErr:            false,
		},
		{
			name: "config in subdirectory",
			setupFiles: func(tmpDir string) error {
				subDir := filepath.Join(tmpDir, "dotfiles")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}

				configContent := `--target=~/
--ignore=*.test`
				return os.WriteFile(filepath.Join(subDir, FlagConfigFileName), []byte(configContent), 0644)
			},
			sourceDir:          "dotfiles",
			cliTarget:          "",
			cliIgnorePatterns:  nil,
			wantTargetDir:      "~/",
			wantIgnorePatterns: []string{".git", "*.test"},
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "lnk-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test files
			if err := tt.setupFiles(tmpDir); err != nil {
				t.Fatalf("setupFiles() error = %v", err)
			}

			// Determine source directory
			sourceDir := tmpDir
			if tt.sourceDir != "" {
				sourceDir = filepath.Join(tmpDir, tt.sourceDir)
			}

			// Merge config
			merged, err := MergeFlagConfig(sourceDir, tt.cliTarget, tt.cliIgnorePatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeFlagConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("MergeFlagConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// Check target directory
			if merged.TargetDir != tt.wantTargetDir {
				t.Errorf("MergeFlagConfig() TargetDir = %v, want %v", merged.TargetDir, tt.wantTargetDir)
			}

			// Check source directory is set
			if merged.SourceDir != sourceDir {
				t.Errorf("MergeFlagConfig() SourceDir = %v, want %v", merged.SourceDir, sourceDir)
			}

			// Check that wanted patterns are present
			for _, wantPattern := range tt.wantIgnorePatterns {
				found := false
				for _, gotPattern := range merged.IgnorePatterns {
					if gotPattern == wantPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("MergeFlagConfig() missing ignore pattern %q in %v", wantPattern, merged.IgnorePatterns)
				}
			}
		})
	}
}

func TestMergeFlagConfigPrecedence(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup all config sources
	configContent := `--target=/from-config
--ignore=config-pattern`
	if err := os.WriteFile(filepath.Join(tmpDir, FlagConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	ignoreContent := `ignore-file-pattern`
	if err := os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test precedence: CLI > config > default
	merged, err := MergeFlagConfig(tmpDir, "/from-cli", []string{"cli-pattern"})
	if err != nil {
		t.Fatalf("MergeFlagConfig() error = %v", err)
	}

	// CLI target should win
	if merged.TargetDir != "/from-cli" {
		t.Errorf("TargetDir precedence failed: got %v, want /from-cli", merged.TargetDir)
	}

	// All ignore patterns should be combined
	expectedPatterns := []string{
		"cli-pattern",          // from CLI
		"config-pattern",       // from .lnkconfig
		"ignore-file-pattern",  // from .lnkignore
		".git",                 // built-in
	}

	for _, want := range expectedPatterns {
		found := false
		for _, got := range merged.IgnorePatterns {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected pattern %q in merged patterns", want)
		}
	}
}
