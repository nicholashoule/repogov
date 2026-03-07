# AI Agent Setup Audit

Reference file for tracking supported AI agent configuration patterns.
Consult and update this file when adding or modifying agent support in repogov.

## Sources

- GitHub Copilot: https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot
- Cursor: https://cursor.com/docs/context/rules
- Claude Code: https://code.claude.com/docs/en/settings and https://code.claude.com/docs/en/memory
- Windsurf: docs unavailable (all documented URLs return 404 as of 2026-03-07)

## Support Matrix

| Agent | Root Config | Multi-file Dir | repogov Preset | Verified |
|-------|-------------|----------------|----------------|:--------:|
| GitHub Copilot | `.github/copilot-instructions.md` | `.github/instructions/*.instructions.md` | `DefaultGitHubLayout()` | Yes |
| Cursor | none (use rules dir) | `.cursor/rules/*.md` or `*.mdc` | `DefaultCursorLayout()` | Yes |
| Windsurf | `.windsurfrules` (legacy) | `.windsurf/rules/*.md` | `DefaultWindsurfLayout()` | No |
| Claude Code | `CLAUDE.md` or `.claude/CLAUDE.md` | `.claude/rules/*.md`, `.claude/agents/*.md` | `DefaultClaudeLayout()` | Yes |
| AGENTS.md (all agents) | `AGENTS.md` (root or nested) | none | cross-platform, seeded by `init` | Yes |
| Aider | `.aider.conf.yml` | none (single file) | none | n/a |
| OpenAI Codex | `AGENTS.md` | none (single file) | none | n/a |

All four preset agents enforce a 300-line limit on scoped instruction files.
`AGENTS.md` is governed at 200 lines (root) and the default (300) for nested files.

## Per-Agent Details

### GitHub Copilot

Source: official GitHub docs (verified 2026-03-07)

- `.github/copilot-instructions.md` -- repo-wide context; 50-line limit.
- `.github/instructions/*.instructions.md` -- path-specific rules; `applyTo` frontmatter glob.
  Subdirectories under `.github/instructions/` are supported for organization.
- `.github/prompts/*.prompt.md` -- reusable prompt templates.
- `AGENTS.md` anywhere in the repo -- agent-mode instructions (shared standard with Codex/Cursor).
- Seeded by `init` with default instruction files and `copilot-instructions.md`.

### Cursor

Source: official Cursor docs (verified 2026-03-07)

- `.cursor/rules/` -- project rules; version-controlled.
  - Both `.md` (plain) and `.mdc` (with YAML frontmatter) extensions are supported.
  - `.mdc` frontmatter: `description`, `globs`, `alwaysApply`.
  - Files can be organized into subdirectories within `.cursor/rules/`.
- `AGENTS.md` at project root or subdirectory -- simple alternative to `.cursor/rules/`.
- Official best-practice size limit is 500 lines; repogov enforces 300.
- `DefaultCursorLayout()` uses an empty glob for `rules/` to accept both `.md` and `.mdc`.
- `init` creates the `.cursor/rules/` directory.

### Windsurf (Codeium)

Source: unverified -- Windsurf docs returning 404 as of 2026-03-07.

- `.windsurfrules` -- documented legacy single-file format.
- `.windsurf/rules/*.md` -- reported multi-file format; `applyTo` frontmatter.
- `DefaultWindsurfLayout()` validates `.windsurf/rules/` with `*.md` glob.
- `init` creates the `.windsurf/rules/` directory.
- ACTION: re-verify against official Windsurf/Codeium docs when available.

### Claude Code (Anthropic)

Source: official Claude Code docs (verified 2026-03-07)

- `CLAUDE.md` at repo root OR `.claude/CLAUDE.md` -- primary instruction file; loaded every
  session. Target under 200 lines. Use `@path` imports to reference other files.
- `.claude/rules/*.md` -- scoped instruction files; optional `paths` YAML frontmatter for
  per-file-pattern scoping. Subdirectories supported. Governed at 300 lines.
- `.claude/agents/*.md` -- subagent definitions with YAML frontmatter. Governed at 300 lines.
- `.claude/settings.json` -- project settings (permissions, env vars, hooks, MCP servers).
  Hooks are defined here under the `hooks` key -- NOT as a separate directory.
- `.claude/settings.local.json` -- personal overrides; gitignored.
- `.mcp.json` at repo root -- project-scoped MCP server configuration (outside `.claude/`).
- `DefaultClaudeLayout()` validates `.claude/` with `rules/` and `agents/` dirs.
- `init` creates the `.claude/` directory structure.

### Aider

- `.aider.conf.yml` -- primary configuration at repo root.
- `--read` accepts any file (e.g., `CONVENTIONS.md`) as additional context.
- No multi-file directory pattern. No repogov preset.

### OpenAI Codex / ChatGPT Agents

- `AGENTS.md` at repo root or subdirectory -- auto-loaded by Codex agents.
  Also recognized by GitHub Copilot, Cursor, and Claude Code.
- No multi-file directory pattern. No repogov preset.

### AGENTS.md (Cross-Agent Open Standard)

Source: https://agents.md (verified 2026-03-07)

- `AGENTS.md` at the repo root -- loaded by all major coding agents as a
  "README for agents". Plain Markdown; no frontmatter required.
- **Nested scoping**: an `AGENTS.md` in any subdirectory provides
  directory-scoped instructions. Agents walk up from the current file to the
  repo root and load every `AGENTS.md` found; the closer file takes precedence.
- Typical sections: context/overview, dev environment tips, testing, PR
  instructions. Links to `docs/` and platform instruction dirs are encouraged.
- repogov governs `AGENTS.md` at 200 lines; nested files use the default (300).
- `init` seeds a `AGENTS.md` at the repo root for all platforms, with links to
  `docs/`, `README.md`, and the schema's instruction/rules directory.

## Maintenance Notes

When a new agent version introduces a multi-file directory pattern:

1. Add a `Default<Agent>Layout()` function to `presets.go`.
2. Add applicable glob rules to `DefaultConfig()` in `repogov.go`.
3. Update `.github/repogov-config.json` rules to match.
4. Add `TestInitLayout_<Agent>Schema` to `init_test.go`.
5. Update the matrix and per-agent section above with the source URL and date.

When an existing agent deprecates or changes its config format:

1. Update the per-agent section with source URL and date verified.
2. Adjust the preset `Optional` list or `Dirs` map as needed.
3. Add `Naming.Exceptions` for any mandated uppercase filenames.
