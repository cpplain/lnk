package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateLinks creates symlinks from the repository to the home directory
func CreateLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	fmt.Println("Creating links with smart defaults...")

	// Require LinkMappings to be defined
	if len(config.LinkMappings) == 0 {
		return fmt.Errorf("no link mappings defined in configuration")
	}

	// Process each link mapping
	for _, mapping := range config.LinkMappings {
		// Expand the target path (handle ~/)
		targetPath, err := ExpandPath(mapping.Target)
		if err != nil {
			return fmt.Errorf("expanding target path for mapping %s: %w", mapping.Source, err)
		}

		// Build the source path
		sourcePath := filepath.Join(configRepo, mapping.Source)

		// Check if source directory exists
		if info, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Skipping mapping %s: source directory does not exist\n", mapping.Source)
				continue
			}
			return fmt.Errorf("checking source directory for mapping %s: %w", mapping.Source, err)
		} else if !info.IsDir() {
			return fmt.Errorf("source path for mapping %s is not a directory: %s", mapping.Source, sourcePath)
		}

		fmt.Printf("Processing mapping: %s -> %s\n", mapping.Source, mapping.Target)
		if err := processDirectoryWithMapping(sourcePath, targetPath, configRepo, &mapping, config, dryRun); err != nil {
			return fmt.Errorf("processing mapping %s: %w", mapping.Source, err)
		}
	}

	return nil
}

// RemoveLinks removes all symlinks pointing to the config repository
func RemoveLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	return removeLinks(configRepo, config, dryRun, false)
}

// removeLinks is the internal implementation that allows skipping confirmation
func removeLinks(configRepo string, config *Config, dryRun bool, skipConfirm bool) error {
	fmt.Println("Removing all links...")

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

	if len(externalLinks) == 0 {
		fmt.Println("No links found to remove.")
		return nil
	}

	// Show all links that will be removed
	fmt.Printf("Found %d symlinks to remove:\n", len(externalLinks))
	for _, link := range externalLinks {
		fmt.Printf("  %s -> %s\n", link.Link, link.Target)
	}
	fmt.Println()

	// Confirm if not in dry-run mode and not skipping confirmation
	if !dryRun && !skipConfirm {
		if !ConfirmPrompt("Remove all symlinks?") {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove links
	for _, link := range externalLinks {
		if dryRun {
			fmt.Printf("[DRY RUN] Would remove: %s\n", link.Link)
		} else {
			if err := os.Remove(link.Link); err != nil {
				fmt.Printf("%s: %v\n", Red("Error removing"), err)
				continue
			}
			fmt.Printf("Removed: %s\n", link.Link)
		}
	}

	return nil
}

// PruneLinks removes broken symlinks pointing to the config repository
func PruneLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := findManagedLinks(homeDir, configRepo, config)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	// Collect all broken links first
	var brokenLinks []LinkInfo
	for _, link := range links {
		// Skip internal links
		if link.IsInternal {
			continue
		}

		// Check if link is broken
		if link.IsBroken {
			brokenLinks = append(brokenLinks, link)
		}
	}

	// If no broken links found, report and return
	if len(brokenLinks) == 0 {
		fmt.Println("No broken links found.")
		return nil
	}

	// Display all broken links
	fmt.Printf("Found %d broken symlinks:\n", len(brokenLinks))
	for _, link := range brokenLinks {
		fmt.Printf("  %s -> %s (target missing)\n", link.Link, link.Target)
	}

	// In dry-run mode, just show what would be removed
	if dryRun {
		fmt.Println("\n[DRY RUN] Would remove the above broken symlinks.")
		return nil
	}

	// Ask for confirmation (default to "yes" in test mode for compatibility)
	if !ConfirmPromptWithTestDefault("\nRemove all broken symlinks?", true) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove the broken links
	for _, link := range brokenLinks {
		if err := os.Remove(link.Link); err != nil {
			fmt.Printf("%s: %v\n", Red("Error removing"), err)
			continue
		}
		fmt.Printf("Removed: %s\n", link.Link)
	}

	return nil
}

// processDirectoryWithMapping recursively processes a directory for linking using a specific mapping
func processDirectoryWithMapping(sourceBase, targetBase, repoRoot string, mapping *LinkMapping, config *Config, dryRun bool) error {
	entries, err := os.ReadDir(sourceBase)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", sourceBase, err)
	}

	for _, entry := range entries {
		sourceItem := filepath.Join(sourceBase, entry.Name())
		targetItem := filepath.Join(targetBase, entry.Name())

		// Calculate relative path from the mapping's source directory
		relativePath := strings.TrimPrefix(sourceItem, filepath.Join(repoRoot, mapping.Source)+"/")

		// Check if ignored by pattern
		relPathForIgnore := strings.TrimPrefix(sourceItem, repoRoot+"/")
		if mapping != nil && mapping.Source != "." {
			relPathForIgnore = strings.TrimPrefix(relPathForIgnore, mapping.Source+"/")
		}
		if config.ShouldIgnore(relPathForIgnore) {
			continue
		}

		// Handle directory
		if entry.IsDir() {
			// Check if this directory should be linked as a whole
			shouldLinkAsDir := false
			for _, dir := range mapping.LinkAsDirectory {
				if relativePath == dir {
					shouldLinkAsDir = true
					break
				}
			}

			if shouldLinkAsDir {
				// Link the entire directory as a unit
				if err := linkItem(sourceItem, targetItem, true, dryRun); err != nil {
					fmt.Printf("%s linking directory %s: %v\n", Yellow("Warning"), targetItem, err)
				}
				// Don't recurse into this directory since we linked it as a whole
			} else {
				// Not configured to link as directory, so recurse into it
				// Create target directory if it doesn't exist
				if _, err := os.Stat(targetItem); os.IsNotExist(err) {
					if dryRun {
						fmt.Printf("[DRY RUN] Would create directory: %s\n", targetItem)
					} else {
						if err := os.MkdirAll(targetItem, 0755); err != nil {
							return fmt.Errorf("creating directory %s: %w", targetItem, err)
						}
					}
				}
				// Recursively process directory contents
				if err := processDirectoryWithMapping(sourceItem, targetItem, repoRoot, mapping, config, dryRun); err != nil {
					return err
				}
			}
		} else {
			// Handle file - link all files
			if err := linkItem(sourceItem, targetItem, false, dryRun); err != nil {
				fmt.Printf("%s linking file %s: %v\n", Yellow("Warning"), targetItem, err)
			}
		}
	}

	return nil
}

// linkItem creates a symlink from target to source
func linkItem(source, target string, isDir bool, dryRun bool) error {
	// Check if target exists
	if info, err := os.Lstat(target); err == nil {
		// If it's already a symlink pointing to our source, nothing to do
		if info.Mode()&os.ModeSymlink != 0 {
			if existingTarget, err := os.Readlink(target); err == nil && existingTarget == source {
				return nil
			}
		} else {
			// Target exists and is not a symlink
			return fmt.Errorf("%s exists and is not a symlink, skipping", target)
		}
	}

	// Create the link
	if dryRun {
		itemType := "file"
		if isDir {
			itemType = "directory"
		}
		fmt.Printf("[DRY RUN] Would link %s: %s -> %s\n", itemType, target, source)
	} else {
		// Remove existing symlink if it exists
		if _, err := os.Lstat(target); err == nil {
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("removing existing link: %w", err)
			}
		}

		// Create new symlink
		if err := os.Symlink(source, target); err != nil {
			return fmt.Errorf("creating symlink: %w", err)
		}

		itemType := "file"
		if isDir {
			itemType = "directory"
		}
		fmt.Printf("Linked %s: %s\n", itemType, target)
	}

	return nil
}
