package lnk

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStatusJSON(t *testing.T) {
	// Save original format and verbosity
	originalFormat := GetOutputFormat()
	originalVerbosity := GetVerbosity()
	defer func() {
		SetOutputFormat(originalFormat)
		SetVerbosity(originalVerbosity)
	}()

	// Set JSON format and quiet mode
	SetOutputFormat(FormatJSON)
	SetVerbosity(VerbosityQuiet)

	// Create test environment
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")

	// Override home directory for test
	oldHome := os.Getenv("HOME")
	testHome := filepath.Join(tmpDir, "home")
	os.Setenv("HOME", testHome)
	defer os.Setenv("HOME", oldHome)

	// Create directories
	os.MkdirAll(filepath.Join(repoDir, "home"), 0755)
	os.MkdirAll(testHome, 0755)

	// Create test files
	testFile1 := filepath.Join(repoDir, "home", ".bashrc")
	testFile2 := filepath.Join(repoDir, "home", ".vimrc")
	os.WriteFile(testFile1, []byte("test"), 0644)
	os.WriteFile(testFile2, []byte("test"), 0644)

	// Create symlinks
	link1 := filepath.Join(testHome, ".bashrc")
	link2 := filepath.Join(testHome, ".vimrc")
	os.Symlink(testFile1, link1)
	os.Symlink(testFile2, link2)

	// Create config
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: filepath.Join(repoDir, "home"), Target: "~/"},
		},
	}

	// Redirect stdout to capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run status
	err := Status(config)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Parse JSON output
	var output StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify output
	if output.Summary.Total != 2 {
		t.Errorf("Expected 2 total links, got %d", output.Summary.Total)
	}
	if output.Summary.Active != 2 {
		t.Errorf("Expected 2 active links, got %d", output.Summary.Active)
	}
	if output.Summary.Broken != 0 {
		t.Errorf("Expected 0 broken links, got %d", output.Summary.Broken)
	}
	if len(output.Links) != 2 {
		t.Errorf("Expected 2 links in array, got %d", len(output.Links))
	}
}

func TestStatusJSONEmpty(t *testing.T) {
	// Save original format and verbosity
	originalFormat := GetOutputFormat()
	originalVerbosity := GetVerbosity()
	defer func() {
		SetOutputFormat(originalFormat)
		SetVerbosity(originalVerbosity)
	}()

	// Set JSON format and quiet mode
	SetOutputFormat(FormatJSON)
	SetVerbosity(VerbosityQuiet)

	// Create test environment
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")

	// Override home directory for test
	oldHome := os.Getenv("HOME")
	testHome := filepath.Join(tmpDir, "home")
	os.Setenv("HOME", testHome)
	defer os.Setenv("HOME", oldHome)

	// Create directories
	os.MkdirAll(repoDir, 0755)
	os.MkdirAll(testHome, 0755)

	// Create config with no mappings
	config := &Config{
		LinkMappings: []LinkMapping{
			{Source: "home", Target: "~/"},
		},
	}

	// Redirect stdout to capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run status
	err := Status(config)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Parse JSON output
	var output StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify output
	if output.Summary.Total != 0 {
		t.Errorf("Expected 0 total links, got %d", output.Summary.Total)
	}
	if output.Links == nil {
		t.Error("Expected empty array for links, got nil")
	}
	if len(output.Links) != 0 {
		t.Errorf("Expected 0 links in array, got %d", len(output.Links))
	}
}
