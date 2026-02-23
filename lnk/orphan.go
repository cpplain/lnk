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

// Orphan removes files from package management using the new options-based interface
func Orphan(opts OrphanOptions) error {
	PrintCommandHeader("Orphaning Files")

	// Validate inputs
	if len(opts.Paths) == 0 {
		return NewValidationErrorWithHint("paths", "", "at least one file path is required",
			"Specify which files to orphan, e.g.: lnk -O ~/.bashrc")
	}

	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	absSourceDir, absTargetDir := paths.SourceDir, paths.TargetDir
	PrintVerbose("Source directory: %s", absSourceDir)
	PrintVerbose("Target directory: %s", absTargetDir)

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
			managed, err := FindManagedLinks(absPath, sources)
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
			IsBroken: false,
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
	if err := RemoveSymlink(link.Path); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Move content from repo to original location
	if err := MoveFile(link.Target, link.Path); err != nil {
		// Try to restore symlink on error
		if rollbackErr := os.Symlink(link.Target, link.Path); rollbackErr != nil {
			return fmt.Errorf("failed to move from repository: %v (rollback failed, symlink lost: %v)", err, rollbackErr)
		}
		return fmt.Errorf("failed to move from repository: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(link.Path, targetInfo.Mode()); err != nil {
		PrintWarning("Failed to set permissions: %v", err)
	}

	PrintSuccess("Orphaned: %s", ContractPath(link.Path))

	return nil
}
