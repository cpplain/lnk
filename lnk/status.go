package lnk

import (
	"fmt"
	"sort"
)

// Status displays the status of managed symlinks for the source directory
func Status(opts LinkOptions) error {
	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	sourceDir, targetDir := paths.SourceDir, paths.TargetDir

	PrintCommandHeader("Symlink Status")
	PrintVerbose("Source directory: %s", sourceDir)
	PrintVerbose("Target directory: %s", targetDir)

	// Find all symlinks for the source directory
	managedLinks, err := FindManagedLinks(targetDir, []string{sourceDir})
	if err != nil {
		return fmt.Errorf("failed to find managed links: %w", err)
	}

	// Sort by link path
	sort.Slice(managedLinks, func(i, j int) bool {
		return managedLinks[i].Path < managedLinks[j].Path
	})

	// Display links
	if len(managedLinks) > 0 {
		// Separate active and broken links
		var activeLinks, brokenLinks []ManagedLink
		for _, link := range managedLinks {
			if link.IsBroken {
				brokenLinks = append(brokenLinks, link)
			} else {
				activeLinks = append(activeLinks, link)
			}
		}

		// Display active links
		if len(activeLinks) > 0 {
			for _, link := range activeLinks {
				if ShouldSimplifyOutput() {
					// For piped output, use simple format
					fmt.Printf("active %s\n", ContractPath(link.Path))
				} else {
					PrintSuccess("Active: %s", ContractPath(link.Path))
				}
			}
		}

		// Display broken links
		if len(brokenLinks) > 0 {
			if len(activeLinks) > 0 && !ShouldSimplifyOutput() {
				fmt.Println()
			}
			for _, link := range brokenLinks {
				if ShouldSimplifyOutput() {
					// For piped output, use simple format
					fmt.Printf("broken %s\n", ContractPath(link.Path))
				} else {
					PrintError("Broken: %s", ContractPath(link.Path))
				}
			}
		}

		// Summary
		if !ShouldSimplifyOutput() {
			fmt.Println()
			PrintInfo("Total: %s (%s active, %s broken)",
				Bold(fmt.Sprintf("%d links", len(managedLinks))),
				Green(fmt.Sprintf("%d", len(activeLinks))),
				Red(fmt.Sprintf("%d", len(brokenLinks))))
		}
	} else {
		PrintEmptyResult("active links")
	}

	return nil
}
