package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
			tmpDir := t.TempDir()

			if err := tt.setupFile(tmpDir); err != nil {
				t.Fatalf("setupFile() error = %v", err)
			}

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

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name               string
		setupFiles         func(tmpDir string) error
		cliIgnorePatterns  []string
		wantIgnorePatterns []string // patterns to check (subset)
		wantErr            bool
		errContains        string
	}{
		{
			name: "no config files, use defaults",
			setupFiles: func(tmpDir string) error {
				return nil
			},
			cliIgnorePatterns:  nil,
			wantIgnorePatterns: []string{".git", ".DS_Store", ".lnkignore"},
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
			cliIgnorePatterns:  nil,
			wantIgnorePatterns: []string{".git", "node_modules/", "dist/", ".env"},
			wantErr:            false,
		},
		{
			name: "CLI ignore patterns added",
			setupFiles: func(tmpDir string) error {
				return nil
			},
			cliIgnorePatterns:  []string{"*.local", "secrets/"},
			wantIgnorePatterns: []string{".git", "*.local", "secrets/"},
			wantErr:            false,
		},
		{
			name: ".lnkignore and CLI combined",
			setupFiles: func(tmpDir string) error {
				ignoreContent := `node_modules/
.env`
				return os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644)
			},
			cliIgnorePatterns:  []string{"*.local"},
			wantIgnorePatterns: []string{".git", "node_modules/", ".env", "*.local"},
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if err := tt.setupFiles(tmpDir); err != nil {
				t.Fatalf("setupFiles() error = %v", err)
			}

			config, err := LoadConfig(tmpDir, tt.cliIgnorePatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// SourceDir should be resolved to absolute path
			if !filepath.IsAbs(config.SourceDir) {
				t.Errorf("LoadConfig() SourceDir = %v, want absolute path", config.SourceDir)
			}

			// TargetDir should be the home directory (expanded)
			homeDir, _ := os.UserHomeDir()
			if config.TargetDir != homeDir {
				t.Errorf("LoadConfig() TargetDir = %v, want %v", config.TargetDir, homeDir)
			}

			// Check that wanted patterns are present
			for _, wantPattern := range tt.wantIgnorePatterns {
				found := false
				for _, gotPattern := range config.IgnorePatterns {
					if gotPattern == wantPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("LoadConfig() missing ignore pattern %q in %v", wantPattern, config.IgnorePatterns)
				}
			}
		})
	}
}

func TestLoadConfigSourceDirResolution(t *testing.T) {
	t.Run("resolves relative path to absolute", func(t *testing.T) {
		tmpDir := t.TempDir()

		config, err := LoadConfig(tmpDir, nil)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !filepath.IsAbs(config.SourceDir) {
			t.Errorf("SourceDir should be absolute, got %v", config.SourceDir)
		}
	})

	t.Run("missing directory returns error", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/that/does/not/exist", nil)
		if err == nil {
			t.Fatal("LoadConfig() expected error for missing directory")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("LoadConfig() error = %v, want error containing 'does not exist'", err)
		}
	})

	t.Run("file instead of directory returns error", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "lnk-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		_, err = LoadConfig(tmpFile.Name(), nil)
		if err == nil {
			t.Fatal("LoadConfig() expected error for file path")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("LoadConfig() error = %v, want error containing 'not a directory'", err)
		}
	})
}

func TestLoadConfigIgnorePatternOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .lnkignore with a pattern
	ignoreContent := `ignore-file-pattern`
	if err := os.WriteFile(filepath.Join(tmpDir, IgnoreFileName), []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(tmpDir, []string{"cli-pattern"})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify order: built-in patterns first, then .lnkignore, then CLI
	builtIn := getBuiltInIgnorePatterns()
	expectedOrder := append(builtIn, "ignore-file-pattern", "cli-pattern")

	if len(config.IgnorePatterns) != len(expectedOrder) {
		t.Fatalf("LoadConfig() IgnorePatterns length = %d, want %d\ngot: %v\nwant: %v",
			len(config.IgnorePatterns), len(expectedOrder), config.IgnorePatterns, expectedOrder)
	}

	for i, want := range expectedOrder {
		if config.IgnorePatterns[i] != want {
			t.Errorf("LoadConfig() IgnorePatterns[%d] = %q, want %q", i, config.IgnorePatterns[i], want)
		}
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"tilde only", "~", homeDir, false},
		{"tilde with path", "~/foo", filepath.Join(homeDir, "foo"), false},
		{"absolute path", "/tmp/foo", "/tmp/foo", false},
		{"relative path", "foo/bar", "foo/bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContractPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"home dir", homeDir, "~"},
		{"subpath of home", filepath.Join(homeDir, "foo"), "~/foo"},
		{"non-home path", "/tmp/foo", "/tmp/foo"},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContractPath(tt.path)
			if got != tt.want {
				t.Errorf("ContractPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
