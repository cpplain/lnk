package cfgman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateAdoptSource validates the source path and checks if it's already adopted
func validateAdoptSource(absSource, absConfigRepo string) error {
	// Check if source exists
	sourceInfo, err := os.Lstat(absSource)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source does not exist: %s", absSource)
		}
		return fmt.Errorf("checking source: %w", err)
	}

	// Check if source is already a symlink
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absSource)
		if err != nil {
			return fmt.Errorf("reading symlink: %w", err)
		}

		// Check if it's already managed using proper path comparison
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(absSource), target)
		}
		if cleanTarget, err := filepath.Abs(absTarget); err == nil {
			if relPath, err := filepath.Rel(absConfigRepo, cleanTarget); err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
				return NewLinkError("adopt", absSource, target, ErrAlreadyAdopted)
			}
		}
	}
	return nil
}

// determineRelativePath determines the relative path from home directory
func determineRelativePath(absSource string) (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("getting home directory: %w", err)
	}

	relPath, err := getRelativePathFromHome(absSource, homeDir)
	if err != nil {
		return "", "", fmt.Errorf("source must be within home directory: %w", err)
	}

	return relPath, homeDir, nil
}

// getRelativePathFromHome attempts to get a relative path from the given home directory
func getRelativePathFromHome(absSource, homeDir string) (string, error) {
	relPath, err := filepath.Rel(homeDir, absSource)
	if err != nil {
		return "", err
	}

	// Ensure the path doesn't escape the home directory
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path is outside home directory")
	}

	return relPath, nil
}

// ensureSourceDirExists ensures the source directory exists in the repository
func ensureSourceDirExists(configRepo, sourceDir string, config *Config) (*LinkMapping, error) {
	// Validate sourceDir exists in config mappings
	mapping := config.GetMapping(sourceDir)
	if mapping == nil {
		return nil, fmt.Errorf("source directory '%s' not found in config mappings. Please add it to .cfgman.json first with a mapping like: {\"source\": \"%s\", \"target\": \"~/\"}", sourceDir, sourceDir)
	}

	// Check if source directory exists in the repository
	sourceDirPath := filepath.Join(configRepo, sourceDir)
	if _, err := os.Stat(sourceDirPath); os.IsNotExist(err) {
		// Create the source directory if it doesn't exist
		if err := os.MkdirAll(sourceDirPath, 0755); err != nil {
			return nil, fmt.Errorf("creating source directory %s: %w", sourceDirPath, err)
		}
	}

	return mapping, nil
}

// performAdoption performs the actual file move and symlink creation
func performAdoption(absSource, destPath string) error {
	// Check if source is a directory
	sourceInfo, err := os.Stat(absSource)
	if err != nil {
		return fmt.Errorf("checking source: %w", err)
	}

	if sourceInfo.IsDir() {
		// For directories, adopt each file individually
		return performDirectoryAdoption(absSource, destPath)
	}

	// For files, use the original logic
	return performFileAdoption(absSource, destPath)
}

// performFileAdoption handles adoption of a single file
func performFileAdoption(absSource, destPath string) error {
	// Create parent directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Move file to repo
	if err := os.Rename(absSource, destPath); err != nil {
		// If rename fails (e.g., cross-device), fall back to copy and remove
		if err := copyAndVerify(absSource, destPath); err != nil {
			return err
		}
	}

	// Create symlink back
	if err := os.Symlink(destPath, absSource); err != nil {
		// Rollback: move file back
		if rollbackErr := os.Rename(destPath, absSource); rollbackErr != nil {
			return fmt.Errorf("creating symlink failed: %v (rollback also failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("creating symlink: %w", err)
	}

	return nil
}

// performDirectoryAdoption recursively adopts all files in a directory
func performDirectoryAdoption(absSource, destPath string) error {
	// First, create the destination directory structure
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Walk the source directory
	return filepath.Walk(absSource, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from source root
		relPath, err := filepath.Rel(absSource, sourcePath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Calculate destination path
		destItemPath := filepath.Join(destPath, relPath)

		if info.IsDir() {
			// Create directory in destination
			if err := os.MkdirAll(destItemPath, info.Mode()); err != nil {
				return fmt.Errorf("creating directory %s: %w", destItemPath, err)
			}
			// Directory will be created in original location after all files are moved
			return nil
		}

		// It's a file - check if it's already adopted
		sourceFileInfo, err := os.Lstat(sourcePath)
		if err != nil {
			return fmt.Errorf("checking file %s: %w", relPath, err)
		}

		// Skip if it's already a symlink
		if sourceFileInfo.Mode()&os.ModeSymlink != 0 {
			// Check if it points to our destination
			if target, err := os.Readlink(sourcePath); err == nil && target == destItemPath {
				log.Debug("Skipping already adopted file: %s", relPath)
				return nil
			}
		}

		// Check if destination already exists
		if _, err := os.Stat(destItemPath); err == nil {
			log.Info("  Skipping %s: already exists in repo", relPath)
			return nil
		}

		// Move file to repo
		if err := os.Rename(sourcePath, destItemPath); err != nil {
			// If rename fails (e.g., cross-device), fall back to copy and remove
			if err := copyAndVerify(sourcePath, destItemPath); err != nil {
				return fmt.Errorf("moving file %s: %w", relPath, err)
			}
		}

		// Create parent directory in original location if needed
		sourceDir := filepath.Dir(sourcePath)
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			// Rollback: move file back
			os.Rename(destItemPath, sourcePath)
			return fmt.Errorf("creating parent directory for symlink: %w", err)
		}

		// Create symlink back
		if err := os.Symlink(destItemPath, sourcePath); err != nil {
			// Rollback: move file back
			if rollbackErr := os.Rename(destItemPath, sourcePath); rollbackErr != nil {
				return fmt.Errorf("creating symlink failed: %v (rollback also failed: %v)", err, rollbackErr)
			}
			return fmt.Errorf("creating symlink for %s: %w", relPath, err)
		}

		return nil
	})
}

// copyAndVerify copies a file and verifies the copy succeeded
func copyAndVerify(absSource, destPath string) error {
	// First, try to copy the file
	if copyErr := copyPath(absSource, destPath); copyErr != nil {
		return fmt.Errorf("copying to repo: %w", copyErr)
	}

	// Verify the copy succeeded by comparing file info
	srcInfo, err := os.Stat(absSource)
	if err != nil {
		// Source disappeared? Clean up and fail
		os.RemoveAll(destPath)
		return fmt.Errorf("source file disappeared during copy: %w", err)
	}
	dstInfo, err := os.Stat(destPath)
	if err != nil {
		// Copy didn't complete properly
		os.RemoveAll(destPath)
		return fmt.Errorf("destination file not created properly: %w", err)
	}

	// For files, verify size matches
	if !srcInfo.IsDir() && srcInfo.Size() != dstInfo.Size() {
		os.RemoveAll(destPath)
		return fmt.Errorf("copy verification failed: size mismatch (src: %d, dst: %d)", srcInfo.Size(), dstInfo.Size())
	}

	// Now try to remove the original
	if err := os.RemoveAll(absSource); err != nil {
		// Removal failed - we now have the file in both places
		// Try to clean up the copy
		if cleanupErr := os.RemoveAll(destPath); cleanupErr != nil {
			// Both the original removal and cleanup failed
			return fmt.Errorf("critical: file exists in both locations. Failed to remove original (%v) and failed to clean up copy (%v). Manual intervention required", err, cleanupErr)
		}
		return fmt.Errorf("removing original after copy: %w", err)
	}

	return nil
}

// Adopt moves a file or directory into the configuration repository and creates a symlink back
func Adopt(source string, configRepo string, config *Config, sourceDir string, dryRun bool) error {
	// Convert to absolute paths
	absConfigRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

	// Validate source and check if already adopted
	if err := validateAdoptSource(absSource, absConfigRepo); err != nil {
		return err
	}

	// Determine relative path from home directory
	relPath, _, err := determineRelativePath(absSource)
	if err != nil {
		return err
	}

	// Ensure source directory exists in repository
	_, err = ensureSourceDirExists(configRepo, sourceDir, config)
	if err != nil {
		return err
	}

	destPath := filepath.Join(absConfigRepo, sourceDir, relPath)

	// Check if source is a directory for proper dry-run output
	sourceInfo, err := os.Stat(absSource)
	if err != nil {
		return fmt.Errorf("checking source: %w", err)
	}

	// Check if destination already exists (only for files, not directories)
	if !sourceInfo.IsDir() {
		if _, err := os.Stat(destPath); err == nil {
			return fmt.Errorf("destination already exists in repo: %s", destPath)
		}
	}

	// Validate symlink creation
	if err := ValidateSymlinkCreation(absSource, destPath); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if dryRun {
		log.Info("%s Would adopt: %s", Yellow(DryRunPrefix), absSource)
		if sourceInfo.IsDir() {
			log.Info("  Move directory contents to: %s", destPath)
			log.Info("  Create individual symlinks for each file")
		} else {
			log.Info("  Move to: %s", destPath)
			log.Info("  Create symlink: %s -> %s", absSource, destPath)
		}
		return nil
	}

	// Perform the adoption
	log.Info("Adopting: %s", absSource)

	if err := performAdoption(absSource, destPath); err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		log.Info("  %s Moved directory contents to: %s", Green(SuccessIcon), destPath)
		log.Info("  %s Created individual symlinks for each file", Green(SuccessIcon))
	} else {
		log.Info("  %s Moved to: %s", Green(SuccessIcon), destPath)
		log.Info("  %s Created symlink: %s -> %s", Green(SuccessIcon), absSource, destPath)
	}

	return nil
}
