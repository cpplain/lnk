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
	Source          string   `json:"source"`
	Target          string   `json:"target"`
	LinkAsDirectory []string `json:"link_as_directory"`
}

// Config represents the link configuration
type Config struct {
	IgnorePatterns []string      `json:"ignore_patterns"` // Gitignore-style patterns to ignore
	LinkMappings   []LinkMapping `json:"link_mappings"`   // Flexible mapping system
}

// LoadConfig reads the configuration from a JSON file in the specified directory
func LoadConfig(configRepo string) (*Config, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(configRepo)
	if err != nil {
		return nil, fmt.Errorf("resolving config directory: %w", err)
	}
	configRepo = absPath
	// Check for .cfgman.json
	cfgmanPath := filepath.Join(configRepo, ".cfgman.json")
	if _, err := os.Stat(cfgmanPath); err != nil {
		// Config file doesn't exist
		return nil, fmt.Errorf("no configuration file found: please create .cfgman.json in %s", configRepo)
	}

	// Load the config
	data, err := os.ReadFile(cfgmanPath)
	if err != nil {
		return nil, fmt.Errorf("reading .cfgman.json: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing .cfgman.json: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to a JSON file
func (c *Config) Save(configRepo string) error {
	// Always save to .cfgman.json
	cfgmanPath := filepath.Join(configRepo, ".cfgman.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(cfgmanPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
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

// AddDirectoryLinkToMapping adds a directory to a specific mapping's LinkAsDirectory list
func (c *Config) AddDirectoryLinkToMapping(source, relativePath string) error {
	// Normalize path (remove leading ./ if present)
	relativePath = strings.TrimPrefix(relativePath, "./")

	mapping := c.GetMapping(source)
	if mapping == nil {
		return fmt.Errorf("no mapping found for source: %s", source)
	}

	// Check if already exists
	for _, dir := range mapping.LinkAsDirectory {
		if dir == relativePath {
			return fmt.Errorf("path already configured to link as directory: %s", relativePath)
		}
	}

	mapping.LinkAsDirectory = append(mapping.LinkAsDirectory, relativePath)
	return nil
}

// ShouldLinkAsDirectoryForMapping checks if a path should be linked as a directory for a specific mapping
func (c *Config) ShouldLinkAsDirectoryForMapping(source, relativePath string) bool {
	// Normalize path (remove leading ./ if present)
	relativePath = strings.TrimPrefix(relativePath, "./")

	mapping := c.GetMapping(source)
	if mapping == nil {
		return false
	}

	for _, dir := range mapping.LinkAsDirectory {
		if relativePath == dir {
			return true
		}
	}
	return false
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
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}
