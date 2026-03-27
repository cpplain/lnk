# lnk Specifications

Design documentation for `lnk`, an opinionated symlink manager for dotfiles.

## Index

| Spec                                   | Description                                                                 |
| -------------------------------------- | --------------------------------------------------------------------------- |
| [cli.md](cli.md)                       | Command-line interface: subcommands, flags, help, version, typo suggestions |
| [config.md](config.md)                 | Configuration system: `.lnkignore`, ignore patterns, built-in defaults      |
| [create.md](create.md)                 | `create` command: symlink creation with 3-phase execution                   |
| [remove.md](remove.md)                 | `remove` command: removing managed symlinks                                 |
| [status.md](status.md)                 | `status` command: displaying managed symlink status                         |
| [prune.md](prune.md)                   | `prune` command: removing broken symlinks                                   |
| [adopt.md](adopt.md)                   | `adopt` command: adopting files into the source directory                   |
| [orphan.md](orphan.md)                 | `orphan` command: removing files from management                            |
| [error-handling.md](error-handling.md) | Error types, hints, exit codes, and per-operation error type mappings       |
| [output.md](output.md)                 | Output system: verbosity, color, piped format                               |
| [internals.md](internals.md)           | Internal helpers: `FindManagedLinks`, `CreateSymlink`, `MoveFile`, etc.     |
| [stdlib.md](stdlib.md)                 | Standard library usage: which packages/functions to use and why             |
| [testing.md](testing.md)               | Testing strategy: TDD workflow, test levels, conventions, helpers           |

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
