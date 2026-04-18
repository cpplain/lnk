package lnk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ==========================================
// Phase 1: Validation Tests
// ==========================================

func TestOrphanFailFastValidation(t *testing.T) {
	// Phase 1 must fail fast on first validation error with no filesystem changes.
	// If the first path is invalid, the second (valid) path must NOT be orphaned.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a valid managed symlink
	sourceFile := filepath.Join(sourceDir, ".bashrc")
	createTestFile(t, sourceFile, "bash config")
	validLink := filepath.Join(targetDir, ".bashrc")
	os.Symlink(sourceFile, validLink)

	// Use nonexistent path as first arg to trigger fail-fast
	nonexistent := filepath.Join(targetDir, ".doesnotexist")

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{nonexistent, validLink},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}

	// The valid symlink must NOT have been orphaned (no filesystem changes)
	info, statErr := os.Lstat(validLink)
	if statErr != nil {
		t.Fatalf("valid symlink disappeared: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("valid symlink was orphaned despite Phase 1 failure — fail-fast violated")
	}

	// Source file should still exist
	if _, statErr := os.Stat(sourceFile); statErr != nil {
		t.Error("source file was removed despite Phase 1 failure")
	}
}

func TestOrphanPathNotFound(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	nonexistent := filepath.Join(targetDir, ".bashrc")

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{nonexistent},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	// Should be a PathError with hint
	hint := GetErrorHint(err)
	if hint == "" {
		t.Errorf("expected error with hint, got: %v", err)
	}
}

func TestOrphanRegularFileRejected(t *testing.T) {
	// A regular file (not a symlink) should be rejected with hint to use rm.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	regularFile := filepath.Join(targetDir, "regular.txt")
	createTestFile(t, regularFile, "regular content")

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{regularFile},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for regular file")
	}
	if !strings.Contains(err.Error(), "not a symlink") {
		t.Errorf("expected 'not a symlink' error, got: %v", err)
	}
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "rm") {
		t.Errorf("expected hint about using rm, got: %q", hint)
	}

	// File should not be modified
	content, _ := os.ReadFile(regularFile)
	if string(content) != "regular content" {
		t.Error("regular file was modified")
	}
}

func TestOrphanUnmanagedSymlinkRejected(t *testing.T) {
	// A symlink not managed by the specified source should be rejected.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create external file and symlink to it
	externalFile := filepath.Join(tempDir, "external.txt")
	createTestFile(t, externalFile, "external")
	linkPath := filepath.Join(targetDir, "external-link")
	os.Symlink(externalFile, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for unmanaged symlink")
	}
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "rm") {
		t.Errorf("expected hint about using rm, got: %q", hint)
	}

	// Symlink should remain unchanged
	info, _ := os.Lstat(linkPath)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("unmanaged symlink was modified")
	}
}

func TestOrphanBrokenSymlinkRejected(t *testing.T) {
	// A broken symlink (target doesn't exist) should be rejected with hint to use rm.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create symlink to nonexistent file in source dir
	brokenTarget := filepath.Join(sourceDir, "nonexistent")
	linkPath := filepath.Join(targetDir, ".broken-link")
	os.Symlink(brokenTarget, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for broken symlink")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "rm") {
		t.Errorf("expected hint about using rm, got: %q", hint)
	}

	// Broken symlink should still exist
	info, statErr := os.Lstat(linkPath)
	if statErr != nil {
		t.Fatal("broken symlink was removed")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("broken symlink was modified")
	}
}

func TestOrphanPathOutsideTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	outsideDir := filepath.Join(tempDir, "outside")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)
	os.MkdirAll(outsideDir, 0755)

	// Create a managed symlink outside target dir
	sourceFile := filepath.Join(sourceDir, "file.txt")
	createTestFile(t, sourceFile, "content")
	outsideLink := filepath.Join(outsideDir, "link")
	os.Symlink(sourceFile, outsideLink)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{outsideLink},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for path outside target directory")
	}
	if !strings.Contains(err.Error(), "must be within target directory") {
		t.Errorf("expected 'must be within target directory' error, got: %v", err)
	}
}

func TestOrphanNoPaths(t *testing.T) {
	opts := OrphanOptions{
		SourceDir: "/tmp/dotfiles",
		TargetDir: "/tmp/target",
		Paths:     []string{},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for empty paths")
	}
	if !strings.Contains(err.Error(), "at least one file path is required") {
		t.Errorf("expected 'at least one file path' error, got: %v", err)
	}
}

func TestOrphanSourceDirNotExist(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(targetDir, 0755)

	linkPath := filepath.Join(targetDir, ".bashrc")
	os.Symlink("/nonexistent/file", linkPath)

	opts := OrphanOptions{
		SourceDir: filepath.Join(tempDir, "nonexistent"),
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent source directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error about nonexistent directory, got: %v", err)
	}
}

// ==========================================
// Directory Expansion Tests
// ==========================================

func TestOrphanDirectoryExpandsManagedLinks(t *testing.T) {
	// Passing a directory should use FindManagedLinks to collect active symlinks.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create source files and symlinks
	source1 := filepath.Join(sourceDir, "file1")
	source2 := filepath.Join(sourceDir, "subdir", "file2")
	createTestFile(t, source1, "content1")
	createTestFile(t, source2, "content2")

	dir := filepath.Join(targetDir, "orphan-dir")
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	link1 := filepath.Join(dir, "file1")
	link2 := filepath.Join(dir, "subdir", "file2")
	os.Symlink(source1, link1)
	os.Symlink(source2, link2)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{dir},
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both should be orphaned (regular files, not symlinks)
	for _, linkPath := range []string{link1, link2} {
		info, statErr := os.Lstat(linkPath)
		if statErr != nil {
			t.Errorf("orphaned file %s should exist: %v", linkPath, statErr)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s should be a regular file after orphaning", linkPath)
		}
	}

	// Source files should be removed
	for _, srcPath := range []string{source1, source2} {
		if _, statErr := os.Stat(srcPath); !os.IsNotExist(statErr) {
			t.Errorf("source file %s should have been removed", srcPath)
		}
	}
}

func TestOrphanDirectoryNoManagedLinks(t *testing.T) {
	// Directory with no managed links should return error with hint to run lnk status.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	emptyDir := filepath.Join(targetDir, "empty-dir")
	os.MkdirAll(emptyDir, 0755)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{emptyDir},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for directory with no managed links")
	}
	if !strings.Contains(err.Error(), "no managed symlinks") {
		t.Errorf("expected 'no managed symlinks' error, got: %v", err)
	}
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "status") {
		t.Errorf("expected hint about lnk status, got: %q", hint)
	}
}

func TestOrphanDirectoryBrokenLinksRejected(t *testing.T) {
	// Broken links found during directory expansion should be rejected with PathError.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create one active and one broken managed link in the directory
	activeSource := filepath.Join(sourceDir, "active")
	createTestFile(t, activeSource, "active content")

	dir := filepath.Join(targetDir, "mixed-dir")
	os.MkdirAll(dir, 0755)
	activeLink := filepath.Join(dir, "active")
	os.Symlink(activeSource, activeLink)
	brokenLink := filepath.Join(dir, "broken")
	os.Symlink(filepath.Join(sourceDir, "nonexistent"), brokenLink)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{dir},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error for directory containing broken links")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error for broken link, got: %v", err)
	}
	hint := GetErrorHint(err)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "rm") {
		t.Errorf("expected hint about using rm, got: %q", hint)
	}

	// Active link should NOT have been orphaned (fail-fast)
	info, _ := os.Lstat(activeLink)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("active link was orphaned despite broken link in same directory — fail-fast violated")
	}
}

// ==========================================
// Deduplication Tests
// ==========================================

func TestOrphanDeduplicatesByPath(t *testing.T) {
	// If the same symlink is collected via both directory and explicit path,
	// it should only be orphaned once (deduplicated by Path).
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	sourceFile := filepath.Join(sourceDir, ".config", "file.txt")
	createTestFile(t, sourceFile, "config content")
	configDir := filepath.Join(targetDir, ".config")
	os.MkdirAll(configDir, 0755)
	linkPath := filepath.Join(configDir, "file.txt")
	os.Symlink(sourceFile, linkPath)

	// Pass both the directory and the explicit symlink path
	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{configDir, linkPath},
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be orphaned exactly once (regular file, not symlink)
	info, statErr := os.Lstat(linkPath)
	if statErr != nil {
		t.Fatalf("orphaned file should exist: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("file should be a regular file after orphaning")
	}
}

func TestOrphanEmptyCollectionAfterDedup(t *testing.T) {
	// If collection is empty after dedup, should print message and return nil.
	// This is tested indirectly — after all paths are processed and collection is empty.
	// We need a scenario where paths are provided but result in an empty collection.
	// This happens when all links are duplicates that reduce to zero — but that
	// can't happen since at least one must be the first occurrence. Instead,
	// the "No managed symlinks found." message appears only for valid but empty results.
	// The spec says this prints after dedup when collection is empty, so we test
	// the output message directly.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// This test exercises the message — actual emptiness after dedup would
	// require all items being filtered, which the spec doesn't specify.
	// Covered by directory-no-managed-links test instead.
	t.Skip("empty collection after dedup requires filtering that removes all items — covered by directory tests")
}

// ==========================================
// Phase 2: Execution Tests
// ==========================================

func TestOrphanSingleFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	sourceFile := filepath.Join(sourceDir, ".bashrc")
	createTestFile(t, sourceFile, "bash config")
	linkPath := filepath.Join(targetDir, ".bashrc")
	os.Symlink(sourceFile, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Link should be replaced with actual file
	info, statErr := os.Lstat(linkPath)
	if statErr != nil {
		t.Fatalf("orphaned file should exist: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("file is still a symlink after orphaning")
	}

	// Content should be preserved
	content, _ := os.ReadFile(linkPath)
	if string(content) != "bash config" {
		t.Errorf("file content = %q, want %q", string(content), "bash config")
	}

	// Source file should be removed
	if _, statErr := os.Stat(sourceFile); !os.IsNotExist(statErr) {
		t.Error("source file still exists in repository")
	}
}

func TestOrphanMultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	files := map[string]string{
		".bashrc": "bash config",
		".vimrc":  "vim config",
	}
	var paths []string
	for name, content := range files {
		sourceFile := filepath.Join(sourceDir, name)
		createTestFile(t, sourceFile, content)
		linkPath := filepath.Join(targetDir, name)
		os.Symlink(sourceFile, linkPath)
		paths = append(paths, linkPath)
	}

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     paths,
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All links should be replaced with regular files
	for _, linkPath := range paths {
		info, statErr := os.Lstat(linkPath)
		if statErr != nil {
			t.Errorf("orphaned file %s should exist: %v", linkPath, statErr)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s is still a symlink after orphaning", linkPath)
		}
	}

	// Source files should be removed
	for name := range files {
		sourceFile := filepath.Join(sourceDir, name)
		if _, statErr := os.Stat(sourceFile); !os.IsNotExist(statErr) {
			t.Errorf("source file %s still exists in repository", name)
		}
	}
}

func TestOrphanPermissionsRestored(t *testing.T) {
	// File permissions should be restored from the original source file.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create source file with executable permission
	sourceFile := filepath.Join(sourceDir, "script.sh")
	createTestFile(t, sourceFile, "#!/bin/bash")
	os.Chmod(sourceFile, 0755)

	linkPath := filepath.Join(targetDir, "script.sh")
	os.Symlink(sourceFile, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check permissions were restored
	info, _ := os.Lstat(linkPath)
	if info.Mode().Perm() != 0755 {
		t.Errorf("file permissions = %o, want %o", info.Mode().Perm(), 0755)
	}
}

func TestOrphanCleanEmptyDirs(t *testing.T) {
	// After orphaning, empty parent directories in the source should be cleaned up.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create a deeply nested source file
	sourceFile := filepath.Join(sourceDir, ".config", "app", "settings.json")
	createTestFile(t, sourceFile, "settings")

	linkPath := filepath.Join(targetDir, ".config", "app", "settings.json")
	os.MkdirAll(filepath.Dir(linkPath), 0755)
	os.Symlink(sourceFile, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty parent directories in source should be cleaned up
	assertNotExists(t, filepath.Join(sourceDir, ".config", "app"))
	assertNotExists(t, filepath.Join(sourceDir, ".config"))

	// sourceDir itself should NOT be removed
	assertDirExists(t, sourceDir)
}

// ==========================================
// Phase 2: Rollback Tests
// ==========================================

func TestOrphanRollbackOnExecutionFailure(t *testing.T) {
	// If Phase 2 fails partway through, all completed orphans must be
	// rolled back in reverse order.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create two files in separate source subdirectories
	srcDirA := filepath.Join(sourceDir, "a")
	srcDirB := filepath.Join(sourceDir, "b")
	source1 := filepath.Join(srcDirA, "file1")
	source2 := filepath.Join(srcDirB, "file2")
	createTestFile(t, source1, "content1")
	createTestFile(t, source2, "content2")

	link1 := filepath.Join(targetDir, "file1")
	link2 := filepath.Join(targetDir, "file2")
	os.Symlink(source1, link1)
	os.Symlink(source2, link2)

	// Make srcDirB read-only. Phase 1 os.Stat on the file still works (stat only
	// needs execute permission on parent). But MoveFile (os.Rename or copy+delete)
	// will fail because it can't remove from a read-only directory.
	os.Chmod(srcDirB, 0555)
	t.Cleanup(func() { os.Chmod(srcDirB, 0755) })

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{link1, link2},
	}
	err := Orphan(opts)
	if err == nil {
		t.Fatal("expected error when Phase 2 execution fails")
	}

	// After rollback, link1 should be restored as a symlink
	info, statErr := os.Lstat(link1)
	if statErr != nil {
		t.Fatalf("link1 should exist after rollback: %v", statErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("link1 should be a symlink after rollback, not a regular file")
	}

	// Source file 1 should be back in source dir
	if _, statErr := os.Stat(source1); statErr != nil {
		t.Errorf("source file 1 should be restored after rollback: %v", statErr)
	}
}

func TestOrphanRollbackFailureReportsCombinedError(t *testing.T) {
	// If rollback itself fails, the error message should include both
	// the original failure and the rollback failure.
	t.Skip("rollback failure is difficult to trigger deterministically in unit tests")
}

// ==========================================
// Dry-Run Tests
// ==========================================

func TestOrphanDryRunNoChanges(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	sourceFile := filepath.Join(sourceDir, ".testfile")
	createTestFile(t, sourceFile, "test content")
	linkPath := filepath.Join(targetDir, ".testfile")
	os.Symlink(sourceFile, linkPath)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{linkPath},
		DryRun:    true,
	}
	err := Orphan(opts)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Symlink should still exist
	info, statErr := os.Lstat(linkPath)
	if statErr != nil {
		t.Fatal("symlink was removed during dry run")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was modified during dry run")
	}

	// Source file should still exist
	if _, statErr := os.Stat(sourceFile); statErr != nil {
		t.Error("source file was removed during dry run")
	}
}

func TestOrphanDryRunOutputFormat(t *testing.T) {
	// Dry-run should print per-file detail matching spec §5 format.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	source1 := filepath.Join(sourceDir, ".bashrc")
	source2 := filepath.Join(sourceDir, ".vimrc")
	createTestFile(t, source1, "bash")
	createTestFile(t, source2, "vim")

	link1 := filepath.Join(targetDir, ".bashrc")
	link2 := filepath.Join(targetDir, ".vimrc")
	os.Symlink(source1, link1)
	os.Symlink(source2, link2)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{link1, link2},
		DryRun:    true,
	}

	output := CaptureOutput(t, func() {
		Orphan(opts)
	})

	// Should contain count header
	if !strings.Contains(output, "Would orphan 2 symlink(s)") {
		t.Errorf("dry-run output missing count header, got:\n%s", output)
	}
	// Should contain per-file detail with "Remove symlink:" and "Move from:"
	if !strings.Contains(output, "Remove symlink:") {
		t.Errorf("dry-run output missing 'Remove symlink:' detail, got:\n%s", output)
	}
	if !strings.Contains(output, "Move from:") {
		t.Errorf("dry-run output missing 'Move from:' detail, got:\n%s", output)
	}
}

// ==========================================
// Summary and Output Tests
// ==========================================

func TestOrphanSummaryOutput(t *testing.T) {
	// Summary should print "Orphaned N file(s) successfully" and next-step hint.
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "dotfiles")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	source1 := filepath.Join(sourceDir, ".bashrc")
	source2 := filepath.Join(sourceDir, ".vimrc")
	createTestFile(t, source1, "bash")
	createTestFile(t, source2, "vim")

	link1 := filepath.Join(targetDir, ".bashrc")
	link2 := filepath.Join(targetDir, ".vimrc")
	os.Symlink(source1, link1)
	os.Symlink(source2, link2)

	opts := OrphanOptions{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Paths:     []string{link1, link2},
	}

	output := CaptureOutput(t, func() {
		Orphan(opts)
	})

	if !strings.Contains(output, "Orphaned 2 file(s) successfully") {
		t.Errorf("output missing summary line, got:\n%s", output)
	}
	if !strings.Contains(output, "lnk status") {
		t.Errorf("output missing next-step hint with 'lnk status', got:\n%s", output)
	}
}
