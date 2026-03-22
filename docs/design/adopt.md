# Adopt Command Specification

---

## 1. Overview

### Purpose

The `adopt` command brings existing files under `lnk` management: it moves each
specified file from the home directory into the source (repository) directory and
creates a symlink in the original location pointing to the new repository copy.

### Goals

- **Atomic**: all paths succeed together or none are changed — no partial state left on disk
- **Non-destructive**: files are moved, not deleted; the symlink preserves access from the original location
- **Rollback on failure**: if any operation fails, all completed adoptions are reversed
- **Already-adopted detection**: clear error if a file is already managed by `lnk`
- **Directory support**: adopting a directory adopts each file within it individually
- **Dry-run support**: preview all moves and symlinks before executing

### Non-Goals

- Adopting files outside the home directory
- Merging file contents
- Adopting symlinks that already point elsewhere

---

## 2. Interface

### CLI

```
lnk adopt [flags] <source-dir> <path...>
```

`source-dir` is the repository directory to move files into (required). One or more
paths are required after the source directory. Each path may be a file or directory and
must be within `~`.

### Go Function

```go
func Adopt(opts AdoptOptions) error
```

```go
type AdoptOptions struct {
    SourceDir string   // repository directory to move files into
    TargetDir string   // home directory where files currently live (always ~ from CLI; configurable in tests)
    Paths     []string // one or more file/directory paths to adopt (must be within TargetDir)
    DryRun    bool     // preview mode
}
```

---

## 3. Behavior

`Adopt` executes in two sequential phases. If Phase 1 fails for any path, Phase 2 does
not run. If any operation in Phase 2 fails, all completed adoptions are rolled back and
the error is returned — no partial state is left on disk.

### Phase 1: Collect and Validate

For each path in `opts.Paths`:

1. **Expand** the path using `ExpandPath`
2. **Stat** with `os.Lstat`:
   - If path does not exist: return error with hint to check the path
3. **If directory** (not itself a symlink): walk it recursively and collect each regular file
   within (`d.Type().IsRegular()`); symlinks and other non-regular entries are skipped;
   apply steps 4–8 to each collected file. If no files are found after walking,
   return error `"no files to adopt in <path>"` with hint to check that the
   directory contains regular files
4. **Validate** via `validateAdoptSource(absPath, absSourceDir)`:
   - If path is a symlink already pointing into `sourceDir`: return error
     `"file already adopted"` with hint to run `lnk status`
   - If `validateAdoptSource` returns nil but the path is a symlink (detected via
     the `os.Lstat` result from step 2): return `PathError` (op: `"adopt"`, path,
     err: a descriptive error) with hint to remove the symlink first — adopting
     symlinks that point outside `sourceDir` is not supported
5. **Compute relative path** from `opts.TargetDir` to `absPath`:
   - If the path is not within `TargetDir`: return error with hint that only files
     within the target directory can be adopted
6. **Compute destination**: `destPath = filepath.Join(absSourceDir, relPath)`
7. **Check destination**: if `destPath` already exists, return error with hint to
   remove it first
8. **Validate symlink** via `ValidateSymlinkCreation(destPath, absPath)` — checks for
   circular references and overlapping paths (source=destPath, the real file after the
   move; target=absPath, the symlink location)

After collecting all paths, **deduplicate** by absolute path — if the same file was
collected more than once (e.g., via both a directory argument and an explicit file
argument), keep only the first occurrence.

If any validation fails, return the error immediately. No filesystem changes are made.

### Dry-Run Mode

Print a count header, then per-file detail:

```
Adopting Files

[DRY RUN] Would adopt 2 file(s):
[DRY RUN] Would adopt: ~/.bashrc
  Move to: ~/git/dotfiles/.bashrc
  Create symlink: ~/.bashrc -> ~/git/dotfiles/.bashrc
[DRY RUN] Would adopt: ~/.vimrc
  Move to: ~/git/dotfiles/.vimrc
  Create symlink: ~/.vimrc -> ~/git/dotfiles/.vimrc

No changes made in dry-run mode
```

### Phase 2: Execute

For each planned adoption in order:

1. **Verify source still exists** (`os.Lstat(absPath)`): if gone, return error with hint
2. Create parent directory of `destPath` (`os.MkdirAll`, mode `0755`)
3. Move file from `absPath` to `destPath` via `MoveFile`
4. Create symlink via `CreateSymlink(destPath, absPath)` — `source=destPath` (the real
   file in the repository), `target=absPath` (where the symlink appears)
5. On success: print `"Adopted: <absPath>"`

If any step fails:

- Roll back in reverse order all adoptions up to and including the failing one
  (the per-step conditionals handle partial state):
  - Remove the symlink (if created)
  - Move `destPath` back to `absPath` via `MoveFile` (if moved)
  - If a rollback step also fails: return a combined error reporting both the
    original failure and the rollback failure (e.g.,
    `"adopt failed: <err>; rollback failed: <err>"`)
- Call `CleanEmptyDirs` on parent directories of rolled-back destinations, bounded
  by `sourceDir`, but only for directories that were **created by `MkdirAll` during
  this operation** (track newly created dirs before calling `MkdirAll` by checking
  existence first)
- Return error describing the failure

After all adoptions succeed:

- Print summary `"Adopted N file(s) successfully"` and next-step hint

---

## 4. Already-Adopted Detection

A file is considered already adopted if:

1. `os.Lstat` shows it is a symlink, AND
2. Resolving the symlink target to an absolute path and computing
   `filepath.Rel(absSourceDir, cleanTarget)` yields a path that does not start with `..`
   and is not `.`

---

## 5. MoveFile Behavior

`MoveFile(src, dst)` attempts:

1. `os.Rename(src, dst)` — fast path (same filesystem)
2. If rename fails (e.g., cross-device): copy then delete
   - Read `src` file mode via `os.Lstat`
   - Copy file contents from `src` to `dst`
   - Apply the original file mode to `dst` via `os.Chmod`; if `os.Chmod` fails,
     log a warning via `PrintVerbose` and continue — permission restoration is
     best-effort and does not abort the copy
   - Verify the copy by comparing file sizes
   - If copy or verification fails: call `os.Remove(dst)` (ignore any removal error) before returning the error
   - Remove `src` only after a successful, verified copy

---

## 6. Path Behavior

- `SourceDir` is resolved to an absolute path: first `ExpandPath` (tilde expansion),
  then `filepath.Abs` (relative-to-absolute conversion)
- `SourceDir` must exist and be a directory; checked after path resolution via
  `os.Stat` — returns `ValidationError` with hint if missing or not a directory
- Each `Path` is resolved to an absolute path: first `ExpandPath`, then `filepath.Abs`
- Each path must reside within `TargetDir` (always `~` from CLI); paths outside produce an error
- Displayed paths use `ContractPath`

---

## 7. Examples

```sh
# Adopt a single file
lnk adopt . ~/.bashrc

# Adopt multiple files
lnk adopt . ~/.bashrc ~/.vimrc ~/.gitconfig

# Adopt with explicit source directory
lnk adopt ~/git/dotfiles ~/.bashrc

# Adopt a directory (adopts each file individually)
lnk adopt . ~/.config/nvim

# Dry-run to preview what would happen
lnk adopt -n . ~/.bashrc ~/.vimrc
```

---

## 8. Output

```
Adopting Files

✓ Adopted: ~/.bashrc
✓ Adopted: ~/.vimrc

✓ Adopted 2 file(s) successfully
Next: Run 'lnk status <source-dir>' to view adopted files
```

---

## 9. Error Cases

| Scenario                      | Error Message                                                                     |
| ----------------------------- | --------------------------------------------------------------------------------- |
| File does not exist           | `adopt <path>: no such file or directory` + hint to check path                    |
| File already adopted          | `adopt <path>: file already adopted` + hint to run `lnk status`                   |
| Path is a non-adopted symlink | `adopt <path>: cannot adopt a symlink` + hint to remove the symlink first         |
| Path outside target directory | `path <path> must be within target directory` + hint                              |
| Destination already exists    | `destination <dest> already exists` + hint to remove first                        |
| Empty directory argument      | `no files to adopt in <path>` + hint to check directory contains regular files    |
| Source vanishes at execute    | error with hint to check path; all completed adoptions rolled back + dirs cleaned |
| Permission denied             | OS error wrapped in `PathError` with permission hint                              |

---

## 10. Related Specifications

- [orphan.md](orphan.md) — The inverse operation
- [create.md](create.md) — Creating symlinks after adoption
- [status.md](status.md) — Verifying adopted files
- [error-handling.md](error-handling.md) — Error types and rollback behavior
- [output.md](output.md) — Output functions and verbosity
