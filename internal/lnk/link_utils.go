package lnk

import (
	"os"
	"path/filepath"
	"strings"
)

// ManagedLink represents a symlink managed by lnk
type ManagedLink struct {
	Path     string // The symlink path
	Target   string // The target path (what the symlink points to)
	IsBroken bool   // Whether the link is broken
	Source   string // Source mapping name (e.g., "home", "work")
}

// FindManagedLinks finds all symlinks within a directory that point to configured source directories
func FindManagedLinks(startPath string, config *Config) ([]ManagedLink, error) {
	var links []ManagedLink
	var fileCount int

	// Use ShowProgress to handle the 1-second delay
	err := ShowProgress("Searching for managed links", func() error {
		return filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
			fileCount++
			if err != nil {
				// Log the error but continue walking
				PrintVerbose("Error walking path %s: %v", path, err)
				return nil
			}

			// Skip certain directories
			if info.IsDir() {
				name := filepath.Base(path)
				// Skip specific system directories but allow other dot directories
				if name == LibraryDir || name == TrashDir {
					return filepath.SkipDir
				}
				return nil
			}

			// Check if it's a symlink
			if info.Mode()&os.ModeSymlink != 0 {
				if link := checkManagedLink(path, config); link != nil {
					links = append(links, *link)
				}
			}

			return nil
		})
	})

	return links, err
}

// checkManagedLink checks if a symlink points to any configured source directory and returns its info
func checkManagedLink(linkPath string, config *Config) *ManagedLink {
	target, err := os.Readlink(linkPath)
	if err != nil {
		PrintVerbose("Failed to read symlink %s: %v", linkPath, err)
		return nil
	}

	// Get absolute target path
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(linkPath), target)
	}
	cleanTarget, err := filepath.Abs(absTarget)
	if err != nil {
		PrintVerbose("Failed to get absolute path for target %s: %v", target, err)
		return nil
	}

	// Check if it points to any of our configured source directories
	var managedBySource string
	for _, mapping := range config.LinkMappings {
		absSource, err := ExpandPath(mapping.Source)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(absSource, cleanTarget)
		if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
			// This link is managed by this source directory
			managedBySource = mapping.Source
			break
		}
	}

	// Not managed by any configured source
	if managedBySource == "" {
		return nil
	}

	link := &ManagedLink{
		Path:   linkPath,
		Target: target,
		Source: managedBySource,
	}

	// Check if link is broken by checking if the target exists
	if _, err := os.Stat(cleanTarget); err != nil {
		link.IsBroken = true
	}

	return link
}

// FindManagedLinksForSources finds all symlinks in startPath that point to any of the specified source directories.
// This is a package-based version of FindManagedLinks that works with explicit source paths instead of Config.
// sources should be absolute paths (use ExpandPath first if needed).
func FindManagedLinksForSources(startPath string, sources []string) ([]ManagedLink, error) {
	var links []ManagedLink

	err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			PrintVerbose("Error walking path %s: %v", path, err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			name := filepath.Base(path)
			// Skip specific system directories
			if name == LibraryDir || name == TrashDir {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink == 0 {
			return nil
		}

		// Read symlink target
		target, err := os.Readlink(path)
		if err != nil {
			PrintVerbose("Failed to read symlink %s: %v", path, err)
			return nil
		}

		// Get absolute target path
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(path), target)
		}
		cleanTarget, err := filepath.Abs(absTarget)
		if err != nil {
			PrintVerbose("Failed to get absolute path for target %s: %v", target, err)
			return nil
		}

		// Check if target points to any of our sources
		var managedBySource string
		for _, source := range sources {
			relPath, err := filepath.Rel(source, cleanTarget)
			if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
				managedBySource = source
				break
			}
		}

		if managedBySource == "" {
			return nil
		}

		link := ManagedLink{
			Path:   path,
			Target: target,
			Source: managedBySource,
		}

		// Check if link is broken
		if _, err := os.Stat(cleanTarget); err != nil {
			link.IsBroken = true
		}

		links = append(links, link)
		return nil
	})

	return links, err
}
