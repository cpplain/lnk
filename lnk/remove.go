package lnk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// collectManagedLinks walks SourceDir and returns target paths that are managed symlinks.
// A target symlink is "managed" if it resolves to the corresponding source file.
func collectManagedLinks(sourceDir, targetDir string) ([]string, error) {
	// Resolve sourceDir so comparisons work when EvalSymlinks resolves OS-level
	// symlinks (e.g., macOS /var -> /private/var)
	resolvedSourceDir, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolving source directory: %w", err)
	}

	var managed []string

	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Compute expected target path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}
		targetPath := filepath.Join(targetDir, relPath)

		// Check if target is a symlink
		info, err := os.Lstat(targetPath)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return nil // doesn't exist or not a symlink — skip
		}

		// Verify the symlink points into sourceDir
		resolved, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			return nil // broken or inaccessible — skip
		}
		rel, _ := filepath.Rel(resolvedSourceDir, resolved)
		if strings.HasPrefix(rel, "..") || rel == "." {
			return nil // not managed by this source
		}

		managed = append(managed, targetPath)
		return nil
	})

	return managed, err
}

// RemoveLinks removes symlinks managed by the source directory
func RemoveLinks(opts LinkOptions) error {
	PrintCommandHeader("Removing Symlinks")

	// Expand and validate paths
	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	sourceDir, targetDir := paths.SourceDir, paths.TargetDir

	// Walk source dir to find managed links
	PrintVerbose("Walking source directory %s to find managed links", sourceDir)
	managed, err := collectManagedLinks(sourceDir, targetDir)
	if err != nil {
		return fmt.Errorf("walking source directory: %w", err)
	}

	if len(managed) == 0 {
		PrintEmptyResult("symlinks to remove")
		return nil
	}

	// Show what will be removed in dry-run mode
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would remove %d symlink(s):", len(managed))
		for _, path := range managed {
			PrintDryRun("Would remove: %s", ContractPath(path))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Track results for summary
	var removed, failed int
	var removedParents []string

	// Remove links
	for _, path := range managed {
		if err := RemoveSymlink(path); err != nil {
			PrintError("Failed to remove %s: %v", ContractPath(path), err)
			failed++
			continue
		}
		PrintSuccess("Removed: %s", ContractPath(path))
		removed++
		removedParents = append(removedParents, filepath.Dir(path))
	}

	// Clean empty parent directories
	CleanEmptyDirs(removedParents, targetDir)

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
