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
    Target   string // absolute path of the symlink's resolved target (never relative)
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

This function is only needed by commands that must scan the target directory for
symlinks whose source files may no longer exist (`prune`) or for discovering
managed symlinks within a directory argument (`orphan`). Commands that know their
source directory (`create`, `remove`) walk the source directory instead.
See [stdlib.md](stdlib.md) for the source-dir vs target-dir traversal strategy.

### Behavior

1. Uses `filepath.WalkDir` to traverse the directory tree rooted at `startPath`.
   `fs.DirEntry.Type()` provides the file type without an extra `Lstat` syscall,
   allowing non-symlink entries to be skipped cheaply.
2. Skips non-symlink entries: `d.Type()&fs.ModeSymlink == 0`
3. On macOS, skips `Library` and `.Trash` directories entirely (`filepath.SkipDir`)
4. For each symlink found:
   - Calls `filepath.EvalSymlinks` to resolve the full symlink chain to a clean
     absolute path
   - Checks if `filepath.Rel(source, resolvedTarget)` does not start with `..` and
     is not `.` for any source in `sources`
   - If matched: creates a `ManagedLink`; sets `Target` to the resolved absolute path
     returned by `filepath.EvalSymlinks`
   - Sets `IsBroken` based on whether the target file exists (see broken link handling below)
5. Walk errors (e.g., permission denied on a subdirectory) are logged at verbose
   level and do not abort the walk — results may be incomplete

### Broken Link Handling

`filepath.EvalSymlinks` fails on broken symlinks (the chain cannot be resolved). When
it fails, fall back to manual resolution to still check containment and mark the link
broken:

1. Call `os.Readlink(symlinkPath)` to get the raw target string
2. If the target is relative, resolve it: `filepath.Join(filepath.Dir(symlinkPath), rawTarget)`
3. Call `filepath.Abs` to normalize
4. Check containment: for any source in `sources`, check that
   `filepath.Rel(source, resolvedTarget)` does not start with `..` and is not `.`
5. If matched: create `ManagedLink` with `IsBroken: true`, `Target` set to the
   normalized absolute path computed in step 3, and `Source` set to the matching source

This ensures broken managed symlinks (e.g., from deleted source files) are still
discovered and reported by `status` and `prune`.

### Return Value

Returns the collected `[]ManagedLink` and the error from `filepath.WalkDir` (nil
unless the root `startPath` itself cannot be walked). An empty slice with nil
error means no managed links were found.

### Usage

```go
links, err := FindManagedLinks(targetDir, []string{sourceDir})
```

Used by: `status`, `prune`, `orphan`.

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
3. On success: returns nil (the caller is responsible for printing output)

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
   - Reads `src` file mode via `os.Lstat`
   - Copies file contents from `src` to `dst`
   - Applies the original file mode to `dst` via `os.Chmod`
   - Verifies the copy by comparing file sizes
   - If copy or verification fails: call `os.Remove(dst)` (ignore any removal error) before returning the error
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

Used by: `remove`, `prune`, `adopt` (rollback only), `orphan`.

---

## 8. ValidateSymlinkCreation

```go
func ValidateSymlinkCreation(source, target string) error
```

Validates that creating a symlink at `target` pointing to `source` would not produce
an invalid or dangerous filesystem state.

### Behavior

1. Returns `ValidationError` if `source == target`
2. Returns `ValidationError` if `source` is inside `target` (circular reference)
3. Returns `ValidationError` if `target` is inside `source` (overlapping paths)

All paths are resolved to absolute paths before comparison.

### Usage

Used by: `create` (Phase 2 validation), `adopt` (Phase 1 validation).

---

## 9. PatternMatcher

```go
type PatternMatcher struct { ... }

func NewPatternMatcher(patterns []string) *PatternMatcher
func (m *PatternMatcher) Matches(relPath string) bool
```

Matches relative paths against a list of gitignore-style ignore patterns.

### Behavior

- Patterns are applied in order; later patterns can negate earlier ones with `!pattern`
- `*.swp` — matches any `.swp` file anywhere in the tree
- `local/` — matches a directory named `local` and all files within it
- `dir/file` — matches only at that specific relative path
- `**` — matches across directory boundaries
- `!pattern` — negates a previously matched pattern; the path is included if the last matching pattern is a negation

`Matches` returns `true` if the path should be ignored (excluded from linking).

### Usage

Used by: `create` (Phase 1 collection).

---

## 10. validateAdoptSource

```go
func validateAdoptSource(absPath, absSourceDir string) error
```

Checks whether a path is already adopted (i.e., a symlink that already points into
`absSourceDir`).

### Behavior

1. Calls `os.Lstat(absPath)` — if path does not exist or is not a symlink, returns nil
   (not already adopted)
2. Reads the symlink target via `os.Readlink`
3. Resolves to an absolute path: if `rawTarget` is relative, compute
   `filepath.Join(filepath.Dir(absPath), rawTarget)` then `filepath.Clean`
   to normalize; if `rawTarget` is already absolute, use it directly
4. Checks `filepath.Rel(absSourceDir, cleanTarget)` — if the result does not start
   with `..` and is not `.`, the file is already adopted
5. Returns `LinkError` with `ErrAlreadyAdopted` and hint to run `lnk status`

### Usage

Used by: `adopt` (Phase 1 validation).

---

## 11. LoadIgnoreFile

```go
func LoadIgnoreFile(sourceDir string) ([]string, error)
```

Loads ignore patterns from `<sourceDir>/.lnkignore`. Returns an empty slice without
error if the file does not exist.

### Behavior

1. Resolves `<sourceDir>/.lnkignore` to an absolute path
2. If the file does not exist: logs via `PrintVerbose` and returns `[]string{}, nil`
3. Opens and reads the file; if any I/O error occurs (e.g., permission denied):
   returns `nil, err` — propagates to `LoadConfig`, which returns the error to
   the caller and aborts the operation
4. Parses the file content line by line:
   - Skips empty lines and lines beginning with `#`
   - Each non-comment line is appended as a pattern
5. Returns the collected patterns

### Usage

Used by: `LoadConfig` in `config.go`.

---

## 12. Related Specifications

- [create.md](create.md) — Uses `CreateSymlink`, `ValidateSymlinkCreation`, `PatternMatcher`
- [remove.md](remove.md) — Uses `RemoveSymlink`, `CleanEmptyDirs` (source-dir walk)
- [status.md](status.md) — Uses `FindManagedLinks`
- [prune.md](prune.md) — Uses `FindManagedLinks`, `RemoveSymlink`, `CleanEmptyDirs`
- [adopt.md](adopt.md) — Uses `MoveFile`, `CleanEmptyDirs` (rollback), `ValidateSymlinkCreation`, `validateAdoptSource`
- [orphan.md](orphan.md) — Uses `FindManagedLinks`, `RemoveSymlink`, `MoveFile`, `CleanEmptyDirs`
- [config.md](config.md) — Uses `LoadIgnoreFile`
- [error-handling.md](error-handling.md) — Error types returned by these functions
- [stdlib.md](stdlib.md) — Standard library functions used by these helpers
