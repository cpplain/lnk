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
must be within the user's home directory (`~`).

### Go Function

```go
func Adopt(opts AdoptOptions) error
```

```go
type AdoptOptions struct {
    SourceDir string   // repository directory to move files into
    Paths     []string // one or more file/directory paths to adopt (must be within ~)
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
3. **If directory** (not itself a symlink): walk it and collect each file within;
   apply steps 4–8 to each collected file
4. **Validate** via `validateAdoptSource(absPath, absSourceDir)`:
   - If path is a symlink already pointing into `sourceDir`: return error
     `"file already adopted"` with hint to run `lnk status`
5. **Compute relative path** from the user's home directory (`os.UserHomeDir`) to `absPath`:
   - If the path is not within `~`: return error with hint that only files
     within the home directory can be adopted
6. **Compute destination**: `destPath = filepath.Join(absSourceDir, relPath)`
7. **Check destination**: if `destPath` already exists, return error with hint to
   remove it first
8. **Validate symlink** via `ValidateSymlinkCreation(absPath, destPath)` — checks for
   circular references and overlapping paths

If any validation fails, return the error immediately. No filesystem changes are made.

If no files are collected after expansion (e.g., empty directory), print
`"No files to adopt found."` and return nil.

### Dry-Run Mode

For each collected file, print:

```
[DRY RUN] Would adopt: ~/.bashrc
  Move to: ~/git/dotfiles/.bashrc
  Create symlink: ~/.bashrc -> ~/git/dotfiles/.bashrc
```

End with `PrintDryRunSummary()`.

### Phase 2: Execute

For each planned adoption in order:

1. Create parent directory of `destPath` (`os.MkdirAll`, mode `0755`)
2. Move file from `absPath` to `destPath` via `MoveFile`
3. Create symlink: `absPath` → `destPath`
4. On success: print `"Adopted: <absPath>"`

If any step fails:

- Roll back all completed adoptions in reverse order:
  - Remove the symlink (if created)
  - Move `destPath` back to `absPath` via `MoveFile`
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
   - Copy verifies file size matches after copy
   - Original is removed only after successful copy

---

## 6. Path Behavior

- `SourceDir` is expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory
- Each `Path` is expanded with `ExpandPath` before processing
- Each path must reside within the user's home directory (`~`); paths outside produce an error
- The home directory is resolved internally via `os.UserHomeDir()`
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
Next: Run 'lnk status' to view adopted files
```

---

## 9. Error Cases

| Scenario                    | Error Message                                                   |
| --------------------------- | --------------------------------------------------------------- |
| File does not exist         | `adopt <path>: no such file or directory` + hint to check path  |
| File already adopted        | `adopt <path>: file already adopted` + hint to run `lnk status` |
| Path outside home directory | `path <path> must be within home directory` + hint              |
| Destination already exists  | `destination <dest> already exists` + hint to remove first      |
| Permission denied           | OS error wrapped in `PathError` with permission hint            |

---

## 10. Related Specifications

- [orphan.md](orphan.md) — The inverse operation
- [create.md](create.md) — Creating symlinks after adoption
- [status.md](status.md) — Verifying adopted files
- [error-handling.md](error-handling.md) — Error types and rollback behavior
- [output.md](output.md) — Output functions and verbosity
