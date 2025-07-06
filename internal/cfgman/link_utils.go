package cfgman

import (
	"os"
	"path/filepath"
	"strings"
)

// ManagedLink represents a symlink managed by cfgman
type ManagedLink struct {
	Path     string // The symlink path
	Target   string // The target path (what the symlink points to)
	IsBroken bool   // Whether the link is broken
	Source   string // Source mapping name (e.g., "home", "work")
}

// FindManagedLinks finds all symlinks within a directory that point to the config repo
func FindManagedLinks(startPath, configRepo string, config *Config) ([]ManagedLink, error) {
	var links []ManagedLink
	var fileCount int

	// Create progress indicator
	progress := NewProgressIndicator("Searching for managed links")

	// Use ShowProgress to handle the 1-second delay
	err := ShowProgress("Searching for managed links", func() error {
		return filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
			fileCount++
			if fileCount%100 == 0 {
				progress.Update(fileCount)
			}
			if err != nil {
				// Debug the error but continue walking
				Debug("Error walking path %s: %v", path, err)
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
				if link := checkManagedLink(path, configRepo, config); link != nil {
					links = append(links, *link)
				}
			}

			return nil
		})
	})

	return links, err
}

// checkManagedLink checks if a symlink points to the config repo and returns its info
func checkManagedLink(linkPath, configRepo string, config *Config) *ManagedLink {
	target, err := os.Readlink(linkPath)
	if err != nil {
		Debug("Failed to read symlink %s: %v", linkPath, err)
		return nil
	}

	// Check if it points to our config repo using proper path comparison
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(linkPath), target)
	}
	cleanTarget, err := filepath.Abs(absTarget)
	if err != nil {
		Debug("Failed to get absolute path for target %s: %v", target, err)
		return nil
	}

	relPath, err := filepath.Rel(configRepo, cleanTarget)
	if err != nil || strings.HasPrefix(relPath, "..") || relPath == "." {
		// Not managed by this repo
		return nil
	}

	link := &ManagedLink{
		Path:   linkPath,
		Target: target,
	}

	// Determine source mapping based on target path
	if config != nil {
		link.Source = DetermineSourceMapping(target, configRepo, config)
	}

	// Check if link is broken by checking if the target exists
	if _, err := os.Stat(cleanTarget); err != nil {
		link.IsBroken = true
	}

	return link
}
