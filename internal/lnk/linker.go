package lnk

import (
	"fmt"
	"os"
	"path/filepath"
)

// PlannedLink represents a source file and its target symlink location
type PlannedLink struct {
	Source string
	Target string
}

// CreateLinks creates symlinks from the source directories to the target directories
func CreateLinks(config *Config, dryRun bool) error {
	PrintHeader("Creating Symlinks")

	// Require LinkMappings to be defined
	if len(config.LinkMappings) == 0 {
		return NewValidationErrorWithHint("link mappings", "", "no link mappings defined",
			"Add at least one mapping to your .lnk.json file. Example: {\"source\": \"home\", \"target\": \"~/\"}")
	}

	// Phase 1: Collect all files to link
	PrintVerbose("Starting phase 1: collecting files to link")
	var plannedLinks []PlannedLink
	for _, mapping := range config.LinkMappings {
		PrintVerbose("Processing mapping: %s -> %s", mapping.Source, mapping.Target)

		// Expand the source path (handle ~/)
		sourcePath, err := ExpandPath(mapping.Source)
		if err != nil {
			return fmt.Errorf("expanding source path for mapping %s: %w", mapping.Source, err)
		}
		PrintVerbose("Source path: %s", sourcePath)

		// Expand the target path (handle ~/)
		targetPath, err := ExpandPath(mapping.Target)
		if err != nil {
			return fmt.Errorf("expanding target path for mapping %s: %w", mapping.Source, err)
		}
		PrintVerbose("Expanded target path: %s", targetPath)

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
		links, err := collectPlannedLinks(sourcePath, targetPath, &mapping, config)
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
func RemoveLinks(config *Config, dryRun bool, force bool) error {
	return removeLinks(config, dryRun, !force)
}

// removeLinks is the internal implementation that allows skipping confirmation
func removeLinks(config *Config, dryRun bool, skipConfirm bool) error {
	PrintHeader("Removing Symlinks")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return NewPathErrorWithHint("get home directory", "~", err,
			"Check that the HOME environment variable is set correctly")
	}

	// Find all symlinks pointing to configured source directories
	links, err := FindManagedLinks(homeDir, config)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	if len(links) == 0 {
		PrintInfo("No symlinks found to remove.")
		return nil
	}

	// Show what will be removed in dry-run mode
	if dryRun {
		for _, link := range links {
			PrintDryRun("Would remove: %s", ContractPath(link.Path))
		}
		return nil
	}

	// Confirm action if not skipped
	if !skipConfirm {
		fmt.Println()
		var prompt string
		if len(links) == 1 {
			prompt = fmt.Sprintf("This will remove 1 symlink. Continue? (y/N): ")
		} else {
			prompt = fmt.Sprintf("This will remove %d symlink(s). Continue? (y/N): ", len(links))
		}

		confirmed, err := ConfirmAction(prompt)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			PrintInfo("Operation cancelled.")
			return nil
		}
	}

	// Track results for summary
	var removed, failed int

	// Remove links
	for _, link := range links {
		if err := os.Remove(link.Path); err != nil {
			PrintError("Failed to remove %s: %v", ContractPath(link.Path), err)
			failed++
			continue
		}
		PrintSuccess("Removed: %s", ContractPath(link.Path))
		removed++
	}

	// Print summary
	fmt.Println()
	if removed > 0 {
		PrintSuccess("Removed %d symlink(s) successfully", removed)
		PrintInfo("Next: Run 'lnk create' to recreate links or 'lnk status' to see remaining links")
	}
	if failed > 0 {
		PrintWarning("Failed to remove %d symlink(s)", failed)
	}

	return nil
}

// PruneLinks removes broken symlinks pointing to configured source directories
func PruneLinks(config *Config, dryRun bool, force bool) error {
	PrintHeader("Pruning Broken Symlinks")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return NewPathErrorWithHint("get home directory", "~", err,
			"Check that the HOME environment variable is set correctly")
	}

	// Find all symlinks pointing to configured source directories
	links, err := FindManagedLinks(homeDir, config)
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

	// Show what will be pruned in dry-run mode
	if dryRun {
		for _, link := range brokenLinks {
			PrintDryRun("Would prune: %s", ContractPath(link.Path))
		}
		return nil
	}

	// Confirm action if not forced
	if !force {
		fmt.Println()
		var prompt string
		if len(brokenLinks) == 1 {
			prompt = fmt.Sprintf("This will remove 1 broken symlink. Continue? (y/N): ")
		} else {
			prompt = fmt.Sprintf("This will remove %d broken symlink(s). Continue? (y/N): ", len(brokenLinks))
		}

		confirmed, err := ConfirmAction(prompt)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			PrintInfo("Operation cancelled.")
			return nil
		}
	}

	// Track results for summary
	var pruned, failed int

	// Remove the broken links
	for _, link := range brokenLinks {
		if err := os.Remove(link.Path); err != nil {
			PrintError("Failed to remove %s: %v", ContractPath(link.Path), err)
			failed++
			continue
		}
		PrintSuccess("Pruned: %s", ContractPath(link.Path))
		pruned++
	}

	// Print summary
	fmt.Println()
	if pruned > 0 {
		PrintSuccess("Pruned %d broken symlink(s) successfully", pruned)
		PrintInfo("Next: Run 'lnk status' to check remaining links")
	}
	if failed > 0 {
		PrintWarning("Failed to prune %d symlink(s)", failed)
	}

	return nil
}

// shouldIgnoreEntry determines if an entry should be ignored based on patterns
func shouldIgnoreEntry(sourceItem, sourcePath string, mapping *LinkMapping, config *Config) bool {
	// Get relative path from source directory
	relPath, err := filepath.Rel(sourcePath, sourceItem)
	if err != nil {
		// If we can't get relative path, don't ignore
		return false
	}
	return config.ShouldIgnore(relPath)
}

// collectPlannedLinks walks a source directory and collects all files that should be linked
func collectPlannedLinks(sourcePath, targetPath string, mapping *LinkMapping, config *Config) ([]PlannedLink, error) {
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
		if shouldIgnoreEntry(path, sourcePath, mapping, config) {
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

	processLinks := func() error {
		for _, link := range links {
			// Create parent directory if needed
			parentDir := filepath.Dir(link.Target)
			if !createdDirs[parentDir] {
				if err := os.MkdirAll(parentDir, 0755); err != nil {
					return NewPathErrorWithHint("create directory", parentDir, err,
						"Check that you have write permissions in the parent directory")
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
		return nil
	}

	// Use ShowProgress to handle the 1-second delay
	if err := ShowProgress("Creating symlinks", processLinks); err != nil {
		return err
	}

	// Print summary
	if created > 0 {
		PrintSuccess("Created %d symlink(s) successfully", created)
		PrintInfo("Next: Run 'lnk status' to verify links")
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
				return NewLinkErrorWithHint("remove existing link", source, target, err,
					"Check file permissions and ensure you have write access to the target directory")
			}
		} else {
			// Target exists and is not a symlink
			return NewLinkErrorWithHint("create symlink", source, target,
				fmt.Errorf("file already exists and is not a symlink"),
				fmt.Sprintf("Use 'lnk adopt %s <source-dir>' to adopt this file first", target))
		}
	}

	// Create new symlink
	if err := os.Symlink(source, target); err != nil {
		return NewLinkErrorWithHint("create symlink", source, target, err,
			"Check that the parent directory exists and you have write permissions")
	}

	PrintSuccess("Created: %s", ContractPath(target))
	return nil
}
