package lnk

import (
	"fmt"
)

// RemoveLinks removes symlinks managed by the source directory
func RemoveLinks(opts LinkOptions) error {
	PrintCommandHeader("Removing Symlinks")

	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	sourceDir, targetDir := paths.SourceDir, paths.TargetDir

	// Find all managed links for the source directory
	PrintVerbose("Searching for managed links in %s", targetDir)
	links, err := FindManagedLinks(targetDir, []string{sourceDir})
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
		if err := RemoveSymlink(link.Path); err != nil {
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
		return fmt.Errorf("failed to remove %d symlink(s)", failed)
	}

	return nil
}
