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

	// Create config with mappings
	config := &Config{
		LinkMappings: []LinkMapping{
			{
				Source:          "home",
				Target:          "~/",
				LinkAsDirectory: []string{".config/nvim"},
			},
			{
				Source:          "work",
				Target:          "~/",
				LinkAsDirectory: []string{},
			},
		},
	}

	// Capture output
	output := CaptureOutput(t, func() {
		err := Status(configRepo, config)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
	})

	// Debug: print the actual output
	t.Logf("Status output:\n%s", output)

	// Verify output contains source mappings
	if !strings.Contains(output, "[home]") {
		t.Errorf("Output should contain [home] source mapping")
	}
	if !strings.Contains(output, "[work]") {
		t.Errorf("Output should contain [work] source mapping")
	}
	if !strings.Contains(output, ".bashrc") {
		t.Errorf("Output should contain .bashrc")
	}
	if !strings.Contains(output, ".gitconfig") {
		t.Errorf("Output should contain .gitconfig")
	}

	// Verify directories linked as units section
	if !strings.Contains(output, "Directories linked as units:") {
		t.Errorf("Output should contain 'Directories linked as units:' section")
	}
	if !strings.Contains(output, ".config/nvim") {
		t.Errorf("Output should show .config/nvim as directory linked as unit")
	}
}

func TestDetermineSourceMapping(t *testing.T) {
	configRepo := "/tmp/dotfiles"
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/"},
			{Source: "work", Target: "~/"},
			{Source: "private/home", Target: "~/"},
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
			expected: "home",
		},
		{
			name:     "work mapping",
			target:   "/tmp/dotfiles/work/.gitconfig",
			expected: "work",
		},
		{
			name:     "private/home mapping",
			target:   "/tmp/dotfiles/private/home/.ssh/config",
			expected: "private/home",
		},
		{
			name:     "unknown mapping",
			target:   "/tmp/dotfiles/other/file",
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSourceMapping(tt.target, configRepo, config)
			if result != tt.expected {
				t.Errorf("determineSourceMapping(%s) = %s; want %s", tt.target, result, tt.expected)
			}
		})
	}
}
