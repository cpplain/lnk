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
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	PrintHeader("Creating Symlinks")

	// Require LinkMappings to be defined
	if len(config.LinkMappings) == 0 {
		return ErrNoLinkMappings
	}

	// Phase 1: Collect all files to link
	PrintVerbose("Starting phase 1: collecting files to link")
	var plannedLinks []PlannedLink
	for _, mapping := range config.LinkMappings {
		PrintVerbose("Processing mapping: %s -> %s", mapping.Source, mapping.Target)

		// Expand the target path (handle ~/)
		targetPath, err := ExpandPath(mapping.Target)
		if err != nil {
			return fmt.Errorf("expanding target path for mapping %s: %w", mapping.Source, err)
		}
		PrintVerbose("Expanded target path: %s", targetPath)

		// Build the source path
		sourcePath := filepath.Join(absConfigRepo, mapping.Source)
		PrintVerbose("Source path: %s", sourcePath)

		// Check if source directory exists
		if info, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				PrintSkip("Skipping %s: source directory does not exist", mapping.Source)
				continue
			}
			return fmt.Errorf("failed to check source directory for mapping %s: %w", mapping.Source, err)
		} else if !info.IsDir() {
			return fmt.Errorf("failed to process mapping %s: source path is not a directory: %s", mapping.Source, sourcePath)
		}

		Debug("Processing mapping: %s -> %s", mapping.Source, mapping.Target)

		// Collect files from this mapping
		links, err := collectPlannedLinks(sourcePath, targetPath, absConfigRepo, &mapping, config)
		if err != nil {
			return fmt.Errorf("collecting files for mapping %s: %w", mapping.Source, err)
		}
		plannedLinks = append(plannedLinks, links...)
	}

	if len(plannedLinks) == 0 {
		PrintInfo("No files to link.")
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
		fmt.Println()
		PrintDryRun("Would create %d symlink(s):", len(plannedLinks))
		for _, link := range plannedLinks {
			PrintDryRun("Would link: %s -> %s", ContractPath(link.Target), ContractPath(link.Source))
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
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	return removeLinks(absConfigRepo, config, dryRun, false)
}

// removeLinks is the internal implementation that allows skipping confirmation
func removeLinks(configRepo string, config *Config, dryRun bool, skipConfirm bool) error {
	PrintHeader("Removing Symlinks")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := FindManagedLinks(homeDir, configRepo, config)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	if len(links) == 0 {
		PrintInfo("No symlinks found to remove.")
		return nil
	}

	// Track results for summary
	var removed, failed int

	// Remove links
	for _, link := range links {
		if dryRun {
			PrintDryRun("Would remove: %s", ContractPath(link.Path))
		} else {
			if err := os.Remove(link.Path); err != nil {
				PrintError("Failed to remove %s: %v", ContractPath(link.Path), err)
				failed++
				continue
			}
			PrintSuccess("Removed: %s", ContractPath(link.Path))
			removed++
		}
	}

	// Print summary for non-dry-run
	if !dryRun {
		fmt.Println()
		if removed > 0 {
			PrintSuccess("Removed %d symlink(s) successfully", removed)
		}
		if failed > 0 {
			PrintWarning("Failed to remove %d symlink(s)", failed)
		}
	}

	return nil
}

// PruneLinks removes broken symlinks pointing to the config repository
func PruneLinks(configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	PrintHeader("Pruning Broken Symlinks")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find all symlinks pointing to our repo
	links, err := FindManagedLinks(homeDir, absConfigRepo, config)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
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
		PrintInfo("No broken symlinks found.")
		return nil
	}

	// Track results for summary
	var pruned, failed int

	// Remove the broken links
	for _, link := range brokenLinks {
		if dryRun {
			PrintDryRun("Would prune: %s", ContractPath(link.Path))
		} else {
			if err := os.Remove(link.Path); err != nil {
				PrintError("Failed to remove %s: %v", ContractPath(link.Path), err)
				failed++
				continue
			}
			PrintSuccess("Pruned: %s", ContractPath(link.Path))
			pruned++
		}
	}

	// Print summary for non-dry-run
	if !dryRun {
		fmt.Println()
		if pruned > 0 {
			PrintSuccess("Pruned %d broken symlink(s) successfully", pruned)
		}
		if failed > 0 {
			PrintWarning("Failed to prune %d symlink(s)", failed)
		}
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
			return fmt.Errorf("failed to calculate relative path: %w", err)
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

	// Track results for summary
	var created, failed int

	for _, link := range links {
		// Create parent directory if needed
		parentDir := filepath.Dir(link.Target)
		if !createdDirs[parentDir] {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
			}
			createdDirs[parentDir] = true
		}

		// Create the symlink
		if err := createLink(link.Source, link.Target); err != nil {
			if _, ok := err.(LinkExistsError); ok {
				// Link already exists with correct target - skip silently
				continue
			}
			// Print warning but continue with other links
			PrintWarning("Failed to link %s: %v", ContractPath(link.Target), err)
			failed++
		} else {
			created++
		}
	}

	// Print summary
	fmt.Println()
	if created > 0 {
		PrintSuccess("Created %d symlink(s) successfully", created)
	}
	if failed > 0 {
		PrintWarning("Failed to create %d symlink(s)", failed)
	}

	return nil
}

// LinkExistsError indicates a symlink already exists with the correct target
type LinkExistsError struct {
	target string
}

func (e LinkExistsError) Error() string {
	return fmt.Sprintf("symlink already exists: %s", e.target)
}

// createLink creates a single symlink, handling existing files/links
func createLink(source, target string) error {
	// Check if target exists
	if info, err := os.Lstat(target); err == nil {
		// If it's already a symlink pointing to our source, nothing to do
		if info.Mode()&os.ModeSymlink != 0 {
			if existingTarget, err := os.Readlink(target); err == nil && existingTarget == source {
				return LinkExistsError{target: target}
			}
			// Remove existing symlink pointing elsewhere
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("failed to remove existing link: %w", err)
			}
		} else {
			// Target exists and is not a symlink
			return fmt.Errorf("failed to create symlink: %s already exists and is not a symlink. Use 'cfgman adopt' to adopt this file first", target)
		}
	}

	// Create new symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	PrintSuccess("Created: %s", ContractPath(target))
	return nil
}
