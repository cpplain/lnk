package lnk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AdoptOptions holds options for adopting files into the source directory
type AdoptOptions struct {
	SourceDir string   // base directory for dotfiles (e.g., ~/git/dotfiles)
	TargetDir string   // where files currently are (default: ~)
	Paths     []string // files to adopt (e.g., ["~/.bashrc", "~/.vimrc"])
	DryRun    bool     // preview mode
}

// validateAdoptSource checks if a path is already adopted (a symlink pointing into sourceDir).
// Returns ErrAlreadyAdopted if so, nil otherwise. The caller is responsible for
// checking existence and handling non-adopted symlinks separately.
func validateAdoptSource(absPath, absSourceDir string) error {
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	target, err := os.Readlink(absPath)
	if err != nil {
		return nil
	}
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(absPath), target)
	}
	if cleanTarget, err := filepath.Abs(absTarget); err == nil {
		if relPath, err := filepath.Rel(absSourceDir, cleanTarget); err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
			return NewLinkErrorWithHint("adopt", absPath, target, ErrAlreadyAdopted,
				"This file is already managed by lnk. Use 'lnk status' to see managed files")
		}
	}
	return nil
}

// plannedAdoption represents a file to be adopted, validated in Phase 1.
type plannedAdoption struct {
	absPath  string // original location (becomes symlink)
	destPath string // destination in source dir (real file after move)
}

// Adopt adopts files into the source directory using two-phase transactional execution.
func Adopt(opts AdoptOptions) error {
	PrintCommandHeader("Adopting Files")

	if len(opts.Paths) == 0 {
		return NewValidationErrorWithHint("paths", "", "at least one file path is required",
			"Specify which files to adopt, e.g.: lnk adopt <source-dir> ~/.bashrc ~/.vimrc")
	}

	paths, err := ResolvePaths(opts.SourceDir, opts.TargetDir)
	if err != nil {
		return err
	}
	absSourceDir, absTargetDir := paths.SourceDir, paths.TargetDir
	PrintVerbose("Source directory: %s", absSourceDir)
	PrintVerbose("Target directory: %s", absTargetDir)

	// Phase 1: Collect and Validate
	var planned []plannedAdoption
	seen := make(map[string]bool)

	for _, path := range opts.Paths {
		absPath, err := ExpandPath(path)
		if err != nil {
			return WithHint(
				fmt.Errorf("failed to expand path %s: %w", path, err),
				"Check that the path is valid")
		}
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return WithHint(
				fmt.Errorf("failed to resolve path %s: %w", path, err),
				"Check that the path is valid")
		}

		info, err := os.Lstat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return NewPathErrorWithHint("adopt", absPath, err,
					"Check that the file path is correct and the file exists")
			}
			return NewPathError("adopt", absPath, err)
		}

		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			// Walk directory and collect regular files
			var files []string
			walkErr := filepath.WalkDir(absPath, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type().IsRegular() {
					files = append(files, p)
				}
				return nil
			})
			if walkErr != nil {
				return NewPathError("adopt", absPath, walkErr)
			}
			if len(files) == 0 {
				return WithHint(
					fmt.Errorf("no files to adopt in %s", ContractPath(absPath)),
					"Check that the directory contains regular files")
			}
			for _, f := range files {
				if err := collectAdoption(f, absSourceDir, absTargetDir, nil, seen, &planned); err != nil {
					return err
				}
			}
		} else {
			if err := collectAdoption(absPath, absSourceDir, absTargetDir, info, seen, &planned); err != nil {
				return err
			}
		}
	}

	// Dry-run
	if opts.DryRun {
		fmt.Println()
		PrintDryRun("Would adopt %d file(s):", len(planned))
		for _, p := range planned {
			PrintDryRun("Would adopt: %s", ContractPath(p.absPath))
			PrintDetail("Move to: %s", ContractPath(p.destPath))
			PrintDetail("Create symlink: %s -> %s", ContractPath(p.absPath), ContractPath(p.destPath))
		}
		fmt.Println()
		PrintDryRunSummary()
		return nil
	}

	// Phase 2: Execute with rollback
	type completedAdoption struct {
		absPath   string
		destPath  string
		moved     bool
		symlinked bool
	}
	var completed []completedAdoption
	var createdDirs []string

	rollback := func(originalErr error) error {
		var rollbackErrors []string
		for i := len(completed) - 1; i >= 0; i-- {
			c := completed[i]
			if c.symlinked {
				if err := os.Remove(c.absPath); err != nil {
					rollbackErrors = append(rollbackErrors, fmt.Sprintf("remove symlink %s: %v", ContractPath(c.absPath), err))
				}
			}
			if c.moved {
				if err := MoveFile(c.destPath, c.absPath); err != nil {
					rollbackErrors = append(rollbackErrors, fmt.Sprintf("restore %s: %v", ContractPath(c.absPath), err))
				}
			}
		}
		if len(createdDirs) > 0 {
			CleanEmptyDirs(createdDirs, absSourceDir)
		}
		if len(rollbackErrors) > 0 {
			return fmt.Errorf("adopt failed: %v; rollback failed: %s", originalErr, strings.Join(rollbackErrors, "; "))
		}
		return originalErr
	}

	for _, p := range planned {
		// Verify source still exists
		if _, err := os.Lstat(p.absPath); err != nil {
			return rollback(WithHint(
				NewPathError("adopt", p.absPath, err),
				"Check that the file path is correct and the file exists"))
		}

		// Create parent directory, tracking if newly created
		destDir := filepath.Dir(p.destPath)
		_, statErr := os.Stat(destDir)
		dirExisted := statErr == nil

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return rollback(NewPathError("adopt", destDir, fmt.Errorf("failed to create directory: %w", err)))
		}
		if !dirExisted {
			createdDirs = append(createdDirs, destDir)
		}

		c := completedAdoption{absPath: p.absPath, destPath: p.destPath}

		// Move file
		if err := MoveFile(p.absPath, p.destPath); err != nil {
			completed = append(completed, c)
			return rollback(err)
		}
		c.moved = true

		// Create symlink
		if err := CreateSymlink(p.destPath, p.absPath); err != nil {
			completed = append(completed, c)
			return rollback(err)
		}
		c.symlinked = true
		completed = append(completed, c)

		PrintSuccess("Adopted: %s", ContractPath(p.absPath))
	}

	PrintSummary("Adopted %d file(s) successfully", len(planned))
	PrintNextStep("status", "view adopted files")
	return nil
}

// collectAdoption validates a single file for adoption and adds it to the planned list.
// Returns an error immediately if validation fails (fail-fast).
func collectAdoption(absPath, absSourceDir, absTargetDir string, info os.FileInfo, seen map[string]bool, planned *[]plannedAdoption) error {
	// Deduplicate by absolute path
	if seen[absPath] {
		return nil
	}

	// Get file info if not provided (files from directory walk)
	if info == nil {
		var err error
		info, err = os.Lstat(absPath)
		if err != nil {
			return NewPathError("adopt", absPath, err)
		}
	}

	// Check already-adopted
	if err := validateAdoptSource(absPath, absSourceDir); err != nil {
		return err
	}

	// Check non-adopted symlink (caller's responsibility per spec)
	if info.Mode()&os.ModeSymlink != 0 {
		return NewPathErrorWithHint("adopt", absPath,
			fmt.Errorf("cannot adopt a symlink"),
			"Remove the symlink first, then adopt the target file")
	}

	// Check path is within target directory
	relPath, err := filepath.Rel(absTargetDir, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return WithHint(
			fmt.Errorf("path %s must be within target directory %s", ContractPath(absPath), ContractPath(absTargetDir)),
			"Only files within the target directory can be adopted")
	}

	// Compute destination
	destPath := filepath.Join(absSourceDir, relPath)

	// Check destination doesn't already exist
	if _, err := os.Stat(destPath); err == nil {
		return WithHint(
			fmt.Errorf("destination %s already exists", ContractPath(destPath)),
			"Remove the existing file first or choose a different file")
	}

	// Validate symlink creation (source=destPath, target=absPath per spec)
	if err := ValidateSymlinkCreation(destPath, absPath); err != nil {
		return err
	}

	seen[absPath] = true
	*planned = append(*planned, plannedAdoption{absPath: absPath, destPath: destPath})
	return nil
}
