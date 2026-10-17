# Claude Code (Anthropic) Audit

Source: https://code.claude.com/docs/en/settings and https://code.claude.com/docs/en/memory
Verified: 2026-10-17
(settings, memory, permissions, skills pages fetched 2026-10-17)

## Configuration Files

- `CLAUDE.md` at repo root OR `.claude/CLAUDE.md` — primary instruction file; loaded every
  session. Target under 200 lines. Use `@path` imports to reference other files.
- `.claude/rules/*.md` — scoped instruction files; optional `paths` YAML frontmatter for
  per-file-pattern scoping. Subdirectories and symlinks supported.
- `.claude/agents/*.md` — subagent definitions with YAML frontmatter. Governed at 300 lines.
- `.claude/settings.json` — project settings (permissions, env vars, hooks, MCP servers).
  Hooks are defined here under the `hooks` key — NOT as a separate directory.
- `.claude/settings.local.json` — personal overrides; gitignored.
  `claudeMdExcludes` key here can suppress specific `CLAUDE.md` files in monorepos.
- `CLAUDE.local.md` at repo root - personal per-project instructions, loaded after `CLAUDE.md`
  at the same level; should be gitignored. NOT governed by repogov (outside `.claude/`).
- `.claude/skills/<name>/SKILL.md` - skills (frontmatter: `description`, `when_to_use`,
  `allowed-tools`, `paths`, `context`, etc.; keep `SKILL.md` under 500 lines). Nested
  `<subdir>/.claude/skills/` also supported.
- `.claude/commands/*.md` - legacy custom commands, now merged into skills (still work;
  docs recommend skills for new work).
- `.mcp.json` at repo root — project-scoped MCP server configuration (outside `.claude/`).

### User-level rules

- `~/.claude/CLAUDE.md` — global instruction file; applied to all projects.
- `~/.claude/rules/` — user-level rules directory (same format as `.claude/rules/*.md`).
  Separate path from `~/.claude/CLAUDE.md`; rules here are user-personal, not project files.

### Managed policy CLAUDE.md (enterprise / IT)

| OS | Path |
|----|------|
| macOS | `/Library/Application Support/ClaudeCode/CLAUDE.md` |
| Linux | `/etc/claude-code/CLAUDE.md` |
| Windows | `C:\Program Files\ClaudeCode\CLAUDE.md` |

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `CLAUDE.md`, `rules/*.md`, `agents/*.md` |
| `.json` | `settings.json` / `settings.local.json` for permissions and hooks |

## Memory Configuration

Claude Code has two separate memory mechanisms:

### Static CLAUDE.md instructions

`CLAUDE.md` (or `.claude/CLAUDE.md`) is the primary instruction file — loaded every session
without any auto-updating by the agent. This is the correct place for stable project context.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Project (primary) | `CLAUDE.md` or `.claude/CLAUDE.md` | Yes — loaded every session |
| Project (scoped) | `.claude/rules/*.md` | Yes — matched by `paths:` frontmatter glob |
| Settings / hooks | `.claude/settings.json` | Yes — permissions, env vars, hooks |
| Personal overrides | `.claude/settings.local.json` | Yes — gitignored |
| User global | `~/.claude/CLAUDE.md` and `~/.claude/rules/*.md` | Yes — applied to all projects |
| System policy | OS-specific managed CLAUDE.md (see Configuration Files) | Yes — IT-managed, read-only |

### Auto memory system

Claude Code can automatically write and maintain a memory file as it works across sessions.
This is **machine-local and git-repo-scoped** — never committed to version control.

| Item | Detail |
|------|---------|
| Storage location | `~/.claude/projects/<project-hash>/memory/` |
| Index file | `MEMORY.md` — first 200 lines or 25KB (whichever first) loaded at every session start |
| Topic files | e.g. `debugging.md`, `testing.md` — loaded on demand when relevant |
| Trigger | Claude writes to memory when it learns something useful for future sessions |
| Toggle | `autoMemoryEnabled` in settings (default: on), or env `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` |
| Inspect / edit | `/memory` slash command lists all active CLAUDE.md files and memory entries |

Repogov does not govern auto-memory files (they are outside the repo). The CLAUDE.md
instruction files in the repo remain the authoritative team-shared context layer.

Claude Code uses `@path` import syntax inside `CLAUDE.md` to compose larger context
from smaller focused files.

## AGENTS.md Interop

Claude Code reads a repo's `AGENTS.md` only when no `CLAUDE.md` / `.claude/CLAUDE.md` /
`CLAUDE.local.md` exists in the working directory or above (default `claude-md-or-agents-md`).
The `claudeMd` key in `managed-settings.json` can also embed managed instructions.

## Settings Precedence and Permission Syntax

Settings precedence (highest first): managed settings, `claude --settings`,
`.claude/settings.local.json`, `.claude/settings.json`, `~/.claude/settings.json`.
Permission rules evaluate deny, then ask, then allow; first match wins, so an allow rule
cannot carve an exception out of a deny rule.

Bash rule syntax: the current form is `Bash(npm run *)` (space + wildcard; `*` may appear at
any position). The legacy `Bash(npm run:*)` suffix is documented as equivalent and is only
recognized at the end of a pattern; it is not formally deprecated.

## CheckClaudeSettings (repogov)

`CheckClaudeSettings(root)` in `settings.go` is a read-only audit of `.claude/settings.json`
and `.claude/settings.local.json` (strict allowlist; symlinks and non-regular files are not
followed). It reports line/rule counts, invalid JSON (Fail), duplicate rules, rules made
redundant by a broader `:*` prefix rule, and allow entries overridden by deny (Warn), each
with a `-- FIX:` hint. `.claude/.credentials.json` is NEVER opened or parsed; its presence is
reported by filename only. Repogov never creates or modifies settings files. Redundancy
detection targets the `:*` form, not the space-wildcard `Bash(x *)` form.

## Seeded Files

`init -agent claude` creates the `.claude/` directory structure and seeds:

| File | Notes |
|------|-------|
| `CLAUDE.md` | Primary instruction file (Required). Generated by `createClaudeMd`; the generic required-file loop skips it to avoid writing placeholder content before `createClaudeMd` runs. |
| `rules/general.md` | Frontmatter: `paths: ["**"]`; general project conventions. |

## Limits

- `.claude/CLAUDE.md`: 200 lines (enforced via `DefaultConfig().Files`, matching per-session instruction file guidance).
- `.claude/rules/*.md` and `.claude/agents/*.md`: 300 lines.

## Preset

`DefaultClaudeLayout()` in `presets.go`. Validates `.claude/` with `rules/` and `agents/` dirs.
`CLAUDE.md` is listed as **Required** in the schema (not Optional); moving it to Optional would
cause `repogov layout` to exit non-zero for any Claude repo.
CLI agent name: `claude`.
