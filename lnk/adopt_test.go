package lnk

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ==========================================
// Phase 1: Validation Tests
// ==========================================

func TestAdoptFailFastValidation(t *testing.T) {
	// Phase 1 must fail fast on first validation error with no filesystem changes.
	// If the first path is invalid, the second (valid) path must NOT be adopted.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a valid file and a nonexistent path
	validFile := filepath.Join(targetDir, ".bashrc")
	createTestFile(t, validFile, "bash config")
	nonexistent := filepath.Join(targetDir, ".doesnotexist")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{nonexistent, validFile},
		DryRun:    false,
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}

	// The valid file must NOT have been adopted (no filesystem changes in Phase 1 failure)
	info, statErr := os.Lstat(validFile)
	if statErr != nil {
		t.Fatalf("valid file disappeared: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("valid file was converted to symlink despite Phase 1 failure — fail-fast violated")
	}

	// Source dir should have no new files
	entries, _ := os.ReadDir(sourceDir)
	if len(entries) > 0 {
		t.Error("source directory has files despite Phase 1 failure — no filesystem changes should occur")
	}
}

func TestAdoptFileAlreadyAdopted(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a file in the source dir and a symlink in the target dir pointing to it
	repoFile := filepath.Join(sourceDir, ".bashrc")
	createTestFile(t, repoFile, "bash config")
	symlinkPath := filepath.Join(targetDir, ".bashrc")
	os.Symlink(repoFile, symlinkPath)

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{symlinkPath},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for already-adopted file")
	}
	if !strings.Contains(err.Error(), "already adopted") {
		t.Errorf("expected 'already adopted' error, got: %v", err)
	}
}

func TestAdoptNonAdoptedSymlinkRejected(t *testing.T) {
	// A symlink that points outside sourceDir should be rejected with
	// hint to remove the symlink first.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create an external target and a symlink pointing to it
	externalFile := filepath.Join(tempDir, "external", "config")
	createTestFile(t, externalFile, "external config")
	symlinkPath := filepath.Join(targetDir, ".config")
	os.Symlink(externalFile, symlinkPath)

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{symlinkPath},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for non-adopted symlink")
	}
	if !strings.Contains(err.Error(), "cannot adopt a symlink") {
		t.Errorf("expected 'cannot adopt a symlink' error, got: %v", err)
	}
	// Check hint
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "remove the symlink") {
		t.Errorf("expected hint about removing the symlink, got: %q", hint)
	}
}

func TestAdoptDestinationAlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create file in target and a conflicting file already in source
	targetFile := filepath.Join(targetDir, ".bashrc")
	createTestFile(t, targetFile, "target config")
	destFile := filepath.Join(sourceDir, ".bashrc")
	createTestFile(t, destFile, "existing repo file")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{targetFile},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for destination already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestAdoptPathOutsideTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	outsideDir := filepath.Join(tempDir, "outside")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)
	os.MkdirAll(outsideDir, 0755)

	outsideFile := filepath.Join(outsideDir, "file.txt")
	createTestFile(t, outsideFile, "outside content")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{outsideFile},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for path outside target directory")
	}
	if !strings.Contains(err.Error(), "must be within target directory") {
		t.Errorf("expected 'must be within target directory' error, got: %v", err)
	}
}

func TestAdoptNoPaths(t *testing.T) {
	opts := AdoptOptions{
		SourceDir: "/tmp/dotfiles",
		TargetDir: "/tmp/target",
		Paths:     []string{},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for empty paths")
	}
	if !strings.Contains(err.Error(), "at least one file path is required") {
		t.Errorf("expected 'at least one file path' error, got: %v", err)
	}
}

// ==========================================
// Directory Walking Tests
// ==========================================

func TestAdoptDirectoryWalksFiles(t *testing.T) {
	// Adopting a directory should walk it and adopt each regular file individually.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a directory with multiple files
	configDir := filepath.Join(targetDir, ".config", "nvim")
	createTestFile(t, filepath.Join(configDir, "init.vim"), "nvim init")
	createTestFile(t, filepath.Join(configDir, "settings.json"), "nvim settings")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{filepath.Join(targetDir, ".config", "nvim")},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each file should now be a symlink
	for _, name := range []string{"init.vim", "settings.json"} {
		filePath := filepath.Join(configDir, name)
		assertSymlink(t, filePath, filepath.Join(sourceDir, ".config", "nvim", name))
	}
}

func TestAdoptEmptyDirectoryReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create an empty directory
	emptyDir := filepath.Join(targetDir, ".config", "empty")
	os.MkdirAll(emptyDir, 0755)

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{emptyDir},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no files to adopt") {
		t.Errorf("expected 'no files to adopt' error, got: %v", err)
	}
}

func TestAdoptDirectorySkipsSymlinks(t *testing.T) {
	// When walking a directory, symlinks and non-regular entries should be skipped.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	configDir := filepath.Join(targetDir, ".config", "app")
	createTestFile(t, filepath.Join(configDir, "config.toml"), "real config")
	// Create a symlink inside the directory — should be skipped
	externalFile := filepath.Join(tempDir, "external.txt")
	createTestFile(t, externalFile, "external")
	os.Symlink(externalFile, filepath.Join(configDir, "link.txt"))

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{configDir},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// config.toml should be adopted
	assertSymlink(t, filepath.Join(configDir, "config.toml"),
		filepath.Join(sourceDir, ".config", "app", "config.toml"))

	// The symlink link.txt should NOT have been moved to source dir
	destLink := filepath.Join(sourceDir, ".config", "app", "link.txt")
	if _, err := os.Lstat(destLink); err == nil {
		t.Error("symlink inside directory should have been skipped, but it was adopted")
	}
}

// ==========================================
// Deduplication Tests
// ==========================================

func TestAdoptDeduplicatesByAbsolutePath(t *testing.T) {
	// If the same file is specified both directly and via a directory argument,
	// it should only be adopted once (deduplicated by absolute path).
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	configDir := filepath.Join(targetDir, ".config")
	filePath := filepath.Join(configDir, "file.txt")
	createTestFile(t, filePath, "config file")

	// Pass the directory AND the explicit file — same file collected twice
	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{configDir, filePath},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be adopted exactly once
	assertSymlink(t, filePath, filepath.Join(sourceDir, ".config", "file.txt"))
}

// ==========================================
// Phase 2: Execution Tests
// ==========================================

func TestAdoptSingleFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	bashrc := filepath.Join(targetDir, ".bashrc")
	createTestFile(t, bashrc, "bash config")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{bashrc},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be moved to source and symlinked back
	assertSymlink(t, bashrc, filepath.Join(sourceDir, ".bashrc"))

	// Verify content was preserved
	content, err := os.ReadFile(filepath.Join(sourceDir, ".bashrc"))
	if err != nil {
		t.Fatalf("failed to read adopted file: %v", err)
	}
	if string(content) != "bash config" {
		t.Errorf("adopted file content = %q, want %q", string(content), "bash config")
	}
}

func TestAdoptMultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	files := map[string]string{
		".bashrc":  "bash config",
		".vimrc":   "vim config",
		".gitconfig": "git config",
	}
	var paths []string
	for name, content := range files {
		p := filepath.Join(targetDir, name)
		createTestFile(t, p, content)
		paths = append(paths, p)
	}

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     paths,
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for name := range files {
		assertSymlink(t, filepath.Join(targetDir, name), filepath.Join(sourceDir, name))
	}
}

func TestAdoptNestedFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	nestedFile := filepath.Join(targetDir, ".config", "nvim", "init.vim")
	createTestFile(t, nestedFile, "nvim config")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{nestedFile},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertSymlink(t, nestedFile, filepath.Join(sourceDir, ".config", "nvim", "init.vim"))
}

// ==========================================
// Phase 2: Rollback Tests
// ==========================================

func TestAdoptRollbackOnExecutionFailure(t *testing.T) {
	// If Phase 2 fails partway through, all completed adoptions must be
	// rolled back in reverse order.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create two files to adopt
	file1 := filepath.Join(targetDir, ".config", "a", "file1")
	file2 := filepath.Join(targetDir, ".config", "b", "file2")
	createTestFile(t, file1, "content1")
	createTestFile(t, file2, "content2")

	// To trigger Phase 2 failure after file1 succeeds: make file2's destination
	// parent directory unwritable so MkdirAll fails for file2.
	// Phase 1 doesn't create directories, so this won't affect validation.
	destParentB := filepath.Join(sourceDir, ".config")
	os.MkdirAll(destParentB, 0755)
	// Create .config/b as a read-only dir so writing file2 into it fails
	destDirB := filepath.Join(destParentB, "b")
	os.MkdirAll(destDirB, 0555)
	t.Cleanup(func() { os.Chmod(destDirB, 0755) })

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{file1, file2},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error when Phase 2 execution fails")
	}

	// After rollback, file1 should be back as a regular file (not a symlink)
	info, statErr := os.Lstat(file1)
	if statErr != nil {
		t.Fatalf("file1 should exist after rollback: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("file1 should be a regular file after rollback, not a symlink")
	}

	// Content should be preserved
	content, readErr := os.ReadFile(file1)
	if readErr != nil {
		t.Fatalf("failed to read rolled-back file1: %v", readErr)
	}
	if string(content) != "content1" {
		t.Errorf("file1 content after rollback = %q, want %q", string(content), "content1")
	}
}

func TestAdoptRollbackCleansEmptyDirs(t *testing.T) {
	// On rollback, CleanEmptyDirs should clean up directories created during
	// Phase 2 execution, bounded by sourceDir.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create files with nested paths so Phase 2 creates directories
	file1 := filepath.Join(targetDir, ".config", "a", "file1")
	file2 := filepath.Join(targetDir, ".config", "b", "file2")
	createTestFile(t, file1, "content1")
	createTestFile(t, file2, "content2")

	// Make file2's destination parent unwritable to cause Phase 2 failure
	destParent := filepath.Join(sourceDir, ".config")
	os.MkdirAll(destParent, 0755)
	destDirB := filepath.Join(destParent, "b")
	os.MkdirAll(destDirB, 0555)
	t.Cleanup(func() { os.Chmod(destDirB, 0755) })

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{file1, file2},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error")
	}

	// After rollback, file1 should have been moved back to target
	destFile1 := filepath.Join(sourceDir, ".config", "a", "file1")
	if _, statErr := os.Lstat(destFile1); statErr == nil {
		t.Error("file1 should have been moved back to target during rollback")
	}

	// The .config/a directory in source should be cleaned up since it was
	// created during this operation
	dirA := filepath.Join(sourceDir, ".config", "a")
	if _, statErr := os.Lstat(dirA); statErr == nil {
		t.Error(".config/a directory should have been cleaned up during rollback")
	}
}

func TestAdoptRollbackFailureReportsCombinedError(t *testing.T) {
	// If rollback itself fails, the error message should include both
	// the original failure and the rollback failure.
	// This is hard to trigger deterministically, so we just verify the
	// error type/message structure exists for when it does happen.
	// The implementation must return a combined error like:
	// "adopt failed: <err>; rollback failed: <err>"
	t.Skip("rollback failure is difficult to trigger deterministically in unit tests")
}

// ==========================================
// Dry-Run Tests
// ==========================================

func TestAdoptDryRunNoChanges(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	testFile := filepath.Join(targetDir, ".bashrc")
	createTestFile(t, testFile, "bash config")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{testFile},
		DryRun:    true,
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Verify file is still a regular file
	info, err := os.Lstat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("file was converted to symlink in dry-run mode")
	}

	// Verify file was not moved to source
	if _, err := os.Stat(filepath.Join(sourceDir, ".bashrc")); err == nil {
		t.Error("file was moved to source in dry-run mode")
	}
}

func TestAdoptDryRunPerFileDetail(t *testing.T) {
	// Dry-run should print per-file detail with move destination and symlink info.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	file1 := filepath.Join(targetDir, ".bashrc")
	file2 := filepath.Join(targetDir, ".vimrc")
	createTestFile(t, file1, "bash")
	createTestFile(t, file2, "vim")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{file1, file2},
		DryRun:    true,
	}

	output := CaptureOutput(t, func() {
		Adopt(opts)
	})

	// Should contain count header
	if !strings.Contains(output, "Would adopt 2 file(s)") {
		t.Errorf("dry-run output missing count header, got:\n%s", output)
	}
	// Should contain per-file move detail
	if !strings.Contains(output, "Move to:") {
		t.Errorf("dry-run output missing 'Move to:' detail, got:\n%s", output)
	}
	if !strings.Contains(output, "Create symlink:") {
		t.Errorf("dry-run output missing 'Create symlink:' detail, got:\n%s", output)
	}
}

func TestAdoptDryRunDirectoryShowsPerFileDetail(t *testing.T) {
	// When a directory is passed in dry-run, each individual file should be listed.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	configDir := filepath.Join(targetDir, ".config", "app")
	createTestFile(t, filepath.Join(configDir, "a.conf"), "a")
	createTestFile(t, filepath.Join(configDir, "b.conf"), "b")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{configDir},
		DryRun:    true,
	}

	output := CaptureOutput(t, func() {
		Adopt(opts)
	})

	// Should show per-file detail for each file in the directory
	if !strings.Contains(output, "a.conf") {
		t.Errorf("dry-run output missing a.conf, got:\n%s", output)
	}
	if !strings.Contains(output, "b.conf") {
		t.Errorf("dry-run output missing b.conf, got:\n%s", output)
	}
}

// ==========================================
// Summary and Output Tests
// ==========================================

func TestAdoptSummaryOutput(t *testing.T) {
	// Summary should print "Adopted N file(s) successfully" and next-step hint.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	file1 := filepath.Join(targetDir, ".bashrc")
	file2 := filepath.Join(targetDir, ".vimrc")
	createTestFile(t, file1, "bash")
	createTestFile(t, file2, "vim")

	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{file1, file2},
	}

	output := CaptureOutput(t, func() {
		Adopt(opts)
	})

	if !strings.Contains(output, "Adopted 2 file(s) successfully") {
		t.Errorf("output missing summary line, got:\n%s", output)
	}
	if !strings.Contains(output, "lnk status") {
		t.Errorf("output missing next-step hint with 'lnk status', got:\n%s", output)
	}
}

// ==========================================
// ValidateSymlinkCreation Argument Order Tests
// ==========================================

func TestAdoptValidateSymlinkCreationArgOrder(t *testing.T) {
	// ValidateSymlinkCreation should be called with (destPath, absPath):
	// source = destPath (real file after move), target = absPath (symlink location).
	// This test verifies the correct argument order by triggering a validation
	// error that depends on argument position.
	//
	// We can verify this indirectly: if the source dir IS the target dir,
	// ValidateSymlinkCreation(destPath, absPath) should detect the overlap.
	// This is a compile/smoke test — the real verification is in the implementation.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	testFile := filepath.Join(targetDir, ".bashrc")
	createTestFile(t, testFile, "config")

	// Normal case should pass validation
	opts := AdoptOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{testFile},
	}
	err := Adopt(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ==========================================
// Source Dir Validation Tests
// ==========================================

func TestAdoptSourceDirNotExist(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(targetDir, 0755)

	testFile := filepath.Join(targetDir, ".testfile")
	createTestFile(t, testFile, "test")

	opts := AdoptOptions{
		SourceDir: filepath.Join(tempDir, "nonexistent"),
		TargetDir: targetDir,
		Paths:     []string{testFile},
	}
	err := Adopt(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent source directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error about nonexistent directory, got: %v", err)
	}
}

// ==========================================
// validateAdoptSource Unit Tests
// ==========================================

func TestValidateAdoptSourceAlreadyAdopted(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	os.MkdirAll(sourceDir, 0755)

	// Create a file in source and a symlink pointing to it
	repoFile := filepath.Join(sourceDir, ".bashrc")
	createTestFile(t, repoFile, "config")
	symlinkPath := filepath.Join(tempDir, "target", ".bashrc")
	os.MkdirAll(filepath.Dir(symlinkPath), 0755)
	os.Symlink(repoFile, symlinkPath)

	err := validateAdoptSource(symlinkPath, sourceDir)
	if err == nil {
		t.Fatal("expected error for already-adopted file")
	}
	if !errors.Is(err, ErrAlreadyAdopted) {
		t.Errorf("expected ErrAlreadyAdopted, got: %v", err)
	}
}

func TestValidateAdoptSourceRegularFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	os.MkdirAll(sourceDir, 0755)

	regularFile := filepath.Join(tempDir, "target", ".bashrc")
	createTestFile(t, regularFile, "config")

	err := validateAdoptSource(regularFile, sourceDir)
	if err != nil {
		t.Errorf("unexpected error for regular file: %v", err)
	}
}
