# Internal Functions Specification

---

## 1. Overview

This document describes internal Go functions shared across multiple operation
implementations. These are not user-facing commands but are referenced throughout
the operation specs.

---

## 2. ManagedLink

```go
type ManagedLink struct {
    Path     string // absolute path of the symlink in the target directory
    Target   string // raw symlink target value (as stored on disk)
    IsBroken bool   // true if the resolved target file does not exist
    Source   string // absolute source directory that manages this link
}
```

---

## 3. FindManagedLinks

```go
func FindManagedLinks(startPath string, sources []string) ([]ManagedLink, error)
```

Walks `startPath` recursively and returns all symlinks whose resolved absolute
target is inside any of the specified `sources` directories.

### Behavior

1. Uses `filepath.Walk` to traverse the directory tree rooted at `startPath`
2. Skips non-symlink entries (regular files, directories)
3. On macOS, skips `Library` and `.Trash` directories entirely (`filepath.SkipDir`)
4. For each symlink found:
   - Reads the target via `os.Readlink`
   - Resolves relative targets relative to the symlink's parent directory
   - Calls `filepath.Abs` to produce a clean absolute path
   - Checks if `filepath.Rel(source, cleanTarget)` does not start with `..` and
     is not `.` for any source in `sources`
   - If matched: creates a `ManagedLink`; sets `IsBroken` if `os.Stat(cleanTarget)` fails
5. Walk errors (e.g., permission denied on a subdirectory) are logged at verbose
   level and do not abort the walk — results may be incomplete

### Return Value

Returns the collected `[]ManagedLink` and the error from `filepath.Walk` (nil
unless the root `startPath` itself cannot be walked). An empty slice with nil
error means no managed links were found.

### Usage

```go
links, err := FindManagedLinks(targetDir, []string{sourceDir})
```

Used by: `remove`, `status`, `prune`, `orphan`.

---

## 4. CreateSymlink

```go
func CreateSymlink(source, target string) error
```

Creates a symlink at `target` pointing to `source`.

### Behavior

1. Calls `os.Lstat(target)` to check if the target path already exists:
   - If it is a symlink already pointing to `source`: return `LinkExistsError`
     (non-fatal signal; caller skips silently)
   - If it is a symlink pointing elsewhere: remove it via `os.Remove`, then
     create the new symlink
   - If it is a regular file or directory: return `LinkError` with hint to use
     `lnk adopt` first
2. Creates the symlink via `os.Symlink(source, target)`
3. On success: prints `"Created: <ContractPath(target)>"`

### Errors

- `LinkExistsError`: symlink already correct — caller skips silently, no output
- `LinkError`: collision with regular file, or symlink removal/creation failure

---

## 5. RemoveSymlink

```go
func RemoveSymlink(path string) error
```

Removes the symlink at `path`.

### Behavior

1. Calls `os.Lstat(path)` — returns `PathError` if path does not exist
2. Verifies the entry is a symlink — returns `PathError` with `ErrNotSymlink` if not
3. Calls `os.Remove(path)` — returns the OS error on failure

---

## 6. MoveFile

```go
func MoveFile(src, dst string) error
```

Moves a file from `src` to `dst`.

### Behavior

1. Attempts `os.Rename(src, dst)` — fast path, works on the same filesystem
2. If rename fails (e.g., cross-device): falls back to copy-then-delete:
   - Copies file contents from `src` to `dst`
   - Verifies the copy by comparing file sizes
   - Removes `src` only after a successful, verified copy

---

## 7. CleanEmptyDirs

```go
func CleanEmptyDirs(dirs []string, boundaryDir string) int
```

Removes empty parent directories left behind after symlink removal or file moves.

### Behavior

1. For each directory path in `dirs`:
   - Start at `current = dir`
   - Loop while `current != boundaryDir`:
     - Read directory entries via `os.ReadDir(current)`: if non-empty or error, break
     - Call `os.Remove(current)` — only succeeds on empty directories (safe by design)
     - On success: log via `PrintVerbose("Removed empty directory: %s", ContractPath(current))`
       and increment the removed counter
     - On failure: log via `PrintVerbose` and break
     - Advance: `current = filepath.Dir(current)`
2. Return the total count of removed directories

`boundaryDir` is never removed. If a parent directory was already cleaned by
an earlier entry in `dirs`, `os.ReadDir` will fail and the loop breaks gracefully
— no deduplication needed.

### Usage

```go
// remove / prune: clean target-side empty dirs
CleanEmptyDirs(parentDirsOfRemovedLinks, targetDir)

// orphan: clean source-side empty dirs
CleanEmptyDirs(parentDirsOfOrphanedTargets, sourceDir)
```

Used by: `remove`, `prune`, `orphan`.

---

## 8. Related Specifications

- [create.md](create.md) — Uses `CreateSymlink`
- [remove.md](remove.md) — Uses `FindManagedLinks`, `RemoveSymlink`, `CleanEmptyDirs`
- [status.md](status.md) — Uses `FindManagedLinks`
- [prune.md](prune.md) — Uses `FindManagedLinks`, `RemoveSymlink`, `CleanEmptyDirs`
- [adopt.md](adopt.md) — Uses `MoveFile`
- [orphan.md](orphan.md) — Uses `FindManagedLinks`, `RemoveSymlink`, `MoveFile`, `CleanEmptyDirs`
- [error-handling.md](error-handling.md) — Error types returned by these functions
