package cfgman

import (
	"path/filepath"
	"strings"
)

// PatternMatcher handles gitignore-style pattern matching
type PatternMatcher struct {
	patterns []compiledPattern
}

// compiledPattern represents a parsed pattern with its properties
type compiledPattern struct {
	pattern    string
	isNegation bool
	isDir      bool
	hasSlash   bool
	isGlob     bool
}

// NewPatternMatcher creates a new pattern matcher with the given patterns
func NewPatternMatcher(patterns []string) *PatternMatcher {
	pm := &PatternMatcher{
		patterns: make([]compiledPattern, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		if compiled := compilePattern(pattern); compiled != nil {
			pm.patterns = append(pm.patterns, *compiled)
		}
	}

	return pm
}

// MatchesPattern checks if a path matches any of the patterns
// Returns true if the path should be ignored
func MatchesPattern(path string, patterns []string) bool {
	pm := NewPatternMatcher(patterns)
	return pm.Matches(path)
}

// Matches checks if a path matches any of the patterns
func (pm *PatternMatcher) Matches(path string) bool {
	// Normalize the path
	path = normalizePathForMatching(path)

	// Check each pattern in order, tracking match state
	// Patterns are processed sequentially, with later patterns overriding earlier ones
	matched := false

	for _, pattern := range pm.patterns {
		if pattern.isNegation {
			// Negation patterns only apply if we're currently matched
			if matched && matchesPattern(path, compiledPattern{
				pattern:    pattern.pattern,
				isDir:      pattern.isDir,
				hasSlash:   pattern.hasSlash,
				isGlob:     pattern.isGlob,
				isNegation: false, // Treat as non-negated for matching
			}) {
				matched = false
			}
		} else {
			// Regular patterns
			if matchesPattern(path, pattern) {
				matched = true
			}
		}
	}

	return matched
}

// compilePattern parses a pattern string into a compiledPattern
func compilePattern(pattern string) *compiledPattern {
	pattern = strings.TrimSpace(pattern)

	// Skip empty lines and comments
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return nil
	}

	cp := &compiledPattern{
		pattern: pattern,
	}

	// Check for negation
	if strings.HasPrefix(pattern, "!") {
		cp.isNegation = true
		pattern = pattern[1:]
		cp.pattern = pattern
	}

	// Check if it's a directory pattern
	if strings.HasSuffix(pattern, "/") {
		cp.isDir = true
		pattern = strings.TrimSuffix(pattern, "/")
		cp.pattern = pattern
	}

	// Check if pattern contains a slash (affects matching behavior)
	cp.hasSlash = strings.Contains(cp.pattern, "/")

	// Check if it's a glob pattern
	cp.isGlob = strings.ContainsAny(cp.pattern, "*?[")

	return cp
}

// normalizePathForMatching prepares a path for pattern matching
func normalizePathForMatching(path string) string {
	// Convert backslashes to forward slashes for cross-platform consistency
	// filepath.ToSlash only works on Windows, so we do it manually
	path = strings.ReplaceAll(path, "\\", "/")

	// Remove leading ./ if present
	path = strings.TrimPrefix(path, "./")

	// Remove trailing slash for consistency
	path = strings.TrimSuffix(path, "/")

	return path
}

// matchesPattern checks if a path matches a single compiled pattern
func matchesPattern(path string, pattern compiledPattern) bool {
	// Directory patterns
	if pattern.isDir {
		// If pattern has a slash, it's an absolute path from root
		if pattern.hasSlash {
			// Check if the path is the directory itself or a file within it
			if path == pattern.pattern {
				return true
			}
			return strings.HasPrefix(path, pattern.pattern+"/")
		}

		// Pattern without slash can match directory anywhere
		// Check if pattern matches any directory component in the path
		parts := strings.Split(path, "/")
		for i, part := range parts {
			if part == pattern.pattern {
				// Found matching directory, check if path continues into it
				if i < len(parts)-1 {
					return true
				}
			}
		}

		// Also check if the entire path is the directory
		return path == pattern.pattern
	}

	// Handle double wildcard patterns
	if strings.Contains(pattern.pattern, "**") {
		return matchesDoubleWildcard(path, pattern.pattern)
	}

	// If pattern contains a slash, match against the full path
	if pattern.hasSlash {
		if pattern.isGlob {
			matched, _ := filepath.Match(pattern.pattern, path)
			return matched
		}
		return path == pattern.pattern
	}

	// Pattern without slash matches against basename or any path ending
	basename := filepath.Base(path)

	if pattern.isGlob {
		// First try matching against basename
		if matched, _ := filepath.Match(pattern.pattern, basename); matched {
			return true
		}

		// Also try matching against the full path for patterns like "*.log"
		// This allows "*.log" to match "dir/file.log"
		if matched, _ := filepath.Match(pattern.pattern, path); matched {
			return true
		}

		// For patterns without slashes, also check if any parent directory matches
		// This allows "node_modules" to match "path/to/node_modules/file.js"
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if matched, _ := filepath.Match(pattern.pattern, part); matched {
				return true
			}
		}
	} else {
		// Literal match against basename
		if basename == pattern.pattern {
			return true
		}

		// Also check if any parent directory matches exactly
		// This allows "node_modules" to match "path/to/node_modules/file.js"
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if part == pattern.pattern {
				return true
			}
		}
	}

	return false
}

// matchesDoubleWildcard handles patterns containing **
func matchesDoubleWildcard(path, pattern string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	if len(parts) == 2 {
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")

		// Handle cases like "**/node_modules"
		if prefix == "" {
			// Pattern starts with **, match suffix anywhere in path
			if suffix == "" {
				return true // ** matches everything
			}

			// Check if suffix matches end of path
			if strings.HasSuffix(path, suffix) {
				return true
			}

			// Check if suffix matches any path component
			if strings.Contains(path, "/"+suffix+"/") {
				return true
			}

			// Check if path starts with suffix
			if strings.HasPrefix(path, suffix+"/") {
				return true
			}

			// Check exact match
			if path == suffix {
				return true
			}
		}

		// Handle cases like "src/**/test"
		if prefix != "" && suffix != "" {
			if strings.HasPrefix(path, prefix+"/") {
				remaining := strings.TrimPrefix(path, prefix+"/")
				// Check if suffix matches anywhere in the remaining path
				if strings.Contains(remaining, suffix) {
					return true
				}
			}
		}

		// Handle cases like "src/**"
		if prefix != "" && suffix == "" {
			return strings.HasPrefix(path, prefix+"/") || path == prefix
		}
	}

	return false
}
