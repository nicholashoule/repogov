# Cursor Audit

Source: https://cursor.com/docs/context/rules
Verified: 2026-03-13

## Configuration Files

- `.cursor/rules/` — project rules; version-controlled.
  - Both `.md` (plain) and `.mdc` (with YAML frontmatter) extensions are supported.
  - `.mdc` frontmatter keys: `description`, `globs`, `alwaysApply`.
  - Files can be organized into **subdirectories** within `.cursor/rules/` for grouping.
- `AGENTS.md` at project root or subdirectory — simple alternative to `.cursor/rules/`.

### Team Rules (Team/Enterprise)

Cursor Team and Enterprise plans support dashboard-managed rules:
- Defined and edited in the Cursor dashboard (not local files)
- Applied to all team members automatically (can be enforced)
- **Remote rules via GitHub** — rules can also be imported and synced from any GitHub
  repository, enabling a centralized team rule source outside the project repo

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | Plain markdown rules |
| `.mdc` | Supports YAML frontmatter (`description`, `globs`, `alwaysApply`) |

## Seeded Files

`init -agent cursor` creates the `.cursor/rules/` directory and seeds:

| File | Frontmatter | Purpose |
|------|-------------|---------|
| `general.mdc` | `description`, `globs: ""`, `alwaysApply: true` | General project conventions |

## Memory Configuration

Cursor has no dedicated memory file or runtime memory system. The `.cursor/rules/` directory
is the sole persistent context layer. There is no `memory.md` that Cursor writes to or reads
from at runtime.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Project | `.cursor/rules/*.md` or `*.mdc` | Yes — controlled by `alwaysApply` or `globs` frontmatter |
| Global | Settings UI only (no documented global filesystem path) | — |

For always-on project context, set `alwaysApply: true` in the rule's YAML frontmatter.
There is no session-level memory mechanism.

## Limits

- Official best-practice size limit is 500 lines; repogov enforces 300.

## Preset

`DefaultCursorLayout()` in `presets.go`. Uses an empty glob for `rules/` to accept both `.md` and `.mdc`.
CLI agent name: `cursor`.
