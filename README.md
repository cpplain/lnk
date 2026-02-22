# lnk

An opinionated symlink manager for dotfiles.

## Key Features

- **Simple CLI** - Flag-based interface with sensible defaults
- **Recursive file linking** - Links individual files throughout directory trees
- **Package-based organization** - Support for multiple packages (public/private configs)
- **Safety first** - Dry-run mode and clear status reporting
- **Flexible configuration** - Optional config files with CLI override
- **No dependencies** - Single binary, stdlib only (git integration optional)

## Installation

```bash
brew install cpplain/tap/lnk
```

## Quick Start

```bash
# From your dotfiles directory
cd ~/git/dotfiles
lnk .                    # Flat repo: link everything
lnk home                 # Nested repo: link home/ package
lnk home private/home    # Multiple packages
```

## Usage

```bash
lnk [options] <packages...>
```

At least one package is required for link operations. Use `.` for flat repository (all files in source directory) or specify subdirectories for nested repository (e.g., `home`, `private/home`).

### Action Flags (mutually exclusive)

| Flag                | Description                        |
| ------------------- | ---------------------------------- |
| `-C, --create`      | Create symlinks (default action)   |
| `-R, --remove`      | Remove symlinks                    |
| `-S, --status`      | Show status of symlinks            |
| `-P, --prune`       | Remove broken symlinks             |
| `-A, --adopt`       | Adopt files into package           |
| `-O, --orphan PATH` | Remove file from management        |

### Directory Flags

| Flag               | Description                                   |
| ------------------ | --------------------------------------------- |
| `-s, --source DIR` | Source directory (default: current directory) |
| `-t, --target DIR` | Target directory (default: `~`)               |

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
# Flat repository (all files in source directory)
lnk .

# Nested repository
lnk home

# Multiple packages
lnk home private/home

# Specify source directory
lnk -s ~/dotfiles home

# Specify target directory
lnk -t ~ home

# Dry-run to preview changes
lnk -n home

# Add ignore pattern
lnk --ignore '*.swp' home
```

### Removing Links

```bash
# Remove links from package
lnk -R home

# Dry-run to preview removal
lnk -n -R home
```

### Checking Status

```bash
# Show status of links in package
lnk -S home

# Show status with verbose output
lnk -v -S home
```

### Pruning Broken Links

```bash
# Remove broken symlinks
lnk -P

# Dry-run to preview pruning
lnk -n -P
```

### Adopting Files

```bash
# Adopt files into package
lnk -A home ~/.bashrc ~/.vimrc

# Adopt with dry-run
lnk -n -A home ~/.gitconfig
```

### Orphaning Files

```bash
# Remove file from management
lnk -O ~/.bashrc

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

lnk recursively traverses your source directories and creates individual symlinks for each file (not directories). This approach:

- Allows multiple packages to map to the same target directory
- Preserves your ability to have local-only files alongside managed configs
- Creates parent directories as needed (never as symlinks)

**Example:** With package `home` mapped from `~/dotfiles/home` to `~`:

```
Source:                              Target:
~/dotfiles/home/
  .bashrc                     →      ~/.bashrc (symlink)
  .config/
    git/
      config                  →      ~/.config/git/config (symlink)
    nvim/
      init.vim                →      ~/.config/nvim/init.vim (symlink)
```

The directories `.config`, `.config/git`, and `.config/nvim` are created as regular directories, not symlinks. This allows you to have local configs in `~/.config/localapp/` that aren't managed by lnk.

### Package Organization

lnk uses a package-based approach. A package is a subdirectory in your source that maps to a target directory. Common patterns:

**Flat Repository:**
```
~/dotfiles/
  .bashrc
  .vimrc
  .gitconfig
```
Use: `lnk .`

**Nested Repository:**
```
~/dotfiles/
  home/          # Public configs
    .bashrc
    .vimrc
  private/
    home/        # Private configs
      .ssh/
        config
```
Use: `lnk home private/home`

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
lnk -n home private/home

# 4. Create links for real
lnk home private/home
```

### Adding New Configurations

```bash
# Adopt a new app config into your repository
lnk -A home ~/.config/newapp

# This will:
# 1. Move ~/.config/newapp to ~/dotfiles/home/.config/newapp
# 2. Create symlinks for each file in the directory tree
# 3. Preserve the directory structure
```

### Managing Public and Private Configs

```bash
# Keep work/private configs separate using git submodule
cd ~/dotfiles
git submodule add git@github.com:you/dotfiles-private.git private

# Structure:
# ~/dotfiles/home/         (public configs)
# ~/dotfiles/private/home/ (private configs via submodule)

# Adopt to appropriate location
lnk -A home ~/.bashrc              # Public config
lnk -A private/home ~/.ssh/config  # Private config
```

### Migrating from Other Dotfile Managers

```bash
# 1. Remove existing links from old manager
stow -D home  # Example: GNU Stow

# 2. Create links with lnk
lnk home

# lnk creates individual file symlinks instead of directory symlinks,
# so you can gradually migrate and test package by package
```

## Tips

- **Always dry-run first** - Use `-n` to preview changes before making them
- **Check status regularly** - Use `-S` to check for broken links
- **Use packages** - Separate public and private configs into different packages
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
# Remove old links
lnk -R home

# Recreate from new location
cd /new/path/to/dotfiles
lnk home
```

### Some Files Not Linking

```bash
# Check if they're ignored
lnk -v home  # Verbose mode shows ignored files

# Check .lnkignore and .lnkconfig
cat .lnkignore
cat .lnkconfig
```

### Permission Denied Errors

```bash
# Check file permissions in source
ls -la ~/dotfiles/home/.ssh

# Files should be readable
chmod 600 ~/dotfiles/home/.ssh/config
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
