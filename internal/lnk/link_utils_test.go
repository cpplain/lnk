package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestFindManagedLinksForSources(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (startPath string, sources []string, cleanup func())
		expectedLinks int
		validateFunc  func(t *testing.T, links []ManagedLink)
	}{
		{
			name: "find links from single source",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-sources-test")
				sourceDir := filepath.Join(tmpDir, "repo", "home")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create file in source
				sourceFile := filepath.Join(sourceDir, "config.txt")
				os.WriteFile(sourceFile, []byte("config"), 0644)

				// Create symlink
				linkPath := filepath.Join(targetDir, ".config")
				os.Symlink(sourceFile, linkPath)

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
			validateFunc: func(t *testing.T, links []ManagedLink) {
				if len(links) != 1 {
					t.Fatalf("Expected 1 link, got %d", len(links))
				}
				if links[0].IsBroken {
					t.Error("Link should not be broken")
				}
			},
		},
		{
			name: "find links from multiple sources",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-multi-sources-test")
				source1 := filepath.Join(tmpDir, "repo", "home")
				source2 := filepath.Join(tmpDir, "repo", "private")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(source1, 0755)
				os.MkdirAll(source2, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create files and links from both sources
				file1 := filepath.Join(source1, "bashrc")
				os.WriteFile(file1, []byte("bashrc"), 0644)
				os.Symlink(file1, filepath.Join(targetDir, ".bashrc"))

				file2 := filepath.Join(source2, "secret.key")
				os.WriteFile(file2, []byte("secret"), 0600)
				os.Symlink(file2, filepath.Join(targetDir, ".secret"))

				return targetDir, []string{source1, source2}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 2,
		},
		{
			name: "find no links when sources don't match",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-no-match-test")
				sourceDir := filepath.Join(tmpDir, "repo", "home")
				externalDir := filepath.Join(tmpDir, "external")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(externalDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create external symlink (not managed)
				externalFile := filepath.Join(externalDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)
				os.Symlink(externalFile, filepath.Join(targetDir, "external-link"))

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 0,
		},
		{
			name: "detect broken links",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-broken-sources-test")
				sourceDir := filepath.Join(tmpDir, "repo", "home")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create symlink to non-existent file
				targetPath := filepath.Join(sourceDir, "missing.txt")
				linkPath := filepath.Join(targetDir, "broken-link")
				os.Symlink(targetPath, linkPath)

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
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
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-skip-sources-test")
				sourceDir := filepath.Join(tmpDir, "repo", "home")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create link in regular directory
				sourceFile1 := filepath.Join(sourceDir, "file1.txt")
				os.WriteFile(sourceFile1, []byte("file1"), 0644)
				os.Symlink(sourceFile1, filepath.Join(targetDir, "link1"))

				// Create link in Library directory (should be skipped)
				libraryDir := filepath.Join(targetDir, "Library")
				os.MkdirAll(libraryDir, 0755)
				sourceFile2 := filepath.Join(sourceDir, "file2.txt")
				os.WriteFile(sourceFile2, []byte("file2"), 0644)
				os.Symlink(sourceFile2, filepath.Join(libraryDir, "link2"))

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1, // Only the one outside Library
		},
		{
			name: "handle relative symlinks",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-relative-sources-test")
				sourceDir := filepath.Join(tmpDir, "repo", "home")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create file and relative symlink
				sourceFile := filepath.Join(sourceDir, "relative.txt")
				os.WriteFile(sourceFile, []byte("relative"), 0644)

				linkPath := filepath.Join(targetDir, "relative-link")
				relPath, _ := filepath.Rel(targetDir, sourceFile)
				os.Symlink(relPath, linkPath)

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
		},
		{
			name: "handle nested package paths",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-nested-sources-test")
				sourceDir := filepath.Join(tmpDir, "repo", "private", "home")
				targetDir := filepath.Join(tmpDir, "home")

				os.MkdirAll(sourceDir, 0755)
				os.MkdirAll(targetDir, 0755)

				// Create nested source file
				sourceFile := filepath.Join(sourceDir, "secret.key")
				os.WriteFile(sourceFile, []byte("secret"), 0600)

				// Create parent directory for symlink
				sshDir := filepath.Join(targetDir, ".ssh")
				os.MkdirAll(sshDir, 0755)
				os.Symlink(sourceFile, filepath.Join(sshDir, "id_rsa"))

				return targetDir, []string{sourceDir}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 1,
			validateFunc: func(t *testing.T, links []ManagedLink) {
				if len(links) != 1 {
					t.Fatalf("Expected 1 link, got %d", len(links))
				}
				if !strings.Contains(links[0].Source, "private") {
					t.Errorf("Source = %q, want to contain 'private'", links[0].Source)
				}
			},
		},
		{
			name: "handle empty sources list",
			setupFunc: func(t *testing.T) (string, []string, func()) {
				tmpDir, _ := os.MkdirTemp("", "lnk-empty-sources-test")
				targetDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(targetDir, 0755)

				// Create some symlink that won't match
				externalFile := filepath.Join(tmpDir, "external.txt")
				os.WriteFile(externalFile, []byte("external"), 0644)
				os.Symlink(externalFile, filepath.Join(targetDir, "link"))

				return targetDir, []string{}, func() { os.RemoveAll(tmpDir) }
			},
			expectedLinks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startPath, sources, cleanup := tt.setupFunc(t)
			defer cleanup()

			links, err := FindManagedLinksForSources(startPath, sources)
			if err != nil {
				t.Fatalf("FindManagedLinksForSources error: %v", err)
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
