---
status: test
---

# Task: Rewrite `adopt` to use two-phase transactional execution

## Behavior

`Adopt` must use two-phase execution matching `docs/design/features/adopt.md` §5:

- **Phase 1 (Collect and Validate)**: For each path, expand, lstat, handle
  directories (walk and collect regular files), validate via
  `validateAdoptSource`, check path is within `TargetDir`, compute destination,
  check destination doesn't exist, validate via `ValidateSymlinkCreation`.
  Fail-fast: if any validation fails, return the error immediately with no
  filesystem changes. After collecting, deduplicate by absolute path.

- **Dry-run**: If `DryRun`, print planned adoptions and return.

- **Phase 2 (Execute with rollback)**: For each planned adoption: verify source
  still exists, `MkdirAll` parent of destination (track newly created dirs),
  `MoveFile`, `CreateSymlink`. If any step fails: roll back ALL completed
  adoptions in reverse order (remove symlink, move file back), call
  `CleanEmptyDirs` on parents of rolled-back destinations (only dirs created
  during this operation), return error.

Current code uses continue-on-failure (prints errors, increments counter,
continues to next path). The `performAdoption` and `performDirectoryAdoption`
helper functions also need replacement — they mix validation with execution
and have per-item rollback instead of batch rollback.

Also: `validateAdoptSource` currently calls `os.Lstat` internally, but per the
spec, the caller does `os.Lstat` in step 2 and passes the result context. The
spec's step 4 calls `validateAdoptSource` to check already-adopted status, then
separately checks if the path is a non-adopted symlink. The function should only
check for already-adopted symlinks; the non-adopted-symlink check belongs in the
caller using the lstat result from step 2.

## Acceptance Criteria

- Phase 1 validates ALL paths before any filesystem changes
- Phase 1 fails fast on first validation error (returns error, no changes)
- Directory arguments are walked and individual regular files collected
- Empty directory argument returns error `"no files to adopt in <path>"`
- Non-adopted symlinks rejected with hint to remove symlink first
- Paths deduplicated by absolute path after collection
- Phase 2 executes moves and symlinks for all collected files
- Phase 2 failure triggers reverse-order rollback of all completed adoptions
- Rollback calls `CleanEmptyDirs` on parents of destinations, bounded by sourceDir,
  only for directories created during this operation
- Rollback failure produces combined error message
- Dry-run prints per-file detail with move destination and symlink info
- Summary prints `"Adopted N file(s) successfully"` and next-step hint
- `ValidateSymlinkCreation` called with `(destPath, absPath)` — source is the
  destination (real file after move), target is the original path (symlink location)

## Context

- Current implementation: `lnk/adopt.go` — uses continue-on-failure model
- `validateAdoptSource` at line 19 — needs adjustment per spec step 4
- `performAdoption` and `performDirectoryAdoption` — to be replaced
- Spec: `docs/design/features/adopt.md` §5
- Tests: `lnk/adopt_test.go` — existing tests will need updating to match
  two-phase behavior (e.g., errors return immediately, no partial adoption)
- Related: `lnk/orphan.go` may serve as a reference for two-phase pattern
  (task 04 will fix orphan separately)

## Log

### Planning

- Current `Adopt` iterates paths with continue-on-failure: validates and executes
  each path independently, prints errors, counts failures.
- Spec requires strict two-phase: collect all → validate all → execute all (or
  roll back all).
- `performDirectoryAdoption` mixes walking, validation, and execution in a single
  `filepath.Walk` — needs to be split into collection (Phase 1) and execution
  (Phase 2).
- `ValidateSymlinkCreation` is currently called with `(absPath, destPath)` but
  spec says `(destPath, absPath)` — source=destPath (real file), target=absPath
  (symlink).

### Testing

- ...

### Implementation

- ...
