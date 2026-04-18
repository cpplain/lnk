package lnk

import (
	"fmt"
	"path/filepath"
)

// Prune removes broken symlinks managed by the source directory
func Prune(opts LinkOptions) error {
	PrintCommandHeader("Pruning Broken Symlinks")

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
	var removedParents []string

	// Remove the broken links
	for _, link := range brokenLinks {
		if err := RemoveSymlink(link.Path); err != nil {
			PrintWarningWithHint(fmt.Errorf("Failed to prune %s: %w", ContractPath(link.Path), err))
			failed++
			continue
		}
		PrintSuccess("Pruned: %s", ContractPath(link.Path))
		pruned++
		removedParents = append(removedParents, filepath.Dir(link.Path))
	}

	// Clean empty parent directories
	CleanEmptyDirs(removedParents, targetDir)

	// Print summary
	if pruned > 0 {
		PrintSummary("Pruned %d broken symlink(s) successfully", pruned)
	}
	if failed > 0 {
		PrintWarning("Failed to prune %d symlink(s)", failed)
		return fmt.Errorf("failed to prune %d symlink(s)", failed)
	}
	if failed == 0 {
		PrintNextStep("status", sourceDir, "view remaining managed files")
	}

	return nil
}
