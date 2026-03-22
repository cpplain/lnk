# Standard Library Usage Specification

---

## 1. Overview

### Purpose

`lnk` uses Go's standard library exclusively — no external dependencies. This document
specifies which stdlib packages and functions are used for each category of operation,
and why. It is the authoritative reference for implementation choices involving stdlib.

### Goals

- **No external dependencies**: stdlib only
- **Leverage stdlib correctly**: prefer stdlib functions over hand-rolled equivalents
- **Document decisions**: explain why specific functions were chosen and where trade-offs exist

---

## 2. Directory Traversal

### `filepath.WalkDir` (not `filepath.Walk`)

All directory traversal uses `filepath.WalkDir` from the `path/filepath` package.

```go
import "path/filepath"
import "io/fs"

filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if d.Type()&fs.ModeSymlink != 0 {
        // handle symlink
    }
    // ...
})
```

**Why not `filepath.Walk`**: `filepath.Walk` calls `os.Lstat` on every entry to produce
`os.FileInfo`. `filepath.WalkDir` passes `fs.DirEntry` instead — `DirEntry.Type()` reads
the file type directly from the directory entry without an extra syscall on most platforms.
For source-dir walks (`create`, `remove`), this avoids stat-ing ignored files. For
`FindManagedLinks` (target-dir walk), filtering with `d.Type()&fs.ModeSymlink` skips
regular files and directories in `~` without stat-ing them.

**Note on symlink traversal**: `filepath.WalkDir` does not follow symlinks into directories.
This is the correct behavior for `lnk` — only files are linked, never directories.

### Source-dir vs target-dir walking

Different commands use different traversal strategies:

| Command  | Traversal strategy                   | Why                                                     |
| -------- | ------------------------------------ | ------------------------------------------------------- |
| `create` | Walk source dir                      | Enumerate files to link                                 |
| `remove` | Walk source dir                      | Symmetric with create; compute expected target symlinks |
| `status` | Walk target dir (`FindManagedLinks`) | Must show all managed symlinks including broken ones    |
| `prune`  | Walk target dir (`FindManagedLinks`) | Must find broken symlinks whose source files are gone   |
| `orphan` | Walk target dir for directory args   | Must find managed symlinks within a directory           |

`create` and `remove` walk the source directory and compute where each file's symlink
should be in `~`. This is efficient (source dirs are small) and avoids scanning all of
`~`. The trade-off for `remove` is that broken symlinks left by deleted source files are
out of scope — those are handled by `prune`.

`status`, `prune`, and `orphan` use `FindManagedLinks` to walk the target directory
(`~`). This is necessary when the command needs to discover symlinks that may point to
files no longer present in the source directory.

---

## 3. Symlink Resolution

### `filepath.EvalSymlinks`

When resolving whether a symlink's target falls within a source directory,
use `filepath.EvalSymlinks`:

```go
import "path/filepath"

resolved, err := filepath.EvalSymlinks(path)
```

**Why**: `filepath.EvalSymlinks` resolves the complete symlink chain and returns the
real (fully resolved) path. This handles relative symlink targets, chains of symlinks,
and produces a clean absolute path suitable for `filepath.Rel` comparison — all in one
call. The manual sequence of `os.Readlink` + relative resolution + `filepath.Abs`
achieves the same result but requires handling edge cases that `EvalSymlinks` already
covers.

**When not to use it**: `filepath.EvalSymlinks` follows the entire chain, so it will
fail if any link in the chain is broken. For checking symlink metadata (is this a
symlink? what is its raw target?) use `os.Lstat` and `os.Readlink` directly.

**Single-level resolution in `adopt`/`orphan`**: these commands use `os.Readlink` +
manual path resolution instead of `filepath.EvalSymlinks`. They need to verify where
a specific symlink points (one level), not where a chain of symlinks resolves to.
This is deliberate and does not contradict the general preference for `EvalSymlinks`.

### Broken link detection

A managed symlink is broken when `os.Stat(resolvedTarget)` returns `os.IsNotExist`.
`os.Stat` follows symlinks (unlike `os.Lstat`), so it checks whether the ultimate
target file exists:

```go
_, err := os.Stat(resolvedTarget)
isBroken := err != nil && os.IsNotExist(err)
```

---

## 4. Symlink Operations

| Operation           | Function      | Notes                                              |
| ------------------- | ------------- | -------------------------------------------------- |
| Create symlink      | `os.Symlink`  | `os.Symlink(source, target)`                       |
| Read symlink target | `os.Readlink` | Returns raw target string (may be relative)        |
| Inspect path type   | `os.Lstat`    | Does not follow symlinks; use for symlink metadata |
| Remove symlink      | `os.Remove`   | Works on symlinks; does not follow them            |
| Check target exists | `os.Stat`     | Follows symlinks; use for broken link detection    |

---

## 5. Path Operations

| Operation                         | Function         | Notes                                           |
| --------------------------------- | ---------------- | ----------------------------------------------- |
| Check if path is within directory | `filepath.Rel`   | Path is within dir if result doesn't start `..` |
| Normalize to absolute path        | `filepath.Abs`   | Cleans and absolutizes a path                   |
| Join path segments                | `filepath.Join`  | OS-appropriate separator; cleans result         |
| Parent directory                  | `filepath.Dir`   | Used by `CleanEmptyDirs` to walk upward         |
| Expand and normalize              | `filepath.Clean` | Resolves `.` and `..` without filesystem access |

### Checking containment with `filepath.Rel`

To check if `child` is within `parent`:

```go
rel, err := filepath.Rel(parent, child)
isWithin := err == nil && !strings.HasPrefix(rel, "..") && rel != "."
```

---

## 6. Pattern Matching

### Decision: custom `PatternMatcher` (not `filepath.Match`)

`filepath.Match` supports simple glob patterns (`*`, `?`, `[...]`) but does not support:

- `**` — cross-directory matching
- `!pattern` — negation
- Trailing `/` — directory-only matching

The ignore pattern system requires at minimum `**` (for patterns like `*.swp` matching
anywhere in the tree) and `!pattern` (for users to negate built-in defaults). Therefore,
`filepath.Match` is insufficient and a custom `PatternMatcher` is required.

`PatternMatcher` implements gitignore-style semantics:

- `*.swp` — matches any `.swp` file anywhere in the tree (implicit `**`)
- `local/` — matches a directory named `local` and all files within it
- `dir/file` — matches only at that specific relative path
- `**` — matches across directory boundaries
- `!pattern` — negates a previously matched pattern

---

## 7. File Operations

| Operation              | Function              | Notes                                              |
| ---------------------- | --------------------- | -------------------------------------------------- |
| Move file (same FS)    | `os.Rename`           | Fast path; fails across filesystems                |
| Read directory entries | `os.ReadDir`          | Returns `[]fs.DirEntry`; used by `CleanEmptyDirs`  |
| Create directory tree  | `os.MkdirAll`         | Creates all missing parents; mode `0755`           |
| Copy file contents     | `io.Copy`             | Used by `MoveFile` cross-device fallback           |
| Get file mode          | `os.Lstat` → `Mode()` | Used by `MoveFile` to preserve permissions on copy |
| Restore permissions    | `os.Chmod`            | Used by `orphan` to restore original file mode     |

### `os.Stat` vs `os.Lstat`

- **`os.Stat`**: follows symlinks — use when you want to know about the file the symlink points to (e.g., does the target exist?)
- **`os.Lstat`**: does not follow symlinks — use when you want to know about the symlink itself (e.g., is this path a symlink?)

---

## 8. What Requires Custom Implementation

These operations have no suitable stdlib equivalent:

| Custom function           | Why no stdlib equivalent                                                 |
| ------------------------- | ------------------------------------------------------------------------ |
| `MoveFile`                | `os.Rename` fails cross-device; fallback requires copy + verify + delete |
| `CleanEmptyDirs`          | No stdlib function walks upward removing empty dirs to a boundary        |
| Transactional rollback    | No stdlib filesystem transaction support                                 |
| `ValidateSymlinkCreation` | Domain-specific: same-path, circular reference, overlapping path checks  |
| `PatternMatcher`          | `filepath.Match` lacks `**` and `!` negation                             |

---

## 9. Package Import Summary

```go
import (
    "io"            // io.Copy (MoveFile cross-device fallback)
    "io/fs"         // fs.DirEntry, fs.ModeSymlink (WalkDir callbacks)
    "os"            // file operations, stat, symlinks
    "path/filepath" // WalkDir, EvalSymlinks, Rel, Abs, Join, Dir, Match
    "strings"       // filepath.Rel result prefix checks
)
```

---

## 10. Related Specifications

- [internals.md](internals.md) — Internal helper functions and their stdlib usage
- [config.md](config.md) — `LoadIgnoreFile` and path handling
- [create.md](create.md) — Source-dir traversal pattern
- [remove.md](remove.md) — Source-dir traversal for removal
- [status.md](status.md) — Target-dir traversal via `FindManagedLinks`
- [prune.md](prune.md) — Target-dir traversal via `FindManagedLinks`
- [orphan.md](orphan.md) — Target-dir traversal via `FindManagedLinks`
