package lnk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ManagedLink represents a symlink managed by lnk
type ManagedLink struct {
	Path     string // The symlink path
	Target   string // The target path (what the symlink points to)
	IsBroken bool   // Whether the link is broken
	Source   string // Source mapping name (e.g., "home", "work")
}

// FindManagedLinks finds all symlinks in startPath that point to any of the specified source directories.
// sources should be absolute paths (use ExpandPath first if needed).
func FindManagedLinks(startPath string, sources []string) ([]ManagedLink, error) {
	// Resolve sources to canonical paths so containment checks work
	// when EvalSymlinks resolves path components (e.g., /var → /private/var on macOS)
	resolvedSources := make([]string, len(sources))
	for i, s := range sources {
		resolved, err := filepath.EvalSymlinks(s)
		if err != nil {
			resolvedSources[i] = s
		} else {
			resolvedSources[i] = resolved
		}
	}

	var links []ManagedLink
	var walkErrors []error

	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			PrintVerbose("Error walking path %s: %v", path, err)
			walkErrors = append(walkErrors, err)
			return nil
		}

		// Skip directories
		if d.IsDir() {
			name := filepath.Base(path)
			// Skip specific system directories
			if name == LibraryDir || name == TrashDir {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a symlink
		if d.Type()&fs.ModeSymlink == 0 {
			return nil
		}

		// Try filepath.EvalSymlinks first for non-broken links
		var resolvedTarget string
		var isBroken bool

		evalTarget, evalErr := filepath.EvalSymlinks(path)
		if evalErr == nil {
			// EvalSymlinks succeeded — link is not broken
			resolvedTarget = evalTarget
		} else {
			// EvalSymlinks failed — likely a broken link; fall back to manual resolution
			rawTarget, err := os.Readlink(path)
			if err != nil {
				PrintVerbose("Failed to read symlink %s: %v", path, err)
				return nil
			}

			absTarget := rawTarget
			if !filepath.IsAbs(rawTarget) {
				absTarget = filepath.Join(filepath.Dir(path), rawTarget)
			}
			cleanTarget, err := filepath.Abs(absTarget)
			if err != nil {
				PrintVerbose("Failed to get absolute path for target %s: %v", rawTarget, err)
				return nil
			}
			// Resolve the parent directory to handle path symlinks (e.g., /var → /private/var)
			parentDir := filepath.Dir(cleanTarget)
			if resolvedParent, err := filepath.EvalSymlinks(parentDir); err == nil {
				resolvedTarget = filepath.Join(resolvedParent, filepath.Base(cleanTarget))
			} else {
				resolvedTarget = cleanTarget
			}

			// Verify the target is genuinely missing vs other error
			if _, statErr := os.Stat(resolvedTarget); statErr != nil {
				if os.IsNotExist(statErr) {
					isBroken = true
				} else {
					// Other error (e.g., permission denied) — skip, can't classify
					return nil
				}
			}
		}

		// Check if target points to any of our sources.
		// Try resolved sources first (for EvalSymlinks-resolved targets), then
		// fall back to original sources (for manually resolved broken link targets
		// where parent directory resolution may have failed).
		var managedBySource string
		for i, resolved := range resolvedSources {
			relPath, err := filepath.Rel(resolved, resolvedTarget)
			if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
				managedBySource = sources[i]
				break
			}
		}
		if managedBySource == "" {
			for _, source := range sources {
				relPath, err := filepath.Rel(source, resolvedTarget)
				if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
					managedBySource = source
					break
				}
			}
		}

		if managedBySource == "" {
			return nil
		}

		links = append(links, ManagedLink{
			Path:     path,
			Target:   resolvedTarget,
			IsBroken: isBroken,
			Source:   managedBySource,
		})
		return nil
	})

	// Warn if there were errors during walk
	if len(walkErrors) > 0 {
		PrintVerbose("Encountered %d errors during filesystem walk - results may be incomplete", len(walkErrors))
	}

	return links, err
}

// LinkExistsError indicates a symlink already exists with the correct target
type LinkExistsError struct {
	target string
}

func (e LinkExistsError) Error() string {
	return fmt.Sprintf("symlink already exists: %s", e.target)
}

// CreateSymlink creates a single symlink, handling existing files/links
func CreateSymlink(source, target string) error {
	// Check if target exists
	if info, err := os.Lstat(target); err == nil {
		// If it's already a symlink pointing to our source, nothing to do
		if info.Mode()&os.ModeSymlink != 0 {
			if existingTarget, err := os.Readlink(target); err == nil && existingTarget == source {
				return LinkExistsError{target: target}
			}
			// Remove existing symlink pointing elsewhere
			if err := os.Remove(target); err != nil {
				return NewLinkErrorWithHint("remove existing link", source, target, err,
					"Check file permissions and ensure you have write access to the target directory")
			}
		} else {
			// Target exists and is not a symlink
			return NewLinkErrorWithHint("create symlink", source, target,
				fmt.Errorf("file already exists and is not a symlink"),
				fmt.Sprintf("Use 'lnk adopt %s <source-dir>' to adopt this file first", target))
		}
	}

	// Create new symlink
	if err := os.Symlink(source, target); err != nil {
		return NewLinkErrorWithHint("create symlink", source, target, err,
			"Check that the parent directory exists and you have write permissions")
	}

	PrintSuccess("Created: %s", ContractPath(target))
	return nil
}

// RemoveSymlink removes a symlink at the given path.
// Returns error if path is not a symlink or removal fails.
func RemoveSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return NewPathError("remove symlink", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return NewPathErrorWithHint("remove symlink", path, fmt.Errorf("not a symlink"),
			"Only symlinks can be removed with this operation")
	}
	return os.Remove(path)
}
