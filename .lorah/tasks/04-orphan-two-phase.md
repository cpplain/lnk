---
status: test
---

# Task: Rewrite `orphan` to use two-phase transactional execution

## Behavior

`Orphan` must use two-phase execution matching `docs/design/features/orphan.md` §5:

- **Phase 1 (Collect and Validate)**: For each path, expand, lstat, validate
  within target directory, handle directories (call `FindManagedLinks`, reject
  broken links, collect active links), handle files (must be symlink, readlink,
  resolve to absolute, verify within source dir, verify not broken). Fail-fast:
  if any validation fails, return the error immediately with no filesystem
  changes. After collecting, deduplicate by `Path`.

- **Dry-run**: If `DryRun`, print planned orphans and return.

- **Phase 2 (Execute with rollback)**: For each managed link call
  `orphanManagedLink`: verify target still exists, read original file mode via
  `os.Lstat`, `RemoveSymlink`, `MoveFile`, `os.Chmod` (best-effort). If any
  step fails: roll back ALL completed orphans in reverse order (move file back,
  recreate symlink), return error. After success, call `CleanEmptyDirs` on
  source-side parent directories bounded by `sourceDir`.

Current code uses continue-on-failure (prints errors via `PrintErrorWithHint`,
increments nothing, `continue`s to next path). The `orphanManagedLink` helper
has per-item rollback instead of batch rollback. Missing deduplication, missing
broken-link rejection in directory expansion, missing `CleanEmptyDirs` after
success.

## Acceptance Criteria

- Phase 1 validates ALL paths before any filesystem changes
- Phase 1 fails fast on first validation error (returns error, no changes)
- Directory arguments use `FindManagedLinks` to collect managed symlinks
- Directory with no managed links returns error with hint to run `lnk status`
- Broken links found during directory expansion are rejected with `PathError`
  and hint to use `rm`
- File paths validated: must be symlink, must be managed by source, must not be
  broken
- Paths outside target directory return `ValidationError`
- Paths deduplicated by `Path` after collection, first occurrence kept
- Empty collection after dedup prints `"No managed symlinks found."` and returns nil
- Phase 2 executes remove-symlink + move-file + chmod for all collected links
- Phase 2 failure triggers reverse-order rollback of all completed orphans
- Rollback moves file back and recreates symlink; rollback failure produces
  combined error `"orphan failed: <err>; rollback failed: <err>"`
- Permission restoration uses `os.Lstat` on `link.Target` before removal, logged
  via `PrintVerbose` on failure (not `PrintWarning`)
- After all orphans succeed, `CleanEmptyDirs` called on parent directories of
  `link.Target` paths, bounded by `sourceDir`
- Summary prints `"Orphaned N file(s) successfully"` and next-step hint
- Dry-run prints per-file detail matching spec §5 format

## Context

- Current implementation: `lnk/orphan.go` — uses continue-on-failure model
- `orphanManagedLink` at line 191 — has per-item rollback, needs batch rollback
- Spec: `docs/design/features/orphan.md` §5
- Tests: `lnk/orphan_test.go` — existing tests will need updating to match
  two-phase behavior (errors return immediately, no partial orphaning)
- Reference: `lnk/adopt.go` — completed two-phase rewrite (task 03) is the
  pattern to follow for Phase 2 rollback structure
- `PrintNextStep` currently takes 2 args; spec requires 3. Use current 2-arg
  signature for now — `PrintNextStep` signature change is a separate task.

## Log

### Planning

- Current `Orphan` iterates paths with continue-on-failure: validates each path
  independently, prints errors, continues to next.
- Spec requires strict two-phase: collect all → validate all → execute all (or
  roll back all).
- `orphanManagedLink` has per-item rollback (restores symlink if move fails) —
  needs to become batch rollback like adopt's pattern.
- Missing: deduplication by `Path`, broken-link rejection during directory
  expansion, `CleanEmptyDirs` call on source-side parents after success.
- `os.Chmod` failure currently uses `PrintWarning` — spec says `PrintVerbose`.
- Dry-run output says "Copy from" — spec says "Move from".
- Adopt task (03) provides the two-phase pattern to follow.

### Testing

- ...

### Implementation

- ...
