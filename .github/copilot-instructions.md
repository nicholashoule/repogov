# Copilot Instructions

This file provides repository-level context to GitHub Copilot and compatible AI coding agents.

## Scoped Instructions

See modular instruction files in `.github/rules/` for scoped rules.
Use the `*.md` naming convention and set `applyTo` in the YAML frontmatter (e.g. `applyTo: "**/*.go"`) to scope each file.

Long-form project context lives in [README.md](../README.md) and [docs/](../docs/).

## File Constraints

**`.github` File Limit**: All files in `.github/` must not exceed the configured line limit (see `repogov-config.json`). Use `docs/` for detailed explanations.

**Handling the line limit (priority order):**

1. **Refactor First** - Remove redundancy, condense verbose explanations
2. **Move to `docs/`** - Relocate detailed content to `docs/` directory
3. **Link, Don't Repeat** - Reference external docs instead of duplicating
4. **Split Only When Necessary** - Only when content is a distinct concern; use cohesive files, descriptive names, and cross-references

Run checks: `go run ./cmd/repogov -agent copilot`
Re-scaffold missing files: `go run ./cmd/repogov -agent copilot init`

## File Naming Conventions

**Prefer lowercase filenames** in `docs/` and `.github/` directories:

- Use `kebab-case` or `snake_case` for multi-word filenames
- Exception: `*_AUDIT.md` files in `docs/` may use uppercase
- Exception: GitHub-mandated filenames remain uppercase (e.g., `CODEOWNERS`)

## Repository Conventions

- Follow existing code style and patterns
- Keep files within configured line limits (see repogov-config.json)
- Write tests for new functionality
