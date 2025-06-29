package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LinkInfo represents information about a symlink
type LinkInfo struct {
	Link     string
	Target   string
	IsBroken bool
	Source   string // Source mapping name (e.g., "home", "work")
}

// Status displays the status of all managed symlinks
func Status(configRepo string, config *Config) error {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	log.Info(Bold("=== Dotfiles Status ==="))
	log.Info("")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	managedLinks, err := FindManagedLinks(homeDir, absConfigRepo, config)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	// Convert to LinkInfo
	var links []LinkInfo
	for _, ml := range managedLinks {
		link := LinkInfo{
			Link:     ml.Path,
			Target:   ml.Target,
			IsBroken: ml.IsBroken,
			Source:   ml.Source,
		}
		links = append(links, link)
	}

	// Sort by link path
	sort.Slice(links, func(i, j int) bool {
		return links[i].Link < links[j].Link
	})

	// Display links
	if len(links) > 0 {
		log.Info(Bold("Active links:"))
		for _, link := range links {
			displayLink(link)
		}
	} else {
		log.Info("No active links found.")
	}

	return nil
}

func displayLink(link LinkInfo) {
	// Show source mapping
	sourceLabel := "[" + link.Source + "]"
	sourceColor := Cyan(sourceLabel)

	status := Green("OK")
	if link.IsBroken {
		status = Red("BROKEN")
	}

	// Format for aligned output
	log.Info("  %-50s -> %-40s %-15s %s",
		link.Link,
		link.Target,
		sourceColor,
		status)
}
