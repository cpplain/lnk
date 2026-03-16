# Status Command Specification

---

## 1. Overview

### Purpose

The `status` command displays all symlinks in the target directory that are managed
by the specified source directory, categorized as active (link target exists) or
broken (link target does not exist).

### Goals

- **Read-only**: status never modifies any files
- **Sorted output**: links displayed in alphabetical order by path
- **Broken link visibility**: broken links are clearly distinguished from active links
- **Simplified piped output**: reduced formatting when stdout is not a terminal
- **Summary**: always shows total counts

### Non-Goals

- Showing unmanaged files in the target directory
- Showing what files would be linked (use `create --dry-run`)
- JSON or structured format output

---

## 2. Interface

### CLI

```
lnk status [flags] <source-dir>
```

`source-dir` is the source directory to check (required). The target directory is
always `~`. `--dry-run` is accepted but has no effect (status is always read-only).

### Go Function

```go
func Status(opts LinkOptions) error
```

```go
type LinkOptions struct {
    SourceDir      string   // source directory to check
    TargetDir      string   // where to search for symlinks (always ~ from CLI; configurable in tests)
    IgnorePatterns []string // not used by status
    DryRun         bool     // accepted but ignored
}
```

---

## 3. Behavior

### Step 1: Discover Managed Links

Call `FindManagedLinks(targetDir, []string{sourceDir})` to collect all symlinks
in `targetDir` pointing into `sourceDir`.

Each entry carries:

```go
type ManagedLink struct {
    Path     string // absolute path of the symlink in target
    Target   string // absolute path of the symlink's resolved target (never relative)
    IsBroken bool   // true if the target file does not exist
    Source   string // absolute source directory that manages this link
}
```

### Step 2: Sort

Sort all managed links by `Path` (lexicographic ascending).

### Step 3: Display

Split managed links into two groups: active and broken.

#### Terminal Output

Active links are printed first, then a blank line separator (if both groups are
non-empty), then broken links:

```
Symlink Status

✓ Active: ~/.bashrc
✓ Active: ~/.vimrc
✓ Active: ~/.config/git/config

✗ Broken: ~/.zshrc

✓ Total: 4 links (3 active, 1 broken)
```

#### Piped Output

When `ShouldSimplifyOutput()` is true (stdout is not a terminal), each link is
printed as a space-separated `status path` pair with no icons. Paths use
`ContractPath` (`~/`) consistent with terminal output:

```
active ~/.bashrc
active ~/.vimrc
active ~/.config/git/config
broken ~/.zshrc
```

No summary line is printed in piped mode.

### Empty Result

If no managed links are found:

```
Symlink Status

No managed links found.
```

---

## 4. Exit Code

`status` exits 0 whenever it successfully reports, even when broken links are found.
Broken links are informational — not a runtime error. Exit 1 only on actual failures
(e.g., the target directory cannot be read). Users who want to act on broken links
programmatically can use piped output:

```sh
lnk status . | grep ^broken
```

---

## 5. Path Behavior

- `SourceDir` and `TargetDir` are expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory; validation error otherwise
- Walk skips `Library` and `.Trash` directories on macOS
- All displayed paths use `ContractPath` (home directory shown as `~`)

---

## 6. Broken Link Detection

A link is broken when `os.Stat(resolvedTarget)` returns `os.IsNotExist`. This follows
symlinks (unlike `os.Lstat`), so a broken link is one whose ultimate target does not
exist.

---

## 7. Examples

```sh
# Status of current directory
lnk status .

# Status of a specific source
lnk status ~/git/dotfiles

# Verbose: show source and target dirs before listing
lnk status -v ~/git/dotfiles

# Pipe to grep to find broken links
lnk status ~/git/dotfiles | grep ^broken
```

---

## 8. Output

```
Symlink Status

✓ Active: ~/.bashrc
✓ Active: ~/.config/git/config
✓ Active: ~/.vimrc

✓ Total: 3 links (3 active, 0 broken)
```

With broken links:

```
Symlink Status

✓ Active: ~/.bashrc
✓ Active: ~/.vimrc

✗ Broken: ~/.zshrc

✓ Total: 3 links (2 active, 1 broken)
```

---

## 9. Related Specifications

- [create.md](create.md) — Creating the links shown by status
- [remove.md](remove.md) — Removing active links
- [prune.md](prune.md) — Removing broken links shown by status
- [output.md](output.md) — Terminal vs. machine-readable output rules
- [stdlib.md](stdlib.md) — `filepath.WalkDir` and `filepath.EvalSymlinks` used by `FindManagedLinks`
