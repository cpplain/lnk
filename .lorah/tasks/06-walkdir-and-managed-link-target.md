---
status: implement
---

# Task: Replace filepath.Walk with WalkDir and fix ManagedLink.Target

## Behavior

Migrate all `filepath.Walk` calls to `filepath.WalkDir` per `docs/design/stdlib.md`
§2, and fix `FindManagedLinks` to set `ManagedLink.Target` to the normalized absolute
path per `docs/design/internals.md` §3.

Three changes:

1. **`lnk/symlink.go` — `FindManagedLinks`**: Replace `filepath.Walk` with
   `filepath.WalkDir`. The callback changes from `func(path string, info os.FileInfo,
   err error)` to `func(path string, d fs.DirEntry, err error)`. Use
   `d.Type()&fs.ModeSymlink` to detect symlinks instead of `info.Mode()&os.ModeSymlink`.
   Use `d.IsDir()` instead of `info.IsDir()`.

2. **`lnk/symlink.go` — `FindManagedLinks`**: Set `ManagedLink.Target` to the
   normalized absolute path (`cleanTarget`) instead of the raw readlink value
   (`target`). The spec says Target stores "absolute path of the symlink's resolved
   target (never relative)".

3. **`lnk/create.go` — `collectPlannedLinksWithPatterns`**: Replace `filepath.Walk`
   with `filepath.WalkDir`. Same callback signature change. Use `d.IsDir()` instead
   of `info.IsDir()`.

Additionally, per the spec, `FindManagedLinks` should try `filepath.EvalSymlinks`
first for non-broken links (to get the fully resolved path), and only fall back to
manual `os.Readlink` + resolution for broken links. Currently it always uses the
manual path. Review whether the current approach satisfies the spec or needs
restructuring.

## Acceptance Criteria

- No `filepath.Walk` calls remain in any `.go` files (only `filepath.WalkDir`)
- `FindManagedLinks` callback uses `fs.DirEntry` and `d.Type()&fs.ModeSymlink`
- `ManagedLink.Target` is always an absolute path (never a raw relative readlink value)
- `collectPlannedLinksWithPatterns` callback uses `fs.DirEntry` and `d.IsDir()`
- All existing tests pass (`make check`)

## Context

- `filepath.Walk` is used in exactly 2 places: `lnk/symlink.go:24` and `lnk/create.go:31`
- `ManagedLink.Target` is set at `lnk/symlink.go:80` using raw `target` variable;
  should use `cleanTarget` instead
- The spec in `internals.md` §3 describes using `filepath.EvalSymlinks` first, with
  manual fallback for broken links — the current code only does manual resolution
- Import `io/fs` will be needed for `fs.DirEntry` and `fs.ModeSymlink`

## Log

### Planning

- Two `filepath.Walk` call sites identified, both straightforward signature changes
- `ManagedLink.Target` fix is a one-line change (`target` → `cleanTarget`) but
  couples naturally with the WalkDir migration since both are in the same function
- The `FindManagedLinks` resolution logic may need restructuring to try
  `filepath.EvalSymlinks` first per the spec — evaluate during implementation

### Testing

- Added 4 tests to `lnk/symlink_test.go`:
  - `TestFindManagedLinksTargetIsAbsolute`: relative symlink → Target must be absolute (FAILS: stores raw readlink)
  - `TestFindManagedLinksTargetAbsoluteForAbsoluteSymlinks`: absolute symlink → Target is absolute (passes)
  - `TestFindManagedLinksBrokenLinkTargetIsAbsolute`: broken relative symlink → Target must be absolute (FAILS: stores raw readlink)
  - `TestFindManagedLinksUsesEvalSymlinks`: non-broken link Target matches `filepath.EvalSymlinks` result (FAILS: manual resolution doesn't resolve macOS `/var` → `/private/var`)
- No tests for WalkDir migration itself (same observable behavior as Walk; verified by acceptance criteria code check)
- Existing `TestFindManagedLinks` table tests all pass unchanged

### Implementation

- ...
