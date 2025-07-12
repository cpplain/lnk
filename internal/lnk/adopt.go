package lnk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateAdoptSource validates the source path and checks if it's already adopted
func validateAdoptSource(absSource, absSourceDir string) error {
	// Check if source exists
	sourceInfo, err := os.Lstat(absSource)
	if err != nil {
		if os.IsNotExist(err) {
			return NewPathErrorWithHint("adopt", absSource, err,
				"Check that the file path is correct and the file exists")
		}
		return fmt.Errorf("failed to check source: %w", err)
	}

	// Check if source is already a symlink
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absSource)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}

		// Check if it's already managed using proper path comparison
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(absSource), target)
		}
		if cleanTarget, err := filepath.Abs(absTarget); err == nil {
			if relPath, err := filepath.Rel(absSourceDir, cleanTarget); err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
				return NewLinkErrorWithHint("adopt", absSource, target, ErrAlreadyAdopted,
					"This file is already managed by lnk. Use 'lnk status' to see managed files")
			}
		}
	}
	return nil
}

// determineRelativePath determines the relative path from home directory
func determineRelativePath(absSource string) (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	relPath, err := getRelativePathFromHome(absSource, homeDir)
	if err != nil {
		return "", "", NewPathErrorWithHint("adopt", absSource,
			fmt.Errorf("source must be within home directory: %w", err),
			"lnk can only manage files within your home directory")
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
		return nil, NewValidationErrorWithHint("source directory", sourceDir,
			"not found in config mappings",
			fmt.Sprintf("Add it to .lnk.json first with a mapping like: {\"source\": \"%s\", \"target\": \"~/\"}", sourceDir))
	}

	// Check if source directory exists in the repository
	sourceDirPath := filepath.Join(configRepo, sourceDir)
	if _, err := os.Stat(sourceDirPath); os.IsNotExist(err) {
		// Create the source directory if it doesn't exist
		if err := os.MkdirAll(sourceDirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create source directory %s: %w", sourceDirPath, err)
		}
	}

	return mapping, nil
}

// performAdoption performs the actual file move and symlink creation
func performAdoption(absSource, destPath string) error {
	// Check if source is a directory
	sourceInfo, err := os.Stat(absSource)
	if err != nil {
		return fmt.Errorf("failed to check source: %w", err)
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
		return fmt.Errorf("failed to create destination directory: %w", err)
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
			return fmt.Errorf("failed to create symlink: %v (rollback also failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// performDirectoryAdoption recursively adopts all files in a directory
func performDirectoryAdoption(absSource, destPath string) error {
	// First, create the destination directory structure
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Track results
	var adopted, skipped int
	var walkErr error
	var fileCount int

	// Walk the source directory
	processFiles := func() error {
		return filepath.Walk(absSource, func(sourcePath string, info os.FileInfo, err error) error {
			fileCount++
			if err != nil {
				return err
			}

			// Calculate relative path from source root
			relPath, err := filepath.Rel(absSource, sourcePath)
			if err != nil {
				return fmt.Errorf("failed to calculate relative path: %w", err)
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
					return fmt.Errorf("failed to create directory %s: %w", destItemPath, err)
				}
				// Directory will be created in original location after all files are moved
				return nil
			}

			// It's a file - check if it's already adopted
			sourceFileInfo, err := os.Lstat(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to check file %s: %w", relPath, err)
			}

			// Skip if it's already a symlink
			if sourceFileInfo.Mode()&os.ModeSymlink != 0 {
				// Check if it points to our destination
				if target, err := os.Readlink(sourcePath); err == nil && target == destItemPath {
					Debug("Skipping already adopted file: %s", relPath)
					skipped++
					return nil
				}
			}

			// Check if destination already exists
			if _, err := os.Stat(destItemPath); err == nil {
				PrintSkip("Skipping %s: file already exists in repository at %s", ContractPath(sourcePath), ContractPath(destItemPath))
				skipped++
				return nil
			}

			// Move file to repo
			if err := os.Rename(sourcePath, destItemPath); err != nil {
				// If rename fails (e.g., cross-device), fall back to copy and remove
				if err := copyAndVerify(sourcePath, destItemPath); err != nil {
					return fmt.Errorf("failed to move file %s: %w", relPath, err)
				}
			}

			// Create parent directory in original location if needed
			sourceDir := filepath.Dir(sourcePath)
			if err := os.MkdirAll(sourceDir, 0755); err != nil {
				// Rollback: move file back
				os.Rename(destItemPath, sourcePath)
				return fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}

			// Create symlink back
			if err := os.Symlink(destItemPath, sourcePath); err != nil {
				// Rollback: move file back
				if rollbackErr := os.Rename(destItemPath, sourcePath); rollbackErr != nil {
					return fmt.Errorf("failed to create symlink: %v (rollback also failed: %v)", err, rollbackErr)
				}
				return fmt.Errorf("failed to create symlink for %s: %w", relPath, err)
			}

			PrintSuccess("Adopted: %s", ContractPath(sourcePath))
			adopted++
			return nil
		})
	}

	// Use ShowProgress to handle the 1-second delay
	walkErr = ShowProgress("Scanning files to adopt", processFiles)

	// Print summary if we adopted multiple files
	if walkErr == nil && (adopted > 0 || skipped > 0) {
		if adopted > 0 {
			PrintSuccess("Successfully adopted %d file(s)", adopted)
			PrintInfo("Next: Run 'lnk create' to create symlinks")
		}
		if skipped > 0 {
			PrintInfo("Skipped %d file(s) (already adopted or exist in repo)", skipped)
		}
	}

	return walkErr
}

// copyAndVerify copies a file and verifies the copy succeeded
func copyAndVerify(absSource, destPath string) error {
	// First, try to copy the file
	if copyErr := copyPath(absSource, destPath); copyErr != nil {
		return fmt.Errorf("failed to copy to repository: %w", copyErr)
	}

	// Verify the copy succeeded by comparing file info
	srcInfo, err := os.Stat(absSource)
	if err != nil {
		// Source disappeared? Clean up and fail
		os.RemoveAll(destPath)
		return fmt.Errorf("failed to copy: source file disappeared during operation: %w", err)
	}
	dstInfo, err := os.Stat(destPath)
	if err != nil {
		// Copy didn't complete properly
		os.RemoveAll(destPath)
		return fmt.Errorf("failed to copy: destination file not created properly: %w", err)
	}

	// For files, verify size matches
	if !srcInfo.IsDir() && srcInfo.Size() != dstInfo.Size() {
		os.RemoveAll(destPath)
		return fmt.Errorf("failed to verify copy: size mismatch (src: %d, dst: %d)", srcInfo.Size(), dstInfo.Size())
	}

	// Now try to remove the original
	if err := os.RemoveAll(absSource); err != nil {
		// Removal failed - we now have the file in both places
		// Try to clean up the copy
		if cleanupErr := os.RemoveAll(destPath); cleanupErr != nil {
			// Both the original removal and cleanup failed
			return fmt.Errorf("failed to complete adoption: file exists in both locations. Failed to remove original (%v) and failed to clean up copy (%v). Manual intervention required", err, cleanupErr)
		}
		return fmt.Errorf("failed to remove original after copy: %w", err)
	}

	return nil
}

// Adopt moves a file or directory into the source directory and creates a symlink back
func Adopt(source string, config *Config, sourceDir string, dryRun bool) error {
	// Convert to absolute paths
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}
	PrintHeader("Adopting Files")

	// Ensure sourceDir is absolute
	absSourceDir, err := ExpandPath(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to expand source directory: %w", err)
	}

	// Validate that sourceDir exists in config mappings
	var mapping *LinkMapping
	for i := range config.LinkMappings {
		expandedSource, err := ExpandPath(config.LinkMappings[i].Source)
		if err != nil {
			continue
		}
		if expandedSource == absSourceDir {
			mapping = &config.LinkMappings[i]
			break
		}
	}

	if mapping == nil {
		return NewValidationErrorWithHint("source directory", sourceDir,
			"not found in config mappings",
			fmt.Sprintf("Add it to .lnk.json first with a mapping like: {\"source\": \"%s\", \"target\": \"~/\"}", sourceDir))
	}

	// Validate source and check if already adopted
	if err := validateAdoptSource(absSource, absSourceDir); err != nil {
		return err
	}

	// Determine relative path from home directory
	relPath, _, err := determineRelativePath(absSource)
	if err != nil {
		return err
	}

	// Create source directory if it doesn't exist
	if _, err := os.Stat(absSourceDir); os.IsNotExist(err) {
		if err := os.MkdirAll(absSourceDir, 0755); err != nil {
			return fmt.Errorf("failed to create source directory %s: %w", absSourceDir, err)
		}
	}

	destPath := filepath.Join(absSourceDir, relPath)

	// Check if source is a directory for proper dry-run output
	sourceInfo, err := os.Stat(absSource)
	if err != nil {
		return fmt.Errorf("failed to check source: %w", err)
	}

	// Check if destination already exists (only for files, not directories)
	if !sourceInfo.IsDir() {
		if _, err := os.Stat(destPath); err == nil {
			return NewPathErrorWithHint("adopt", destPath,
				fmt.Errorf("destination already exists in repo"),
				"Remove the existing file first or choose a different source directory")
		}
	}

	// Validate symlink creation
	if err := ValidateSymlinkCreation(absSource, destPath); err != nil {
		return fmt.Errorf("failed to validate adoption: %w", err)
	}

	if dryRun {
		PrintDryRun("Would adopt: %s", ContractPath(absSource))
		if sourceInfo.IsDir() {
			PrintDetail("Move directory contents to: %s", ContractPath(destPath))
			PrintDetail("Create individual symlinks for each file")
		} else {
			PrintDetail("Move to: %s", ContractPath(destPath))
			PrintDetail("Create symlink: %s -> %s", ContractPath(absSource), ContractPath(destPath))
		}
		return nil
	}

	// Perform the adoption
	if err := performAdoption(absSource, destPath); err != nil {
		return err
	}

	if !sourceInfo.IsDir() {
		PrintSuccess("Adopted: %s", ContractPath(absSource))
		PrintInfo("Next: Run 'lnk create' to create symlinks")
	}

	return nil
}
