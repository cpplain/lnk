package lnk

import (
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		// Basic literal matches
		{
			name:     "exact file match",
			path:     "file.txt",
			patterns: []string{"file.txt"},
			want:     true,
		},
		{
			name:     "exact file no match",
			path:     "other.txt",
			patterns: []string{"file.txt"},
			want:     false,
		},
		{
			name:     "file in subdirectory",
			path:     "dir/file.txt",
			patterns: []string{"file.txt"},
			want:     true,
		},

		// Wildcard patterns
		{
			name:     "wildcard extension match",
			path:     "test.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "wildcard extension match in subdir",
			path:     "logs/app.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "wildcard prefix match",
			path:     "test_file.txt",
			patterns: []string{"test_*"},
			want:     true,
		},
		{
			name:     "multiple wildcards",
			path:     "test_file_2023.log",
			patterns: []string{"test_*.log"},
			want:     true,
		},

		// Directory patterns
		{
			name:     "directory pattern matches dir",
			path:     "temp",
			patterns: []string{"temp/"},
			want:     true,
		},
		{
			name:     "directory pattern matches file in dir",
			path:     "temp/file.txt",
			patterns: []string{"temp/"},
			want:     true,
		},
		{
			name:     "directory pattern matches nested file",
			path:     "temp/subdir/file.txt",
			patterns: []string{"temp/"},
			want:     true,
		},
		{
			name:     "directory pattern no match",
			path:     "temporary/file.txt",
			patterns: []string{"temp/"},
			want:     false,
		},

		// Double wildcard patterns
		{
			name:     "double wildcard matches any depth",
			path:     "src/deep/nested/node_modules/file.js",
			patterns: []string{"**/node_modules"},
			want:     true,
		},
		{
			name:     "double wildcard at root",
			path:     "node_modules/file.js",
			patterns: []string{"**/node_modules"},
			want:     true,
		},
		{
			name:     "double wildcard exact match",
			path:     "node_modules",
			patterns: []string{"**/node_modules"},
			want:     true,
		},
		{
			name:     "double wildcard with prefix",
			path:     "src/components/test/file.js",
			patterns: []string{"src/**/test"},
			want:     true,
		},
		{
			name:     "double wildcard with suffix",
			path:     "src/anything/here.txt",
			patterns: []string{"src/**"},
			want:     true,
		},
		{
			name:     "double wildcard matches all",
			path:     "any/path/to/file.txt",
			patterns: []string{"**"},
			want:     true,
		},

		// Negation patterns
		{
			name:     "negation overrides previous match",
			path:     "important.log",
			patterns: []string{"*.log", "!important.log"},
			want:     false,
		},
		{
			name:     "negation with no previous match",
			path:     "file.txt",
			patterns: []string{"*.log", "!important.log"},
			want:     false,
		},
		{
			name:     "multiple negations",
			path:     "test.log",
			patterns: []string{"*.log", "!important.log", "!critical.log"},
			want:     true,
		},
		{
			name:     "negation then re-ignore",
			path:     "test.log",
			patterns: []string{"*.log", "!test.log", "test.log"},
			want:     true,
		},

		// Comments and empty lines
		{
			name:     "comments are ignored",
			path:     "file.txt",
			patterns: []string{"# This is a comment", "file.txt"},
			want:     true,
		},
		{
			name:     "empty patterns ignored",
			path:     "file.txt",
			patterns: []string{"", "   ", "file.txt"},
			want:     true,
		},

		// Path with slashes
		{
			name:     "pattern with slash exact match",
			path:     "src/test.js",
			patterns: []string{"src/test.js"},
			want:     true,
		},
		{
			name:     "pattern with slash no match different path",
			path:     "lib/test.js",
			patterns: []string{"src/test.js"},
			want:     false,
		},
		{
			name:     "pattern with slash and wildcard",
			path:     "src/components/Button.js",
			patterns: []string{"src/*.js"},
			want:     false, // Doesn't match because * doesn't cross directory boundaries
		},
		{
			name:     "pattern with slash and wildcard direct match",
			path:     "src/index.js",
			patterns: []string{"src/*.js"},
			want:     true,
		},

		// Special cases
		{
			name:     "hidden files",
			path:     ".gitignore",
			patterns: []string{".gitignore"},
			want:     true,
		},
		{
			name:     "hidden directories",
			path:     ".git/config",
			patterns: []string{".git/"},
			want:     true,
		},
		{
			name:     "match directory name anywhere",
			path:     "src/node_modules/package/file.js",
			patterns: []string{"node_modules"},
			want:     true,
		},
		{
			name:     "leading slash removed from path",
			path:     "./src/file.js",
			patterns: []string{"src/file.js"},
			want:     true,
		},

		// Complex patterns
		{
			name:     "multiple patterns with mixed results",
			path:     "build/temp/output.log",
			patterns: []string{"build/", "!build/output/", "*.log"},
			want:     true,
		},
		{
			name:     "nested gitignore behavior",
			path:     "docs/api/private/secret.md",
			patterns: []string{"docs/", "!docs/api/", "private/"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPattern(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("MatchesPattern(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestPatternMatcher(t *testing.T) {
	// Test with pre-compiled pattern matcher for performance testing
	patterns := []string{
		"*.log",
		"*.tmp",
		"node_modules/",
		"**/target",
		"!important.log",
		"# Comment line",
		"",
		"build/",
	}

	pm := NewPatternMatcher(patterns)

	testPaths := []struct {
		path string
		want bool
	}{
		{"test.log", true},
		{"important.log", false},
		{"test.tmp", true},
		{"node_modules/package.json", true},
		{"src/node_modules/test.js", true},
		{"project/target/classes", true},
		{"target", true},
		{"build/output.txt", true},
		{"README.md", false},
	}

	for _, test := range testPaths {
		got := pm.Matches(test.path)
		if got != test.want {
			t.Errorf("PatternMatcher.Matches(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestCompilePattern(t *testing.T) {
	tests := []struct {
		pattern    string
		wantNil    bool
		isNegation bool
		isDir      bool
		hasSlash   bool
		isGlob     bool
	}{
		{"file.txt", false, false, false, false, false},
		{"*.log", false, false, false, false, true},
		{"!important.log", false, true, false, false, false},
		{"temp/", false, false, true, false, false},
		{"src/test.js", false, false, false, true, false},
		{"**/node_modules", false, false, false, true, true},
		{"# comment", true, false, false, false, false},
		{"", true, false, false, false, false},
		{"   ", true, false, false, false, false},
		{"src/*.js", false, false, false, true, true},
		{"!src/", false, true, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := compilePattern(tt.pattern)

			if tt.wantNil {
				if got != nil {
					t.Errorf("compilePattern(%q) = %v, want nil", tt.pattern, got)
				}
				return
			}

			if got == nil {
				t.Errorf("compilePattern(%q) = nil, want non-nil", tt.pattern)
				return
			}

			if got.isNegation != tt.isNegation {
				t.Errorf("compilePattern(%q).isNegation = %v, want %v", tt.pattern, got.isNegation, tt.isNegation)
			}
			if got.isDir != tt.isDir {
				t.Errorf("compilePattern(%q).isDir = %v, want %v", tt.pattern, got.isDir, tt.isDir)
			}
			if got.hasSlash != tt.hasSlash {
				t.Errorf("compilePattern(%q).hasSlash = %v, want %v", tt.pattern, got.hasSlash, tt.hasSlash)
			}
			if got.isGlob != tt.isGlob {
				t.Errorf("compilePattern(%q).isGlob = %v, want %v", tt.pattern, got.isGlob, tt.isGlob)
			}
		})
	}
}

func TestNormalizePathForMatching(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"file.txt", "file.txt"},
		{"./file.txt", "file.txt"},
		{"./src/file.txt", "src/file.txt"},
		{"src/", "src"},
		{"./src/", "src"},
		{"path\\to\\file.txt", "path/to/file.txt"},
	}

	for _, tt := range tests {
		got := normalizePathForMatching(tt.input)
		if got != tt.want {
			t.Errorf("normalizePathForMatching(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func BenchmarkMatchesPattern(b *testing.B) {
	patterns := []string{
		"*.log",
		"*.tmp",
		"*.cache",
		"node_modules/",
		"vendor/",
		"**/target",
		"**/build",
		"!important.log",
		"!critical.tmp",
	}

	paths := []string{
		"test.log",
		"src/main.go",
		"vendor/github.com/user/repo/file.go",
		"node_modules/package/index.js",
		"project/target/output.jar",
		"important.log",
		"path/to/deep/file.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			MatchesPattern(path, patterns)
		}
	}
}

func BenchmarkPatternMatcherReuse(b *testing.B) {
	patterns := []string{
		"*.log",
		"*.tmp",
		"*.cache",
		"node_modules/",
		"vendor/",
		"**/target",
		"**/build",
		"!important.log",
		"!critical.tmp",
	}

	paths := []string{
		"test.log",
		"src/main.go",
		"vendor/github.com/user/repo/file.go",
		"node_modules/package/index.js",
		"project/target/output.jar",
		"important.log",
		"path/to/deep/file.txt",
	}

	pm := NewPatternMatcher(patterns)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			pm.Matches(path)
		}
	}
}
