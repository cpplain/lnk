package lnk

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

// LinkOptions holds configuration for linking operations
type LinkOptions struct {
	SourceDir      string   // source directory - what to link from (e.g., ~/git/dotfiles)
	TargetDir      string   // where to create links (default: ~)
	IgnorePatterns []string // combined ignore patterns from all sources
	DryRun         bool     // preview mode without making changes
}

// collectPlannedLinksWithPatterns walks a source directory and collects all files that should be linked
// Uses ignore patterns directly instead of a Config object
func collectPlannedLinksWithPatterns(sourcePath, targetPath string, ignorePatterns []string) ([]PlannedLink, error) {
	var links []PlannedLink

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories - we only link files
		if info.IsDir() {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Check if this file should be ignored
		if MatchesPattern(relPath, ignorePatterns) {
			return nil
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

// CreateLinks creates symlinks using the provided options
func CreateLinks(opts LinkOptions) error {
	PrintCommandHeader("Creating Symlinks")

	// Expand source and target directories
	sourceDir, err := ExpandPath(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("expanding source directory %s: %w", opts.SourceDir, err)
	}

	targetDir, err := ExpandPath(opts.TargetDir)
	if err != nil {
		return fmt.Errorf("expanding target directory %s: %w", opts.TargetDir, err)
	}

	// Check if source directory exists
	if info, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return NewValidationErrorWithHint("source directory", sourceDir, "directory does not exist",
				"Ensure the source directory exists or specify a different path")
		}
		return fmt.Errorf("failed to check source directory: %w", err)
	} else if !info.IsDir() {
		return NewValidationErrorWithHint("source directory", sourceDir, "path is not a directory",
			"The source path must be a directory containing files to link")
	}

	// Phase 1: Collect all files to link
	PrintVerbose("Starting phase 1: collecting files to link")
	PrintVerbose("Source directory: %s", sourceDir)
	PrintVerbose("Target directory: %s", targetDir)

	plannedLinks, err := collectPlannedLinksWithPatterns(sourceDir, targetDir, opts.IgnorePatterns)
	if err != nil {
		return fmt.Errorf("collecting files to link: %w", err)
	}

	if len(plannedLinks) == 0 {
		PrintEmptyResult("files to link")
		return nil
	}

	// Phase 2: Validate all targets
	for _, link := range plannedLinks {
		if err := ValidateSymlinkCreation(link.Source, link.Target); err != nil {
			return fmt.Errorf("validation failed for %s -> %s: %w", link.Target, link.Source, err)
		}
	}

	// Phase 3: Execute (or show dry-run)
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would create %d symlink(s):", len(plannedLinks))
		for _, link := range plannedLinks {
			PrintDryRun("Would link: %s -> %s", ContractPath(link.Target), ContractPath(link.Source))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Execute the plan
	return executePlannedLinks(plannedLinks)
}

// findManagedLinksForSource finds all symlinks in targetDir that point to the source directory
func findManagedLinksForSource(targetDir, sourceDir string) ([]ManagedLink, error) {
	var links []ManagedLink

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
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

		// Check if target points to our source directory
		relPath, err := filepath.Rel(sourceDir, cleanTarget)
		if err != nil || strings.HasPrefix(relPath, "..") || relPath == "." {
			return nil
		}

		link := ManagedLink{
			Path:   path,
			Target: target,
			Source: sourceDir,
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

// RemoveLinks removes symlinks managed by the source directory
func RemoveLinks(opts LinkOptions) error {
	PrintCommandHeader("Removing Symlinks")

	// Expand source and target directories
	sourceDir, err := ExpandPath(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("expanding source directory %s: %w", opts.SourceDir, err)
	}

	targetDir, err := ExpandPath(opts.TargetDir)
	if err != nil {
		return fmt.Errorf("expanding target directory %s: %w", opts.TargetDir, err)
	}

	// Check if source directory exists
	if info, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return NewValidationErrorWithHint("source directory", sourceDir, "directory does not exist",
				"Ensure the source directory exists or specify a different path")
		}
		return fmt.Errorf("failed to check source directory: %w", err)
	} else if !info.IsDir() {
		return NewValidationErrorWithHint("source directory", sourceDir, "path is not a directory",
			"The source path must be a directory")
	}

	// Find all managed links for the source directory
	PrintVerbose("Searching for managed links in %s", targetDir)
	links, err := findManagedLinksForSource(targetDir, sourceDir)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	if len(links) == 0 {
		PrintEmptyResult("symlinks to remove")
		return nil
	}

	// Show what will be removed in dry-run mode
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would remove %d symlink(s):", len(links))
		for _, link := range links {
			PrintDryRun("Would remove: %s", ContractPath(link.Path))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
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
	if removed > 0 {
		PrintSummary("Removed %d symlink(s) successfully", removed)
	}
	if failed > 0 {
		PrintWarning("Failed to remove %d symlink(s)", failed)
	}

	return nil
}

// Prune removes broken symlinks managed by the source directory
func Prune(opts LinkOptions) error {
	PrintCommandHeader("Pruning Broken Symlinks")

	// Expand source and target directories
	sourceDir, err := ExpandPath(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("expanding source directory %s: %w", opts.SourceDir, err)
	}

	targetDir, err := ExpandPath(opts.TargetDir)
	if err != nil {
		return fmt.Errorf("expanding target directory %s: %w", opts.TargetDir, err)
	}

	// Check if source directory exists
	if info, err := os.Stat(sourceDir); err != nil {
		if os.IsNotExist(err) {
			return NewValidationErrorWithHint("source directory", sourceDir, "directory does not exist",
				"Ensure the source directory exists or specify a different path")
		}
		return fmt.Errorf("failed to check source directory: %w", err)
	} else if !info.IsDir() {
		return NewValidationErrorWithHint("source directory", sourceDir, "path is not a directory",
			"The source path must be a directory")
	}

	// Find all managed links for the source directory
	PrintVerbose("Searching for managed links in %s", targetDir)
	links, err := findManagedLinksForSource(targetDir, sourceDir)
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	// Filter to only broken links
	var brokenLinks []ManagedLink
	for _, link := range links {
		if link.IsBroken {
			brokenLinks = append(brokenLinks, link)
		}
	}

	if len(brokenLinks) == 0 {
		PrintEmptyResult("broken symlinks")
		return nil
	}

	// Show what will be pruned in dry-run mode
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would prune %d broken symlink(s):", len(brokenLinks))
		for _, link := range brokenLinks {
			PrintDryRun("Would prune: %s", ContractPath(link.Path))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Track results for summary
	var pruned, failed int

	// Remove the broken links
	for _, link := range brokenLinks {
		if err := os.Remove(link.Path); err != nil {
			PrintError("Failed to prune %s: %v", ContractPath(link.Path), err)
			failed++
			continue
		}
		PrintSuccess("Pruned: %s", ContractPath(link.Path))
		pruned++
	}

	// Print summary
	if pruned > 0 {
		PrintSummary("Pruned %d broken symlink(s) successfully", pruned)
	}
	if failed > 0 {
		PrintWarning("Failed to prune %d symlink(s)", failed)
	}

	return nil
}

// shouldIgnoreEntry determines if an entry should be ignored based on patterns

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
		PrintSummary("Created %d symlink(s) successfully", created)
		PrintNextStep("status", "verify links")
	} else if failed == 0 {
		// All links were skipped (already exist)
		PrintInfo("All symlinks already exist")
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
