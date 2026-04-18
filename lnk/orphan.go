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

// Orphan removes files from package management using two-phase transactional execution.
func Orphan(opts OrphanOptions) error {
	PrintCommandHeader("Orphaning Files")

	// Validate inputs
	if len(opts.Paths) == 0 {
		return NewValidationErrorWithHint("paths", "", "at least one file path is required",
			"Specify which files to orphan, e.g.: lnk orphan <source-dir> ~/.bashrc")
	}

	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	absSourceDir, absTargetDir := paths.SourceDir, paths.TargetDir
	PrintVerbose("Source directory: %s", absSourceDir)
	PrintVerbose("Target directory: %s", absTargetDir)

	// Phase 1: Collect and Validate
	var managedLinks []ManagedLink
	seen := make(map[string]bool)

	for _, path := range opts.Paths {
		absPath, err := ExpandPath(path)
		if err != nil {
			return WithHint(
				fmt.Errorf("failed to expand path %s: %w", path, err),
				"Check that the path is valid")
		}

		// Stat with Lstat (don't follow symlinks)
		linkInfo, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return NewPathErrorWithHint("orphan", absPath, err,
					"Check that the file path is correct")
			}
			return NewPathError("orphan", absPath, err)
		}

		// Validate target directory
		relToTarget, err := filepath.Rel(absTargetDir, absPath)
		if err != nil || strings.HasPrefix(relToTarget, "..") {
			return NewValidationErrorWithHint("path", ContractPath(absPath),
				fmt.Sprintf("path %s must be within target directory", ContractPath(absPath)),
				"Only paths within the target directory can be orphaned")
		}

		// Handle directories
		if linkInfo.IsDir() && linkInfo.Mode()&os.ModeSymlink == 0 {
			sources := []string{absSourceDir}
			managed, err := FindManagedLinks(absPath, sources)
			if err != nil {
				return NewPathError("orphan", absPath, err)
			}
			if len(managed) == 0 {
				return WithHint(
					fmt.Errorf("no managed symlinks found in %s", ContractPath(absPath)),
					"Use 'lnk status' to see managed links")
			}
			// Reject broken links
			for _, link := range managed {
				if link.IsBroken {
					return NewPathErrorWithHint("orphan", link.Path,
						fmt.Errorf("symlink target does not exist"),
						"Use 'rm' to remove the broken symlink directly")
				}
			}
			// Add active links, deduplicating
			for _, link := range managed {
				if !seen[link.Path] {
					seen[link.Path] = true
					managedLinks = append(managedLinks, link)
				}
			}
			continue
		}

		// Handle files
		if linkInfo.Mode()&os.ModeSymlink == 0 {
			return NewPathErrorWithHint("orphan", absPath, ErrNotSymlink,
				"Only symlinks can be orphaned. Use 'rm' to remove regular files")
		}

		// Read symlink target
		rawTarget, err := os.Readlink(absPath)
		if err != nil {
			return WithHint(
				fmt.Errorf("failed to read symlink %s: %w", ContractPath(absPath), err),
				"Check symlink permissions")
		}

		// Resolve to absolute path
		resolvedTarget := rawTarget
		if !filepath.IsAbs(rawTarget) {
			resolvedTarget = filepath.Join(filepath.Dir(absPath), rawTarget)
		}
		resolvedTarget = filepath.Clean(resolvedTarget)

		// Verify target is within source directory
		relPath, err := filepath.Rel(absSourceDir, resolvedTarget)
		if err != nil || strings.HasPrefix(relPath, "..") || relPath == "." {
			return NewLinkErrorWithHint("orphan", absPath, rawTarget,
				fmt.Errorf("not managed by source"),
				"This symlink was not created by lnk from this source. Use 'rm' to remove it directly")
		}

		// Verify target exists (not broken)
		if _, err := os.Stat(resolvedTarget); os.IsNotExist(err) {
			return NewPathErrorWithHint("orphan", absPath,
				fmt.Errorf("symlink target does not exist"),
				"The file in the repository has been deleted. Use 'rm' to remove the broken symlink")
		}

		// Deduplicate and add
		if !seen[absPath] {
			seen[absPath] = true
			managedLinks = append(managedLinks, ManagedLink{
				Path:     absPath,
				Target:   resolvedTarget,
				IsBroken: false,
				Source:   absSourceDir,
			})
		}
	}

	// Empty collection after dedup
	if len(managedLinks) == 0 {
		PrintInfo("No managed symlinks found.")
		return nil
	}

	// Dry-run
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would orphan %d symlink(s):", len(managedLinks))
		for _, link := range managedLinks {
			PrintDryRun("Would orphan: %s", ContractPath(link.Path))
			PrintDetail("Remove symlink: %s", ContractPath(link.Path))
			PrintDetail("Move from: %s", ContractPath(link.Target))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Phase 2: Execute with rollback
	type completedOrphan struct {
		link           ManagedLink
		symlinkRemoved bool
		fileMoved      bool
	}
	var completed []completedOrphan

	rollback := func(originalErr error) error {
		var rollbackErrors []string
		for i := len(completed) - 1; i >= 0; i-- {
			c := completed[i]
			if c.fileMoved {
				if err := MoveFile(c.link.Path, c.link.Target); err != nil {
					rollbackErrors = append(rollbackErrors, fmt.Sprintf("restore %s: %v", ContractPath(c.link.Target), err))
					continue
				}
			}
			if c.symlinkRemoved {
				if err := os.Symlink(c.link.Target, c.link.Path); err != nil {
					rollbackErrors = append(rollbackErrors, fmt.Sprintf("recreate symlink %s: %v", ContractPath(c.link.Path), err))
				}
			}
		}
		if len(rollbackErrors) > 0 {
			return fmt.Errorf("orphan failed: %v; rollback failed: %s", originalErr, strings.Join(rollbackErrors, "; "))
		}
		return originalErr
	}

	for _, link := range managedLinks {
		c := completedOrphan{link: link}

		// Verify target still exists
		targetInfo, err := os.Lstat(link.Target)
		if err != nil {
			completed = append(completed, c)
			return rollback(WithHint(
				fmt.Errorf("orphan failed: symlink target does not exist: %s", ContractPath(link.Target)),
				"Use 'rm' to remove the broken symlink"))
		}
		originalMode := targetInfo.Mode()

		// Remove symlink
		if err := RemoveSymlink(link.Path); err != nil {
			completed = append(completed, c)
			return rollback(fmt.Errorf("failed to remove symlink: %w", err))
		}
		c.symlinkRemoved = true

		// Move file from source to target
		if err := MoveFile(link.Target, link.Path); err != nil {
			completed = append(completed, c)
			return rollback(err)
		}
		c.fileMoved = true
		completed = append(completed, c)

		// Restore permissions (best-effort)
		if err := os.Chmod(link.Path, originalMode); err != nil {
			PrintVerbose("Failed to restore permissions for %s: %v", ContractPath(link.Path), err)
		}

		PrintSuccess("Orphaned: %s", ContractPath(link.Path))
	}

	// Clean empty source-side parent directories
	var parentDirs []string
	for _, link := range managedLinks {
		parentDirs = append(parentDirs, filepath.Dir(link.Target))
	}
	CleanEmptyDirs(parentDirs, absSourceDir)

	PrintSummary("Orphaned %d file(s) successfully", len(managedLinks))
	PrintNextStep("status", "view remaining managed files")
	return nil
}
