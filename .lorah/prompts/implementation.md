## YOUR ROLE - BUILD PHASE

You are continuing work on a Go refactoring project. Each session starts fresh with no memory of previous sessions.

### STEP 1: Get Your Bearings

Run these commands to understand the current state:

```bash
pwd && ls -la
cat .lorah/spec.md
cat .lorah/tasks.json
cat .lorah/progress.md
git log --oneline -10
```

Review `CLAUDE.md` for project conventions and build commands.

### STEP 2: Choose ONE Task

Find a task with `"passes": false` in tasks.json. Pick the most logical next one based on:

- Dependencies (e.g., LinkOptions must exist before \*WithOptions functions)
- Phase order (Phase 1 before Phase 2, etc.)

IMPORTANT: Only work on ONE task per session.

### STEP 3: Implement & Test

1. **Write the code** - Follow existing patterns in `internal/lnk/`
2. **Build**: Run `make build` to verify compilation
3. **Test**: Run `make test` to verify all tests pass
4. **Unit tests**: Add unit tests for new functions
5. **E2E tests**: Update e2e tests if CLI behavior changed

Refer to CLAUDE.md for:

- Commit message format (Conventional Commits)
- Code standards (error handling, output, path handling)
- Testing structure (unit tests in `internal/lnk/*_test.go`, e2e in `e2e/`)

### STEP 4: Update Progress

Only after verifying the task works:

1. **Update tasks.json** - Change `"passes": false` to `"passes": true` for THIS task only
   - DO NOT modify task names or descriptions
   - DO NOT add or remove tasks
   - ONLY flip the `passes` field
2. **Update progress.md** - Add what you implemented, any issues discovered, next steps
3. **Commit** - Use `/commit` skill to generate conventional commit message

### CRITICAL RULES

- **Leave code working**: All tests must pass before session end
- **One task at a time**: Complete one feature fully before moving to next
- **No task modifications**: Only change the `passes` field in tasks.json
- **Follow conventions**: Use error types from errors.go, output functions from output.go
- **Test thoroughly**: Run both `make test-unit` and `make test-e2e`

### EXAMPLE WORKFLOW

```bash
# 1. Orient
cat .lorah/tasks.json | grep -A1 '"passes": false' | head -4

# 2. Implement (example: LinkOptions struct)
# - Edit internal/lnk/linker.go
# - Add LinkOptions struct

# 3. Test
make build
make test-unit

# 4. Update
# - Edit .lorah/tasks.json (flip passes to true)
# - Edit .lorah/progress.md

# 5. Commit
# Use commit skill to help generate a commit message.
/commit
git commit -m <message>
```
