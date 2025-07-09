package cfgman

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config with new LinkMappings format
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source: "home",
				Target: "~/",
			},
			{
				Source: "private/home",
				Target: "~/",
			},
		},
	}

	// Save config
	if err := config.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists - should be .cfgman.json for new format
	configPath := filepath.Join(tmpDir, ".cfgman.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(tmpDir)
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
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config with new format
	config := &Config{
		IgnorePatterns: []string{"*.tmp", "backup/"},
		LinkMappings: []LinkMapping{
			{
				Source: "home",
				Target: "~/",
			},
		},
	}

	// Save config - should create .cfgman.json
	if err := config.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify .cfgman.json exists
	cfgmanPath := filepath.Join(tmpDir, ".cfgman.json")
	if _, err := os.Stat(cfgmanPath); err != nil {
		t.Fatalf(".cfgman.json not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(loaded.IgnorePatterns) != 2 {
		t.Errorf("IgnorePatterns length = %d, want 2", len(loaded.IgnorePatterns))
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Load config from directory without config file
	_, err = LoadConfig(tmpDir)
	if err == nil {
		t.Fatal("LoadConfig() should return error when no config file exists")
	}

	// Should return error about missing config file
	if !strings.Contains(err.Error(), "configuration file not found") {
		t.Errorf("LoadConfig() error = %v, want error about missing config file", err)
	}
}

func TestLoadConfigNewFormat(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create new format config file
	newConfig := map[string]interface{}{
		"ignore_patterns": []string{"*.tmp", "backup/", ".DS_Store"},
		"link_mappings": []map[string]interface{}{
			{
				"source": "home",
				"target": "~/",
			},
		},
	}

	data, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmpDir, ".cfgman.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	loaded, err := LoadConfig(tmpDir)
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
			{Source: "home", Target: "~/"},
			{Source: "private/home", Target: "~/"},
			{Source: "config", Target: "~/.config"},
		},
	}

	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{"existing home", "home", true},
		{"existing private", "private/home", true},
		{"existing config", "config", true},
		{"non-existing", "other", false},
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
					{Source: "home", Target: "~/"},
					{Source: "private/home", Target: "~/"},
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
			errContains: "must be a relative path without '..'",
		},
		{
			name: "absolute source",
			config: &Config{
				LinkMappings: []LinkMapping{
					{Source: "/home", Target: "~/"},
				},
			},
			wantErr:     true,
			errContains: "must be a relative path without '..'",
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
					{Source: "home", Target: "~/"},
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
					{Source: "home", Target: "~/"},
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
					{Source: "home", Target: "/opt/configs"},
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
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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
		{Source: "home", Target: "~/"},
		{Source: "config", Target: "~/.config/"},
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
		"README*", "LICENSE*", "CHANGELOG*", ".cfgman.json",
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
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create XDG config directory
	xdgConfigDir := filepath.Join(tmpDir, ".config", "cfgman")
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
		LinkMappings:   []LinkMapping{{Source: "repo", Target: "~/"}},
	}
	repoConfigPath := filepath.Join(repoDir, ".cfgman.json")
	if err := writeConfigFile(repoConfigPath, repoConfig); err != nil {
		t.Fatal(err)
	}

	// Create XDG config
	xdgConfig := &Config{
		IgnorePatterns: []string{"*.xdg"},
		LinkMappings:   []LinkMapping{{Source: "xdg", Target: "~/"}},
	}
	xdgConfigPath := filepath.Join(xdgConfigDir, "config.json")
	if err := writeConfigFile(xdgConfigPath, xdgConfig); err != nil {
		t.Fatal(err)
	}

	// Create explicit config file
	explicitConfig := &Config{
		IgnorePatterns: []string{"*.explicit"},
		LinkMappings:   []LinkMapping{{Source: "explicit", Target: "~/"}},
	}
	explicitConfigPath := filepath.Join(tmpDir, "explicit.json")
	if err := writeConfigFile(explicitConfigPath, explicitConfig); err != nil {
		t.Fatal(err)
	}

	// Test 1: --config flag has highest precedence
	options := &ConfigOptions{
		ConfigPath: explicitConfigPath,
		RepoDir:    repoDir,
	}

	// Set XDG_CONFIG_HOME to our test directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
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

	// Test 2: Repo directory has second precedence
	options.ConfigPath = ""
	config, source, err = LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "repo directory" {
		t.Errorf("Expected source 'repo directory', got %s", source)
	}

	if len(config.IgnorePatterns) != 1 || config.IgnorePatterns[0] != "*.repo" {
		t.Errorf("Expected repo config to be loaded, got ignore patterns: %v", config.IgnorePatterns)
	}

	// Test 3: XDG config has third precedence (remove repo config)
	if err := os.Remove(repoConfigPath); err != nil {
		t.Fatal(err)
	}

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

func TestLoadConfigWithOptions_EnvironmentVariables(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config file
	testConfig := &Config{
		IgnorePatterns: []string{"*.env"},
		LinkMappings:   []LinkMapping{{Source: "env", Target: "~/"}},
	}
	configPath := filepath.Join(tmpDir, "env.json")
	if err := writeConfigFile(configPath, testConfig); err != nil {
		t.Fatal(err)
	}

	// Set environment variables
	originalEnvs := map[string]string{
		"CFGMAN_CONFIG":     os.Getenv("CFGMAN_CONFIG"),
		"CFGMAN_REPO_DIR":   os.Getenv("CFGMAN_REPO_DIR"),
		"CFGMAN_SOURCE_DIR": os.Getenv("CFGMAN_SOURCE_DIR"),
		"CFGMAN_TARGET_DIR": os.Getenv("CFGMAN_TARGET_DIR"),
		"CFGMAN_IGNORE":     os.Getenv("CFGMAN_IGNORE"),
	}

	// Clean up environment variables at the end
	defer func() {
		for key, value := range originalEnvs {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Test environment variables
	os.Setenv("CFGMAN_CONFIG", configPath)
	os.Setenv("CFGMAN_REPO_DIR", tmpDir)
	os.Setenv("CFGMAN_SOURCE_DIR", "env-source")
	os.Setenv("CFGMAN_TARGET_DIR", "~/.env/")
	os.Setenv("CFGMAN_IGNORE", "*.env1,*.env2,*.env3")

	// Test with empty options - should pick up environment variables
	options := &ConfigOptions{}
	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "command line flag" {
		t.Errorf("Expected source 'command line flag', got %s", source)
	}

	// Verify environment variables were applied
	if len(config.LinkMappings) != 1 {
		t.Errorf("Expected 1 link mapping, got %d", len(config.LinkMappings))
	} else {
		mapping := config.LinkMappings[0]
		if mapping.Source != "env-source" || mapping.Target != "~/.env/" {
			t.Errorf("Expected source/target override, got %+v", mapping)
		}
	}

	if len(config.IgnorePatterns) != 3 {
		t.Errorf("Expected 3 ignore patterns, got %d", len(config.IgnorePatterns))
	} else {
		expected := []string{"*.env1", "*.env2", "*.env3"}
		for i, pattern := range expected {
			if config.IgnorePatterns[i] != pattern {
				t.Errorf("Ignore pattern %d: expected %s, got %s", i, pattern, config.IgnorePatterns[i])
			}
		}
	}
}

func TestLoadConfigWithOptions_FlagOverrides(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config file
	testConfig := &Config{
		IgnorePatterns: []string{"*.original"},
		LinkMappings:   []LinkMapping{{Source: "original", Target: "~/"}},
	}
	configPath := filepath.Join(tmpDir, "test.json")
	if err := writeConfigFile(configPath, testConfig); err != nil {
		t.Fatal(err)
	}

	// Test flag overrides
	options := &ConfigOptions{
		ConfigPath:     configPath,
		RepoDir:        tmpDir,
		SourceDir:      "override-source",
		TargetDir:      "~/.override/",
		IgnorePatterns: []string{"*.flag1", "*.flag2"},
	}

	config, source, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	if source != "command line flag" {
		t.Errorf("Expected source 'command line flag', got %s", source)
	}

	// Verify flag overrides were applied
	if len(config.LinkMappings) != 1 {
		t.Errorf("Expected 1 link mapping, got %d", len(config.LinkMappings))
	} else {
		mapping := config.LinkMappings[0]
		if mapping.Source != "override-source" || mapping.Target != "~/.override/" {
			t.Errorf("Expected flag overrides, got %+v", mapping)
		}
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

func TestLoadConfigWithOptions_FlagsPrecedeEnvironment(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set environment variables
	originalEnvs := map[string]string{
		"CFGMAN_REPO_DIR":   os.Getenv("CFGMAN_REPO_DIR"),
		"CFGMAN_SOURCE_DIR": os.Getenv("CFGMAN_SOURCE_DIR"),
		"CFGMAN_TARGET_DIR": os.Getenv("CFGMAN_TARGET_DIR"),
		"CFGMAN_IGNORE":     os.Getenv("CFGMAN_IGNORE"),
	}

	defer func() {
		for key, value := range originalEnvs {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	os.Setenv("CFGMAN_REPO_DIR", "env-repo")
	os.Setenv("CFGMAN_SOURCE_DIR", "env-source")
	os.Setenv("CFGMAN_TARGET_DIR", "~/.env/")
	os.Setenv("CFGMAN_IGNORE", "*.env")

	// Test with flags that should override environment
	options := &ConfigOptions{
		RepoDir:        tmpDir,             // Should override CFGMAN_REPO_DIR
		SourceDir:      "flag-source",      // Should override CFGMAN_SOURCE_DIR
		TargetDir:      "~/.flag/",         // Should override CFGMAN_TARGET_DIR
		IgnorePatterns: []string{"*.flag"}, // Should override CFGMAN_IGNORE
	}

	config, _, err := LoadConfigWithOptions(options)
	if err != nil {
		t.Fatalf("LoadConfigWithOptions() error = %v", err)
	}

	// Verify flags took precedence over environment
	if len(config.LinkMappings) != 1 {
		t.Errorf("Expected 1 link mapping, got %d", len(config.LinkMappings))
	} else {
		mapping := config.LinkMappings[0]
		if mapping.Source != "flag-source" || mapping.Target != "~/.flag/" {
			t.Errorf("Expected flag values, got %+v", mapping)
		}
	}

	if len(config.IgnorePatterns) != 1 || config.IgnorePatterns[0] != "*.flag" {
		t.Errorf("Expected flag ignore pattern, got %v", config.IgnorePatterns)
	}
}

func TestLoadConfigWithOptions_PartialOverrides(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cfgman-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test partial overrides (only source dir specified)
	options := &ConfigOptions{
		RepoDir:   tmpDir,
		SourceDir: "partial-source",
		// TargetDir not specified - should use defaults
	}

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
		{Source: "home", Target: "~/"},
		{Source: "config", Target: "~/.config/"},
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
	expected := "/custom/config/cfgman"
	result := getXDGConfigDir()
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with XDG_CONFIG_HOME not set, HOME set
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", "/home/user")
	expected = "/home/user/.config/cfgman"
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
