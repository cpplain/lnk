package cfgman

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyPath recursively copies a file or directory
func copyPath(src, dst string) error {
	// Validate and clean paths
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to validate source path: %w", err)
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("failed to validate destination path: %w", err)
	}

	// Prevent copying a directory into itself
	relPath, err := filepath.Rel(absSrc, absDst)
	if err == nil && !strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("failed to copy: cannot copy directory into itself")
	}

	srcInfo, err := os.Stat(absSrc)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(absSrc, absDst)
	}
	return copyFile(absSrc, absDst)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info before creating destination
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	// Ensure destination file is closed and cleaned up on error
	var copyErr error
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil && copyErr == nil {
			copyErr = fmt.Errorf("failed to close destination file: %w", closeErr)
		}
		// If there was an error during copy, remove the partial file
		if copyErr != nil {
			os.Remove(dst)
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		copyErr = fmt.Errorf("failed to copy file contents: %w", err)
		return copyErr
	}

	// Set file permissions
	if err = os.Chmod(dst, srcInfo.Mode()); err != nil {
		copyErr = fmt.Errorf("failed to set file permissions: %w", err)
		return copyErr
	}

	return copyErr
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
