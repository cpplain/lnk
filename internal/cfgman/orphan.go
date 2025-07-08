package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
)

// Orphan removes a file or directory from repository management
func Orphan(link string, configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute paths
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}
	absLink, err := filepath.Abs(link)
	if err != nil {
		return fmt.Errorf("failed to resolve link path: %w", err)
	}
	PrintHeader("Orphaning Files")

	// Check if path exists
	linkInfo, err := os.Lstat(absLink)
	if err != nil {
		return NewPathError("orphan", absLink, err)
	}

	// Collect managed links to orphan
	var managedLinks []ManagedLink

	if linkInfo.IsDir() && linkInfo.Mode()&os.ModeSymlink == 0 {
		// For directories, find all managed symlinks within
		managed, err := FindManagedLinks(absLink, absConfigRepo, config)
		if err != nil {
			return fmt.Errorf("failed to find managed links: %w", err)
		}
		if len(managed) == 0 {
			return fmt.Errorf("failed to orphan: no managed symlinks found in directory: %s", absLink)
		}
		managedLinks = managed
	} else {
		// For single files, validate it's a managed symlink
		if linkInfo.Mode()&os.ModeSymlink == 0 {
			return NewPathErrorWithHint("orphan", absLink, ErrNotSymlink,
				"Only symlinks can be orphaned. Use 'rm' to remove regular files")
		}

		if link := checkManagedLink(absLink, absConfigRepo, config); link != nil {
			// Check if the link is broken
			if link.IsBroken {
				return WithHint(
					fmt.Errorf("failed to orphan: symlink target does not exist: %s", ContractPath(link.Target)),
					"The file in the repository has been deleted. Use 'rm' to remove the broken symlink")
			}
			managedLinks = []ManagedLink{*link}
		} else {
			// Read symlink to provide better error message
			target, err := os.Readlink(absLink)
			if err != nil {
				return fmt.Errorf("failed to read symlink: %w", err)
			}
			return WithHint(
				fmt.Errorf("failed to orphan: symlink is not managed by this repository: %s -> %s", absLink, target),
				"This symlink was not created by cfgman. Use 'rm' to remove it manually")
		}
	}

	// Handle dry-run
	if dryRun {
		fmt.Println()
		PrintDryRun("Would orphan %d symlink(s)", len(managedLinks))
		for _, link := range managedLinks {
			fmt.Println()
			PrintDryRun("Would orphan: %s", ContractPath(link.Path))
			PrintDetail("Remove symlink: %s", ContractPath(link.Path))
			PrintDetail("Copy from: %s", ContractPath(link.Target))
			PrintDetail("Remove from repository: %s", ContractPath(link.Target))
		}
		return nil
	}

	// Process each link
	errors := []string{}
	var orphaned int

	for _, link := range managedLinks {
		err := orphanManagedLink(link)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", ContractPath(link.Path), err))
		} else {
			orphaned++
		}
	}

	// Report summary (only show summary if we processed multiple links)
	if len(managedLinks) > 1 {
		fmt.Println()
		if orphaned > 0 {
			PrintSuccess("Successfully orphaned %d file(s)", orphaned)
		}
	}
	if len(errors) > 0 {
		fmt.Println()
		PrintError("Failed to orphan %d file(s):", len(errors))
		for _, err := range errors {
			PrintDetail("• %s", err)
		}
		return fmt.Errorf("failed to complete all orphan operations")
	}

	return nil
}

// orphanManagedLink performs the actual orphaning of a validated managed link
func orphanManagedLink(link ManagedLink) error {
	// Check if target exists (in case it became broken since discovery)
	targetInfo, err := os.Stat(link.Target)
	if err != nil {
		if os.IsNotExist(err) {
			return WithHint(
				fmt.Errorf("failed to orphan: symlink target does not exist: %s", ContractPath(link.Target)),
				"The file in the repository has been deleted. Use 'rm' to remove the broken symlink")
		}
		return fmt.Errorf("failed to check target: %w", err)
	}

	// Remove the symlink first
	if err := os.Remove(link.Path); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Copy content from repo to original location
	if err := copyPath(link.Target, link.Path); err != nil {
		// Try to restore symlink on error
		os.Symlink(link.Target, link.Path)
		return fmt.Errorf("failed to copy from repository: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(link.Path, targetInfo.Mode()); err != nil {
		PrintWarning("Failed to set permissions: %v", err)
	}

	// Remove from repository
	if err := removeFromRepository(link.Target); err != nil {
		PrintWarning("Failed to remove from repository: %v", err)
		PrintWarning("You may need to manually remove: %s", ContractPath(link.Target))
		return fmt.Errorf("failed to remove file from repository")
	}

	PrintSuccess("Orphaned: %s", ContractPath(link.Path))

	return nil
}
