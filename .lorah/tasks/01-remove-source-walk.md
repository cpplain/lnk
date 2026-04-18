---
status: completed
---

# Task: Fix `remove` to use source-walk traversal and call `CleanEmptyDirs`

## Behavior

`RemoveLinks` must walk `SourceDir` (not `TargetDir`) to find managed symlinks,
matching `docs/design/features/remove.md` section 5. For each source file, it
computes the expected symlink path in `TargetDir`, checks if it's a symlink
pointing to the source file, and removes it if so.

After removal, it must call `CleanEmptyDirs` on parent directories of removed
symlinks with `targetDir` as the boundary.

References:
- `docs/design/features/remove.md` section 5, steps 1-2
- `docs/design/features/remove.md` section 6 (managed link detection)

## Acceptance Criteria

- `remove` walks `SourceDir` using `filepath.WalkDir` (not `FindManagedLinks`)
- For each source file, the corresponding target path is checked: if it's a
  symlink pointing to the source file, it's removed; otherwise skipped
- `CleanEmptyDirs` is called on parent directories of successfully removed
  symlinks, with `targetDir` as the boundary
- Broken symlinks from deleted source files are NOT found (by design of
  source-walk — only current source files are checked)
- Walk errors abort immediately (source dirs should be fully readable)
- Existing tests updated to match source-walk behavior

## Context

**Current implementation** (`lnk/remove.go`): Uses `FindManagedLinks(targetDir,
[]string{sourceDir})` which is a target-walk strategy. This is wrong per spec.

**Correct pattern**: Follow `create.go`'s `collectPlannedLinksWithPatterns` —
walk source dir with `filepath.Walk` (needs update to `filepath.WalkDir`), compute
target paths, but instead of creating symlinks, check if target is a managed
symlink and remove it.

**Related functions**:
- `RemoveSymlink` in `lnk/symlink.go` — already exists, use as-is
- `CleanEmptyDirs` in `lnk/file_ops.go` — already exists, use as-is
- `filepath.EvalSymlinks` — for verifying symlink targets (see spec section 6)

**Note**: This task does NOT fix `PrintWarningWithHint` (doesn't exist yet) or
`PrintNextStep` signature — those are separate tasks. Use existing error output
functions for now.

## Log

### Planning

- This is the highest-priority fix: `remove` uses fundamentally wrong traversal
  strategy (target-walk vs source-walk)
- Bundling `CleanEmptyDirs` call because it's part of the same execution flow
  and both are critical acceptance criteria
- NOT bundling `PrintWarningWithHint` or `PrintNextStep` changes — those are
  output system tasks that affect multiple commands

### Testing

- Updated `lnk/remove_test.go` with tests for source-walk behavior
- Added `CleanEmptyDirs` implementation in `lnk/file_ops.go` (needed for tests
  to verify cleanup; implemented per `docs/design/internals.md` §7)
- New test cases added:
  - `skips non-managed symlinks in target` — only managed links removed
  - `skips target path that is a regular file` — regular files untouched
  - `broken symlinks from deleted source files are not found by source walk` —
    validates source-walk only finds current source files (FAILS: expected)
  - `empty parent directories cleaned after removal` — validates CleanEmptyDirs
    is called (FAILS: expected)
  - `non-empty parent directories preserved after removal` — boundary behavior
- Expected failures: 2 tests fail against current target-walk implementation,
  will pass after source-walk implementation

### Implementation

- Replaced `FindManagedLinks` (target-walk) with `collectManagedLinks` (source-walk)
- `collectManagedLinks` uses `filepath.WalkDir` on `SourceDir`, computes expected
  target paths, checks each with `os.Lstat` + `filepath.EvalSymlinks` for managed
  link detection (per spec §6)
- Resolves `sourceDir` via `filepath.EvalSymlinks` before comparison to handle
  OS-level symlinks (macOS `/var` -> `/private/var`)
- Added `CleanEmptyDirs` call on parent directories of removed symlinks with
  `targetDir` as boundary
- All 10 tests pass, `make check` passes
