# lnk

An opinionated symlink manager for dotfiles.

## Key Features

- **Simple CLI** - POSIX-style interface with flags before paths
- **Recursive file linking** - Links individual files throughout directory trees
- **Flexible organization** - Support for multiple source directories
- **Safety first** - Dry-run mode and clear status reporting
- **Flexible configuration** - Optional config files with CLI override
- **No dependencies** - Single binary, stdlib only (git integration optional)

## Installation

```bash
brew install cpplain/tap/lnk
```

## Quick Start

```bash
# Link from current directory (default action is create)
cd ~/git/dotfiles
lnk .                         # Link everything from current directory

# Link from a specific subdirectory
lnk home                      # Link everything from home/ subdirectory

# Link from an absolute path
lnk ~/git/dotfiles           # Link from specific path
```

## Usage

```bash
lnk [action] [flags] <path>
```

Path can be a relative or absolute directory. For create/remove/status operations, the path is the source directory to link from. For adopt/orphan operations, paths are the files to manage.

### Action Flags (mutually exclusive)

| Flag            | Description                      |
| --------------- | -------------------------------- |
| `-C, --create`  | Create symlinks (default action) |
| `-R, --remove`  | Remove symlinks                  |
| `-S, --status`  | Show status of symlinks          |
| `-P, --prune`   | Remove broken symlinks           |
| `-A, --adopt`   | Adopt files into source          |
| `-O, --orphan`  | Remove files from management     |

### Directory Flags

| Flag               | Description                                           |
| ------------------ | ----------------------------------------------------- |
| `-s, --source DIR` | Source directory (for adopt/orphan, default: cwd)     |
| `-t, --target DIR` | Target directory (default: `~`)                       |

### Other Flags

| Flag               | Description                            |
| ------------------ | -------------------------------------- |
| `--ignore PATTERN` | Additional ignore pattern (repeatable) |
| `-n, --dry-run`    | Preview changes without making them    |
| `-v, --verbose`    | Enable verbose output                  |
| `-q, --quiet`      | Suppress all non-error output          |
| `--no-color`       | Disable colored output                 |
| `-V, --version`    | Show version information               |
| `-h, --help`       | Show help message                      |

## Examples

### Creating Links

```bash
# Link from current directory
lnk .

# Link from a subdirectory
lnk home

# Link from absolute path
lnk ~/git/dotfiles

# Specify target directory
lnk -t ~ .

# Dry-run to preview changes
lnk -n .

# Add ignore pattern
lnk --ignore '*.swp' .
```

### Removing Links

```bash
# Remove links from source directory
lnk -R .

# Remove links from subdirectory
lnk -R home

# Dry-run to preview removal
lnk -n -R .
```

### Checking Status

```bash
# Show status of links from current directory
lnk -S .

# Show status from subdirectory
lnk -S home

# Show status with verbose output
lnk -v -S .
```

### Pruning Broken Links

```bash
# Remove broken symlinks from current directory
lnk -P

# Remove broken symlinks from specific source
lnk -P home

# Dry-run to preview pruning
lnk -n -P
```

### Adopting Files

```bash
# Adopt files into current directory
lnk -A ~/.bashrc ~/.vimrc

# Adopt into specific source directory
lnk -A -s ~/git/dotfiles ~/.bashrc

# Adopt with dry-run
lnk -n -A ~/.gitconfig
```

### Orphaning Files

```bash
# Remove file from management
lnk -O ~/.bashrc

# Orphan with specific source
lnk -O -s ~/git/dotfiles ~/.bashrc

# Dry-run to preview orphaning
lnk -n -O ~/.config/oldapp
```

## Config Files

lnk supports optional configuration files in your source directory. CLI flags always take precedence over config files.

### .lnkconfig (optional)

Place in source directory. Format: CLI flags, one per line.

```
--target=~
--ignore=local/
--ignore=*.private
--ignore=*.local
```

Each line should be a valid CLI flag. Use `--flag=value` format for flags that take values.

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
- `.lnkconfig`
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

Use: `lnk .` from within the directory

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

Use: `lnk home` to link public configs, or `lnk ~/dotfiles/private` for private configs

### Config File Precedence

Configuration is merged in this order (later overrides earlier):

1. `.lnkconfig` in source directory
2. `.lnkignore` in source directory
3. CLI flags

### Ignore Patterns

lnk supports gitignore-style patterns for excluding files from linking:

- `*.swp` - all swap files
- `local/` - local directory
- `!important.swp` - negation (include this specific file)
- `**/*.log` - all .log files recursively

Patterns can be specified via:

- `.lnkignore` file (one pattern per line)
- `.lnkconfig` file (`--ignore=pattern`)
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
lnk -n .

# 4. Create links for real
lnk .
```

### Adding New Configurations

```bash
# Adopt a new app config into your repository
lnk -A ~/.config/newapp

# This will:
# 1. Move ~/.config/newapp to ~/dotfiles/.config/newapp (from cwd)
# 2. Create symlinks for each file in the directory tree
# 3. Preserve the directory structure

# Or specify source directory explicitly
lnk -A -s ~/dotfiles ~/.config/newapp
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
lnk public

# Link private configs
lnk private

# Or adopt to appropriate location
cd ~/dotfiles/public
lnk -A ~/.bashrc              # Public config to current dir

cd ~/dotfiles/private
lnk -A ~/.ssh/config          # Private config to current dir
```

### Migrating from Other Dotfile Managers

```bash
# 1. Remove existing links from old manager
stow -D home  # Example: GNU Stow

# 2. Create links with lnk
cd ~/dotfiles
lnk .

# lnk creates individual file symlinks instead of directory symlinks,
# so you can gradually migrate and test
```

## Tips

- **Always dry-run first** - Use `-n` to preview changes before making them
- **Check status regularly** - Use `-S` to check for broken links
- **Organize your dotfiles** - Separate public and private configs into subdirectories
- **Leverage .lnkignore** - Exclude build artifacts, local configs, and README files
- **Test on VM first** - When setting up a new machine, test in a VM before production
- **Version your configs** - Keep `.lnkconfig` and `.lnkignore` in git for reproducibility
- **Use verbose mode for debugging** - Add `-v` to see what lnk is doing

## Comparison with Other Tools

### vs. GNU Stow

- **lnk**: Creates individual file symlinks, allows mixing configs from multiple sources
- **stow**: Creates directory symlinks, simpler but less flexible

### vs. chezmoi

- **lnk**: Simple symlinks, no templates, what you see is what you get
- **chezmoi**: Templates, encryption, complex state management

### vs. dotbot

- **lnk**: Flag-based CLI, built-in adopt/orphan operations
- **dotbot**: YAML-based config, more explicit control

lnk is designed for users who:

- Want a simple, flag-based CLI
- Prefer symlinks over copying
- Need to mix public and private configs
- Want built-in adopt/orphan workflows
- Value clarity over configurability

## Troubleshooting

### Broken Links After Moving Dotfiles

```bash
# Remove old links (from original location)
cd /old/path/to/dotfiles
lnk -R .

# Recreate from new location
cd /new/path/to/dotfiles
lnk .
```

### Some Files Not Linking

```bash
# Check if they're ignored
lnk -v .  # Verbose mode shows ignored files

# Check .lnkignore and .lnkconfig
cat .lnkignore
cat .lnkconfig
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
