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
				Source:          "home",
				Target:          "~/",
				LinkAsDirectory: []string{".config/nvim", ".config/fish"},
			},
			{
				Source:          "private/home",
				Target:          "~/",
				LinkAsDirectory: []string{".config/work", ".ssh"},
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

		if len(loadedMapping.LinkAsDirectory) != len(mapping.LinkAsDirectory) {
			t.Errorf("LinkMapping[%d].LinkAsDirectory length = %d, want %d", i, len(loadedMapping.LinkAsDirectory), len(mapping.LinkAsDirectory))
		}

		for j, dir := range mapping.LinkAsDirectory {
			if j >= len(loadedMapping.LinkAsDirectory) || loadedMapping.LinkAsDirectory[j] != dir {
				t.Errorf("LinkMapping[%d].LinkAsDirectory[%d] = %q, want %q", i, j, loadedMapping.LinkAsDirectory[j], dir)
			}
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
				Source:          "home",
				Target:          "~/",
				LinkAsDirectory: []string{".config/nvim"},
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
	if !strings.Contains(err.Error(), "no configuration file found") {
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
				"source":            "home",
				"target":            "~/",
				"link_as_directory": []string{".config/nvim"},
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

func TestAddDirectoryLinkToMapping(t *testing.T) {
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/", LinkAsDirectory: []string{".config/existing"}},
		},
	}

	tests := []struct {
		name    string
		source  string
		path    string
		wantErr bool
	}{
		{"add to existing mapping", "home", ".config/new", false},
		{"add duplicate", "home", ".config/existing", true},
		{"add to non-existing mapping", "other", ".config/test", true},
		{"add with ./ prefix", "home", "./.config/another", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to avoid state pollution
			testConfig := &Config{
				LinkMappings: make([]LinkMapping, len(config.LinkMappings)),
			}
			for i, m := range config.LinkMappings {
				testConfig.LinkMappings[i] = LinkMapping{
					Source:          m.Source,
					Target:          m.Target,
					LinkAsDirectory: append([]string{}, m.LinkAsDirectory...),
				}
			}

			err := testConfig.AddDirectoryLinkToMapping(tt.source, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddDirectoryLinkToMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldLinkAsDirectoryForMapping(t *testing.T) {
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source:          "home",
				Target:          "~/",
				LinkAsDirectory: []string{".config/nvim", ".config/fish"},
			},
			{
				Source:          "private/home",
				Target:          "~/",
				LinkAsDirectory: []string{".ssh", ".gnupg"},
			},
		},
	}

	tests := []struct {
		name   string
		source string
		path   string
		want   bool
	}{
		{"home existing", "home", ".config/nvim", true},
		{"home not existing", "home", ".config/other", false},
		{"private existing", "private/home", ".ssh", true},
		{"private not existing", "private/home", ".config/nvim", false},
		{"non-existing mapping", "other", ".config/nvim", false},
		{"with ./ prefix", "home", "./.config/nvim", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.ShouldLinkAsDirectoryForMapping(tt.source, tt.path); got != tt.want {
				t.Errorf("ShouldLinkAsDirectoryForMapping(%q, %q) = %v, want %v", tt.source, tt.path, got, tt.want)
			}
		})
	}
}
