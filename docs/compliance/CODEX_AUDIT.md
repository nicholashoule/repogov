# OpenAI Codex (Codex CLI) Audit

Source: https://developers.openai.com/codex/guides/agents-md and https://developers.openai.com/codex/skills
Verified: 2026-10-17

Both source URLs now 308-redirect to https://learn.chatgpt.com/docs/agent-configuration/agents-md and
https://learn.chatgpt.com/docs/build-skills (fetched). Config reference
(https://learn.chatgpt.com/docs/config-file/config-reference) was read only to ~100k of 157k characters;
hooks and subagents pages were fetched on 2026-10-17 (https://learn.chatgpt.com/docs/hooks and
/docs/agent-configuration/subagents) and rules (/docs/agent-configuration/rules); all
project-level surfaces below are now verified.
The product is not deprecated.

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
- Scan covers `$CWD/.agents/skills`, parent folders up to `$REPO_ROOT/.agents/skills`; symlinks followed;
  duplicate `name`s are not merged. `agents/openai.yaml` has `interface`,
  `policy.allow_implicit_invocation` (default true) and `dependencies.tools`.
- Startup skills list budget is ~2% of context (8,000 chars if unknown); `skills.max_context_tokens`
  (up to 10,000) adjusts it. Plugins bundle skills (+ optional MCP) for distribution.
- Skills are lazy-loaded: Codex loads only metadata until a skill is activated (explicit `$skill` or
  implicit match on `description`).

### Global Config

- `~/.codex/AGENTS.md` — global instruction defaults applied to every repository.
- `~/.codex/AGENTS.override.md` — global override; takes precedence over global `AGENTS.md`.
- `~/.codex/config.toml` — agent configuration (`project_doc_fallback_filenames`,
  `project_doc_max_bytes`, `project_root_markers`, `[[skills.config]]` path/enabled to disable a skill).
  `CODEX_HOME` overrides `~/.codex`. Only the first non-empty file per directory is used; discovery
  runs once per session.
- `.codex/config.toml` — project-scoped config; loaded only for trusted projects
  (`projects.<path>.trust_level`). Cannot override `model_provider(s)`, `openai_base_url`,
  `chatgpt_base_url`, `notify`, `profile(s)`, `otel`. Untrusted projects skip project `.codex/` layers
  (config, hooks, rules).
- Hooks: `<repo>/.codex/hooks.json` or inline `[hooks]` in `<repo>/.codex/config.toml` (also under
  `~/.codex/`). "Project-local hooks load only when the project `.codex/` layer is trusted"; user and
  system hooks still load in untrusted projects.
- Custom agents: standalone TOML files, one agent per file, in `.codex/agents/` (project) or
  `~/.codex/agents/` (personal); required fields `name`, `description`, `developer_instructions`.
  The official docs do not state a trust requirement for agent files.
- Rules: `*.rules` files in a `rules/` folder under each active config layer -- user
  `~/.codex/rules/` (TUI allow-list entries land in `~/.codex/rules/default.rules`) and project
  `<repo>/.codex/rules/`. Written in Starlark using `prefix_rule()` with `pattern`, `decision`
  (`allow`, `prompt`, `forbidden`), optional `justification` and `match`/`not_match` examples.
  They control which commands Codex may run **outside the sandbox**; the most restrictive match
  wins. Project-local rules "load only when the project `.codex/` layer is trusted". Rules are
  enforcement (permissions), distinct from `AGENTS.md`, which is guidance. The rules page does not
  describe any link between the two.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `AGENTS.md`, `AGENTS.override.md`, `SKILL.md` |
| `.rules` | `.codex/rules/*.rules` and `~/.codex/rules/*.rules` (Starlark command-execution policy) |
| `.toml` | `~/.codex/config.toml` (global), `.codex/config.toml` (trusted project) |
| `.json` | `.codex/hooks.json` (hooks; project layer requires trust) |
| `.yaml` | `agents/openai.yaml` (optional skill UI metadata) |

## Limits

- Combined `AGENTS.md` instruction chain: 32 KiB by default (`project_doc_max_bytes`); configurable.
- No per-file line limit enforced by repogov currently (no preset exists).

## Preset

None. No repogov preset exists for OpenAI Codex CLI.

## Action Items

- Update source links to the learn.chatgpt.com docs domain.
- Hooks and rules project paths verified 2026-10-17. If a preset is built, treat `.codex/rules/` and
  `.codex/hooks.json` as permission-bearing files (analogous to Claude `settings.json`): audit
  read-only, never scaffold them.
- Evaluate adding a `DefaultCodexLayout()` preset to validate `.agents/skills/` and `AGENTS.md`.
- Codex CLI skills follow the open [Agent Skills standard](https://agentskills.io/specification); track
  that spec for changes.
