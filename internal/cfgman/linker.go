package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PlannedLink represents a source file and its target symlink location
type PlannedLink struct {
	Source string
	Target string
}

// CreateLinks creates symlinks from the repository to the home directory
func CreateLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	log.Info("Creating links with smart defaults...")

	// Require LinkMappings to be defined
	if len(config.LinkMappings) == 0 {
		return ErrNoLinkMappings
	}

	// Phase 1: Collect all files to link
	var plannedLinks []PlannedLink
	for _, mapping := range config.LinkMappings {
		// Expand the target path (handle ~/)
		targetPath, err := ExpandPath(mapping.Target)
		if err != nil {
			return fmt.Errorf("expanding target path for mapping %s: %w", mapping.Source, err)
		}

		// Build the source path
		sourcePath := filepath.Join(absConfigRepo, mapping.Source)

		// Check if source directory exists
		if info, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				log.Info("Skipping mapping %s: source directory does not exist", mapping.Source)
				continue
			}
			return fmt.Errorf("checking source directory for mapping %s: %w", mapping.Source, err)
		} else if !info.IsDir() {
			return fmt.Errorf("source path for mapping %s is not a directory: %s", mapping.Source, sourcePath)
		}

		log.Info("Processing mapping: %s -> %s", mapping.Source, mapping.Target)

		// Collect files from this mapping
		links, err := collectPlannedLinks(sourcePath, targetPath, absConfigRepo, &mapping, config)
		if err != nil {
			return fmt.Errorf("collecting files for mapping %s: %w", mapping.Source, err)
		}
		plannedLinks = append(plannedLinks, links...)
	}

	if len(plannedLinks) == 0 {
		log.Info("No files to link.")
		return nil
	}

	// Phase 2: Validate all targets
	for _, link := range plannedLinks {
		if err := ValidateSymlinkCreation(link.Source, link.Target); err != nil {
			return fmt.Errorf("validation failed for %s -> %s: %w", link.Target, link.Source, err)
		}
	}

	// Phase 3: Execute (or show dry-run)
	if dryRun {
		log.Info("\n%s Would create %d symlink(s):", Yellow(DryRunPrefix), len(plannedLinks))
		for _, link := range plannedLinks {
			log.Info("%s Would link: %s -> %s", Yellow(DryRunPrefix), link.Target, link.Source)
		}
		return nil
	}

	// Execute the plan
	return executePlannedLinks(plannedLinks)
}

// RemoveLinks removes all symlinks pointing to the config repository
func RemoveLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	return removeLinks(absConfigRepo, config, dryRun, false)
}

// removeLinks is the internal implementation that allows skipping confirmation
func removeLinks(configRepo string, config *Config, dryRun bool, skipConfirm bool) error {
	log.Info("Removing all links...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := FindManagedLinks(homeDir, configRepo, config)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	if len(links) == 0 {
		log.Info("No links found to remove.")
		return nil
	}

	// Show all links that will be removed
	log.Info("Found %d symlinks to remove:", len(links))
	for _, link := range links {
		log.Info("  %s -> %s", link.Path, link.Target)
	}
	log.Info("")

	// Confirm if not in dry-run mode and not skipping confirmation
	if !dryRun && !skipConfirm {
		if !ConfirmPrompt("Remove all symlinks?") {
			log.Info("Cancelled.")
			return nil
		}
	}

	// Remove links
	for _, link := range links {
		if dryRun {
			log.Info("%s Would remove: %s", DryRunPrefix, link.Path)
		} else {
			if err := os.Remove(link.Path); err != nil {
				log.Info("%s: %v", Red("Error removing"), err)
				continue
			}
			log.Info("Removed: %s", link.Path)
		}
	}

	return nil
}

// PruneLinks removes broken symlinks pointing to the config repository
func PruneLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := FindManagedLinks(homeDir, absConfigRepo, config)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	// Collect all broken links first
	var brokenLinks []ManagedLink
	for _, link := range links {
		// Check if link is broken
		if link.IsBroken {
			brokenLinks = append(brokenLinks, link)
		}
	}

	// If no broken links found, report and return
	if len(brokenLinks) == 0 {
		log.Info("No broken links found.")
		return nil
	}

	// Display all broken links
	log.Info("Found %d broken symlinks:", len(brokenLinks))
	for _, link := range brokenLinks {
		log.Info("  %s -> %s (target missing)", link.Path, link.Target)
	}

	// In dry-run mode, just show what would be removed
	if dryRun {
		log.Info("")
		log.Info("%s Would remove the above broken symlinks.", DryRunPrefix)
		return nil
	}

	// Ask for confirmation
	if !ConfirmPrompt("\nRemove all broken symlinks?") {
		log.Info("Cancelled.")
		return nil
	}

	// Remove the broken links
	for _, link := range brokenLinks {
		if err := os.Remove(link.Path); err != nil {
			log.Info("%s: %v", Red("Error removing"), err)
			continue
		}
		log.Info("Removed: %s", link.Path)
	}

	return nil
}

// shouldIgnoreEntry determines if an entry should be ignored based on patterns
func shouldIgnoreEntry(sourceItem, repoRoot string, mapping *LinkMapping, config *Config) bool {
	relPathForIgnore := strings.TrimPrefix(sourceItem, repoRoot+"/")
	if mapping != nil && mapping.Source != "." {
		relPathForIgnore = strings.TrimPrefix(relPathForIgnore, mapping.Source+"/")
	}
	return config.ShouldIgnore(relPathForIgnore)
}

// collectPlannedLinks walks a source directory and collects all files that should be linked
func collectPlannedLinks(sourcePath, targetPath, repoRoot string, mapping *LinkMapping, config *Config) ([]PlannedLink, error) {
	var links []PlannedLink

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories - we only link files
		if info.IsDir() {
			return nil
		}

		// Check if this file should be ignored
		if shouldIgnoreEntry(path, repoRoot, mapping, config) {
			return nil
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}

		// Build target path
		target := filepath.Join(targetPath, relPath)

		links = append(links, PlannedLink{
			Source: path,
			Target: target,
		})

		return nil
	})

	return links, err
}

// executePlannedLinks creates the symlinks according to the plan
func executePlannedLinks(links []PlannedLink) error {
	// Track which directories we've created to avoid redundant checks
	createdDirs := make(map[string]bool)

	for _, link := range links {
		// Create parent directory if needed
		parentDir := filepath.Dir(link.Target)
		if !createdDirs[parentDir] {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", parentDir, err)
			}
			createdDirs[parentDir] = true
		}

		// Create the symlink
		if err := createLink(link.Source, link.Target); err != nil {
			// Log warning but continue with other links
			log.Info("%s linking file %s: %v", Yellow("Warning"), link.Target, err)
		}
	}

	return nil
}

// createLink creates a single symlink, handling existing files/links
func createLink(source, target string) error {
	// Check if target exists
	if info, err := os.Lstat(target); err == nil {
		// If it's already a symlink pointing to our source, nothing to do
		if info.Mode()&os.ModeSymlink != 0 {
			if existingTarget, err := os.Readlink(target); err == nil && existingTarget == source {
				return nil
			}
			// Remove existing symlink pointing elsewhere
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("removing existing link: %w", err)
			}
		} else {
			// Target exists and is not a symlink
			return fmt.Errorf("%s exists and is not a symlink", target)
		}
	}

	// Create new symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}

	log.Info("Linked: %s", target)
	return nil
}
