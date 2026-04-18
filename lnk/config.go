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

// Config represents the final merged configuration from all sources
type Config struct {
	SourceDir      string   // Source directory (resolved absolute path)
	TargetDir      string   // Target directory (always ~; configurable in tests)
	IgnorePatterns []string // Combined ignore patterns from all sources
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

// LoadConfig resolves sourceDir, loads ignore patterns, and returns a fully resolved Config.
// The returned SourceDir is always an absolute, validated path.
// Ignore pattern order: built-in defaults + .lnkignore + CLI --ignore patterns.
func LoadConfig(sourceDir string, cliIgnorePatterns []string) (*Config, error) {
	// Resolve sourceDir: expand tilde, then make absolute
	resolvedDir, err := ExpandPath(sourceDir)
	if err != nil {
		return nil, err
	}
	resolvedDir, err = filepath.Abs(resolvedDir)
	if err != nil {
		return nil, NewPathErrorWithHint("resolve path", sourceDir, err,
			"Check that the path is valid")
	}

	// Validate sourceDir exists and is a directory
	info, err := os.Stat(resolvedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewValidationErrorWithHint("source-dir", ContractPath(resolvedDir),
				"directory does not exist",
				fmt.Sprintf("Check that %s exists and is a directory", ContractPath(resolvedDir)))
		}
		return nil, NewPathErrorWithHint("stat", resolvedDir, err,
			"Check file permissions")
	}
	if !info.IsDir() {
		return nil, NewValidationErrorWithHint("source-dir", ContractPath(resolvedDir),
			"not a directory",
			fmt.Sprintf("%s is a file, not a directory", ContractPath(resolvedDir)))
	}

	PrintVerbose("Source directory: %s", ContractPath(resolvedDir))

	// Load ignore patterns from .lnkignore file (if exists)
	ignoreFilePatterns, err := LoadIgnoreFile(resolvedDir)
	if err != nil {
		return nil, err
	}

	// Combine ignore patterns: built-in + .lnkignore + CLI
	ignorePatterns := []string{}
	ignorePatterns = append(ignorePatterns, getBuiltInIgnorePatterns()...)
	ignorePatterns = append(ignorePatterns, ignoreFilePatterns...)
	ignorePatterns = append(ignorePatterns, cliIgnorePatterns...)

	PrintVerbose("Ignore patterns: %d built-in, %d from .lnkignore, %d from CLI = %d total",
		len(getBuiltInIgnorePatterns()), len(ignoreFilePatterns),
		len(cliIgnorePatterns), len(ignorePatterns))

	// Resolve target directory (always ~)
	targetDir, err := ExpandPath("~")
	if err != nil {
		return nil, err
	}

	return &Config{
		SourceDir:      resolvedDir,
		TargetDir:      targetDir,
		IgnorePatterns: ignorePatterns,
	}, nil
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
		".lnkignore",
	}
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", NewPathErrorWithHint("get home directory", path, err,
				"Check that the HOME environment variable is set correctly")
		}
		if path == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// ContractPath contracts the home directory to ~ in paths for display
func ContractPath(path string) string {
	if path == "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, return the original path
		return path
	}

	// Check if path starts with home directory
	if strings.HasPrefix(path, homeDir) {
		// Replace home directory with ~ and clean up any double slashes
		return filepath.Clean("~" + strings.TrimPrefix(path, homeDir))
	}

	return path
}
