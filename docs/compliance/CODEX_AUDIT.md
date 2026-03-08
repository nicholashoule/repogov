# OpenAI Codex (Codex CLI) Audit

Source: https://developers.openai.com/codex/guides/agents-md and https://developers.openai.com/codex/skills
Verified: 2026-03-07

## Configuration Files

### Instructions

- `AGENTS.md` at repo root or any subdirectory — primary instruction file; loaded every session.
  Codex walks the directory tree from the repo root down to the current working directory, loading
  one file per directory (in precedence order: `AGENTS.override.md` > `AGENTS.md` > fallback names).
  Files closer to the current directory take precedence (later in combined prompt).
- `AGENTS.override.md` — override variant; takes precedence over `AGENTS.md` at the same directory level.
  Useful for temporary overrides without modifying the base file.
- Fallback filenames (e.g. `TEAM_GUIDE.md`) can be configured via `project_doc_fallback_filenames` in
  `~/.codex/config.toml`. Codex checks each directory in order: `AGENTS.override.md`, `AGENTS.md`,
  then each fallback name in order.
- Combined instruction size defaults to 32 KiB (`project_doc_max_bytes`); configurable.

### Skills

- `.agents/skills/` — multi-file skills directory; scanned from CWD up to repo root.
  Each skill is a subdirectory containing a `SKILL.md` file with YAML frontmatter (`name`, `description`).
  Optional: `scripts/`, `references/`, `assets/`, `agents/openai.yaml` (UI metadata + policy).
- User-level skills: `~/.agents/skills/`
- Admin-level skills: `/etc/codex/skills/`
- Skills are lazy-loaded: Codex loads only metadata until a skill is activated (explicit `$skill` or
  implicit match on `description`).

### Global Config

- `~/.codex/AGENTS.md` — global instruction defaults applied to every repository.
- `~/.codex/AGENTS.override.md` — global override; takes precedence over global `AGENTS.md`.
- `~/.codex/config.toml` — agent configuration (`project_doc_fallback_filenames`,
  `project_doc_max_bytes`, skill enable/disable, etc.).

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `AGENTS.md`, `AGENTS.override.md`, `SKILL.md` |
| `.toml` | `~/.codex/config.toml` (global config) |
| `.yaml` | `agents/openai.yaml` (optional skill UI metadata) |

## Limits

- Combined `AGENTS.md` instruction chain: 32 KiB by default (`project_doc_max_bytes`); configurable.
- No per-file line limit enforced by repogov currently (no preset exists).

## Preset

None. No repogov preset exists for OpenAI Codex CLI.

## Action Items

- Evaluate adding a `DefaultCodexLayout()` preset to validate `.agents/skills/` and `AGENTS.md`.
- Codex CLI skills follow the open [Agent Skills standard](https://agentskills.io/specification); track
  that spec for changes.
