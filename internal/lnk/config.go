// Package lnk provides functionality for managing configuration files
// across machines using intelligent symlinks. It handles the adoption of
// existing files into a repository, creation and management of symlinks,
// and tracking of configuration file status.
package lnk

import (
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
	IgnorePatterns []string // Ignore patterns override
}

// FlagConfig represents configuration loaded from flag-based config files
type FlagConfig struct {
	Target         string   // Target directory (default: ~)
	IgnorePatterns []string // Ignore patterns from config file
}

// MergedConfig represents the final merged configuration from all sources
type MergedConfig struct {
	SourceDir      string   // Source directory (from CLI)
	TargetDir      string   // Target directory (CLI > config > default)
	IgnorePatterns []string // Combined ignore patterns from all sources
}

// parseFlagConfigFile parses a flag-based config file (stow-style)
// Format: one flag per line, e.g., "--target=~" or "--ignore=*.swp"
func parseFlagConfigFile(filePath string) (*FlagConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &FlagConfig{
		IgnorePatterns: []string{},
	}

	lines := strings.Split(string(data), "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse flag format: --flag=value or --flag value
		if !strings.HasPrefix(line, "--") {
			return nil, fmt.Errorf("invalid flag format at line %d: %q (flags must start with --)", lineNum+1, line)
		}

		// Remove leading --
		line = strings.TrimPrefix(line, "--")

		// Split on = or space
		var flagName, flagValue string
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			flagName = parts[0]
			flagValue = parts[1]
		} else {
			flagName = line
		}

		// Parse known flags
		switch flagName {
		case "target", "t":
			config.Target = flagValue
		case "ignore":
			if flagValue != "" {
				config.IgnorePatterns = append(config.IgnorePatterns, flagValue)
			}
		default:
			// Ignore unknown flags for forward compatibility
			PrintVerbose("Ignoring unknown flag in config: %s", flagName)
		}
	}

	return config, nil
}

// parseIgnoreFile parses a .lnkignore file (gitignore syntax)
func parseIgnoreFile(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ignore file: %w", err)
	}

	patterns := []string{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, line)
	}

	return patterns, nil
}

// LoadFlagConfig loads configuration from flag-based config files (.lnkconfig)
// Discovery order:
// 1. .lnkconfig in source directory (repo-specific)
// 2. $XDG_CONFIG_HOME/lnk/config or ~/.config/lnk/config
// 3. ~/.lnkconfig
func LoadFlagConfig(sourceDir string) (*FlagConfig, string, error) {
	// Expand source directory path
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path for source dir: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Define search paths in precedence order
	configPaths := []struct {
		path   string
		source string
	}{
		{filepath.Join(absSourceDir, FlagConfigFileName), "source directory"},
		{filepath.Join(getXDGConfigDir(), "config"), "XDG config directory"},
		{filepath.Join(homeDir, ".config", "lnk", "config"), "user config directory"},
		{filepath.Join(homeDir, FlagConfigFileName), "home directory"},
	}

	// Try each path
	for _, cp := range configPaths {
		PrintVerbose("Looking for config at: %s", cp.path)

		if _, err := os.Stat(cp.path); err == nil {
			config, err := parseFlagConfigFile(cp.path)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse config from %s: %w", cp.source, err)
			}

			PrintVerbose("Loaded config from %s: %s", cp.source, cp.path)
			return config, cp.path, nil
		}
	}

	// No config file found - return empty config
	PrintVerbose("No flag-based config file found")
	return &FlagConfig{IgnorePatterns: []string{}}, "", nil
}

// LoadIgnoreFile loads ignore patterns from a .lnkignore file in the source directory
func LoadIgnoreFile(sourceDir string) ([]string, error) {
	// Expand source directory path
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for source dir: %w", err)
	}

	ignoreFilePath := filepath.Join(absSourceDir, IgnoreFileName)

	// Check if ignore file exists
	if _, err := os.Stat(ignoreFilePath); os.IsNotExist(err) {
		PrintVerbose("No .lnkignore file found at: %s", ignoreFilePath)
		return []string{}, nil
	}

	patterns, err := parseIgnoreFile(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .lnkignore: %w", err)
	}

	PrintVerbose("Loaded %d ignore patterns from .lnkignore", len(patterns))
	return patterns, nil
}

// MergeFlagConfig merges CLI options with config files to produce final configuration
// Precedence for target: CLI flag > .lnkconfig > default (~)
// Precedence for ignore patterns: All sources are combined (built-in + config + .lnkignore + CLI)
func MergeFlagConfig(sourceDir, cliTarget string, cliIgnorePatterns []string) (*MergedConfig, error) {
	PrintVerbose("Merging configuration from sourceDir=%s, cliTarget=%s, cliIgnorePatterns=%v",
		sourceDir, cliTarget, cliIgnorePatterns)

	// Load flag-based config from .lnkconfig file (if exists)
	flagConfig, configPath, err := LoadFlagConfig(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load flag config: %w", err)
	}

	// Load ignore patterns from .lnkignore file (if exists)
	ignoreFilePatterns, err := LoadIgnoreFile(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load ignore file: %w", err)
	}

	// Determine target directory with precedence: CLI > config file > default
	targetDir := "~"
	if cliTarget != "" {
		targetDir = cliTarget
		PrintVerbose("Using target from CLI flag: %s", targetDir)
	} else if flagConfig.Target != "" {
		targetDir = flagConfig.Target
		if configPath != "" {
			PrintVerbose("Using target from config file: %s (from %s)", targetDir, configPath)
		}
	} else {
		PrintVerbose("Using default target: %s", targetDir)
	}

	// Combine all ignore patterns from different sources
	// Order: built-in defaults + config file + .lnkignore + CLI flags
	// This allows CLI flags to override earlier patterns using negation (!)
	ignorePatterns := []string{}
	ignorePatterns = append(ignorePatterns, getBuiltInIgnorePatterns()...)
	ignorePatterns = append(ignorePatterns, flagConfig.IgnorePatterns...)
	ignorePatterns = append(ignorePatterns, ignoreFilePatterns...)
	ignorePatterns = append(ignorePatterns, cliIgnorePatterns...)

	PrintVerbose("Merged ignore patterns: %d built-in, %d from config, %d from .lnkignore, %d from CLI = %d total",
		len(getBuiltInIgnorePatterns()), len(flagConfig.IgnorePatterns),
		len(ignoreFilePatterns), len(cliIgnorePatterns), len(ignorePatterns))

	return &MergedConfig{
		SourceDir:      sourceDir,
		TargetDir:      targetDir,
		IgnorePatterns: ignorePatterns,
	}, nil
}

// getXDGConfigDir returns the XDG config directory for lnk
func getXDGConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "lnk")
	}

	// Fall back to ~/.config/lnk
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "lnk")
}

// getBuiltInIgnorePatterns returns the built-in default ignore patterns
func getBuiltInIgnorePatterns() []string {
	return []string{
		".git",
		".gitignore",
		".DS_Store",
		"*.swp",
		"*.tmp",
		"README*",
		"LICENSE*",
		"CHANGELOG*",
		".lnk.json",
		".lnkconfig",
		".lnkignore",
	}
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
