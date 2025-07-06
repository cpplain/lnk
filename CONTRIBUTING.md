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

### CLI Design Guidelines

This project follows the principles outlined in [cpplain/cli-design](https://github.com/cpplain/cli-design).

#### Core Principles

1. **Obvious Over Clever**: Make the most intuitive path the easiest to follow
2. **Helpful Over Minimal**: Provide clear guidance and helpful error messages
3. **Consistent Over Special**: Follow established CLI conventions
4. **Human-First, Machine-Friendly**: Prioritize human usability while ensuring scriptability

#### When Adding Commands

- Use clear, descriptive command names (e.g., `create-links` not `link`)
- Provide comprehensive help text with examples
- Support `--dry-run` for all destructive operations
- Use consistent flag naming across commands
- Include helpful error messages that guide users to solutions
- Ensure all commands work both interactively and in scripts

#### Additional Resources

- [clig.dev](https://clig.dev/) - Command Line Interface Guidelines
- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide)
- [GNU Standards for Command Line Interfaces](https://www.gnu.org/prep/standards/standards.html#Command_002dLine-Interfaces)

Thank you for your understanding.
