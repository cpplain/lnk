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
		return fmt.Errorf("resolving repository path: %w", err)
	}
	absLink, err := filepath.Abs(link)
	if err != nil {
		return fmt.Errorf("resolving link path: %w", err)
	}

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
			return fmt.Errorf("finding managed links: %w", err)
		}
		if len(managed) == 0 {
			return fmt.Errorf("no managed symlinks found in directory: %s", absLink)
		}
		managedLinks = managed
	} else {
		// For single files, validate it's a managed symlink
		if linkInfo.Mode()&os.ModeSymlink == 0 {
			return NewPathError("orphan", absLink, ErrNotSymlink)
		}

		if link := checkManagedLink(absLink, absConfigRepo, config); link != nil {
			// Check if the link is broken
			if link.IsBroken {
				return fmt.Errorf("symlink target does not exist: %s", link.Target)
			}
			managedLinks = []ManagedLink{*link}
		} else {
			// Read symlink to provide better error message
			target, err := os.Readlink(absLink)
			if err != nil {
				return fmt.Errorf("reading symlink: %w", err)
			}
			return fmt.Errorf("symlink is not managed by this repository: %s -> %s", absLink, target)
		}
	}

	// Display what will be orphaned
	if len(managedLinks) == 1 {
		link := managedLinks[0]
		log.Info("Orphaning: %s -> %s %s", link.Path, link.Target, Cyan("["+link.Source+"]"))
	} else {
		log.Info("Found %d managed symlink(s) in %s:", len(managedLinks), absLink)
		for _, link := range managedLinks {
			relPath, _ := filepath.Rel(absLink, link.Path)
			if relPath == "" {
				relPath = filepath.Base(link.Path)
			}
			log.Info("  • %s %s", relPath, Cyan("["+link.Source+"]"))
		}
	}

	// Handle dry-run
	if dryRun {
		log.Info("")
		log.Info("%s Would orphan %d symlink(s)", Yellow(DryRunPrefix), len(managedLinks))
		for _, link := range managedLinks {
			log.Info("")
			log.Info("%s Would orphan: %s", Yellow(DryRunPrefix), link.Path)
			log.Info("  Remove symlink: %s", link.Path)
			log.Info("  Copy from: %s", link.Target)
			log.Info("  Remove from repository: %s", link.Target)
		}
		return nil
	}

	// Confirm with user
	log.Info("")
	log.Info("This will:")
	log.Info("  - Remove symlink(s)")
	log.Info("  - Copy content back to original location(s)")
	log.Info("  - Remove file(s) from repository")
	log.Info("")

	confirmMsg := "Continue?"
	if len(managedLinks) > 1 {
		confirmMsg = fmt.Sprintf("Orphan all %d symlink(s)?", len(managedLinks))
	}
	if !ConfirmPrompt(confirmMsg) {
		return ErrOperationCancelled
	}

	// Process each link
	errors := []string{}
	successCount := 0

	for i, link := range managedLinks {
		if len(managedLinks) > 1 {
			log.Info("")
			log.Info("[%d/%d] Orphaning: %s %s", i+1, len(managedLinks), link.Path, Cyan("["+link.Source+"]"))
		}

		err := orphanManagedLink(link)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", link.Path, err))
		} else {
			successCount++
		}
	}

	// Report results for batch operations
	if len(managedLinks) > 1 {
		log.Info("")
		log.Info("%s Orphaned %d/%d symlink(s)", Green(SuccessIcon), successCount, len(managedLinks))

		if len(errors) > 0 {
			log.Info("")
			log.Info("%s Failed to orphan %d symlink(s):", Red(FailureIcon), len(errors))
			for _, err := range errors {
				log.Info("  • %s", err)
			}
			return fmt.Errorf("some operations failed")
		}
	}

	return nil
}

// orphanManagedLink performs the actual orphaning of a validated managed link
func orphanManagedLink(link ManagedLink) error {
	// Check if target exists (in case it became broken since discovery)
	targetInfo, err := os.Stat(link.Target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("symlink target does not exist: %s", link.Target)
		}
		return fmt.Errorf("checking target: %w", err)
	}

	// Remove the symlink first
	if err := os.Remove(link.Path); err != nil {
		return fmt.Errorf("removing symlink: %w", err)
	}

	// Copy content from repo to original location
	if err := copyPath(link.Target, link.Path); err != nil {
		// Try to restore symlink on error
		os.Symlink(link.Target, link.Path)
		return fmt.Errorf("copying from repo: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(link.Path, targetInfo.Mode()); err != nil {
		log.Info("  %s Warning: Failed to set permissions: %v", Yellow(WarningIcon), err)
	}

	log.Info("  %s Removed symlink: %s", Green(SuccessIcon), link.Path)
	log.Info("  %s Copied content from: %s", Green(SuccessIcon), link.Target)

	// Remove from repository
	if err := removeFromRepository(link.Target); err != nil {
		log.Info("  %s Warning: Failed to remove from repository: %v", Yellow(WarningIcon), err)
		log.Info("  %s You may need to manually remove: %s", Yellow(WarningIcon), link.Target)
	} else {
		log.Info("  %s Removed from repository: %s", Green(SuccessIcon), link.Target)
	}

	return nil
}
