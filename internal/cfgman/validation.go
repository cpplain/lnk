package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateNoCircularSymlink checks if creating a symlink would create a circular reference
func ValidateNoCircularSymlink(source, target string) error {
	// Check if target is already a symlink that points back to source
	targetInfo, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			// Target doesn't exist yet, no circular link possible
			return nil
		}
		return fmt.Errorf("failed to check target: %w", err)
	}

	// If target is a symlink, check where it points
	if targetInfo.Mode()&os.ModeSymlink != 0 {
		linkDest, err := os.Readlink(target)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}

		// Resolve to absolute paths for comparison
		absSource, err := filepath.Abs(source)
		if err != nil {
			return fmt.Errorf("failed to resolve source path: %w", err)
		}

		absLinkDest := linkDest
		if !filepath.IsAbs(linkDest) {
			absLinkDest = filepath.Join(filepath.Dir(target), linkDest)
		}
		absLinkDest, err = filepath.Abs(absLinkDest)
		if err != nil {
			return fmt.Errorf("failed to resolve link destination: %w", err)
		}

		// Check if the symlink points to our source - this is OK, not circular
		if absSource == absLinkDest {
			return nil // Already points to correct location
		}
	}

	// Also check if source is within target directory (would create a loop)
	absSource, _ := filepath.Abs(source)
	absTarget, _ := filepath.Abs(target)

	if strings.HasPrefix(absSource, absTarget+string(filepath.Separator)) {
		return NewValidationErrorWithHint("symlink", absSource,
			"source is inside target directory, would create circular reference",
			"Move the source file to a different location first")
	}

	return nil
}

// ValidateNoOverlappingPaths checks if source and target paths would overlap
func ValidateNoOverlappingPaths(source, target string) error {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Check if paths are the same
	if absSource == absTarget {
		return NewValidationErrorWithHint("symlink", absSource,
			"source and target are the same path",
			"Ensure source and target paths are different")
	}

	// Check if source is inside target
	if strings.HasPrefix(absSource, absTarget+string(filepath.Separator)) {
		return NewValidationErrorWithHint("path overlap", absSource,
			"source path is inside target path",
			"Choose paths that don't overlap")
	}

	// Check if target is inside source
	if strings.HasPrefix(absTarget, absSource+string(filepath.Separator)) {
		return NewValidationErrorWithHint("path overlap", absTarget,
			"target path is inside source path",
			"Choose paths that don't overlap")
	}

	return nil
}

// ValidateSymlinkCreation performs all validation checks before creating a symlink
func ValidateSymlinkCreation(source, target string) error {
	// Check for circular symlinks
	if err := ValidateNoCircularSymlink(source, target); err != nil {
		return err
	}

	// Check for overlapping paths
	if err := ValidateNoOverlappingPaths(source, target); err != nil {
		return err
	}

	return nil
}
