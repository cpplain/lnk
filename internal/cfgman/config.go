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

// LoadConfig reads the configuration from a JSON file in the specified directory
func LoadConfig(configRepo string) (*Config, error) {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config directory: %w", err)
	}
	// Check for .cfgman.json
	cfgmanPath := filepath.Join(absConfigRepo, ConfigFileName)
	if _, err := os.Stat(cfgmanPath); err != nil {
		// Config file doesn't exist
		if os.IsNotExist(err) {
			return nil, NewPathError("load config", cfgmanPath, ErrConfigNotFound)
		}
		return nil, NewPathError("stat config", cfgmanPath, err)
	}

	// Load the config
	data, err := os.ReadFile(cfgmanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read %s: file not found. Run 'cfgman init' to create a config file", ConfigFileName)
		}
		return nil, fmt.Errorf("failed to read %s: %w", ConfigFileName, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, NewPathError("failed to parse config", cfgmanPath, fmt.Errorf("%w: %v", ErrInvalidConfig, err))
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save writes the configuration to a JSON file
func (c *Config) Save(configRepo string) error {
	// Always save to .cfgman.json
	cfgmanPath := filepath.Join(configRepo, ConfigFileName)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cfgmanPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
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
			return "", fmt.Errorf("failed to get home directory: %w", err)
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
			return NewValidationError("link mapping source", "", fmt.Sprintf("empty source in mapping %d", i+1))
		}
		if mapping.Target == "" {
			return NewValidationError("link mapping target", "", fmt.Sprintf("empty target in mapping %d", i+1))
		}

		// Source should not contain ".." or absolute paths
		if strings.Contains(mapping.Source, "..") || filepath.IsAbs(mapping.Source) {
			return NewValidationError("link mapping source", mapping.Source, "must be a relative path without '..'")
		}

		// Target should be a valid path (can be absolute or start with ~/)
		if mapping.Target != "~/" && !strings.HasPrefix(mapping.Target, "~/") && !filepath.IsAbs(mapping.Target) {
			return NewValidationError("link mapping target", mapping.Target, "must be an absolute path or start with ~/")
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
func DetermineSourceMapping(target, configRepo string, config *Config) string {
	// Remove the config repo prefix to get the relative path
	relPath := strings.TrimPrefix(target, configRepo)
	relPath = strings.TrimPrefix(relPath, "/")

	// Check each mapping to find which one contains this path
	for _, mapping := range config.LinkMappings {
		if strings.HasPrefix(relPath, mapping.Source+"/") || relPath == mapping.Source {
			return mapping.Source
		}
	}

	// Default to showing the first directory component
	parts := strings.Split(relPath, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return "unknown"
}
