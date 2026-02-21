## YOUR ROLE - INIT PHASE

You are setting up a refactoring project in an existing Go codebase. This runs once at the start.

### STEP 1: Read the Specification and Codebase

1. Read `.lorah/spec.md` to understand the refactoring goal
2. Read `CLAUDE.md` to understand project structure and conventions
3. Explore the existing codebase in `internal/lnk/` and `cmd/lnk/` to understand current architecture

### STEP 2: Create Task List

Create `.lorah/tasks.json` with testable requirements based on the 4 implementation phases in spec.md:

**Phase 1: Config file support** (tasks for LoadConfig, discovery, parsing, merging)
**Phase 2: Options-based API** (tasks for LinkOptions struct and \*WithOptions functions)
**Phase 3: CLI rewrite** (tasks for flag parsing, action flags, help, validation)
**Phase 4: Update internal functions** (tasks for adopt, orphan, prune updates)
**Testing** (tasks for unit tests and e2e tests)

Format:

```json
[
  {
    "name": "LoadConfig function",
    "description": "Implement config file discovery and parsing in config.go",
    "passes": false
  },
  {
    "name": "LinkOptions struct",
    "description": "Add LinkOptions struct to linker.go with all required fields",
    "passes": false
  }
]
```

Mark ALL tasks as `"passes": false` initially.

### STEP 3: No Git Init Needed

This is an existing repo with git already initialized. Skip git init.

### STEP 4: Create Progress Notes

Create `.lorah/progress.md` with:

- Initial inventory of what exists
- Summary of refactoring plan
- Note that tasks.json has been created
- Mark session complete

### STEP 5: Commit Your Work

Use the `/commit` skill to help create a conventional commit message.
