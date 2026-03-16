# Create Command Specification

---

## 1. Overview

### Purpose

The `create` command recursively traverses a source directory and creates a symlink
in the target directory for every file found, mirroring the directory structure.
Directories themselves are never symlinked — only individual files are.

### Goals

- **File-level linking**: symlink individual files, never directories
- **Non-destructive**: fail with a clear error if a non-symlink file already exists at the target
- **Idempotent**: re-running `create` on an already-linked repo is safe and silent
- **Dry-run first**: all changes can be previewed before execution
- **3-phase execution**: collect, validate, then execute — no partial states

### Non-Goals

- Directory-level symlinking
- Merging or diffing file contents
- Watching for file changes

---

## 2. Interface

### CLI

```
lnk create [flags] <source-dir>
```

`source-dir` is the source directory to link from (required). The target directory
is always `~`.

### Go Function

```go
func CreateLinks(opts LinkOptions) error
```

```go
type LinkOptions struct {
    SourceDir      string   // source directory to link from
    TargetDir      string   // where to create links (always ~ from CLI; configurable in tests)
    IgnorePatterns []string // combined ignore patterns from all sources
    DryRun         bool     // preview mode: show changes without making them
}
```

---

## 3. Execution Phases

`CreateLinks` executes in three sequential phases. If any phase fails, subsequent
phases do not run.

### Phase 1: Collect

Walk `SourceDir` recursively. For each entry:

1. Skip non-regular entries: only collect files where `d.Type().IsRegular()`; directories, symlinks, and other special entries are skipped
2. Compute the relative path from `SourceDir`
3. Check the relative path against ignore patterns via `PatternMatcher`
4. If not ignored, add `PlannedLink{Source: absFile, Target: targetDir/relPath}`

If no files are found after filtering, print `"No files to link found."` and return nil.

```go
type PlannedLink struct {
    Source string // absolute path to file in source directory
    Target string // absolute path where symlink will be created
}
```

### Phase 2: Validate

For each `PlannedLink`, call `ValidateSymlinkCreation(source, target)`:

- Detect circular references (source inside target directory)
- Detect overlapping paths (source == target, source inside target, target inside source)

If any validation fails, return the error immediately without executing any links.
All-or-nothing: the user sees the problem before any filesystem changes are made.

### Phase 3: Execute (or Dry-Run)

#### Dry-Run Mode

Print what would happen without making changes:

```
[DRY RUN] Would create 3 symlink(s):
[DRY RUN] Would link: ~/.bashrc -> ~/git/dotfiles/.bashrc
[DRY RUN] Would link: ~/.vimrc -> ~/git/dotfiles/.vimrc
[DRY RUN] Would link: ~/.config/git/config -> ~/git/dotfiles/.config/git/config

No changes made in dry-run mode
```

#### Execute Mode

For each `PlannedLink`:

1. Create parent directory (`os.MkdirAll`) if it does not exist (mode `0755`)
2. Call `CreateSymlink(source, target)`:
   - If target is already a symlink pointing to `source`: silently skip (`LinkExistsError`)
   - If target is a symlink pointing elsewhere: remove and recreate
   - If target is a regular file or directory: return error with hint to use `adopt`
3. On success: print `"Created: <target>"`
4. On skip (`LinkExistsError`): continue silently
5. On failure: print warning and increment failure counter; continue with remaining links

After all links are processed:

- If `created > 0`: print summary `"Created N symlink(s) successfully"`
- If `created == 0` and `failed == 0`: print `"All symlinks already exist"`
- If `failed > 0`: print warning `"Failed to create N symlink(s)"` and return error
- Print next-step hint only when `failed == 0`

---

## 4. Ignore Pattern Matching

Patterns are applied to the **relative path** from `SourceDir` (not the absolute path).
Pattern matching follows gitignore semantics:

- `*.swp` — matches any `.swp` file anywhere in the tree
- `local/` — matches a directory named `local` and all files within it
- `dir/file` — matches only at that specific relative path
- `!pattern` — negates a previously matched pattern
- `**` — matches across directory boundaries

See [config.md](config.md) for the full list of active patterns and their sources.

---

## 5. Path Behavior

- `SourceDir` and `TargetDir` are expanded with `ExpandPath` before use
- `SourceDir` must exist and be a directory; validation error otherwise
- `TargetDir` does not need to exist; it is created as needed during execution
- Displayed paths use `ContractPath` (home directory shown as `~`)

---

## 6. Collision Handling

| Target state                       | Behavior                                                                                                                                                         |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does not exist                     | Create symlink                                                                                                                                                   |
| Symlink pointing to correct source | Skip silently (`LinkExistsError`)                                                                                                                                |
| Symlink pointing elsewhere         | Remove and recreate                                                                                                                                              |
| Regular file or directory          | Warning printed; link skipped; run continues. Error returned at end if failure count > 0. Hint: `"Use 'lnk adopt <source-dir> <path>' to adopt this file first"` |

Collisions with regular files do not abort the entire run; all other links are still
attempted. The command exits non-zero if any collisions occurred.

---

## 7. Examples

```sh
# Create links from current directory
lnk create .

# Create links from an absolute path
lnk create ~/git/dotfiles

# Dry-run to preview what would happen
lnk create -n ~/git/dotfiles

# Add an extra ignore pattern
lnk create --ignore 'local/' ~/git/dotfiles

# Verbose output
lnk create -v ~/git/dotfiles
```

---

## 8. Output

```
Creating Symlinks

✓ Created: ~/.bashrc
✓ Created: ~/.vimrc
✓ Created: ~/.config/git/config

✓ Created 3 symlink(s) successfully
Next: Run 'lnk status <source-dir>' to verify links
```

Empty source:

```
Creating Symlinks

No files to link found.
```

All links already exist (idempotent re-run):

```
Creating Symlinks

All symlinks already exist
```

Partial success (some created, some failed):

```
Creating Symlinks

✓ Created: ~/.bashrc
✓ Created: ~/.vimrc
! Failed to create symlink: ~/.zshrc -> ~/git/dotfiles/.zshrc: permission denied
  Try: Use 'lnk adopt <source-dir> <path>' to adopt this file first

✓ Created 2 symlink(s) successfully
! Failed to create 1 symlink(s)
```

---

## 9. Related Specifications

- [config.md](config.md) — Ignore pattern sources and loading
- [status.md](status.md) — Verifying links after creation
- [adopt.md](adopt.md) — Adopting existing files before linking
- [error-handling.md](error-handling.md) — Error types used during validation
- [output.md](output.md) — Output functions and verbosity
