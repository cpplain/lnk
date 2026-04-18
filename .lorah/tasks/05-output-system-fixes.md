---
status: implement
---

<!-- Valid statuses: test | implement | blocked | completed -->

# Task: Fix Output System and Command Output Behavior

## Behavior

Aligns the output system and continue-on-failure commands with the design specs.
References: `docs/design/output.md` §5, `docs/design/features/create.md` §5,
`docs/design/features/remove.md` §5, `docs/design/features/prune.md` §5,
`docs/design/features/status.md` §5.

### Changes required

1. **`PrintWarningWithHint(err error)`** — add to `lnk/output.go`. Mirrors
   `PrintErrorWithHint` but uses warning icon and `PrintWarning` formatting, always
   writes to stderr.
   - Terminal: `"! <err>"` (yellow icon); if hint: `"  Try: <hint>"` (cyan)
   - Piped: `"warning: <err>"`; if hint: `"hint: <hint>"`

2. **`PrintNextStep` signature** — change from `(command, description string)` to
   `(command, sourceDir, description string)`. Contracts `sourceDir` via
   `ContractPath`. Output: `"Next: Run 'lnk <command> <contractedSourceDir>' to
   <description>"`. Update all callers: `create.go`, `adopt.go`, `remove.go`,
   `prune.go`, `orphan.go`.

3. **`status` broken links → stdout** — `status.go` currently uses `PrintError`
   (stderr) for broken links. Replace with direct stdout output using `✗` icon and
   `Red` color inline (terminal) or `"broken <path>"` (piped). Spec: `status.md` §5
   step 3.

4. **`create`, `remove`, `prune` use `PrintWarningWithHint`** — replace `PrintWarning`
   per-item failure calls with `PrintWarningWithHint(fmt.Errorf(...))`. Callers must
   wrap the error with path context before passing.

5. **`remove` and `prune` next-step hints** — both commands currently omit the
   next-step hint entirely. Add `PrintNextStep(...)` call after the summary, printed
   only when `failed == 0`.

6. **`status` empty result message** — currently uses `PrintEmptyResult` with a
   generic item type. Change to `PrintInfo("No managed links found.")` directly, as
   the spec requires specific phrasing (`status.md` §5 empty result).

## Acceptance Criteria

- `PrintWarningWithHint(err error)` exists in `lnk/output.go` and writes to stderr.
  When the error has a hint, appends `"  Try: <hint>"` (terminal) or `"hint: <hint>"`
  (piped).
- `PrintNextStep` takes three arguments `(command, sourceDir, description string)` and
  contracts `sourceDir` via `ContractPath`. All existing callers updated.
- `status` prints broken links to stdout, not stderr (no `PrintError`).
- `status` empty result prints `"No managed links found."`.
- `create`, `remove`, `prune` call `PrintWarningWithHint` for per-item execution
  failures.
- `remove` and `prune` call `PrintNextStep` after the summary only when `failed == 0`.

## Context

- **`lnk/output.go`**: add `PrintWarningWithHint`; change `PrintNextStep` signature.
- **`lnk/status.go`**: fix broken-link output stream; fix empty-result message.
- **`lnk/create.go`**: use `PrintWarningWithHint`; update `PrintNextStep` call.
- **`lnk/remove.go`**: use `PrintWarningWithHint`; update `PrintNextStep` call; add
  next-step after summary when `failed == 0`.
- **`lnk/prune.go`**: use `PrintWarningWithHint`; update `PrintNextStep` call; add
  next-step after summary when `failed == 0`.
- **`lnk/adopt.go`**: update `PrintNextStep` call to include `sourceDir`.
- **`lnk/orphan.go`**: update `PrintNextStep` call to include `sourceDir`.
- `PrintWarningWithHint` does NOT appear in the verbosity table — like
  `PrintErrorWithHint`, it is always shown (not gated by verbosity).
- The `remove.go` and `prune.go` `PrintNextStep` call passes `opts.SourceDir` as
  `sourceDir`.
- For `status` broken-link output in terminal mode: print
  `fmt.Printf("%s Broken: %s\n", Red(FailureIcon), ContractPath(link.Path))` to stdout.
  Piped mode: `fmt.Printf("broken %s\n", ContractPath(link.Path))` (no change needed
  to the piped path since it already writes to stdout via `fmt.Printf`).

## Log

### Planning

- All Critical tasks (01–04) are complete.
- Remaining High items are interconnected: `PrintWarningWithHint` is needed by
  `create`, `remove`, `prune`; `PrintNextStep` signature change forces updates in five
  command files; `status` fixes are small and belong in the same output-consistency
  pass.
- Grouped all six High items into one task for coherence.
- Existing `PrintWarning` calls for per-item failures in `create.go:142`,
  `remove.go` (likely similar), `prune.go` (likely similar) become
  `PrintWarningWithHint` calls.
- `adopt.go:217` and `orphan.go:239` already pass the right description; they just
  need `sourceDir` added.

### Testing

- Created `lnk/output_test.go`: tests for `PrintWarningWithHint` (piped mode: writes
  to stderr as "warning:", emits "hint:" line only when error has hint) and `PrintNextStep`
  (3-arg signature, contracts home dir, formats as "Next: Run 'lnk <cmd> <dir>' to <desc>").
- Updated `lnk/status_test.go`: fixed "no matching links" test to expect "No managed
  links found." (was "No active links found"); added `TestStatusBrokenLinksToStdout`
  (piped mode: broken paths in stdout, not stderr) and `TestStatusEmptyResultMessage`
  (exact empty-result phrasing).
- Added `TestCreateLinksPerItemWarning` to `lnk/create_test.go`: verifies hint line
  appears in stderr — distinguishes `PrintWarningWithHint(%w)` from `PrintWarning(%v)`
  since `CreateSymlink` returns a hinted `LinkError` on permission denied.
- Added `TestRemoveLinksPerItemWarning` and `TestRemoveLinksNextStep` to
  `lnk/remove_test.go`: per-item failure must use "warning:" (not "error:"); next-step
  hint "Next:" must appear after successful removal.
- Added `TestPrunePerItemWarning` and `TestPruneNextStep` to `lnk/prune_test.go`:
  same pattern as remove.
- Added stubs to `lnk/output.go`: `PrintWarningWithHint(err error)` (empty body) and
  changed `PrintNextStep` signature to `(command, sourceDir, description string)`.
- Updated callers to compile: `executePlannedLinks` in `create.go` gains `sourceDir`
  param; `adopt.go` and `orphan.go` pass `absSourceDir`.
- 9 tests fail as expected (no implementation); no panics; e2e suite passes.
- Edge cases covered: piped vs terminal distinction, hint propagation via %w, next-step
  only on full success (failed==0), exact empty-result string.

### Implementation

- ...
