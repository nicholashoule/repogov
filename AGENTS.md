# AGENTS.md

Agent instructions for AI coding assistants working in this repository.
See [agents.md](https://agents.md) for the open format specification.

## Context

- Project overview: [README.md](README.md)
- Extended documentation: [docs/](docs/)
- Scoped instruction files: [.github/instructions/](.github/instructions/) (`applyTo` frontmatter)
- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)

## Nested Instructions

Place an `AGENTS.md` in any subdirectory to provide directory-scoped instructions.
Agents load the nearest `AGENTS.md` walking up to the repo root; more specific
files take precedence over less specific ones.

## .github Layout

Standard files and directories for this repository's `.github/` folder:

- `.github/ISSUE_TEMPLATE/`
- `.github/PULL_REQUEST_TEMPLATE/`
- `.github/workflows/`
- `.github/ISSUE_TEMPLATE.md`
- `.github/pull_request_template.md`
- `.github/CONTRIBUTING.md`
- `.github/CODE_OF_CONDUCT.md`
- `.github/SECURITY.md`
- `.github/SUPPORT.md`
- `.github/FUNDING.yml`
- `.github/CODEOWNERS`
- `.github/dependabot.yml`

## Dev Environment

TODO: describe setup steps (dependencies, tools, environment variables).

## Testing

Validate our boundaries by snapshotting the current `.github/` contents, running all init commands, and inspecting the generated files in a temp path (e.g., `./temp`) for every agent flag and config. Confirm that `AGENTS.md` reflects the correct context for the init‑selected agent.

Testing, edge-cases

The AGENTS.md "## Context" section isn't being updated to reflect the respective configured agent. Fix and validate each agent links to the correct directory or directories when we init.

## PR Instructions

TODO: describe pull request conventions (title format, required checks, review expectations).
