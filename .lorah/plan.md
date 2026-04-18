# Spec-Compliance Fixes

## Scope

Align the implementation with the design specs in `docs/design/`. A branch
review identified 17 issues where the code diverges from the specifications.
This work fixes the implementation — no new features, no spec changes.

The issues fall into these categories:

- **Traversal strategy**: `remove` walks the wrong directory
- **Missing cleanup**: `CleanEmptyDirs` not called after symlink removal
- **Transactional model**: `adopt` and `orphan` use continue-on-failure instead
  of two-phase transactional execution with rollback
- **Output system**: missing `PrintWarningWithHint`, wrong `PrintNextStep`
  signature, broken link output on wrong stream
- **Command details**: wrong error functions, missing deduplication, missing
  broken-link rejection, wrong empty-result messages
- **CLI edge cases**: bare `lnk` exit code, `--ignore=value` parsing before
  command name
- **Stdlib compliance**: `filepath.Walk` used instead of `filepath.WalkDir`
- **Data model**: `ManagedLink.Target` stores raw readlink value instead of
  absolute path

## Boundaries

- Fix implementation only — do not modify design specs in `docs/design/`
- Do not add new features or change public behavior beyond what specs require
- Existing tests that conflict with specs should be updated to match specs
- All changes must pass `make check` (fmt + test + lint)

## Acceptance Criteria

### Critical

- [x] `remove` walks `SourceDir` (not `TargetDir`) to find managed links, matching
      `docs/design/features/remove.md` §5 step 1
- [x] `remove` calls `CleanEmptyDirs` on parent directories of removed symlinks,
      matching `remove.md` §5 step 2
- [x] `prune` calls `CleanEmptyDirs` on parent directories of pruned symlinks,
      matching `docs/design/features/prune.md` §5
- [x] `adopt` uses two-phase execution: Phase 1 validates ALL paths (fail-fast on
      first error), Phase 2 executes with batch rollback on any failure, matching
      `docs/design/features/adopt.md` §5
- [x] `orphan` uses two-phase execution: Phase 1 validates ALL paths (fail-fast on
      first error), Phase 2 executes with batch rollback on any failure, matching
      `docs/design/features/orphan.md` §5

### High

- [ ] `PrintWarningWithHint(err error)` exists in `lnk/output.go`, matching
      `docs/design/output.md` §5 Specialized Functions
- [ ] `PrintNextStep` takes three arguments `(command, sourceDir, description)` and
      contracts `sourceDir` via `ContractPath`, matching `output.md` §5
- [ ] `status` prints broken links to stdout (not stderr), matching
      `docs/design/features/status.md` §5 step 3
- [ ] `create`, `remove`, and `prune` use `PrintWarningWithHint` for per-item
      execution failures, matching their respective specs
- [ ] `remove` and `prune` print next-step hints when `failed == 0`, matching their
      respective specs
- [x] `adopt` and `orphan` deduplicate collected paths by absolute path, matching
      their respective specs
- [x] `orphan` rejects broken links found during directory expansion, matching
      `docs/design/features/orphan.md` §5 phase 1

### Medium

- [ ] All `filepath.Walk` calls replaced with `filepath.WalkDir`, matching
      `docs/design/stdlib.md` §2
- [ ] Bare `lnk` (no arguments) exits with code 0, matching
      `docs/design/cli.md` §3 step 4
- [ ] `extractCommand` in `main.go` handles `--ignore=value` syntax before the
      command name
- [ ] `FindManagedLinks` sets `ManagedLink.Target` to the normalized absolute path,
      matching `docs/design/internals.md` §3
- [ ] `status` empty result prints "No managed links found." matching
      `docs/design/features/status.md`
