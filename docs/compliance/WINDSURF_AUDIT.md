# Windsurf (Codeium) Audit

Source: https://docs.windsurf.com/windsurf/cascade/memories — verified 2026-03-07.

## Configuration Files

### Workspace Rules

- `.windsurf/rules/*.md` — primary multi-file rules directory. Each `.md` file
  requires a `trigger:` field in YAML frontmatter that controls when it is applied.
  Windsurf searches `.windsurf/rules/` in subdirectories and walks up to the git
  root, so nested rule files are also loaded.

  **Trigger modes** (set via `trigger:` frontmatter field):

  | Trigger | Behaviour |
  |---------|-----------|
  | `always_on` | Injected into every Cascade request automatically |
  | `model_decision` | Model decides whether the rule is relevant |
  | `glob` | Applied when the active file matches the `globs:` pattern |
  | `manual` | Injected only when the user explicitly invokes the rule |

  Example frontmatter:
  ```yaml
  ---
  trigger: glob
  globs: "**/*.go"
  ---
  ```

- `AGENTS.md` — supported natively by Windsurf's rules engine. A root-level
  `AGENTS.md` is treated as `always_on`; a subdirectory `AGENTS.md` is treated
  as an auto-glob rule scoped to that directory.

- `.windsurfrules` — legacy single-file format. Not documented in current
  official Windsurf docs (only mentioned in third-party agent cross-compatibility
  notes). Newer projects should use `.windsurf/rules/*.md`.

### Global Rules

- `~/.codeium/windsurf/memories/global_rules.md` — single Markdown file applied
  automatically across all projects. Maximum 6,000 characters.

### Enterprise / System Rules

- OS-specific system-level directory (e.g. `/etc/windsurf/rules/*.md` on Linux).
  Managed by IT/security teams.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `.windsurf/rules/*.md` and global `global_rules.md` |
| (none) | Legacy `.windsurfrules` has no extension |

## Limits

| Scope | Limit |
|-------|-------|
| Workspace rule file | 12,000 characters per `.md` file |
| Global rules file | 6,000 characters (`global_rules.md`) |

## Seeded Files

`init -agent windsurf` creates the `.windsurf/rules/` directory and seeds:

| File | Frontmatter | Purpose |
|------|-------------|---------|
| `general.md` | `trigger: always_on` | General project conventions |

## Preset

`DefaultWindsurfLayout()` in `presets.go`. Validates `.windsurf/rules/` with `*.md` glob.
CLI agent name: `windsurf`.
