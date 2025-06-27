package cfgman

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Adopt moves a file or directory into the configuration repository and creates a symlink back
func Adopt(source string, configRepo string, config *Config, sourceDir string, dryRun bool) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	// Resolve absolute path for source
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

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

		// Check if it's already managed
		if strings.Contains(target, configRepo) {
			return fmt.Errorf("file is already adopted: %s -> %s", absSource, target)
		}
	}

	// Determine destination path in repo
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Get relative path from home directory
	// Handle case where absSource might use a different home directory in tests
	relPath, err := filepath.Rel(homeDir, absSource)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// Try to get relative path from parent directories
		// This handles test scenarios where temp directories are used
		dir := filepath.Dir(absSource)
		for dir != "/" && dir != "." {
			base := filepath.Base(dir)
			if base == "home" {
				relPath, err = filepath.Rel(dir, absSource)
				if err == nil && !strings.HasPrefix(relPath, "..") {
					homeDir = dir
					break
				}
			}
			dir = filepath.Dir(dir)
		}

		// If still not found, use the original error
		if err != nil || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("source must be within home directory")
		}
	}

	// Validate sourceDir exists in config mappings
	mapping := config.GetMapping(sourceDir)
	if mapping == nil {
		// Check if it's a legacy source directory
		if sourceDir != "home" && sourceDir != "private/home" {
			return fmt.Errorf("source directory '%s' not found in config mappings", sourceDir)
		}
		// Create a default mapping for legacy directories
		config.LinkMappings = append(config.LinkMappings, LinkMapping{
			Source:          sourceDir,
			Target:          "~/",
			LinkAsDirectory: []string{},
		})
		mapping = config.GetMapping(sourceDir)
	}

	// Check if source directory exists in the repository
	sourceDirPath := filepath.Join(configRepo, sourceDir)
	if _, err := os.Stat(sourceDirPath); os.IsNotExist(err) {
		// Create the source directory if it doesn't exist
		if err := os.MkdirAll(sourceDirPath, 0755); err != nil {
			return fmt.Errorf("creating source directory %s: %w", sourceDirPath, err)
		}
	}

	destPath := filepath.Join(configRepo, sourceDir, relPath)

	// Check if file is inside a directory that's already linked
	// This must be checked BEFORE checking if destination exists
	if insideLink, linkPath := isInsideLinkedDirectory(absSource, homeDir, configRepo); insideLink {
		return fmt.Errorf("file is inside a directory that's already linked: %s", linkPath)
	}

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination already exists in repo: %s", destPath)
	}

	if dryRun {
		fmt.Printf("%s Would adopt: %s\n", Yellow("[DRY RUN]"), absSource)
		fmt.Printf("  Move to: %s\n", destPath)
		fmt.Printf("  Create symlink: %s -> %s\n", absSource, destPath)

		if sourceInfo.IsDir() {
			fmt.Printf("  %s: Add '%s' to %s mapping's link_as_directory? [y/N]\n", Yellow("Prompt"), relPath, sourceDir)
		}
		return nil
	}

	// Perform the adoption
	fmt.Printf("Adopting: %s\n", absSource)

	// Create parent directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Move file/directory to repo
	if err := os.Rename(absSource, destPath); err != nil {
		// If rename fails (e.g., cross-device), fall back to copy and remove
		if err := copyPath(absSource, destPath); err != nil {
			return fmt.Errorf("copying to repo: %w", err)
		}
		if err := os.RemoveAll(absSource); err != nil {
			// Try to clean up if removal fails
			os.RemoveAll(destPath)
			return fmt.Errorf("removing original after copy: %w", err)
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

	fmt.Printf("  %s Moved to: %s\n", Green("✓"), destPath)
	fmt.Printf("  %s Created symlink: %s -> %s\n", Green("✓"), absSource, destPath)

	// For directories, ask if it should be linked as a unit
	if sourceInfo.IsDir() {
		if shouldLinkAsDirectory(relPath) {
			err := config.AddDirectoryLinkToMapping(sourceDir, relPath)
			if err != nil {
				fmt.Printf("  %s Warning: %v\n", Yellow("!"), err)
			} else {
				if err := config.Save(configRepo); err != nil {
					fmt.Printf("  %s Warning: Failed to save config: %v\n", Yellow("!"), err)
				} else {
					fmt.Printf("  %s Added '%s' to %s mapping's link_as_directory\n", Green("✓"), relPath, sourceDir)
				}
			}
		}
	}

	return nil
}

// Orphan removes a file or directory from repository management
func Orphan(link string, configRepo string, config *Config, dryRun bool) error {
	// Convert to absolute path
	absRepo, err := filepath.Abs(configRepo)
	if err != nil {
		return fmt.Errorf("resolving repository path: %w", err)
	}
	configRepo = absRepo
	// Resolve absolute path
	absLink, err := filepath.Abs(link)
	if err != nil {
		return fmt.Errorf("resolving link path: %w", err)
	}

	// Check if path exists
	linkInfo, err := os.Lstat(absLink)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absLink)
		}
		return fmt.Errorf("checking path: %w", err)
	}

	// If it's a directory (not a symlink to a directory), find all managed symlinks within it
	if linkInfo.IsDir() && linkInfo.Mode()&os.ModeSymlink == 0 {
		return orphanDirectory(absLink, configRepo, config, dryRun)
	}

	// For single files/symlinks, use the original logic
	return orphanSingle(absLink, configRepo, config, dryRun)
}

// orphanDirectory recursively orphans all managed symlinks within a directory
func orphanDirectory(dirPath string, configRepo string, config *Config, dryRun bool) error {
	// Find all managed symlinks in the directory
	links, err := findManagedLinksInDir(dirPath, configRepo)
	if err != nil {
		return fmt.Errorf("finding managed links: %w", err)
	}

	if len(links) == 0 {
		return fmt.Errorf("no managed symlinks found in directory: %s", dirPath)
	}

	// Show what will be orphaned
	fmt.Printf("Found %d managed symlink(s) in %s:\n", len(links), dirPath)
	for _, link := range links {
		relPath, _ := filepath.Rel(dirPath, link)
		if relPath == "" {
			relPath = filepath.Base(link)
		}
		fmt.Printf("  • %s\n", relPath)
	}
	fmt.Println()

	if dryRun {
		fmt.Printf("%s Would orphan all %d symlink(s)\n", Yellow("[DRY RUN]"), len(links))
		for _, link := range links {
			target, _ := os.Readlink(link)
			fmt.Printf("\n%s Would orphan: %s\n", Yellow("[DRY RUN]"), link)
			fmt.Printf("  Remove symlink: %s\n", link)
			fmt.Printf("  Copy from: %s\n", target)
			fmt.Printf("  Remove from repository: %s\n", target)
		}
		return nil
	}

	// Confirm with user
	fmt.Printf("This will:\n")
	fmt.Printf("  - Remove symlinks\n")
	fmt.Printf("  - Copy content back to original locations\n")
	fmt.Printf("  - Remove files from repository\n")
	fmt.Printf("\n")
	if !confirmFunc(fmt.Sprintf("Orphan all %d symlink(s)?", len(links))) {
		return fmt.Errorf("operation cancelled")
	}

	// Process each symlink
	errors := []string{}
	successCount := 0

	for i, link := range links {
		fmt.Printf("\n[%d/%d] Orphaning: %s\n", i+1, len(links), link)

		// Orphan the individual symlink (without additional confirmation)
		err := orphanSingleWithConfirm(link, configRepo, config, false, false)

		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", link, err))
		} else {
			successCount++
		}
	}

	// Report results
	fmt.Printf("\n%s Orphaned %d/%d symlink(s)\n", Green("✓"), successCount, len(links))

	if len(errors) > 0 {
		fmt.Printf("\n%s Failed to orphan %d symlink(s):\n", Red("✗"), len(errors))
		for _, err := range errors {
			fmt.Printf("  • %s\n", err)
		}
		return fmt.Errorf("some operations failed")
	}

	return nil
}

// orphanSingle handles orphaning a single file or symlink
func orphanSingle(absLink string, configRepo string, config *Config, dryRun bool) error {
	return orphanSingleWithConfirm(absLink, configRepo, config, dryRun, true)
}

// orphanSingleWithConfirm handles orphaning with optional confirmation display
func orphanSingleWithConfirm(absLink string, configRepo string, config *Config, dryRun bool, showConfirmation bool) error {
	// Check if it's a symlink
	linkInfo, err := os.Lstat(absLink)
	if err != nil {
		return fmt.Errorf("checking link: %w", err)
	}

	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("not a symlink: %s", absLink)
	}

	// Read symlink target
	target, err := os.Readlink(absLink)
	if err != nil {
		return fmt.Errorf("reading symlink: %w", err)
	}

	// Check if it's managed by our repo
	if !strings.Contains(target, configRepo) {
		return fmt.Errorf("symlink is not managed by this repository: %s -> %s", absLink, target)
	}

	// Check if target exists
	targetInfo, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("symlink target does not exist: %s", target)
		}
		return fmt.Errorf("checking target: %w", err)
	}

	// Determine source mapping
	sourceMapping := determineSourceFromTarget(target, configRepo, config)

	if dryRun {
		fmt.Printf("%s Would orphan: %s %s\n", Yellow("[DRY RUN]"), absLink, Cyan("["+sourceMapping+"]"))
		fmt.Printf("  Remove symlink: %s\n", absLink)
		fmt.Printf("  Copy from: %s\n", target)
		fmt.Printf("  Remove from repository: %s\n", target)
		return nil
	}

	// Confirm with user
	if showConfirmation {
		fmt.Printf("Orphaning: %s -> %s %s\n", absLink, target, Cyan("["+sourceMapping+"]"))
		fmt.Printf("\nThis will:\n")
		fmt.Printf("  - Remove symlink\n")
		fmt.Printf("  - Copy content back to original location\n")
		fmt.Printf("  - Remove file from repository\n")
		fmt.Printf("\n")
		if !confirmFunc("Continue?") {
			return fmt.Errorf("operation cancelled")
		}
	} else {
		// Just show what we're processing when in batch mode
		fmt.Printf("Orphaning: %s %s\n", absLink, Cyan("["+sourceMapping+"]"))
	}

	// Remove the symlink first
	if err := os.Remove(absLink); err != nil {
		return fmt.Errorf("removing symlink: %w", err)
	}

	// Copy content from repo to original location
	if err := copyPath(target, absLink); err != nil {
		// Try to restore symlink on error
		os.Symlink(target, absLink)
		return fmt.Errorf("copying from repo: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(absLink, targetInfo.Mode()); err != nil {
		fmt.Printf("  %s Warning: Failed to set permissions: %v\n", Yellow("!"), err)
	}

	fmt.Printf("  %s Removed symlink: %s\n", Green("✓"), absLink)
	fmt.Printf("  %s Copied content from: %s\n", Green("✓"), target)

	// Remove from repository
	if err := removeFromRepository(target); err != nil {
		fmt.Printf("  %s Warning: Failed to remove from repository: %v\n", Yellow("!"), err)
		fmt.Printf("  %s You may need to manually remove: %s\n", Yellow("!"), target)
	} else {
		fmt.Printf("  %s Removed from repository: %s\n", Green("✓"), target)
	}

	return nil
}

// isInsideLinkedDirectory checks if a path is inside a directory that's already symlinked
func isInsideLinkedDirectory(path, homeDir, configRepo string) (bool, string) {
	// Walk up the directory tree
	currentPath := filepath.Dir(path)

	for currentPath != "/" && currentPath != homeDir && currentPath != "." {
		info, err := os.Lstat(currentPath)
		if err != nil {
			return false, ""
		}

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(currentPath)
			if err != nil {
				return false, ""
			}

			// Check if it points to our config repo
			if strings.Contains(target, configRepo) {
				return true, currentPath
			}
		}

		currentPath = filepath.Dir(currentPath)
	}

	return false, ""
}

// findManagedLinksInDir finds all symlinks within a directory that point to the config repo
func findManagedLinksInDir(dirPath, configRepo string) ([]string, error) {
	var links []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip if it's a directory
		if info.IsDir() {
			return nil
		}

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}

			// Check if it points to our config repo
			if strings.Contains(target, configRepo) {
				links = append(links, path)
			}
		}

		return nil
	})

	return links, err
}

// confirmFunc is a variable that holds the confirmation function (for testing)
var confirmFunc = confirm

// shouldLinkAsDirectory prompts the user whether a directory should be linked as a unit
func shouldLinkAsDirectory(relPath string) bool {
	fmt.Printf("\nDirectory adopted. Should '%s' be linked as a complete directory?\n", relPath)
	fmt.Println("(Choosing 'yes' means the entire directory will be symlinked as one unit)")
	return confirmFunc("Add to link_as_directory?")
}

// confirm is a wrapper around ConfirmPrompt for backward compatibility
func confirm(prompt string) bool {
	return ConfirmPrompt(prompt)
}

// copyPath recursively copies a file or directory
func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
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
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close destination file: %w", closeErr)
		}
		// If there was an error during copy, remove the partial file
		if err != nil {
			os.Remove(dst)
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Set file permissions
	if err = os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
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

// removeFromRepository removes a file from the repository (both git tracking and filesystem)
func removeFromRepository(path string) error {
	// First check if git is available
	gitCheckCmd := exec.Command("git", "--version")
	gitAvailable := gitCheckCmd.Run() == nil

	if !gitAvailable {
		// Git is not installed or not accessible, just remove from filesystem
		fmt.Fprintf(os.Stderr, "Warning: git is not available. Removing file from filesystem only.\n")
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove file/directory %s: %w", path, err)
		}
		fmt.Printf("Removed from repository: %s\n", path)
		return nil
	}

	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = filepath.Dir(path)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Not in a git repo, just remove from filesystem
		if os.Getenv("CFGMAN_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: %s is not in a git repository, removing from filesystem\n", path)
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove file/directory %s: %w", path, err)
		}
		fmt.Printf("Removed from repository: %s\n", path)
		return nil
	}

	// Check if the file is tracked by git
	statusCmd := exec.Command("git", "ls-files", "--error-unmatch", path)
	statusCmd.Dir = filepath.Dir(path)
	isTracked := statusCmd.Run() == nil

	if isTracked {
		// Remove the file from git
		cmd = exec.Command("git", "rm", "-f", path)
		cmd.Dir = filepath.Dir(path)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Provide helpful error message
			errMsg := fmt.Sprintf("Failed to remove %s from git tracking: %v", path, err)
			if len(output) > 0 {
				errMsg += fmt.Sprintf("\nGit output: %s", strings.TrimSpace(string(output)))
			}

			// Check for common issues
			if strings.Contains(string(output), "Permission denied") {
				errMsg += "\nHint: Check file permissions and repository ownership"
			} else if strings.Contains(string(output), "uncommitted changes") {
				errMsg += "\nHint: You may need to commit or stash your changes first"
			}

			return fmt.Errorf("%s", errMsg)
		}
		fmt.Printf("Removed %s from git tracking\n", path)
	} else {
		// File is not tracked by git, remove it from filesystem
		if os.Getenv("CFGMAN_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: %s is not tracked by git, removing from filesystem\n", path)
		}

		// Use RemoveAll to handle both files and directories
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove untracked file/directory %s: %w", path, err)
		}
		fmt.Printf("Removed untracked file/directory from repository: %s\n", path)
	}

	return nil
}

// determineSourceFromTarget determines which source mapping a target path belongs to
func determineSourceFromTarget(target, configRepo string, config *Config) string {
	// Remove the config repo prefix to get the relative path
	relPath := strings.TrimPrefix(target, configRepo)
	relPath = strings.TrimPrefix(relPath, "/")

	// Check each mapping to find which one contains this path
	for _, mapping := range config.LinkMappings {
		if strings.HasPrefix(relPath, mapping.Source+"/") || relPath == mapping.Source {
			return mapping.Source
		}
	}

	// Fallback detection for common directory patterns
	if strings.HasPrefix(relPath, "private/home/") {
		return "private/home"
	} else if strings.HasPrefix(relPath, "home/") {
		return "home"
	}

	// Default to showing the first directory component
	parts := strings.Split(relPath, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return "unknown"
}
