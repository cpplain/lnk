# lnk

An opinionated symlink manager for dotfiles.

## Key Features

- **Simple CLI** - Subcommand-based interface (`lnk <command> [flags] <source-dir>`)
- **Recursive file linking** - Links individual files throughout directory trees
- **Flexible organization** - Support for multiple source directories
- **Safety first** - Dry-run mode and clear status reporting
- **Ignore patterns** - Optional `.lnkignore` file with gitignore syntax
- **No dependencies** - Single binary, stdlib only (git integration optional)

## Installation

```bash
brew install cpplain/tap/lnk
```

## Quick Start

```bash
# Create links from current directory
cd ~/git/dotfiles
lnk create .                  # Link everything from current directory

# Link from a specific subdirectory
lnk create home               # Link everything from home/ subdirectory

# Link from an absolute path
lnk create ~/git/dotfiles     # Link from specific path
```

## Usage

```bash
lnk <command> [flags] <source-dir>
```

### Commands

| Command  | Args                     | Description                           |
| -------- | ------------------------ | ------------------------------------- |
| `create` | `<source-dir>`           | Create symlinks from source to target |
| `remove` | `<source-dir>`           | Remove managed symlinks               |
| `status` | `<source-dir>`           | Show status of managed symlinks       |
| `prune`  | `<source-dir>`           | Remove broken symlinks                |
| `adopt`  | `<source-dir> <path...>` | Adopt files into source directory     |
| `orphan` | `<source-dir> <path...>` | Remove files from management          |

For all commands, `source-dir` is the first required positional argument (the dotfiles directory). The target directory is always `~`.

For `adopt`/`orphan`: one or more file or directory paths within `~` are required as additional positional arguments.

### Flags

| Flag               | Description                                                 |
| ------------------ | ----------------------------------------------------------- |
| `--ignore PATTERN` | Additional ignore pattern (repeatable, only affects create) |
| `-n, --dry-run`    | Preview changes without making them                         |
| `-v, --verbose`    | Enable verbose output                                       |
| `--no-color`       | Disable colored output                                      |
| `-V, --version`    | Show version information                                    |
| `-h, --help`       | Show help message                                           |

## Examples

### Creating Links

```bash
# Link from current directory
lnk create .

# Link from a subdirectory
lnk create home

# Link from absolute path
lnk create ~/git/dotfiles

# Dry-run to preview changes
lnk create -n .

# Add ignore pattern
lnk create --ignore '*.swp' .
```

### Removing Links

```bash
# Remove links from source directory
lnk remove .

# Remove links from subdirectory
lnk remove home

# Dry-run to preview removal
lnk remove -n .
```

### Checking Status

```bash
# Show status of links from current directory
lnk status .

# Show status from subdirectory
lnk status home

# Show status with verbose output
lnk status -v .
```

### Pruning Broken Links

```bash
# Remove broken symlinks from current directory
lnk prune .

# Remove broken symlinks from specific source
lnk prune home

# Dry-run to preview pruning
lnk prune -n .
```

### Adopting Files

```bash
# Adopt files into current directory
lnk adopt . ~/.bashrc ~/.vimrc

# Adopt into specific source directory
lnk adopt ~/git/dotfiles ~/.bashrc

# Adopt with dry-run
lnk adopt -n . ~/.gitconfig
```

### Orphaning Files

```bash
# Remove file from management (current directory as source)
lnk orphan . ~/.bashrc

# Orphan with specific source
lnk orphan ~/git/dotfiles ~/.bashrc

# Dry-run to preview orphaning
lnk orphan -n . ~/.config/oldapp
```

## Config Files

lnk supports an optional ignore file in your source directory.

### .lnkignore (optional)

Place in source directory. Gitignore syntax for files to exclude from linking.

```
.git
*.swp
*~
README.md
scripts/
.DS_Store
```

### Default Ignore Patterns

lnk automatically ignores these patterns:

- `.git`
- `.gitignore`
- `.DS_Store`
- `*.swp`
- `*.tmp`
- `README*`
- `LICENSE*`
- `CHANGELOG*`
- `.lnkignore`

## How It Works

### Recursive File Linking

lnk recursively traverses your source directory and creates individual symlinks for each file (not directories). This approach:

- Allows you to mix managed and unmanaged files in the same target directory
- Preserves your ability to have local-only files alongside managed configs
- Creates parent directories as needed (never as symlinks)

**Example:** Linking from `~/dotfiles` to `~`:

```
Source:                              Target:
~/dotfiles/
  .bashrc                     →      ~/.bashrc (symlink)
  .config/
    git/
      config                  →      ~/.config/git/config (symlink)
    nvim/
      init.vim                →      ~/.config/nvim/init.vim (symlink)
```

The directories `.config`, `.config/git`, and `.config/nvim` are created as regular directories, not symlinks. This allows you to have local configs in `~/.config/localapp/` that aren't managed by lnk.

### Repository Organization

You can organize your dotfiles in different ways:

**Flat Repository:**

```
~/dotfiles/
  .bashrc
  .vimrc
  .gitconfig
```

Use: `lnk create .` from within the directory

**Nested Repository:**

```
~/dotfiles/
  home/          # Public configs
    .bashrc
    .vimrc
  private/       # Private configs (e.g., git submodule)
    .ssh/
      config
```

Use: `lnk create home` to link public configs, or `lnk create ~/dotfiles/private` for private configs

### Configuration

The **target directory** is always `~` and is not configurable.

For **ignore patterns**: all sources are combined — built-in defaults, `.lnkignore`,
and `--ignore` flags are all merged into a single pattern list.

### Ignore Patterns

lnk supports gitignore-style patterns for excluding files from linking:

- `*.swp` - all swap files
- `local/` - local directory
- `!important.swp` - negation (include this specific file)
- `**/*.log` - all .log files recursively

Patterns can be specified via:

- `.lnkignore` file (one pattern per line)
- CLI flags (`--ignore pattern`)

## Common Workflows

### Setting Up a New Machine

```bash
# 1. Clone your dotfiles
git clone https://github.com/you/dotfiles.git ~/dotfiles
cd ~/dotfiles

# 2. If using private submodule
git submodule update --init

# 3. Create links (dry-run first to preview)
lnk create -n .

# 4. Create links for real
lnk create .
```

### Adding New Configurations

```bash
# Adopt a new app config into your repository (current directory as source)
lnk adopt . ~/.config/newapp

# This will:
# 1. Move ~/.config/newapp to ./.config/newapp
# 2. Create symlinks for each file in the directory tree
# 3. Preserve the directory structure

# Or specify source directory explicitly
lnk adopt ~/dotfiles ~/.config/newapp
```

### Managing Public and Private Configs

```bash
# Keep work/private configs separate using git submodule
cd ~/dotfiles
git submodule add git@github.com:you/dotfiles-private.git private

# Structure:
# ~/dotfiles/public/    (public configs)
# ~/dotfiles/private/   (private configs via submodule)

# Link public configs
cd ~/dotfiles
lnk create public

# Link private configs
lnk create private

# Or adopt to appropriate location
lnk adopt ~/dotfiles/public ~/.bashrc    # Public config
lnk adopt ~/dotfiles/private ~/.ssh/config  # Private config
```

### Migrating from Other Dotfile Managers

```bash
# 1. Remove existing links from old manager
stow -D home  # Example: GNU Stow

# 2. Create links with lnk
cd ~/dotfiles
lnk create .

# lnk creates individual file symlinks instead of directory symlinks,
# so you can gradually migrate and test
```

## Tips

- **Always dry-run first** - Use `--dry-run` or `-n` to preview changes before making them
- **Check status regularly** - Use `lnk status .` to check for broken links
- **Organize your dotfiles** - Separate public and private configs into subdirectories
- **Leverage .lnkignore** - Exclude build artifacts, local configs, and README files
- **Test on VM first** - When setting up a new machine, test in a VM before production
- **Version your configs** - Keep `.lnkignore` in git for reproducibility
- **Use verbose mode for debugging** - Add `-v` to see what lnk is doing

## Comparison with Other Tools

### vs. GNU Stow

- **lnk**: Creates individual file symlinks, allows mixing configs from multiple sources
- **stow**: Creates directory symlinks, simpler but less flexible

### vs. chezmoi

- **lnk**: Simple symlinks, no templates, what you see is what you get
- **chezmoi**: Templates, encryption, complex state management

### vs. dotbot

- **lnk**: Subcommand-based CLI, built-in adopt/orphan operations
- **dotbot**: YAML-based config, more explicit control

lnk is designed for users who:

- Want a simple, subcommand-based CLI
- Prefer symlinks over copying
- Need to mix public and private configs
- Want built-in adopt/orphan workflows
- Value clarity over configurability

## Troubleshooting

### Broken Links After Moving Dotfiles

```bash
# Remove old links (from original location)
cd /old/path/to/dotfiles
lnk remove .

# Recreate from new location
cd /new/path/to/dotfiles
lnk create .
```

### Some Files Not Linking

```bash
# Check if they're ignored
lnk create -v .  # Verbose mode shows ignored files

# Check .lnkignore
cat .lnkignore
```

### Permission Denied Errors

```bash
# Check file permissions in source
ls -la ~/dotfiles/.ssh

# Files should be readable
chmod 600 ~/dotfiles/.ssh/config
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
