## YOUR ROLE - INIT PHASE

You are setting up a cleanup project in an existing Go codebase. This runs once at the start.

### STEP 1: Read the Specification and Codebase

1. Read `.lorah/spec.md` to understand the cleanup goal (removing legacy code)
2. Read `CLAUDE.md` to understand project structure and conventions
3. Explore the existing codebase in `internal/lnk/` and `cmd/lnk/` to understand current architecture

### STEP 2: Create Task List

Create `.lorah/tasks.json` with testable requirements based on the 6 phases in spec.md:

**Phase 0: Simplify Naming** (tasks for function/type/constant renames)
**Phase 1: Remove Legacy Types** (tasks for removing old types and constants)
**Phase 2: Remove Legacy Functions** (tasks for removing old functions from each file)
**Phase 3: Clean Up Status** (task for keeping new Status function)
**Phase 4: Update Documentation** (task for README.md rewrite)
**Phase 5: Clean Up Tests** (tasks for removing legacy test code)
**Phase 6: Update CLAUDE.md** (task for updating documentation)

Format:

```json
[
  {
    "name": "Remove legacy Config types",
    "description": "Remove LinkMapping, old Config, and ConfigOptions structs from config.go",
    "passes": false
  },
  {
    "name": "Remove CreateLinks function",
    "description": "Remove legacy CreateLinks(config *Config) from linker.go",
    "passes": false
  }
]
```

IMPORTANT: Order tasks to follow dependencies:

- Remove legacy functions BEFORE renaming new functions
- Update tests AFTER code changes
- Update docs AFTER all code changes

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
