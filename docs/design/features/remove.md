# Remove Command Specification

---

## 1. Overview

### Purpose

The `remove` command walks the source directory, computes where each file's symlink
should be in the target directory, and removes any that are managed by this source.
Only managed symlinks are removed; other files are untouched.

### Goals

- **Scoped removal**: only remove symlinks that point to the specified source directory
- **Non-destructive**: never remove regular files or directories
- **Dry-run support**: preview all removals before committing
- **Partial failure tolerance**: continue removing other links even if one fails

### Non-Goals

- Removing the source files themselves
- Removing symlinks from sources other than the specified one

---

## 2. Scope Fences

### Out of Scope

- `RemoveSymlink` implementation (see [../internals.md](../internals.md))
- `CleanEmptyDirs` implementation (see [../internals.md](../internals.md))
- Removing broken symlinks from deleted source files (see [prune.md](prune.md))
- Error type definitions (see [../error-handling.md](../error-handling.md))
- Output function behavior (see [../output.md](../output.md))

### Do NOT Change

- `LinkOptions` struct shape — shared with `create`, `status`, `prune`
- Source-walk traversal strategy — `remove` walks the source dir, not the target dir
- `CleanEmptyDirs` boundary behavior — `targetDir` is never removed

---

## 3. Dependencies

### Prerequisites

- `LoadConfig` resolves and validates `SourceDir` before `RemoveLinks` is called
- `RemoveSymlink`, `CleanEmptyDirs` from internals
- `PrintSuccess`, `PrintWarningWithHint`, `PrintSummary`, `PrintNextStep`, `PrintDryRun`, `PrintDryRunSummary`, `PrintCommandHeader`, `PrintEmptyResult` from output

---

## 4. Interface

### CLI

```
lnk remove [flags] <source-dir>
```

`source-dir` is the source directory whose managed links to remove (required).
The target directory is always `~`.

### Go Function

```go
func RemoveLinks(opts LinkOptions) error
```

```go
type LinkOptions struct {
    SourceDir      string   // source directory whose managed links to remove
    TargetDir      string   // where to look for symlinks (always ~ from CLI; configurable in tests)
    IgnorePatterns []string // not used by remove; accepted for interface consistency
    DryRun         bool     // preview mode
}
```

---

## 5. Behavior

### Step 1: Collect Managed Links

Walk `SourceDir` recursively using `filepath.WalkDir` — the same traversal strategy
as `create`. For each file found, compute the corresponding symlink path in `TargetDir`. Check each computed path with `os.Lstat`:

- If the path is a symlink pointing to the source file (verified via
  `filepath.EvalSymlinks`): add to the removal list
- Otherwise: skip silently (not managed by this source)

If `filepath.WalkDir` returns an error for any entry (e.g., permission denied), the
walk aborts immediately and `RemoveLinks` returns the error — same rationale as
`create`: source directories should be fully readable.

**Scope**: this approach only removes symlinks for files that currently exist in
`SourceDir`. Broken symlinks left by previously-deleted source files are out of
scope for `remove` and are handled by `prune`.

If no managed links are found, print `"No symlinks to remove found."` and return nil.

### Step 2: Dry-Run or Execute

#### Dry-Run Mode

```
Removing Symlinks

[DRY RUN] Would remove 2 symlink(s):
[DRY RUN] Would remove: ~/.bashrc
[DRY RUN] Would remove: ~/.vimrc

No changes made in dry-run mode
```

#### Execute Mode

For each managed link:

1. Call `RemoveSymlink(path)`:
   - Verifies the path is a symlink before removing
   - Returns error if path is not a symlink or removal fails
2. On success: print `"Removed: <path>"`
3. On failure: call `PrintWarningWithHint(fmt.Errorf("Failed to remove %s: %w", ContractPath(path), err))`;
   increment failure counter; continue with remaining links

After all links are processed:

- Call `CleanEmptyDirs` with the parent directories of all successfully removed
  symlinks and `targetDir` as the boundary. This walks upward from each parent,
  removing empty directories until reaching `targetDir` (which is never removed).
  Each removed directory is logged via `PrintVerbose`.
- If `removed > 0`: print summary `"Removed N symlink(s) successfully"`
- If `failed > 0`: print warning `"Failed to remove N symlink(s)"` via `PrintWarning`
  and return `fmt.Errorf("failed to remove %d symlink(s)", failed)` — plain error,
  no hint (per-item hints already printed inline)
- Print next-step hint only when `failed == 0`

---

## 6. Managed Link Detection

A symlink is "managed" by a source directory if its fully resolved target path
is inside `sourceDir`. Resolution uses `filepath.EvalSymlinks` to follow the complete
symlink chain, then `filepath.Rel` to confirm containment:

```go
resolved, err := filepath.EvalSymlinks(symlinkPath)
if err != nil {
    continue // broken or inaccessible symlink; skip silently
}
rel, _ := filepath.Rel(sourceDir, resolved)
isManaged := !strings.HasPrefix(rel, "..") && rel != "."
```

Links that do not meet this criterion are ignored silently. If `filepath.EvalSymlinks` returns an error (e.g., the symlink is broken), the path is skipped silently — it cannot be confirmed to point into `sourceDir`.

---

## 7. Path Behavior

- `SourceDir` and `TargetDir` are resolved to absolute paths by `LoadConfig`
  (see [../config.md](../config.md) §6) — `SourceDir` is validated to exist and be a
  directory before the command runs
- Displayed paths use `ContractPath` (home directory shown as `~`)

---

## 8. Examples

```sh
# Remove links from current directory
lnk remove .

# Remove links from an absolute path
lnk remove ~/git/dotfiles

# Dry-run to preview what would be removed
lnk remove -n ~/git/dotfiles

# Verbose output
lnk remove -v ~/git/dotfiles
```

---

## 9. Output

```
Removing Symlinks

✓ Removed: ~/.bashrc
✓ Removed: ~/.vimrc

✓ Removed 2 symlink(s) successfully
Next: Run 'lnk status <source-dir>' to verify removal
```

Nothing to remove:

```
Removing Symlinks

No symlinks to remove found.
```

Partial success (some removed, some failed):

```
Removing Symlinks

✓ Removed: ~/.bashrc
! Failed to remove symlink: ~/.vimrc: permission denied

✓ Removed 1 symlink(s) successfully
! Failed to remove 1 symlink(s)
```

---

## 10. Verification

### Test Commands

```bash
go test -v ./lnk -run TestRemoveLinks
go test -v ./test -run TestE2ERemove
```

### Test Scenarios

1. Remove links from a source with multiple managed symlinks — all removed
2. Dry-run — no filesystem changes, output shows planned removals
3. No managed symlinks found — `"No symlinks to remove found."`
4. Mixed managed and unmanaged symlinks — only managed ones removed
5. Broken symlinks from deleted source files — skipped (not managed by source walk)
6. Empty parent directories cleaned up after removal
7. Permission denied on symlink removal — warning, continues with others
8. Walk error on source directory — abort immediately

---

## 11. Related Specifications

- [create.md](create.md) — The inverse operation
- [status.md](status.md) — Verifying links before and after removal
- [prune.md](prune.md) — Removing only broken links
- [../error-handling.md](../error-handling.md) — Error types used during removal
- [../output.md](../output.md) — Output functions and verbosity
- [../stdlib.md](../stdlib.md) — Source-dir traversal strategy and `filepath.EvalSymlinks` usage
