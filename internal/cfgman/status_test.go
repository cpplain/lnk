package cfgman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusWithLinkMappings(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	configRepo := filepath.Join(tmpDir, "dotfiles")
	homeDir := filepath.Join(tmpDir, "home")

	// Create directory structure
	os.MkdirAll(filepath.Join(configRepo, "home"), 0755)
	os.MkdirAll(filepath.Join(configRepo, "work"), 0755)
	os.MkdirAll(homeDir, 0755)

	// Create test files in different mappings
	homeFile := filepath.Join(configRepo, "home", ".bashrc")
	workFile := filepath.Join(configRepo, "work", ".gitconfig")
	os.WriteFile(homeFile, []byte("# bashrc"), 0644)
	os.WriteFile(workFile, []byte("# gitconfig"), 0644)

	// Create symlinks
	os.Symlink(homeFile, filepath.Join(homeDir, ".bashrc"))
	os.Symlink(workFile, filepath.Join(homeDir, ".gitconfig"))

	// Set test env to use our test home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", oldHome)

	// Create config with mappings using absolute paths
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source: filepath.Join(configRepo, "home"),
				Target: "~/",
			},
			{
				Source: filepath.Join(configRepo, "work"),
				Target: "~/",
			},
		},
	}

	// Capture output
	output := CaptureOutput(t, func() {
		err := Status(config)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
	})

	// Debug: print the actual output
	t.Logf("Status output:\n%s", output)

	// Verify the output shows the active links (in simplified format when piped)
	if !strings.Contains(output, "active ~/.bashrc") {
		t.Errorf("Output should show active bashrc link")
	}
	if !strings.Contains(output, "active ~/.gitconfig") {
		t.Errorf("Output should show active gitconfig link")
	}

	// Verify output contains the files and paths
	// We no longer show source mappings in brackets since the full path shows the source
	if !strings.Contains(output, ".bashrc") {
		t.Errorf("Output should contain .bashrc")
	}
	if !strings.Contains(output, ".gitconfig") {
		t.Errorf("Output should contain .gitconfig")
	}

	// Removed directories linked as units section - no longer supported
}

func TestDetermineSourceMapping(t *testing.T) {
	configRepo := "/tmp/dotfiles"
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(configRepo, "home"), Target: "~/"},
			{Source: filepath.Join(configRepo, "work"), Target: "~/"},
			{Source: filepath.Join(configRepo, "private/home"), Target: "~/"},
		},
	}

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "home mapping",
			target:   "/tmp/dotfiles/home/.bashrc",
			expected: filepath.Join(configRepo, "home"),
		},
		{
			name:     "work mapping",
			target:   "/tmp/dotfiles/work/.gitconfig",
			expected: filepath.Join(configRepo, "work"),
		},
		{
			name:     "private/home mapping",
			target:   "/tmp/dotfiles/private/home/.ssh/config",
			expected: filepath.Join(configRepo, "private/home"),
		},
		{
			name:     "unknown mapping",
			target:   "/tmp/dotfiles/other/file",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineSourceMapping(tt.target, config)
			if result != tt.expected {
				t.Errorf("DetermineSourceMapping(%s) = %s; want %s", tt.target, result, tt.expected)
			}
		})
	}
}
