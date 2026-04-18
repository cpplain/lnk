---
status: completed
---

# Task: Add `CleanEmptyDirs` call to `prune` after symlink removal

## Behavior

After pruning broken symlinks, `Prune` must call `CleanEmptyDirs` on the parent
directories of all successfully pruned symlinks, with `targetDir` as the
boundary. This matches `docs/design/features/prune.md` section 5, step 3
(Execute Mode, "After all links are processed" block).

References:
- `docs/design/features/prune.md` §5 Execute Mode — `CleanEmptyDirs` call
- `docs/design/internals.md` §7 — `CleanEmptyDirs` behavior

## Acceptance Criteria

- `Prune` calls `CleanEmptyDirs` with parent directories of successfully pruned
  symlinks and `targetDir` as the boundary
- Empty parent directories are removed after pruning
- Non-empty parent directories are preserved
- `targetDir` itself is never removed (boundary behavior)
- All tests pass, `make check` passes

## Context

**Current implementation** (`lnk/prune.go`): Removes broken symlinks but does
not call `CleanEmptyDirs` afterward. Empty directories are left behind.

**Correct pattern**: Follow `remove.go` lines 98-113 — track parent directories
of successfully removed symlinks, then call `CleanEmptyDirs(removedParents,
targetDir)` after the removal loop.

**Related functions**:
- `CleanEmptyDirs` in `lnk/file_ops.go` — already implemented
- `remove.go` — reference implementation of the same pattern

## Log

### Planning

- This is a small, focused task: add `CleanEmptyDirs` call and track parent dirs
- The pattern is identical to what `remove.go` already does (lines 98-113)
- Existing test "prune broken links in subdirectories" sets up the right
  scenario but doesn't assert directory cleanup — new test needed
- Status: `test` because the behavior (directory cleanup) is testable

### Testing

- Added two test cases to `lnk/prune_test.go`:
  - "empty parent directories cleaned after pruning" — broken symlink in nested
    dir `~/.config/app/settings.conf`; after pruning, asserts both `.config/app`
    and `.config` are removed, but `targetDir` is preserved
  - "non-empty parent directories preserved after pruning" — broken symlink
    alongside an unmanaged file; asserts parent dir is kept
- Both compile cleanly; the empty-dir test fails as expected (no implementation)

### Implementation

- Added `path/filepath` import to `prune.go`
- Added `removedParents` slice to track parent dirs of successfully pruned links
- Appended `filepath.Dir(link.Path)` on each successful prune
- Called `CleanEmptyDirs(removedParents, targetDir)` after the removal loop
- Pattern matches `remove.go` lines 98-113 exactly
- All tests pass including both new directory-cleanup test cases
