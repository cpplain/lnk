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

// TestFindManagedLinksTargetIsAbsolute verifies that ManagedLink.Target is always
// an absolute path, even when the symlink was created with a relative target.
// Per internals.md §2: Target stores "absolute path of the symlink's resolved
// target (never relative)".
func TestFindManagedLinksTargetIsAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "repo", "home")
	targetDir := filepath.Join(tmpDir, "home")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create source file
	sourceFile := filepath.Join(sourceDir, "config.txt")
	os.WriteFile(sourceFile, []byte("config"), 0644)

	// Create symlink with a RELATIVE target
	linkPath := filepath.Join(targetDir, "config-link")
	relTarget, _ := filepath.Rel(targetDir, sourceFile)
	os.Symlink(relTarget, linkPath)

	links, err := FindManagedLinks(targetDir, []string{sourceDir})
	if err != nil {
		t.Fatalf("FindManagedLinks error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	// Target must be absolute, not the raw relative readlink value
	if !filepath.IsAbs(links[0].Target) {
		t.Errorf("ManagedLink.Target should be absolute, got %q", links[0].Target)
	}

	// Target must resolve to the source file (use EvalSymlinks for expected value
	// since Target uses EvalSymlinks which resolves path symlinks like /var → /private/var)
	expectedTarget, _ := filepath.EvalSymlinks(sourceFile)
	if links[0].Target != expectedTarget {
		t.Errorf("ManagedLink.Target = %q, want %q", links[0].Target, expectedTarget)
	}
}

// TestFindManagedLinksTargetAbsoluteForAbsoluteSymlinks verifies Target is the
// normalized absolute path even when the symlink was created with an absolute target.
func TestFindManagedLinksTargetAbsoluteForAbsoluteSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "repo", "home")
	targetDir := filepath.Join(tmpDir, "home")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	sourceFile := filepath.Join(sourceDir, "bashrc")
	os.WriteFile(sourceFile, []byte("bashrc"), 0644)

	// Create symlink with absolute target
	linkPath := filepath.Join(targetDir, ".bashrc")
	os.Symlink(sourceFile, linkPath)

	links, err := FindManagedLinks(targetDir, []string{sourceDir})
	if err != nil {
		t.Fatalf("FindManagedLinks error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	if !filepath.IsAbs(links[0].Target) {
		t.Errorf("ManagedLink.Target should be absolute, got %q", links[0].Target)
	}
	expectedTarget, _ := filepath.EvalSymlinks(sourceFile)
	if links[0].Target != expectedTarget {
		t.Errorf("ManagedLink.Target = %q, want %q", links[0].Target, expectedTarget)
	}
}

// TestFindManagedLinksBrokenLinkTargetIsAbsolute verifies that even broken links
// have an absolute Target path. Per internals.md §3 broken link handling:
// Target is set to "the normalized absolute path computed in step 3".
func TestFindManagedLinksBrokenLinkTargetIsAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "repo", "home")
	targetDir := filepath.Join(tmpDir, "home")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a broken symlink with a relative target pointing into sourceDir
	missingFile := filepath.Join(sourceDir, "missing.txt")
	linkPath := filepath.Join(targetDir, "broken-link")
	relTarget, _ := filepath.Rel(targetDir, missingFile)
	os.Symlink(relTarget, linkPath)

	links, err := FindManagedLinks(targetDir, []string{sourceDir})
	if err != nil {
		t.Fatalf("FindManagedLinks error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	if !links[0].IsBroken {
		t.Error("Link should be marked as broken")
	}
	if !filepath.IsAbs(links[0].Target) {
		t.Errorf("Broken link Target should be absolute, got %q", links[0].Target)
	}
	// For broken links, the parent dir is resolved via EvalSymlinks but the file doesn't exist
	resolvedParent, _ := filepath.EvalSymlinks(sourceDir)
	expectedTarget := filepath.Join(resolvedParent, "missing.txt")
	if links[0].Target != expectedTarget {
		t.Errorf("ManagedLink.Target = %q, want %q", links[0].Target, expectedTarget)
	}
}

// TestFindManagedLinksUsesEvalSymlinks verifies that FindManagedLinks uses
// filepath.EvalSymlinks for non-broken links, which resolves the full symlink
// chain. Per internals.md §3: "Calls filepath.EvalSymlinks to resolve the full
// symlink chain to a clean absolute path".
func TestFindManagedLinksUsesEvalSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "repo", "home")
	targetDir := filepath.Join(tmpDir, "home")

	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create the actual source file
	sourceFile := filepath.Join(sourceDir, "config.txt")
	os.WriteFile(sourceFile, []byte("config"), 0644)

	// Create symlink in targetDir pointing to source file
	linkPath := filepath.Join(targetDir, "config-link")
	os.Symlink(sourceFile, linkPath)

	links, err := FindManagedLinks(targetDir, []string{sourceDir})
	if err != nil {
		t.Fatalf("FindManagedLinks error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	// For non-broken links, Target should be the EvalSymlinks result —
	// a fully resolved, clean absolute path
	expectedTarget, _ := filepath.EvalSymlinks(linkPath)
	if links[0].Target != expectedTarget {
		t.Errorf("ManagedLink.Target = %q, want EvalSymlinks result %q", links[0].Target, expectedTarget)
	}
}

func TestFindManagedLinks(t *testing.T) {
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

			links, err := FindManagedLinks(startPath, sources)
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
