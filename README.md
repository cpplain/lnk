# cfgman

A fast, reliable dotfile management tool. Manage your configuration files across machines using intelligent symlinks.

## Key Features

- **Single binary** - No dependencies required (git integration optional)
- **File-level and directory-level linking** - Links files by default and directories by configuration
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

# 2. Initialize cfgman in your repository
cfgman init
# Edit .cfgman.json to configure your mappings

# 3. Adopt existing configs
cfgman adopt ~/.gitconfig home
cfgman adopt ~/.ssh/config private/home

# 4. Create symlinks on new machines
cfgman create-links
```

## Configuration

cfgman uses a single configuration file `.cfgman.json` in your dotfiles repository that controls linking behavior.

**Important**: cfgman must be run from within the repository directory containing `.cfgman.json`.

### Configuration File (.cfgman.json)

Example configuration (after editing the template created by `cfgman init`):

```json
{
  "ignore_patterns": [".DS_Store", "*.swp", "*~", "Thumbs.db"],
  "link_mappings": [
    {
      "source": "home",
      "target": "~/",
      "link_as_directory": [".config/nvim"]
    },
    {
      "source": "private/home",
      "target": "~/",
      "link_as_directory": [".ssh"]
    }
  ]
}
```

- **ignore_patterns**: Gitignore-style patterns for files to never link
- **source**: Directory in your repo containing configs
- **target**: Where symlinks are created (usually `~/`)
- **link_as_directory**: Directories to link as complete units instead of individual files

## Commands

### Configuration Commands

```bash
cfgman init                          # Create a minimal .cfgman.json template
```

### Basic Commands

```bash
cfgman status                        # Show all managed symlinks
cfgman create-links [--dry-run]      # Create symlinks from repo to home
cfgman remove-links [--dry-run]      # Remove all managed symlinks
cfgman prune-links                   # Remove broken symlinks
```

### File Operations

```bash
# Adopt a file/directory into your repository
cfgman adopt <path> [source_dir] [--dry-run]
cfgman adopt ~/.gitconfig home                    # Adopt to public repo
cfgman adopt ~/.ssh/config private/home           # Adopt to private repo

# Orphan a file/directory (remove from management)
cfgman orphan <path> [--dry-run]
cfgman orphan ~/.config/oldapp                    # Stop managing a config
```

### Global Options

```bash
cfgman --version                        # Show version
cfgman help [command]                   # Get help
```

## How It Works

### File-Level and Directory-Level Linking

By default, cfgman links individual files rather than entire directories. When you need to link entire directories (like `.config/nvim`), add them to `link_as_directory` in your `.cfgman.json`.

### Ignore Patterns

cfgman supports gitignore-style patterns in the `ignore_patterns` field to exclude files from linking. The `cfgman init` command creates an empty `ignore_patterns` array that you can populate with patterns like `.DS_Store`, `*.swp`, and other files you want to exclude.

## Common Workflows

### Setting Up a New Machine

```bash
# 1. Clone your dotfiles
git clone https://github.com/you/dotfiles.git ~/dotfiles
cd ~/dotfiles && git submodule update --init  # If using private submodule

# 2. Create links (must be run from repository directory)
cd ~/dotfiles
cfgman create-links
```

### Adding New Configurations

```bash
# Adopt a new app config (from repository directory)
cd ~/dotfiles
cfgman adopt ~/.config/newapp home

# If it's a directory with plugins/state, link as directory
# You'll be prompted during adoption, or edit .cfgman.json manually
```

### Managing Sensitive Files

```bash
# Keep work/private configs separate
cfgman adopt ~/.ssh/config private/home
cfgman adopt ~/.config/work-vpn.conf private/home
```

## Tips

- Always run cfgman commands from your repository directory
- Use `--dry-run` to preview changes before making them
- Keep sensitive configs in a separate private directory or git submodule
- Run `cfgman status` regularly to check for broken links
- Use `ignore_patterns` in `.cfgman.json` to exclude unwanted files
- Consider separate source directories for different contexts (work, personal)
