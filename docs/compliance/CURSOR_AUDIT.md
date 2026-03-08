# Cursor Audit

Source: https://cursor.com/docs/context/rules
Verified: 2026-03-07

## Configuration Files

- `.cursor/rules/` — project rules; version-controlled.
  - Both `.md` (plain) and `.mdc` (with YAML frontmatter) extensions are supported.
  - `.mdc` frontmatter keys: `description`, `globs`, `alwaysApply`.
  - Files can be organized into subdirectories within `.cursor/rules/`.
- `AGENTS.md` at project root or subdirectory — simple alternative to `.cursor/rules/`.

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

## Limits

- Official best-practice size limit is 500 lines; repogov enforces 300.

## Preset

`DefaultCursorLayout()` in `presets.go`. Uses an empty glob for `rules/` to accept both `.md` and `.mdc`.
CLI agent name: `cursor`.
