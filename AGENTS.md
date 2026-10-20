# AGENTS.md

Agent instructions for AI coding assistants working in this repository.
See [agents.md](https://agents.md) for the open format specification.

<!-- repogov:context:begin -->
## Context

Prose language: American English (en-US). Write documentation (.md, .txt), code comments and commit messages in American English, with its spelling and conventions (American spelling: color, behavior, -ize); keep identifiers, code and quoted text as they are.

- Project overview: [README.md](README.md)
- Extended documentation: [docs/](docs/)
- Copilot rule files: [.github/rules/](.github/rules/)
- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)
<!-- repogov:context:end -->

## Nested Instructions

Place an `AGENTS.md` in any subdirectory to provide directory-scoped instructions. Agents load the nearest `AGENTS.md` walking up to the repo root; more specific files take precedence over less specific ones.
