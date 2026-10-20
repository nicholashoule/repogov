# OpenCode Audit

Sources (all fetched 2026-10-17):
- https://opencode.ai/v2/docs (overview, v2 docs index)
- https://opencode.ai/v2/docs/config, /skills, /agents, /commands, /instructions
- https://opencode.ai/docs/rules/ (v1 docs; banner says "New OpenCode v2 is now available")
Verified: 2026-10-17

OpenCode is an open-source terminal, desktop and web coding agent. It is actively developed and
**not deprecated**. A v2 line exists (Docker `2.0.0`, Homebrew `opencode-v2`); the mise note implies
"OpenCode 1" is still installable. This audit follows the **v2** docs; v1 differences are called out.

## Configuration Files

### Instructions (v2, https://opencode.ai/v2/docs/instructions/)

- **`AGENTS.md` only.** "OpenCode V2 recognizes `AGENTS.md` only" -- no `CLAUDE.md` fallback. (v1 read
  `CLAUDE.md` when no `AGENTS.md` existed and honored `OPENCODE_DISABLE_CLAUDE_CODE`; the v2 page
  mentions neither. v1 source: https://opencode.ai/docs/rules/.)
- Global: `~/.config/opencode/AGENTS.md`. Project: every `AGENTS.md` from the workspace up toward the
  home directory (outside home, the search stops at the project root). Nested files are discovered when the
  agent reads a file or lists a directory, load nearest-first, and are deduplicated.
- Nested files are **combined, not overridden**: "OpenCode combines the files and does not resolve
  conflicts between them."
- `instructions` in `opencode.jsonc`: relative paths, globs, `~/` and absolute paths, and
  `http://`/`https://` URLs. Relative paths and globs are searched from the workspace up to the project
  root. URLs are fetched at config load with a 5 second timeout; missing files are skipped, and a URL that
  fails is skipped or keeps its last fetched content.
- Prompt order: agent/provider system prompt, Code Mode guidance, MCP/reference/skill guidance, global and
  project `AGENTS.md`, configured `instructions` in listed order, built-in date/environment context,
  then API session instructions.
- `OPENCODE_DISABLE_PROJECT_CONFIG=1` skips project `AGENTS.md` discovery (global still loads).
- No size limits are documented.

### Config

- `opencode.json` or `opencode.jsonc` (comments and trailing commas allowed); schema
  `https://opencode.ai/config.json`.
- Locations and precedence (low to high): global `~/.config/opencode/`, project root, nested package
  `opencode.json(c)`, system (`/etc/opencode`, `/Library/Application Support/opencode`,
  `%ProgramData%\opencode`), then managed settings/MDM. Non-conflicting settings merge.
- Project config may live in the project directory or in `.opencode/`; every discovered `.opencode`
  config overrides direct configs, so use one form consistently.

### `.opencode/` directory (v2)

| Path | Contents |
|------|----------|
| `.opencode/agents/<name>.md` | Custom agents; frontmatter `description`, `mode` (`primary`/`subagent`/`all`), `model`, `permissions` |
| `.opencode/commands/*.md` | Custom commands (nested folders become `/`-separated names); `description`, `agent`, `model`, `subagent` (`subtask` deprecated alias) |
| `.opencode/skills/` | Skills: root `*.md` or nested `SKILL.md`; ID derives from the path, not frontmatter `name` |

Global equivalents live under `~/.config/opencode/{agents,commands,skills}/`. Skills are also read from
`.claude/skills` and `.agents/skills` (project) and `~/.claude/skills`, `~/.agents/skills` (global).

### Permissions (v2)

Rules are objects `{action, resource, effect}` with `effect` of `allow`, `ask` or `deny`; wildcards in
`action` and `resource`; **last matching rule wins**. Global `permissions` apply before per-agent
rules. v2 docs say not to use legacy top-level `permission`/`tools` fields in new agent config.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `AGENTS.md`, `CLAUDE.md`, `agents/*.md`, `commands/*.md`, `SKILL.md`, root skill files |
| `.json`, `.jsonc` | `opencode.json(c)` -- config, permissions, MCP, `instructions` |

## Support-Criteria Assessment

| Criterion | Result | Basis |
|-----------|:------:|-------|
| C1 File-based | Pass | `AGENTS.md`, `.opencode/` directories |
| C2 Documented stable path | Pass | v2 docs document every path; v1 differs (CLAUDE.md fallback), so pin the major version |
| C3 Markdown content | Pass | All instruction, agent, command and skill files are Markdown |
| C4 Scannable layout | Pass | Fixed directories under `.opencode/` |

**Verdict: qualifies (Tier 2).** A preset is viable now; v2 reads `AGENTS.md` only, so a preset should not require `CLAUDE.md`.

## Security Notes

- `opencode.json(c)` carries permissions, MCP servers and `instructions`; treat it like Claude's
  `settings.json` -- audit read-only, never scaffold.
- `instructions` accepts remote URLs, so an instruction file can come from outside the repo. A governance
  check could flag `http(s)://` entries.
- Skills are also read from `.claude/skills` and `~/.claude/skills`, so Claude-oriented skills affect
  OpenCode. In v1, `CLAUDE.md` was also a fallback (v2 reads `AGENTS.md` only).

## Action Items

- Done (v0.9.0): the root layout recognizes `opencode.json`/`opencode.jsonc` and treats `.opencode/`,
  `.gemini/`, `.agents/` and `.codex/` as managed directories, so they are no longer reported as unexpected.
- Decide on `DefaultOpenCodeLayout()` covering `AGENTS.md`, `opencode.json(c)`, `.opencode/agents/`,
  `.opencode/commands/` and `.opencode/skills/`.
- Confirm whether `.opencode/themes/`, `tools/` and `plugins/` exist in v2 (only `agents`, `commands`,
  `skills`, `plugins` are documented under the system directory).
