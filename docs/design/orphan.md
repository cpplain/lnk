# Orphan Command Specification

---

## 1. Overview

### Purpose

The `orphan` command removes files from `lnk` management: it removes the symlink at
the target location, moves the actual file from the source (repository) directory
back to the target location, and restores the original file permissions.

### Goals

- **Atomic**: all validations pass before any changes are made; all orphans succeed together or none are changed
- **Safe restoration**: the file is always restored to the target before the source copy is removed
- **Rollback on failure**: if any operation fails during execution, all completed orphans are reversed
- **Managed-only**: only symlinks that point into the specified source directory can be orphaned
- **Directory support**: passing a directory orphans all managed symlinks within it
- **Dry-run support**: preview all operations before executing

### Non-Goals

- Orphaning unmanaged symlinks (use `rm` directly)
- Orphaning broken symlinks (target does not exist to restore)
- Removing source files without restoring them

---

## 2. Interface

### CLI

```
lnk orphan [flags] <source-dir> <path...>
```

`source-dir` is the repository directory that manages the files (required). One or
more paths are required after the source directory. Each path may be a managed symlink
or a directory containing managed symlinks, and must be within the user's home
directory (`~`).

### Go Function

```go
func Orphan(opts OrphanOptions) error
```

```go
type OrphanOptions struct {
    SourceDir string   // repository directory (managed link source)
    TargetDir string   // home directory where symlinks live (always ~ from CLI; configurable in tests)
    Paths     []string // one or more symlink paths to orphan
    DryRun    bool     // preview mode
}
```

---

## 3. Behavior

`Orphan` executes in two sequential phases. If Phase 1 fails for any path, Phase 2
does not run — no filesystem changes are made.

### Phase 1: Collect and Validate

For each path in `opts.Paths`:

1. **Expand** the path using `ExpandPath`
2. **Stat** with `os.Lstat`:
   - If not found: return `PathError` (op: `"orphan"`, path, err: `os.ErrNotExist`) with
     hint to check the path
3. **Validate target directory**: compute `filepath.Rel(opts.TargetDir, absPath)` — if the path
   is not within `TargetDir`, return `ValidationError` with hint that only paths within the
   target directory can be orphaned
4. **If directory** (not itself a symlink): call `FindManagedLinks(absPath, []string{absSourceDir})`
   to find all managed symlinks within. If none found: return error `"no managed symlinks
   found in <path>"` with hint to run `lnk status`. For each found link where
   `link.IsBroken == true`: return `PathError` (op: `"orphan"`, path: `link.Path`,
   err: `"symlink target does not exist"`) with hint to use `rm` directly. Add only
   active links to the collection.
5. **If file**:
   - Must be a symlink: if not, return `PathError` with `ErrNotSymlink` and hint to use
     `rm`
   - Read symlink target with `os.Readlink`
   - Resolve the raw target to an absolute path: if `rawTarget` is relative, resolve it as
     `filepath.Join(filepath.Dir(absPath), rawTarget)`, then call `filepath.Clean` to normalize;
     if `rawTarget` is already absolute, use it directly
   - Verify target is within `absSourceDir`: compute `rel, _ := filepath.Rel(absSourceDir,
     resolvedTarget)`; if `rel` starts with `..` or equals `"."`, return `LinkError`
     with hint to use `rm` directly
   - Verify target file exists (not broken) via `os.Stat`: if broken, return `PathError`
     with hint to use `rm`
   - Add to collection as `ManagedLink{Path, Target, IsBroken: false, Source}`

If any validation step returns an error, return it immediately — no filesystem changes are made.

After processing all paths, **deduplicate** by `Path` — if the same symlink was collected
more than once (e.g., via both a directory argument and an explicit symlink argument), keep
only the first occurrence.

If collection is empty after deduplication, print `"No managed symlinks found."` and return nil.

### Dry-Run Mode

```
Orphaning Files

[DRY RUN] Would orphan 2 symlink(s):
[DRY RUN] Would orphan: ~/.bashrc
  Remove symlink: ~/.bashrc
  Move from: ~/git/dotfiles/.bashrc
[DRY RUN] Would orphan: ~/.vimrc
  Remove symlink: ~/.vimrc
  Move from: ~/git/dotfiles/.vimrc

No changes made in dry-run mode
```

### Execute Mode

`Orphan` executes all operations as a transaction. If any step fails, all completed
orphans are rolled back in reverse order and the error is returned — no partial state
is left on disk.

For each managed link in order, call `orphanManagedLink(link)`:

1. Verify target still exists (`os.Stat(link.Target)`): if gone, return error with
   hint to use `rm` for the broken symlink
2. **Read original file mode** from `link.Target` via `os.Lstat`: store
   `info.Mode()` for use in step 5
3. **Remove symlink** via `RemoveSymlink(link.Path)`
4. **Move file** from `link.Target` to `link.Path` via `MoveFile`
5. **Restore permissions** via `os.Chmod(link.Path, originalMode)`:
   - `originalMode` is the mode read in step 2
   - Failure here is a warning only; log it and continue
6. Print `"Orphaned: <link.Path>"`

If any step (1, 3, or 4) fails:

- Roll back all completed orphans in reverse order:
  - Move `link.Path` back to `link.Target` via `MoveFile` (if file was already moved)
  - Recreate the symlink via `os.Symlink(link.Target, link.Path)` (if symlink was removed)
  - If a rollback step also fails: return a combined error reporting both the original
    failure and the rollback failure (e.g., `"orphan failed: <err>; rollback failed: <err>"`)
- Return error describing the original failure

After all orphans succeed:

- Call `CleanEmptyDirs` with the parent directories of all orphaned files' source
  locations (`link.Target`) and `sourceDir` as the boundary. This walks upward
  from each parent in the repository, removing empty directories until reaching
  `sourceDir` (which is never removed). Each removed directory is logged via
  `PrintVerbose`. The target side is unaffected — the file has been restored there.
- Print summary `"Orphaned N file(s) successfully"` and next-step hint

---

## 4. Managed Link Validation

A symlink is considered managed by `absSourceDir` when:

1. The symlink target resolves to an absolute path
2. `filepath.Rel(absSourceDir, resolvedTarget)` does not start with `..` and is not `.`

This is identical to the detection used by `FindManagedLinks`.

---

## 5. Path Behavior

- `SourceDir` is expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory
- Each `Path` is expanded with `ExpandPath` before processing
- Each path must reside within `TargetDir` (always `~` from CLI); paths outside produce an error
- Displayed paths use `ContractPath`

---

## 6. Examples

```sh
# Orphan a single file
lnk orphan . ~/.bashrc

# Orphan multiple files
lnk orphan . ~/.bashrc ~/.vimrc

# Orphan with explicit source directory
lnk orphan ~/git/dotfiles ~/.bashrc

# Orphan all managed files in a directory
lnk orphan . ~/.config/nvim

# Dry-run to preview
lnk orphan -n . ~/.bashrc
```

---

## 7. Output

```
Orphaning Files

✓ Orphaned: ~/.bashrc
✓ Orphaned: ~/.vimrc

✓ Orphaned 2 file(s) successfully
Next: Run 'lnk status <source-dir>' to view remaining managed files
```

---

## 8. Error Cases

All Phase 1 errors abort the entire operation before any filesystem changes are made.

| Scenario                         | Phase | Error Type        | Error                                                             |
| -------------------------------- | ----- | ----------------- | ----------------------------------------------------------------- |
| Path does not exist              | 1     | `PathError`       | `orphan <path>: no such file or directory` + check path hint      |
| Path outside target directory    | 1     | `ValidationError` | `path <path> must be within target directory` + hint              |
| Path is a regular file           | 1     | `PathError`       | `orphan <path>: not a symlink` + hint to use `rm`                 |
| Symlink not managed by source    | 1     | `LinkError`       | `orphan <path>: not managed by source` + hint to use `rm`         |
| Broken symlink                   | 1     | `PathError`       | `orphan <path>: symlink target does not exist` + hint to use `rm` |
| No managed links in directory    | 1     | error             | `no managed symlinks found in <path>` + hint to run `lnk status`  |
| Move fails (with rollback)       | 2     | error             | Error about failed move; all completed orphans reversed           |
| Move fails (rollback also fails) | 2     | error             | Combined error: `"orphan failed: <err>; rollback failed: <err>"`  |

---

## 9. Related Specifications

- [adopt.md](adopt.md) — The inverse operation
- [status.md](status.md) — Verifying remaining managed files after orphaning
- [remove.md](remove.md) — Removing symlinks without restoring files
- [error-handling.md](error-handling.md) — Error types and rollback behavior
- [output.md](output.md) — Output functions and verbosity
