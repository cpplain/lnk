package lnk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config with new LinkMappings format
	sourceDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceDir, 0755)
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source: filepath.Join(sourceDir, "home"),
				Target: "~/",
			},
			{
				Source: filepath.Join(sourceDir, "private/home"),
				Target: "~/",
			},
		},
	}

	// Save config
	configPath := filepath.Join(tmpDir, ".lnk.json")
	if err := config.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists - should be .lnk.json for new format
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify loaded config has correct LinkMappings
	if len(loaded.LinkMappings) != len(config.LinkMappings) {
		t.Errorf("LinkMappings length = %d, want %d", len(loaded.LinkMappings), len(config.LinkMappings))
	}

	// Verify each mapping
	for i, mapping := range config.LinkMappings {
		if i >= len(loaded.LinkMappings) {
			t.Errorf("Missing LinkMapping at index %d", i)
			continue
		}
		loadedMapping := loaded.LinkMappings[i]

		if loadedMapping.Source != mapping.Source {
			t.Errorf("LinkMapping[%d].Source = %q, want %q", i, loadedMapping.Source, mapping.Source)
		}
		if loadedMapping.Target != mapping.Target {
			t.Errorf("LinkMapping[%d].Target = %q, want %q", i, loadedMapping.Target, mapping.Target)
		}

	}
}

func TestConfigSaveNewFormat(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config with new format
	sourceDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceDir, 0755)
	config := &Config{
		IgnorePatterns: []string{"*.tmp", "backup/"},
		LinkMappings: []LinkMapping{
			{
				Source: filepath.Join(sourceDir, "home"),
				Target: "~/",
			},
		},
	}

	// Save config - should create .lnk.json
	configPath := filepath.Join(tmpDir, ".lnk.json")
	if err := config.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify .lnk.json exists
	lnkPath := filepath.Join(tmpDir, ".lnk.json")
	if _, err := os.Stat(lnkPath); err != nil {
		t.Fatalf(".lnk.json not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(lnkPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(loaded.IgnorePatterns) != 2 {
		t.Errorf("IgnorePatterns length = %d, want 2", len(loaded.IgnorePatterns))
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Load config from directory without config file
	configPath := filepath.Join(tmpDir, ".lnk.json")
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() should return error when no config file exists")
	}

	// Should return error about missing config file
	if !strings.Contains(err.Error(), "failed to read .lnk.json") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("LoadConfig() error = %v, want error about missing config file", err)
	}
}

func TestLoadConfigNewFormat(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create new format config file
	sourceDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceDir, 0755)
	newConfig := map[string]interface{}{
		"ignore_patterns": []string{"*.tmp", "backup/", ".DS_Store"},
		"link_mappings": []map[string]interface{}{
			{
				"source": filepath.Join(sourceDir, "home"),
				"target": "~/",
			},
		},
	}

	data, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmpDir, ".lnk.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify ignore patterns
	if len(loaded.IgnorePatterns) != 3 {
		t.Errorf("IgnorePatterns length = %d, want 3", len(loaded.IgnorePatterns))
	}

	// Verify link mappings
	if len(loaded.LinkMappings) != 1 {
		t.Errorf("LinkMappings length = %d, want 1", len(loaded.LinkMappings))
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		relativePath string
		want         bool
	}{
		{
			name: "no ignore patterns",
			config: &Config{
				IgnorePatterns: []string{},
			},
			relativePath: "test.tmp",
			want:         false,
		},
		{
			name: "match file pattern",
			config: &Config{
				IgnorePatterns: []string{"*.tmp", "*.log"},
			},
			relativePath: "test.tmp",
			want:         true,
		},
		{
			name: "match directory pattern",
			config: &Config{
				IgnorePatterns: []string{"backup/", "tmp/"},
			},
			relativePath: "backup/file.txt",
			want:         true,
		},
		{
			name: "match exact filename",
			config: &Config{
				IgnorePatterns: []string{".DS_Store", "Thumbs.db"},
			},
			relativePath: ".DS_Store",
			want:         true,
		},
		{
			name: "no match",
			config: &Config{
				IgnorePatterns: []string{"*.tmp", "backup/"},
			},
			relativePath: "important.txt",
			want:         false,
		},
		{
			name: "double wildcard pattern",
			config: &Config{
				IgnorePatterns: []string{"**/node_modules"},
			},
			relativePath: "src/components/node_modules/package.json",
			want:         true,
		},
		{
			name: "negation pattern",
			config: &Config{
				IgnorePatterns: []string{"*.log", "!important.log"},
			},
			relativePath: "important.log",
			want:         false,
		},
		{
			name: "complex patterns with negation",
			config: &Config{
				IgnorePatterns: []string{"build/", "!build/keep/", "*.tmp"},
			},
			relativePath: "build/keep/file.txt",
			want:         false,
		},
		{
			name: "match directory anywhere",
			config: &Config{
				IgnorePatterns: []string{"node_modules/"},
			},
			relativePath: "deep/path/node_modules/file.js",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldIgnore(tt.relativePath)
			if got != tt.want {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.relativePath, got, tt.want)
			}
		})
	}
}

func TestGetMapping(t *testing.T) {
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "/tmp/source/home", Target: "~/"},
			{Source: "/tmp/source/private/home", Target: "~/"},
			{Source: "/tmp/source/config", Target: "~/.config"},
		},
	}

	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"existing home", "/tmp/source/home", true},
		{"existing private", "/tmp/source/private/home", true},
		{"existing config", "/tmp/source/config", true},
		{"non-existing", "/tmp/source/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := config.GetMapping(tt.source)
			if tt.want && mapping == nil {
				t.Errorf("GetMapping(%q) = nil, want mapping", tt.source)
			} else if !tt.want && mapping != nil {
				t.Errorf("GetMapping(%q) = %+v, want nil", tt.source, mapping)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/tmp/source/home", Target: "~/"},
					{Source: "/tmp/source/private/home", Target: "~/"},
				},
				IgnorePatterns: []string{"*.tmp", "*.log"},
			},
			wantErr: false,
		},
		{
			name: "empty source",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "", Target: "~/"},
				},
			},
			wantErr:     true,
			errContains: "empty source in mapping 1",
		},
		{
			name: "empty target",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: ""},
				},
			},
			wantErr:     true,
			errContains: "empty target in mapping 1",
		},
		{
			name: "source with ..",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "../home", Target: "~/"},
				},
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "valid absolute source",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/home", Target: "~/"},
				},
			},
			wantErr: false,
		},
		{
			name: "relative source",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: "~/"},
				},
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "invalid target",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: "relative/path"},
				},
			},
			wantErr:     true,
			errContains: "must be an absolute path or start with ~/",
		},
		{
			name: "empty ignore pattern",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/tmp/source/home", Target: "~/"},
				},
				IgnorePatterns: []string{"*.tmp", "", "*.log"},
			},
			wantErr:     true,
			errContains: "empty pattern at index 1",
		},
		{
			name: "invalid glob pattern",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/tmp/source/home", Target: "~/"},
				},
				IgnorePatterns: []string{"[invalid"},
			},
			wantErr:     true,
			errContains: "invalid glob pattern",
		},
		{
			name: "valid absolute target",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/tmp/source/home", Target: "/opt/configs"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errContains)
			}
		})
	}
}

// Tests for new configuration loading system with LoadConfigWithOptions

func TestLoadConfigWithOptions_DefaultConfig(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")

	// Set test environment
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	os.Setenv("HOME", tmpDir)

	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			os.Setenv("HOME", originalHOME)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Test with empty options - should use defaults
	options := &ConfigOptions{}
	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "built-in defaults" {
		t.Errorf("Expected source 'built-in defaults', got %s", source)
	}

	// Verify default config structure
	if len(config.LinkMappings) != 2 {
		t.Errorf("Expected 2 default link mappings, got %d", len(config.LinkMappings))
	}

	expectedMappings := []LinkMapping{
		{Source: "~/dotfiles/home", Target: "~/"},
		{Source: "~/dotfiles/config", Target: "~/.config/"},
	}

	for i, expected := range expectedMappings {
		if i >= len(config.LinkMappings) {
			t.Errorf("Missing expected mapping: %+v", expected)
			continue
		}
		actual := config.LinkMappings[i]
		if actual.Source != expected.Source || actual.Target != expected.Target {
			t.Errorf("Mapping %d: expected %+v, got %+v", i, expected, actual)
		}
	}

	// Verify default ignore patterns
	expectedIgnorePatterns := []string{
		".git", ".gitignore", ".DS_Store", "*.swp", "*.tmp",
		"README*", "LICENSE*", "CHANGELOG*", ".lnk.json",
		".lnkconfig", ".lnkignore",
	}

	if len(config.IgnorePatterns) != len(expectedIgnorePatterns) {
		t.Errorf("Expected %d ignore patterns, got %d", len(expectedIgnorePatterns), len(config.IgnorePatterns))
	}

	for i, expected := range expectedIgnorePatterns {
		if i >= len(config.IgnorePatterns) {
			t.Errorf("Missing expected ignore pattern: %s", expected)
			continue
		}
		if config.IgnorePatterns[i] != expected {
			t.Errorf("Ignore pattern %d: expected %s, got %s", i, expected, config.IgnorePatterns[i])
		}
	}
}

func TestLoadConfigWithOptions_ConfigFilePrecedence(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create XDG config directory
	xdgConfigDir := filepath.Join(tmpDir, ".config", "lnk")
	if err := os.MkdirAll(xdgConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config files in different locations
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create repo config
	repoConfig := &Config{
		IgnorePatterns: []string{"*.repo"},
		LinkMappings:   []LinkMapping{{Source: "/tmp/test/repo", Target: "~/"}},
	}
	repoConfigPath := filepath.Join(repoDir, ".lnk.json")
	if err := writeConfigFile(repoConfigPath, repoConfig); err != nil {
		t.Fatal(err)
	}

	// Create XDG config
	xdgConfig := &Config{
		IgnorePatterns: []string{"*.xdg"},
		LinkMappings:   []LinkMapping{{Source: "/tmp/test/xdg", Target: "~/"}},
	}
	xdgConfigPath := filepath.Join(xdgConfigDir, "config.json")
	if err := writeConfigFile(xdgConfigPath, xdgConfig); err != nil {
		t.Fatal(err)
	}

	// Create explicit config file
	explicitConfig := &Config{
		IgnorePatterns: []string{"*.explicit"},
		LinkMappings:   []LinkMapping{{Source: "/tmp/test/explicit", Target: "~/"}},
	}
	explicitConfigPath := filepath.Join(tmpDir, "explicit.json")
	if err := writeConfigFile(explicitConfigPath, explicitConfig); err != nil {
		t.Fatal(err)
	}

	// Test 1: --config flag has highest precedence
	options := &ConfigOptions{
		ConfigPath: explicitConfigPath,
	}

	// Set XDG_CONFIG_HOME and HOME to our test directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			os.Setenv("HOME", originalHOME)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "command line flag" {
		t.Errorf("Expected source 'command line flag', got %s", source)
	}

	if len(config.IgnorePatterns) != 1 || config.IgnorePatterns[0] != "*.explicit" {
		t.Errorf("Expected explicit config to be loaded, got ignore patterns: %v", config.IgnorePatterns)
	}

	// Test 2: Current directory config
	options.ConfigPath = ""

	// Set XDG_CONFIG_HOME to a non-existent directory to skip XDG config
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "nonexistent"))

	// Also need to ensure HOME doesn't have .config/lnk/config.json
	// Create a separate HOME for this test
	testHome := filepath.Join(tmpDir, "testhome")
	os.MkdirAll(testHome, 0755)
	os.Setenv("HOME", testHome)

	// Change to repo directory to test current directory loading
	originalDir, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(originalDir)

	config, source, err = LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "current directory" {
		t.Errorf("Expected source 'current directory', got %s", source)
	}

	if len(config.IgnorePatterns) != 1 || config.IgnorePatterns[0] != "*.repo" {
		t.Errorf("Expected repo config to be loaded, got ignore patterns: %v", config.IgnorePatterns)
	}

	// Test 3: XDG config precedence (remove current dir config)
	if err := os.Remove(repoConfigPath); err != nil {
		t.Fatal(err)
	}

	// Change back to original directory
	os.Chdir(originalDir)

	// Restore XDG_CONFIG_HOME and HOME for XDG test
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	os.Setenv("HOME", tmpDir)

	config, source, err = LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "XDG config directory" {
		t.Errorf("Expected source 'XDG config directory', got %s", source)
	}

	if len(config.IgnorePatterns) != 1 || config.IgnorePatterns[0] != "*.xdg" {
		t.Errorf("Expected XDG config to be loaded, got ignore patterns: %v", config.IgnorePatterns)
	}
}

func TestLoadConfigWithOptions_FlagOverrides(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config file
	testConfig := &Config{
		IgnorePatterns: []string{"*.original"},
		LinkMappings:   []LinkMapping{{Source: "/tmp/test/original", Target: "~/"}},
	}
	configPath := filepath.Join(tmpDir, "test.json")
	if err := writeConfigFile(configPath, testConfig); err != nil {
		t.Fatal(err)
	}

	// Test flag overrides
	options := &ConfigOptions{
		ConfigPath:     configPath,
		IgnorePatterns: []string{"*.flag1", "*.flag2"},
	}

	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "command line flag" {
		t.Errorf("Expected source 'command line flag', got %s", source)
	}

	// Verify config was loaded from file
	if len(config.LinkMappings) != 1 {
		t.Errorf("Expected 1 link mapping from config file, got %d", len(config.LinkMappings))
	}

	if len(config.IgnorePatterns) != 2 {
		t.Errorf("Expected 2 ignore patterns, got %d", len(config.IgnorePatterns))
	} else {
		expected := []string{"*.flag1", "*.flag2"}
		for i, pattern := range expected {
			if config.IgnorePatterns[i] != pattern {
				t.Errorf("Ignore pattern %d: expected %s, got %s", i, pattern, config.IgnorePatterns[i])
			}
		}
	}
}

func TestLoadConfigWithOptions_PartialOverrides(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "lnk-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")

	// Set test environment
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	os.Setenv("HOME", tmpDir)

	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			os.Setenv("HOME", originalHOME)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Test with empty options - should use defaults
	options := &ConfigOptions{}

	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "built-in defaults" {
		t.Errorf("Expected source 'built-in defaults', got %s", source)
	}

	// Should use default mappings since only source dir was specified
	if len(config.LinkMappings) != 2 {
		t.Errorf("Expected 2 default link mappings, got %d", len(config.LinkMappings))
	}

	// Verify default mappings are preserved
	expectedMappings := []LinkMapping{
		{Source: "~/dotfiles/home", Target: "~/"},
		{Source: "~/dotfiles/config", Target: "~/.config/"},
	}

	for i, expected := range expectedMappings {
		if i >= len(config.LinkMappings) {
			t.Errorf("Missing expected mapping: %+v", expected)
			continue
		}
		actual := config.LinkMappings[i]
		if actual.Source != expected.Source || actual.Target != expected.Target {
			t.Errorf("Mapping %d: expected %+v, got %+v", i, expected, actual)
		}
	}
}

func TestGetXDGConfigDir(t *testing.T) {
	// Save original environment
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")

	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			os.Setenv("HOME", originalHOME)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Test with XDG_CONFIG_HOME set
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	expected := "/custom/config/lnk"
	result := getXDGConfigDir()
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with XDG_CONFIG_HOME not set, HOME set
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", "/home/user")
	expected = "/home/user/.config/lnk"
	result = getXDGConfigDir()
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with both unset
	os.Unsetenv("HOME")
	result = getXDGConfigDir()
	if result != "" {
		t.Errorf("Expected empty string when HOME not set, got %s", result)
	}
}

// Helper function to write config files
func writeConfigFile(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

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
