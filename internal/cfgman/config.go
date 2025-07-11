// Package cfgman provides functionality for managing configuration files
// across machines using intelligent symlinks. It handles the adoption of
// existing files into a repository, creation and management of symlinks,
// and tracking of configuration file status.
package cfgman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LinkMapping represents a mapping from source to target directory
type LinkMapping struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Config represents the link configuration
type Config struct {
	IgnorePatterns []string      `json:"ignore_patterns"` // Gitignore-style patterns to ignore
	LinkMappings   []LinkMapping `json:"link_mappings"`   // Flexible mapping system
}

// ConfigOptions represents all configuration options that can be overridden by flags/env vars
type ConfigOptions struct {
	ConfigPath     string   // Path to config file
	SourceDir      string   // Source directory (absolute path)
	TargetDir      string   // Target directory override
	IgnorePatterns []string // Ignore patterns override
}

// getXDGConfigDir returns the XDG config directory for cfgman
func getXDGConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "cfgman")
	}

	// Fall back to ~/.config/cfgman
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "cfgman")
}

// getDefaultConfig returns the built-in default configuration
func getDefaultConfig() *Config {
	return &Config{
		IgnorePatterns: []string{
			".git",
			".gitignore",
			".DS_Store",
			"*.swp",
			"*.tmp",
			"README*",
			"LICENSE*",
			"CHANGELOG*",
			".cfgman.json",
		},
		LinkMappings: []LinkMapping{
			{Source: "~/dotfiles/home", Target: "~/"},
			{Source: "~/dotfiles/config", Target: "~/.config/"},
		},
	}
}

// LoadConfig reads the configuration from a JSON file
// This function is now deprecated - use LoadConfigWithOptions instead
func LoadConfig(configPath string) (*Config, error) {
	PrintVerbose("Loading configuration from: %s", configPath)

	// Load the config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewPathErrorWithHint("read config", configPath, err,
				"Create a configuration file or use built-in defaults with command-line options")
		}
		return nil, fmt.Errorf("failed to read %s: %w", ConfigFileName, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, NewPathErrorWithHint("parse config", configPath,
			fmt.Errorf("%w: %v", ErrInvalidConfig, err),
			"Check your JSON syntax. Common issues: missing commas, unclosed brackets, or trailing commas")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	PrintVerbose("Successfully loaded config with %d link mappings and %d ignore patterns",
		len(config.LinkMappings), len(config.IgnorePatterns))

	return &config, nil
}

// loadConfigFromFile loads configuration from a specific file path
func loadConfigFromFile(filePath string) (*Config, error) {
	if filePath == "" {
		return nil, fmt.Errorf("config file path is empty")
	}

	PrintVerbose("Attempting to load config from: %s", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Read and parse config file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filePath, err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", filePath, err)
	}

	PrintVerbose("Successfully loaded config from: %s", filePath)
	return &config, nil
}

// applyEnvironmentVariables applies environment variable overrides to options
func applyEnvironmentVariables(options *ConfigOptions) {
	if envConfig := os.Getenv("CFGMAN_CONFIG"); envConfig != "" && options.ConfigPath == "" {
		options.ConfigPath = envConfig
		PrintVerbose("Using config path from CFGMAN_CONFIG: %s", envConfig)
	}

	if envSourceDir := os.Getenv("CFGMAN_SOURCE_DIR"); envSourceDir != "" && options.SourceDir == "" {
		options.SourceDir = envSourceDir
		PrintVerbose("Using source directory from CFGMAN_SOURCE_DIR: %s", envSourceDir)
	}

	if envTargetDir := os.Getenv("CFGMAN_TARGET_DIR"); envTargetDir != "" && options.TargetDir == "" {
		options.TargetDir = envTargetDir
		PrintVerbose("Using target directory from CFGMAN_TARGET_DIR: %s", envTargetDir)
	}

	if envIgnore := os.Getenv("CFGMAN_IGNORE"); envIgnore != "" && len(options.IgnorePatterns) == 0 {
		// Split by comma for multiple patterns
		options.IgnorePatterns = strings.Split(envIgnore, ",")
		for i := range options.IgnorePatterns {
			options.IgnorePatterns[i] = strings.TrimSpace(options.IgnorePatterns[i])
		}
		PrintVerbose("Using ignore patterns from CFGMAN_IGNORE: %v", options.IgnorePatterns)
	}
}

// LoadConfigWithOptions loads configuration using the precedence system
func LoadConfigWithOptions(options *ConfigOptions) (*Config, string, error) {
	PrintVerbose("Loading configuration with options: %+v", options)

	// Apply environment variables (only if not already set by flags)
	applyEnvironmentVariables(options)

	var config *Config
	var configSource string

	// Try to load config from various sources in precedence order
	configPaths := []struct {
		path   string
		source string
	}{
		{options.ConfigPath, "command line flag"},
		{filepath.Join(getXDGConfigDir(), "config.json"), "XDG config directory"},
		{filepath.Join(os.ExpandEnv("$HOME"), ".config", "cfgman", "config.json"), "user config directory"},
		{filepath.Join(os.ExpandEnv("$HOME"), ".cfgman.json"), "user home directory"},
		{filepath.Join(".", ConfigFileName), "current directory"},
	}

	for _, configPath := range configPaths {
		if configPath.path == "" {
			continue
		}

		loadedConfig, err := loadConfigFromFile(configPath.path)
		if err == nil {
			config = loadedConfig
			configSource = configPath.source
			PrintVerbose("Using config from: %s (%s)", configPath.path, configSource)
			break
		}
		PrintVerbose("Config not found at: %s (%s)", configPath.path, configPath.source)
	}

	// If no config file found, use defaults
	if config == nil {
		config = getDefaultConfig()
		configSource = "built-in defaults"
		PrintVerbose("Using built-in default configuration")
	}

	// Apply overrides from options
	if options.SourceDir != "" || options.TargetDir != "" {
		// Create a custom mapping if source/target dirs are specified
		if options.SourceDir != "" && options.TargetDir != "" {
			config.LinkMappings = []LinkMapping{
				{Source: options.SourceDir, Target: options.TargetDir},
			}
			PrintVerbose("Overriding link mappings with: %s -> %s", options.SourceDir, options.TargetDir)
		}
	}

	if len(options.IgnorePatterns) > 0 {
		config.IgnorePatterns = options.IgnorePatterns
		PrintVerbose("Overriding ignore patterns with: %v", options.IgnorePatterns)
	}

	return config, configSource, nil
}

// Save writes the configuration to a JSON file
func (c *Config) Save(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return NewPathErrorWithHint("write config", configPath, err,
			"Check that you have write permissions in this directory")
	}

	return nil
}

// GetMapping finds a mapping by source directory
func (c *Config) GetMapping(source string) *LinkMapping {
	for i := range c.LinkMappings {
		if c.LinkMappings[i].Source == source {
			return &c.LinkMappings[i]
		}
	}
	return nil
}

// ShouldIgnore checks if a path matches any of the ignore patterns
func (c *Config) ShouldIgnore(relativePath string) bool {
	return MatchesPattern(relativePath, c.IgnorePatterns)
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", NewPathErrorWithHint("get home directory", path, err,
				"Check that the HOME environment variable is set correctly")
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate link mappings
	for i, mapping := range c.LinkMappings {
		if mapping.Source == "" {
			return NewValidationErrorWithHint("link mapping source", "",
				fmt.Sprintf("empty source in mapping %d", i+1),
				"Set source to a directory in your repo (e.g., 'home' or 'config')")
		}
		if mapping.Target == "" {
			return NewValidationErrorWithHint("link mapping target", "",
				fmt.Sprintf("empty target in mapping %d", i+1),
				"Set target to where files should be linked (e.g., '~/' for home directory)")
		}

		// Source should be an absolute path or start with ~/
		if mapping.Source != "~/" && !strings.HasPrefix(mapping.Source, "~/") && !filepath.IsAbs(mapping.Source) {
			return NewValidationErrorWithHint("link mapping source", mapping.Source,
				"must be an absolute path or start with ~/",
				"Examples: '~/dotfiles/home' for home configs, '/opt/configs' for system configs")
		}

		// Target should be a valid path (can be absolute or start with ~/)
		if mapping.Target != "~/" && !strings.HasPrefix(mapping.Target, "~/") && !filepath.IsAbs(mapping.Target) {
			return NewValidationErrorWithHint("link mapping target", mapping.Target,
				"must be an absolute path or start with ~/",
				"Examples: '~/' for home, '~/.config' for config directory")
		}
	}

	// Validate ignore patterns (basic check for malformed patterns)
	for i, pattern := range c.IgnorePatterns {
		if pattern == "" {
			return NewValidationError("ignore pattern", "", fmt.Sprintf("empty pattern at index %d", i))
		}
		// Test if the pattern compiles (for glob patterns)
		if strings.ContainsAny(pattern, "*?[") {
			if _, err := filepath.Match(pattern, "test"); err != nil {
				return NewValidationError("ignore pattern", pattern, fmt.Sprintf("invalid glob pattern: %v", err))
			}
		}
	}

	return nil
}

// DetermineSourceMapping determines which source mapping a target path belongs to
func DetermineSourceMapping(target string, config *Config) string {
	// Check each mapping to find which one contains this path
	for _, mapping := range config.LinkMappings {
		// Expand the source to get absolute path
		absSource, err := ExpandPath(mapping.Source)
		if err != nil {
			continue
		}

		// Check if target is within this source directory
		if strings.HasPrefix(target, absSource+"/") || target == absSource {
			return mapping.Source
		}
	}

	return "unknown"
}
