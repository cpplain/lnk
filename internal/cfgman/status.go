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
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	PrintHeader("Dotfile Status")
	fmt.Println()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	managedLinks, err := FindManagedLinks(homeDir, absConfigRepo, config)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
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
		// Separate active and broken links
		var activeLinks, brokenLinks []LinkInfo
		for _, link := range links {
			if link.IsBroken {
				brokenLinks = append(brokenLinks, link)
			} else {
				activeLinks = append(activeLinks, link)
			}
		}

		// Display active links
		if len(activeLinks) > 0 {
			for _, link := range activeLinks {
				PrintSuccess("Active: %s", ContractPath(link.Link))
			}
		}

		// Display broken links
		if len(brokenLinks) > 0 {
			if len(activeLinks) > 0 {
				fmt.Println()
			}
			for _, link := range brokenLinks {
				PrintError("Broken: %s", ContractPath(link.Link))
			}
		}

		// Summary
		fmt.Println()
		PrintInfo("Total: %s (%s active, %s broken)",
			Bold(fmt.Sprintf("%d links", len(links))),
			Green(fmt.Sprintf("%d", len(activeLinks))),
			Red(fmt.Sprintf("%d", len(brokenLinks))))
	} else {
		PrintInfo("No active links found.")
	}

	return nil
}
