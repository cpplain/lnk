# cfgman

A fast, reliable dotfile management tool. Manage your configuration files across machines using intelligent symlinks.

## Key Features

- **Single binary** - No dependencies required (git integration optional)
- **Recursive file linking** - Links individual files throughout directory trees
- **Smart directory adoption** - Adopting directories moves all files and creates individual symlinks
- **Flexible configuration** - Support for public and private config repositories
- **Safety first** - Dry-run mode and clear status reporting
- **Bidirectional operations** - Adopt existing files or orphan managed ones

## Installation

```bash
# Build from source
git clone https://github.com/cpplain/cfgman.git
cd cfgman
make install
```

## Quick Start

```bash
# 1. Set up your config repository
mkdir -p ~/dotfiles/{home,private/home}
cd ~/dotfiles
git init

# 2. Create configuration file (optional - cfgman works with built-in defaults)
# Create .cfgman.json if you need custom mappings:
# {
#   "ignore_patterns": [".DS_Store", "*.swp"],
#   "link_mappings": [
#     {"source": "~/dotfiles/home", "target": "~/"},
#     {"source": "~/dotfiles/private/home", "target": "~/"}
#   ]
# }

# 3. Adopt existing configs
cfgman adopt --path ~/.gitconfig --source-dir ~/dotfiles/home
cfgman adopt --path ~/.ssh/config --source-dir ~/dotfiles/private/home

# 4. Create symlinks on new machines
cfgman link
```

## Configuration

cfgman uses a single configuration file `.cfgman.json` in your dotfiles repository that controls linking behavior.

**Note**: cfgman works with built-in defaults and doesn't require a config file. Create `.cfgman.json` only if you need custom ignore patterns or complex link mappings.

### Configuration File (.cfgman.json)

Example configuration:

```json
{
  "ignore_patterns": [".DS_Store", "*.swp", "*~", "Thumbs.db"],
  "link_mappings": [
    {
      "source": "~/dotfiles/home",
      "target": "~/"
    },
    {
      "source": "~/dotfiles/private/home",
      "target": "~/"
    }
  ]
}
```

- **ignore_patterns**: Gitignore-style patterns for files to never link
- **source**: Absolute path to directory containing configs (supports `~/` expansion)
- **target**: Where symlinks are created (usually `~/`)

## Commands

### Basic Commands

```bash
cfgman status                        # Show all managed symlinks
cfgman link [--dry-run]              # Create symlinks from repo to target dirs
cfgman unlink [--dry-run]            # Remove all managed symlinks
cfgman prune [--dry-run]             # Remove broken symlinks
```

### File Operations

```bash
# Adopt a file/directory into your repository
cfgman adopt --path <path> --source-dir <source_dir> [--dry-run]
cfgman adopt --path ~/.gitconfig --source-dir ~/dotfiles/home                    # Adopt to public repo
cfgman adopt --path ~/.ssh/config --source-dir ~/dotfiles/private/home           # Adopt to private repo

# Orphan a file/directory (remove from management)
cfgman orphan --path <path> [--dry-run]
cfgman orphan --path ~/.config/oldapp                    # Stop managing a config
```

### Global Options

```bash
cfgman --version                        # Show version
cfgman help [command]                   # Get help
```

## How It Works

### Recursive File Linking

cfgman recursively traverses your source directories and creates individual symlinks for each file. This approach:

- Allows mixing files from different sources in the same directory
- Preserves your ability to have local-only files alongside managed configs
- Creates parent directories as needed (never as symlinks)

For example, with source `~/dotfiles/home` mapped to `~/`:

- `~/dotfiles/home/.config/git/config` → `~/.config/git/config` (file symlink)
- `~/dotfiles/home/.config/nvim/init.vim` → `~/.config/nvim/init.vim` (file symlink)
- The directories `.config`, `.config/git`, and `.config/nvim` are created as regular directories, not symlinks

### Ignore Patterns

cfgman supports gitignore-style patterns in the `ignore_patterns` field to exclude files from linking. You can add patterns like `.DS_Store`, `*.swp`, and other files you want to exclude.

## Common Workflows

### Setting Up a New Machine

```bash
# 1. Clone your dotfiles
git clone https://github.com/you/dotfiles.git ~/dotfiles
cd ~/dotfiles && git submodule update --init  # If using private submodule

# 2. Create links
cfgman link
```

### Adding New Configurations

```bash
# Adopt a new app config
cfgman adopt --path ~/.config/newapp --source-dir ~/dotfiles/home

# This will move the entire directory tree to your repo
# and create symlinks for each individual file
```

### Managing Sensitive Files

```bash
# Keep work/private configs separate
cfgman adopt --path ~/.ssh/config --source-dir ~/dotfiles/private/home
cfgman adopt --path ~/.config/work-vpn.conf --source-dir ~/dotfiles/private/home
```

## Tips

- Use `--dry-run` to preview changes before making them
- Keep sensitive configs in a separate private directory or git submodule
- Run `cfgman status` regularly to check for broken links
- Use `ignore_patterns` in `.cfgman.json` to exclude unwanted files
- Consider separate source directories for different contexts (work, personal)
- Source paths can use `~/` for home directory expansion
