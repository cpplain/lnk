# lnk — Design Specifications

Design documentation for `lnk`, an opinionated symlink manager for dotfiles.

---

## Foundation

Read these first. They define conventions and cross-cutting concerns
that all feature specs assume.

| Spec                                   | Description                                                                 |
| -------------------------------------- | --------------------------------------------------------------------------- |
| [cli.md](cli.md)                       | Command-line interface: subcommands, flags, help, version, typo suggestions |
| [config.md](config.md)                 | Configuration system: `.lnkignore`, ignore patterns, built-in defaults      |
| [error-handling.md](error-handling.md) | Error types, hints, exit codes, and per-operation error type mappings       |
| [output.md](output.md)                 | Output system: verbosity, color, piped format                               |
| [internals.md](internals.md)           | Internal helpers: `FindManagedLinks`, `CreateSymlink`, `MoveFile`, etc.     |
| [stdlib.md](stdlib.md)                 | Standard library usage: which packages/functions to use and why             |
| [testing.md](testing.md)               | Testing strategy: TDD workflow, test levels, conventions, helpers           |

## Feature Specs

Each spec covers one command end-to-end: behavior, acceptance criteria,
error cases.

| Spec                                     | Description                              |
| ---------------------------------------- | ---------------------------------------- |
| [features/create.md](features/create.md) | Symlink creation with 3-phase execution  |
| [features/remove.md](features/remove.md) | Removing managed symlinks                |
| [features/status.md](features/status.md) | Displaying managed symlink status        |
| [features/prune.md](features/prune.md)   | Removing broken symlinks                 |
| [features/adopt.md](features/adopt.md)   | Adopting files into the source directory |
| [features/orphan.md](features/orphan.md) | Removing files from management           |

## Glossary

These terms are used consistently across all specs. Source and target follow the
same convention as `ln -s source target` — source is where the real file lives,
target is where the symlink appears.

| Term                 | Definition                                                                          |
| -------------------- | ----------------------------------------------------------------------------------- |
| **source directory** | The dotfiles repository (e.g. `~/git/dotfiles`). This is where the real files live. |
| **target directory** | Where symlinks are created (always `~`). This is where files appear to live.        |
| **managed symlink**  | A symlink whose resolved target is within the source directory.                     |
| **active symlink**   | A managed symlink whose target file exists.                                         |
| **broken symlink**   | A managed symlink whose target file no longer exists.                               |

## Design Principles

- **Obvious over clever**: make intuitive paths easiest
- **Helpful over minimal**: provide clear guidance and error messages
- **Consistent over special**: follow CLI conventions
- All mutating commands support `--dry-run`
- All commands accept all global flags
- Paths display using `~/` contraction for home directory paths
- Error messages include actionable hints wherever possible
