---
status: test
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

- ...

### Implementation

- ...
