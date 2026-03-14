# Kiro CLI Audit

Sources:
- https://kiro.dev/docs/steering/ — verified 2026-03-13
- https://kiro.dev/docs/cli/custom-agents/creating/ — verified 2026-03-13
- https://github.com/aws/amazon-q-developer-cli — deprecation notice verified 2026-03-07

## Background

Kiro CLI is AWS's closed-source successor to the now-deprecated Amazon Q
Developer CLI. It is built on Claude frontier models and targets the same
terminal-based agentic coding workflow. Steering files (`.kiro/steering/`) are
the direct equivalent of rule files in Cursor, Windsurf, and Continue.

See [AMAZONQ_AUDIT.md](AMAZONQ_AUDIT.md) for Amazon Q Developer CLI history.

## Configuration Files

### Workspace Steering

- `.kiro/steering/` — primary multi-file steering directory. Steering files are
  plain Markdown with optional YAML frontmatter. Whether a file is loaded
  depends on its **inclusion mode** (`inclusion:` frontmatter key; default: `always`).

  **Inclusion modes:**

  | Mode | Frontmatter | Behaviour |
  |------|------------|----------|
  | `always` (default) | `inclusion: always` (or omit) | Loaded in every interaction |
  | `fileMatch` | `inclusion: fileMatch` + `fileMatchPattern: "*.tsx"` | Loaded when a context file matches the pattern (string or array) |
  | `manual` | `inclusion: manual` | Invoked via `#steering-file-name` in chat or `/name` slash command |
  | `auto` | `inclusion: auto` + `name:` + `description:` | Model decides when relevant; also available as a slash command |

  Example for a scoped steering file:
  ```yaml
  ---
  inclusion: fileMatch
  fileMatchPattern: "**/*.go"
  ---
  ```

  **File references:** Embed live workspace files inline with
  `#[[file:relative_file_name]]` to keep steering context up-to-date
  without copy-pasting.

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
| `.md` | `.kiro/steering/*.md` — Markdown with optional `inclusion:` frontmatter |
| `.json` | `.kiro/agents/*.json` — Custom agent configuration |
| `.md` | `.kiro/skills/<name>/SKILL.md` — Skill definition files |

## Limits

No documented line/character limits per steering file. The official best-
practice guidance is to keep files focused on a single domain.

## Memory Configuration

Kiro has no dedicated runtime memory file. Context persistence is handled through
steering files in `.kiro/steering/`. Only files with `inclusion: always` (the default)
or matching `inclusion: fileMatch` patterns are loaded automatically; `manual` and
`auto` files require explicit invocation.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Project (always) | `.kiro/steering/*.md` with `inclusion: always` | Yes — every interaction |
| Project (scoped) | `.kiro/steering/*.md` with `inclusion: fileMatch` | Yes — when context files match |
| Project (on-demand) | `.kiro/steering/*.md` with `inclusion: manual`/`auto` | No — invoked explicitly |
| Global | `~/.kiro/steering/*.md` | Yes — loaded in every workspace |
| Agent-specific | `.kiro/agents/*.json` | Only in custom agent sessions (`resources` list) |

For always-on project context, use default `inclusion: always` steering files.
For cross-project instructions, use `~/.kiro/steering/`.
There is no `memory.md` equivalent that Kiro writes to automatically.

## AGENTS.md Compatibility

Kiro explicitly supports the [AGENTS.md](https://agents.md/) cross-agent
standard. Files placed at the workspace root or inside `~/.kiro/steering/` are
picked up automatically and always included (equivalent to `alwaysApply: true`
in other agents).

## Seeded Files

`init -agent kiro` creates the `.kiro/steering/` directory and seeds:

| File | Purpose |
|------|---------|
| `general.md` | General project conventions (no mandatory frontmatter) |

## Limits

No documented line/character limits. repogov enforces 300 lines per `.kiro/steering/*.md` file.

## Preset

`DefaultKiroLayout()` in `presets.go`. Validates `.kiro/steering/` with `*.md` glob.
CLI agent name: `kiro`.
