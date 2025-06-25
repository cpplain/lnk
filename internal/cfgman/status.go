package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LinkInfo represents information about a symlink
type LinkInfo struct {
	Link       string
	Target     string
	IsPrivate  bool
	IsBroken   bool
	IsInternal bool   // Cross-repo symlink
	Source     string // Source mapping name (e.g., "home", "work")
}

// Status displays the status of all managed symlinks
func Status(configRepo string, config *Config) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	fmt.Println(Bold("=== Dotfiles Status ==="))
	fmt.Println()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := findManagedLinks(homeDir, configRepo, config)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	// Filter out internal cross-repo symlinks
	var externalLinks []LinkInfo
	for _, link := range links {
		if !link.IsInternal {
			externalLinks = append(externalLinks, link)
		}
	}

	// Sort by link path
	sort.Slice(externalLinks, func(i, j int) bool {
		return externalLinks[i].Link < externalLinks[j].Link
	})

	// Display links
	if len(externalLinks) > 0 {
		fmt.Println(Bold("Active links:"))
		for _, link := range externalLinks {
			displayLink(link)
		}
	} else {
		fmt.Println("No active links found.")
	}

	// Display directories configured to link as units for each mapping
	if len(config.LinkMappings) > 0 {
		fmt.Println()
		fmt.Println(Bold("Directories linked as units:"))
		for _, mapping := range config.LinkMappings {
			if len(mapping.LinkAsDirectory) > 0 {
				fmt.Printf("  %s:\n", Cyan("["+mapping.Source+"]"))
				for _, dir := range mapping.LinkAsDirectory {
					fmt.Printf("    %s\n", Blue(dir))
				}
			}
		}
	}

	return nil
}

func findManagedLinks(startPath, configRepo string, config *Config) ([]LinkInfo, error) {
	var links []LinkInfo

	err := filepath.Walk(startPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip certain directories
		if info.IsDir() {
			name := filepath.Base(path)
			// Skip specific system directories but allow other dot directories
			if name == "Library" || name == ".Trash" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}

			// Check if it points to our config repo
			if strings.Contains(target, configRepo) {
				link := LinkInfo{
					Link:      path,
					Target:    target,
					IsPrivate: strings.Contains(target, filepath.Join(configRepo, "private")),
				}

				// Determine source mapping based on target path
				link.Source = determineSourceMapping(target, configRepo, config)

				// Check if it's an internal cross-repo symlink
				if strings.HasPrefix(path, configRepo) {
					link.IsInternal = true
				}

				// Check if link is broken
				if _, err := os.Stat(path); os.IsNotExist(err) {
					link.IsBroken = true
				}

				links = append(links, link)
			}
		}

		return nil
	})

	return links, err
}

// determineSourceMapping determines which source mapping a target path belongs to
func determineSourceMapping(target, configRepo string, config *Config) string {
	// Remove the config repo prefix to get the relative path
	relPath := strings.TrimPrefix(target, configRepo)
	relPath = strings.TrimPrefix(relPath, "/")

	// Check each mapping to find which one contains this path
	for _, mapping := range config.LinkMappings {
		if strings.HasPrefix(relPath, mapping.Source+"/") || relPath == mapping.Source {
			return mapping.Source
		}
	}

	// Fallback detection for common directory patterns
	if strings.HasPrefix(relPath, "private/home/") {
		return "private/home"
	} else if strings.HasPrefix(relPath, "home/") {
		return "home"
	}

	// Default to showing the first directory component
	parts := strings.Split(relPath, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return "unknown"
}

func displayLink(link LinkInfo) {
	// Show source mapping
	sourceLabel := "[" + link.Source + "]"
	var sourceColor string
	if link.IsPrivate {
		sourceColor = Yellow(sourceLabel)
	} else {
		sourceColor = Cyan(sourceLabel)
	}

	status := Green("OK")
	if link.IsBroken {
		status = Red("BROKEN")
	}

	// Format for aligned output
	fmt.Printf("  %-50s -> %-40s %-15s %s\n",
		link.Link,
		link.Target,
		sourceColor,
		status)
}
