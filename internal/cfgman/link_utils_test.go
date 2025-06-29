package cfgman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindManagedLinks(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (startPath, configRepo string, config *Config, cleanup func())
		expectedLinks int
		validateFunc  func(t *testing.T, links []ManagedLink)
	}{
		{
			name: "find single managed link",
			setupFunc: func(t *testing.T) (string, string, *Config, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-links-test")
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(configRepo, 0755)
				os.MkdirAll(homeDir, 0755)

				// Create file in repo
				sourceFile := filepath.Join(configRepo, "home", "config.txt")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("config"), 0644)

				// Create symlink
				linkPath := filepath.Join(homeDir, ".config")
				os.Symlink(sourceFile, linkPath)

				config := &Config{
					LinkMappings: []LinkMapping{
						{Source: "home", Target: "~/"},
					},
				}

				return homeDir, configRepo, config, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
			validateFunc: func(t *testing.T, links []ManagedLink) {
				if len(links) != 1 {
					t.Fatalf("Expected 1 link, got %d", len(links))
				}
				link := links[0]
				if link.IsBroken {
					t.Error("Link should not be broken")
				}
				if link.Source != "home" {
					t.Errorf("Source = %q, want %q", link.Source, "home")
				}
			},
		},
		{
			name: "find private links",
			setupFunc: func(t *testing.T) (string, string, *Config, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-private-test")
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(configRepo, 0755)
				os.MkdirAll(homeDir, 0755)

				// Create private file
				privateFile := filepath.Join(configRepo, "private", "home", "secret.key")
				os.MkdirAll(filepath.Dir(privateFile), 0755)
				os.WriteFile(privateFile, []byte("secret"), 0600)

				// Create symlink
				linkPath := filepath.Join(homeDir, ".ssh", "id_rsa")
				os.MkdirAll(filepath.Dir(linkPath), 0755)
				os.Symlink(privateFile, linkPath)

				config := &Config{
					LinkMappings: []LinkMapping{
						{Source: "private/home", Target: "~/"},
					},
				}

				return homeDir, configRepo, config, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
			validateFunc: func(t *testing.T, links []ManagedLink) {
				if len(links) != 1 {
					t.Fatalf("Expected 1 link, got %d", len(links))
				}
				if links[0].Source != "private/home" {
					t.Errorf("Source = %q, want %q", links[0].Source, "private/home")
				}
			},
		},
		{
			name: "find broken links",
			setupFunc: func(t *testing.T) (string, string, *Config, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-broken-test")
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(configRepo, 0755)
				os.MkdirAll(homeDir, 0755)

				// Create symlink to non-existent file
				targetPath := filepath.Join(configRepo, "home", "missing.txt")
				linkPath := filepath.Join(homeDir, "broken-link")
				os.Symlink(targetPath, linkPath)

				return homeDir, configRepo, nil, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
			validateFunc: func(t *testing.T, links []ManagedLink) {
				if len(links) != 1 {
					t.Fatalf("Expected 1 link, got %d", len(links))
				}
				if !links[0].IsBroken {
					t.Error("Link should be marked as broken")
				}
			},
		},
		{
			name: "skip system directories",
			setupFunc: func(t *testing.T) (string, string, *Config, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-skip-test")
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(configRepo, 0755)
				os.MkdirAll(homeDir, 0755)

				// Create links in regular directory
				sourceFile1 := filepath.Join(configRepo, "home", "file1.txt")
				os.MkdirAll(filepath.Dir(sourceFile1), 0755)
				os.WriteFile(sourceFile1, []byte("file1"), 0644)
				os.Symlink(sourceFile1, filepath.Join(homeDir, "link1"))

				// Create links in Library directory (should be skipped)
				libraryDir := filepath.Join(homeDir, "Library")
				os.MkdirAll(libraryDir, 0755)
				sourceFile2 := filepath.Join(configRepo, "home", "file2.txt")
				os.WriteFile(sourceFile2, []byte("file2"), 0644)
				os.Symlink(sourceFile2, filepath.Join(libraryDir, "link2"))

				// Create links in .Trash directory (should be skipped)
				trashDir := filepath.Join(homeDir, ".Trash")
				os.MkdirAll(trashDir, 0755)
				sourceFile3 := filepath.Join(configRepo, "home", "file3.txt")
				os.WriteFile(sourceFile3, []byte("file3"), 0644)
				os.Symlink(sourceFile3, filepath.Join(trashDir, "link3"))

				return homeDir, configRepo, nil, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1, // Only the one outside system directories
		},
		{
			name: "ignore external symlinks",
			setupFunc: func(t *testing.T) (string, string, *Config, func()) {
				tmpDir, _ := os.MkdirTemp("", "cfgman-external-test")
				configRepo := filepath.Join(tmpDir, "repo")
				homeDir := filepath.Join(tmpDir, "home")
				externalDir := filepath.Join(tmpDir, "external")

				os.MkdirAll(configRepo, 0755)
				os.MkdirAll(homeDir, 0755)
				os.MkdirAll(externalDir, 0755)

				// Create managed symlink
				managedFile := filepath.Join(configRepo, "home", "managed.txt")
				os.MkdirAll(filepath.Dir(managedFile), 0755)
				os.WriteFile(managedFile, []byte("managed"), 0644)
				os.Symlink(managedFile, filepath.Join(homeDir, "managed-link"))

				// Create external symlink (should be ignored)
				externalFile := filepath.Join(externalDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)
				os.Symlink(externalFile, filepath.Join(homeDir, "external-link"))

				return homeDir, configRepo, nil, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1, // Only the managed link
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startPath, configRepo, config, cleanup := tt.setupFunc(t)
			defer cleanup()

			links, err := FindManagedLinks(startPath, configRepo, config)
			if err != nil {
				t.Fatalf("FindManagedLinks error: %v", err)
			}

			if len(links) != tt.expectedLinks {
				t.Errorf("Found %d links, expected %d", len(links), tt.expectedLinks)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, links)
			}
		})
	}
}

func TestCheckManagedLink(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "cfgman-check-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configRepo := filepath.Join(tmpDir, "repo")
	os.MkdirAll(configRepo, 0755)

	tests := []struct {
		name      string
		setupFunc func() string
		expectNil bool
	}{
		{
			name: "valid managed link",
			setupFunc: func() string {
				sourceFile := filepath.Join(configRepo, "home", "valid.txt")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("valid"), 0644)

				linkPath := filepath.Join(tmpDir, "valid-link")
				os.Symlink(sourceFile, linkPath)
				return linkPath
			},
			expectNil: false,
		},
		{
			name: "external link",
			setupFunc: func() string {
				externalFile := filepath.Join(tmpDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)

				linkPath := filepath.Join(tmpDir, "external-link")
				os.Symlink(externalFile, linkPath)
				return linkPath
			},
			expectNil: true,
		},
		{
			name: "relative symlink",
			setupFunc: func() string {
				sourceFile := filepath.Join(configRepo, "home", "relative.txt")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("relative"), 0644)

				linkDir := filepath.Join(tmpDir, "links")
				os.MkdirAll(linkDir, 0755)
				linkPath := filepath.Join(linkDir, "relative-link")

				// Create relative symlink
				relPath, _ := filepath.Rel(linkDir, sourceFile)
				os.Symlink(relPath, linkPath)
				return linkPath
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkPath := tt.setupFunc()
			config := &Config{
				LinkMappings: []LinkMapping{
					{Source: "home", Target: "~/"},
				},
			}

			result := checkManagedLink(linkPath, configRepo, config)

			if tt.expectNil && result != nil {
				t.Errorf("Expected nil, got %+v", result)
			}
			if !tt.expectNil && result == nil {
				t.Error("Expected non-nil result, got nil")
			}
		})
	}
}

func TestManagedLinkStruct(t *testing.T) {
	// Test ManagedLink struct fields
	link := ManagedLink{
		Path:     "/home/user/.config",
		Target:   "/repo/home/config",
		IsBroken: false,
		Source:   "private/home",
	}

	if link.Path != "/home/user/.config" {
		t.Errorf("Path = %q, want %q", link.Path, "/home/user/.config")
	}
	if link.Target != "/repo/home/config" {
		t.Errorf("Target = %q, want %q", link.Target, "/repo/home/config")
	}
	if link.IsBroken {
		t.Error("IsBroken should be false")
	}
	if link.Source != "private/home" {
		t.Errorf("Source = %q, want %q", link.Source, "private/home")
	}
}
