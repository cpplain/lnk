# Prune Command Specification

---

## 1. Overview

### Purpose

The `prune` command removes broken symlinks from the target directory that are
managed by the specified source directory. A broken symlink is one whose target
file no longer exists (e.g., after files were deleted from the source repository).

### Goals

- **Targeted cleanup**: only remove symlinks that are both managed and broken
- **Non-destructive**: never remove active symlinks or regular files
- **Dry-run support**: preview broken links before removing them
- **Explicit source**: source directory argument is required

### Non-Goals

- Removing unmanaged broken symlinks
- Removing active managed symlinks (use `remove`)
- Recreating links for missing source files

---

## 2. Interface

### CLI

```
lnk prune [flags] <source-dir> [target-dir]
```

`source-dir` is the source directory whose broken links to prune (required).
`target-dir` is the directory to search for symlinks (optional, default: `~`).

### Go Function

```go
func Prune(opts LinkOptions) error
```

```go
type LinkOptions struct {
    SourceDir      string   // source directory whose broken links to prune
    TargetDir      string   // where to search for symlinks (default: ~)
    IgnorePatterns []string // not used by prune
    DryRun         bool     // preview mode
}
```

---

## 3. Behavior

### Step 1: Discover Managed Links

Call `FindManagedLinks(targetDir, []string{sourceDir})` to collect all symlinks in
`targetDir` pointing into `sourceDir`.

### Step 2: Filter to Broken

Keep only links where `IsBroken == true`.

If no broken links are found among managed links, print `"No broken symlinks found."`
and return nil.

### Step 3: Dry-Run or Execute

#### Dry-Run Mode

```
Pruning Broken Symlinks

[DRY RUN] Would prune 1 broken symlink(s):
[DRY RUN] Would prune: ~/.zshrc

No changes made in dry-run mode
```

#### Execute Mode

For each broken link:

1. Call `RemoveSymlink(path)` to remove it
2. On success: print `"Pruned: <path>"`
3. On failure: print error and increment failure counter; continue with remaining links

After all links are processed:

- If `pruned > 0`: print summary `"Pruned N broken symlink(s) successfully"`
- If `failed > 0`: print warning `"Failed to prune N symlink(s)"` and return error

---

## 4. Broken Link Detection

A link is marked broken during `FindManagedLinks` when `os.Stat(resolvedTarget)`
returns `os.IsNotExist`. This check is performed at discovery time; links that
become broken between discovery and execution are handled gracefully by the remove
step returning an error.

---

## 5. Path Behavior

- `SourceDir` and `TargetDir` are expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory; validation error otherwise
- Walk skips `Library` and `.Trash` directories on macOS
- Displayed paths use `ContractPath` (home directory shown as `~`)

---

## 6. Examples

```sh
# Prune broken links from current directory
lnk prune .

# Prune from a specific source
lnk prune ~/git/dotfiles

# Dry-run to see which broken links would be pruned
lnk prune -n ~/git/dotfiles

# Verbose output
lnk prune -v ~/git/dotfiles
```

---

## 7. Output

```
Pruning Broken Symlinks

✓ Pruned: ~/.zshrc

✓ Pruned 1 broken symlink(s) successfully
```

No broken links found:

```
Pruning Broken Symlinks

No broken symlinks found.
```

---

## 8. Relationship to Other Commands

| Scenario                                   | Use          |
| ------------------------------------------ | ------------ |
| Remove all managed links (active + broken) | `lnk remove` |
| Remove only broken managed links           | `lnk prune`  |
| See which links are broken before pruning  | `lnk status` |

---

## 9. Related Specifications

- [remove.md](remove.md) — Removing all managed links (not just broken)
- [status.md](status.md) — Identifying broken links before pruning
- [error-handling.md](error-handling.md) — Error types used during removal
- [output.md](output.md) — Output functions and verbosity
