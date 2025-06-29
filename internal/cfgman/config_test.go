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
