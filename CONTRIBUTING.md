# Contributing to cfgman

cfgman is an opinionated personal project designed to meet specific workflow needs.

While contributions may be considered, please note:

- This project reflects personal preferences and workflows
- There is no guarantee that contributions will be accepted
- Features and changes are driven by the maintainer's needs
- The project may not follow conventional open source practices

If you find cfgman useful, you're welcome to:

- Fork it for your own needs
- Report bugs via GitHub issues
- Share ideas, though implementation is at maintainer's discretion

## Development Standards

### Commit Messages

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages.

#### Format

```
type[(optional scope)]: description

[optional body]

[optional footer(s)]
```

#### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to the build process or auxiliary tools

#### Examples

```
feat: add support for multiple link mappings

fix: prevent race condition during link creation

docs: add examples to README

feat(adopt): allow adopting entire directories

fix!: change config file format to JSON

BREAKING CHANGE: config files must now use .cfgman.json extension
```

Thank you for your understanding.
