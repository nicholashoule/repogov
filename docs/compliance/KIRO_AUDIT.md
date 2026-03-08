# Kiro CLI Audit

Sources:
- https://kiro.dev/docs/cli/steering/ — verified 2026-03-07
- https://kiro.dev/docs/cli/custom-agents/creating/ — verified 2026-03-07
- https://github.com/aws/amazon-q-developer-cli — deprecation notice verified 2026-03-07

## Background

Kiro CLI is AWS's closed-source successor to the now-deprecated Amazon Q
Developer CLI. It is built on Claude frontier models and targets the same
terminal-based agentic coding workflow. Steering files (`.kiro/steering/`) are
the direct equivalent of rule files in Cursor, Windsurf, and Continue.

See [AMAZONQ_AUDIT.md](AMAZONQ_AUDIT.md) for Amazon Q Developer CLI history.

## Configuration Files

### Workspace Steering

- `.kiro/steering/` — primary multi-file steering directory. All `.md` files
  are loaded automatically in every chat session. Files are plain Markdown
  with no mandatory frontmatter.

  Recommended foundation files (conventional names only — no schema enforcement):

  | Filename | Purpose |
  |----------|---------|
  | `product.md` | Product purpose, target users, business objectives |
  | `tech.md` | Frameworks, libraries, technical constraints |
  | `structure.md` | File organisation, naming conventions, architectural decisions |

  Custom files may use any descriptive name (e.g. `api-standards.md`,
  `testing-patterns.md`).

- `AGENTS.md` at workspace root — also detected and always included.
  Can alternatively be placed inside `~/.kiro/steering/` for global scope.

### Global Steering

- `~/.kiro/steering/` — global steering applied to all workspaces.
  Same `.md` format as workspace steering. Workspace steering takes precedence
  over conflicting global steering.

### Team Steering

Global steering can be pushed to developer machines via MDM/Group Policy or
downloaded from a central repository into `~/.kiro/steering/`. Kiro has no
built-in team-sync mechanism; distribution is manual or tool-assisted.

### Custom Agents

- `.kiro/agents/` (workspace) or `~/.kiro/agents/` (global) — JSON
  configuration files defining custom agents.

  Example agent configuration:
  ```json
  {
    "name": "my-agent",
    "description": "A custom agent for my workflow",
    "tools": ["read", "write"],
    "allowedTools": ["read"],
    "resources": [
      "file://README.md",
      "file://.kiro/steering/**/*.md",
      "skill://.kiro/skills/**/SKILL.md"
    ],
    "prompt": "You are a helpful coding assistant",
    "model": "claude-sonnet-4"
  }
  ```

  Steering files are **not** automatically included in custom agent sessions;
  they must be explicitly listed in `resources`.

### Skills

- `.kiro/skills/` — workspace skills directory. Each skill is a subdirectory
  containing a `SKILL.md` file. Referenced from custom agent `resources` via
  `skill://.kiro/skills/**/SKILL.md`.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `.kiro/steering/*.md` — Markdown steering files (no mandatory frontmatter) |
| `.json` | `.kiro/agents/*.json` — Custom agent configuration |
| `.md` | `.kiro/skills/<name>/SKILL.md` — Skill definition files |

## Limits

No documented line/character limits per steering file. The official best-
practice guidance is to keep files focused on a single domain.

## AGENTS.md Compatibility

Kiro explicitly supports the [AGENTS.md](https://agents.md/) cross-agent
standard. Files placed at the workspace root or inside `~/.kiro/steering/` are
picked up automatically and always included (equivalent to `alwaysApply: true`
in other agents).

## repogov Support Status

Not yet supported. Qualifies for a **Tier 1** preset (see
[AI_AGENTS_AUDIT.md](AI_AGENTS_AUDIT.md#implementation-priority-tiers)).

To add support:

1. Create `DefaultKiroLayout()` in `presets.go`:
   - `Root`: `.kiro`
   - `Dirs`: `steering` → glob `*.md`, min 0, description "Kiro steering files"
   - Optional agents dir: `agents` → glob `*.json`
   - `Naming.Case`: `lowercase`
2. Add `TestInitLayout_KiroSchema` to `init_test.go`.
3. Add `kiro` as a CLI agent name in `cmd/repogov/main.go` mapping to
   `DefaultKiroLayout()`.
4. Scaffold a `cmd/repogov/tmp/kiro/` example directory.
5. Update [AI_AGENTS_AUDIT.md](AI_AGENTS_AUDIT.md) — move this file from
   Planned Backlog to the Supported table.
6. Add Kiro to the Support Matrix and Supported File Extensions tables.
