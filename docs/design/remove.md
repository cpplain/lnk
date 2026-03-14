# Remove Command Specification

---

## 1. Overview

### Purpose

The `remove` command finds all symlinks in the target directory that point into the
source directory and removes them. Only managed symlinks (those created by `lnk` from
the specified source) are removed; other files are untouched.

### Goals

- **Scoped removal**: only remove symlinks that point to the specified source directory
- **Non-destructive**: never remove regular files or directories
- **Dry-run support**: preview all removals before committing
- **Partial failure tolerance**: continue removing other links even if one fails

### Non-Goals

- Removing the source files themselves
- Removing symlinks from sources other than the specified one

---

## 2. Interface

### CLI

```
lnk remove [flags] <source-dir> [target-dir]
```

`source-dir` is the source directory whose managed links to remove (required).
`target-dir` is the directory to search for symlinks (optional, default: `~`).

### Go Function

```go
func RemoveLinks(opts LinkOptions) error
```

```go
type LinkOptions struct {
    SourceDir      string   // source directory whose managed links to remove
    TargetDir      string   // where to look for symlinks (default: ~)
    IgnorePatterns []string // not used by remove; accepted for interface consistency
    DryRun         bool     // preview mode
}
```

---

## 3. Behavior

### Step 1: Discover Managed Links

Call `FindManagedLinks(targetDir, []string{sourceDir})` to walk the target directory
and collect all symlinks whose resolved target path is within `sourceDir`.

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
3. On failure: print error and increment failure counter; continue with remaining links

After all links are processed:

- Call `CleanEmptyDirs` with the parent directories of all successfully removed
  symlinks and `targetDir` as the boundary. This walks upward from each parent,
  removing empty directories until reaching `targetDir` (which is never removed).
  Each removed directory is logged via `PrintVerbose`.
- If `removed > 0`: print summary `"Removed N symlink(s) successfully"` and next-step hint
- If `failed > 0`: print warning `"Failed to remove N symlink(s)"` and return error

---

## 4. Managed Link Detection

A symlink is "managed" by a source directory if its resolved absolute target path
is inside `sourceDir`. Resolution:

1. Read the symlink target with `os.Readlink`
2. If the target is relative, resolve it relative to the symlink's parent directory
3. Call `filepath.Abs` to clean the path
4. Check if `filepath.Rel(sourceDir, cleanTarget)` does not start with `..` and is not `.`

Links that do not meet this criterion are ignored silently.

---

## 5. Path Behavior

- `SourceDir` and `TargetDir` are expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory; validation error otherwise
- Walk skips `Library` and `.Trash` directories on macOS (system directories)
- Displayed paths use `ContractPath` (home directory shown as `~`)

---

## 6. Examples

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

## 7. Output

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

---

## 8. Related Specifications

- [create.md](create.md) — The inverse operation
- [status.md](status.md) — Verifying links before and after removal
- [prune.md](prune.md) — Removing only broken links
- [error-handling.md](error-handling.md) — Error types used during removal
- [output.md](output.md) — Output functions and verbosity
