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

	// Create pattern matcher once before walk for efficiency
	pm := NewPatternMatcher(ignorePatterns)

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
		if pm.Matches(relPath) {
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

	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	sourceDir, targetDir := paths.SourceDir, paths.TargetDir

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
	return executePlannedLinks(plannedLinks, sourceDir)
}

// executePlannedLinks creates the symlinks according to the plan
func executePlannedLinks(links []PlannedLink, sourceDir string) error {
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
			if err := CreateSymlink(link.Source, link.Target); err != nil {
				if _, ok := err.(LinkExistsError); ok {
					// Link already exists with correct target - skip silently
					continue
				}
				// Print warning but continue with other links
				PrintWarningWithHint(fmt.Errorf("Failed to link %s: %w", ContractPath(link.Target), err))
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
		PrintNextStep("status", sourceDir, "verify links")
	} else if failed == 0 {
		// All links were skipped (already exist)
		PrintInfo("All symlinks already exist")
	}
	if failed > 0 {
		PrintWarning("Failed to create %d symlink(s)", failed)
		return fmt.Errorf("failed to create %d symlink(s)", failed)
	}

	return nil
}
