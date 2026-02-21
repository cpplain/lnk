package lnk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OrphanOptions holds options for orphaning files from management
type OrphanOptions struct {
	SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
	TargetDir string   // where symlinks are (default: ~)
	Paths     []string // symlink paths to orphan (e.g., ["~/.bashrc", "~/.vimrc"])
	DryRun    bool     // preview mode
}

// OrphanWithOptions removes files from package management using the new options-based interface
func OrphanWithOptions(opts OrphanOptions) error {
	PrintCommandHeader("Orphaning Files")

	// Validate inputs
	if len(opts.Paths) == 0 {
		return NewValidationErrorWithHint("paths", "", "at least one file path is required",
			"Specify which files to orphan, e.g.: lnk -O ~/.bashrc")
	}

	// Expand paths
	absSourceDir, err := ExpandPath(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("failed to expand source directory: %w", err)
	}
	PrintVerbose("Source directory: %s", absSourceDir)

	absTargetDir, err := ExpandPath(opts.TargetDir)
	if err != nil {
		return fmt.Errorf("failed to expand target directory: %w", err)
	}
	PrintVerbose("Target directory: %s", absTargetDir)

	// Validate source directory exists
	if _, err := os.Stat(absSourceDir); os.IsNotExist(err) {
		return NewValidationErrorWithHint("source directory", absSourceDir, "does not exist",
			fmt.Sprintf("Check the source directory: %s", absSourceDir))
	}

	// Collect managed links to orphan
	var managedLinks []ManagedLink

	for _, path := range opts.Paths {
		// Expand path
		absPath, err := ExpandPath(path)
		if err != nil {
			PrintErrorWithHint(WithHint(
				fmt.Errorf("failed to expand path %s: %w", path, err),
				"Check that the path is valid"))
			continue
		}

		// Check if path exists
		linkInfo, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				PrintErrorWithHint(NewPathErrorWithHint("orphan", absPath, err,
					"Check that the file path is correct"))
			} else {
				PrintErrorWithHint(NewPathError("orphan", absPath, err))
			}
			continue
		}

		// Handle directories by finding all managed symlinks within
		if linkInfo.IsDir() && linkInfo.Mode()&os.ModeSymlink == 0 {
			// For directories, find all managed symlinks within that point to source dir
			sources := []string{absSourceDir}
			managed, err := FindManagedLinksForSources(absPath, sources)
			if err != nil {
				PrintErrorWithHint(WithHint(
					fmt.Errorf("failed to find managed links in %s: %w", path, err),
					"Check directory permissions"))
				continue
			}
			if len(managed) == 0 {
				PrintErrorWithHint(WithHint(
					fmt.Errorf("no managed symlinks found in directory: %s", path),
					"Use 'lnk -S' to see managed links"))
				continue
			}
			managedLinks = append(managedLinks, managed...)
			continue
		}

		// For single files, validate it's a managed symlink
		if linkInfo.Mode()&os.ModeSymlink == 0 {
			PrintErrorWithHint(NewPathErrorWithHint("orphan", absPath, ErrNotSymlink,
				"Only symlinks can be orphaned. Use 'rm' to remove regular files"))
			continue
		}

		// Check if this is a managed link pointing to our source directory
		target, err := os.Readlink(absPath)
		if err != nil {
			PrintErrorWithHint(WithHint(
				fmt.Errorf("failed to read symlink %s: %w", path, err),
				"Check symlink permissions"))
			continue
		}

		// Resolve to absolute target path
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(absPath), target)
		}
		absTarget, err = filepath.Abs(absTarget)
		if err != nil {
			PrintErrorWithHint(WithHint(
				fmt.Errorf("failed to resolve target for %s: %w", path, err),
				"Check symlink target"))
			continue
		}

		// Check if target is within source directory
		relPath, err := filepath.Rel(absSourceDir, absTarget)
		if err != nil || strings.HasPrefix(relPath, "..") {
			PrintErrorWithHint(WithHint(
				fmt.Errorf("symlink is not managed by source directory: %s -> %s", path, target),
				"This symlink was not created by lnk from this source. Use 'rm' to remove it manually"))
			continue
		}

		// Check if link is broken
		isBroken := false
		if _, err := os.Stat(absTarget); os.IsNotExist(err) {
			PrintErrorWithHint(WithHint(
				fmt.Errorf("symlink target does not exist: %s", ContractPath(absTarget)),
				"The file in the repository has been deleted. Use 'rm' to remove the broken symlink"))
			continue
		}

		// Add to managed links
		managedLinks = append(managedLinks, ManagedLink{
			Path:     absPath,
			Target:   absTarget,
			IsBroken: isBroken,
			Source:   absSourceDir,
		})
	}

	// If no managed links found, return
	if len(managedLinks) == 0 {
		PrintInfo("No managed symlinks to orphan")
		return nil
	}

	// Handle dry-run
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would orphan %d symlink(s)", len(managedLinks))
		for _, link := range managedLinks {
			fmt.Println()
			PrintDryRun("Would orphan: %s", ContractPath(link.Path))
			PrintDetail("Remove symlink: %s", ContractPath(link.Path))
			PrintDetail("Copy from: %s", ContractPath(link.Target))
			PrintDetail("Remove from repository: %s", ContractPath(link.Target))
		}
		fmt.Println()
		PrintDryRunSummary()
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

	// Report summary
	if orphaned > 0 {
		PrintSummary("Successfully orphaned %d file(s)", orphaned)
		PrintNextStep("status", "view remaining managed files")
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

// Orphan removes a file or directory from repository management
func Orphan(link string, config *Config, dryRun bool, force bool) error {
	// Convert to absolute paths
	absLink, err := filepath.Abs(link)
	if err != nil {
		return fmt.Errorf("failed to resolve link path: %w", err)
	}
	PrintCommandHeader("Orphaning Files")

	// Check if path exists
	linkInfo, err := os.Lstat(absLink)
	if err != nil {
		return NewPathError("orphan", absLink, err)
	}

	// Collect managed links to orphan
	var managedLinks []ManagedLink

	if linkInfo.IsDir() && linkInfo.Mode()&os.ModeSymlink == 0 {
		// For directories, find all managed symlinks within
		managed, err := FindManagedLinks(absLink, config)
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

		if link := checkManagedLink(absLink, config); link != nil {
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
				"This symlink was not created by lnk. Use 'rm' to remove it manually")
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
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Confirm action if not forced
	if !force {
		fmt.Println()
		var prompt string
		if len(managedLinks) == 1 {
			prompt = fmt.Sprintf("This will orphan 1 file. Continue? (y/N): ")
		} else {
			prompt = fmt.Sprintf("This will orphan %d file(s). Continue? (y/N): ", len(managedLinks))
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
		if orphaned > 0 {
			PrintSummary("Successfully orphaned %d file(s)", orphaned)
			PrintNextStep("status", "see remaining managed files")
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
