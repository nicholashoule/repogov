# Cline Audit

Source: https://docs.cline.bot/features/cline-rules — verified 2026-03-07.

## Configuration Files

### Workspace Rules

- `.clinerules/` — primary multi-file rules directory. Cline loads all `.md`
  and `.txt` files in this directory. Files support YAML frontmatter with a
  `paths:` field for glob-based conditional application.

  Example frontmatter for a conditional rule:
  ```yaml
  ---
  paths:
    - "**/*.go"
  ---
  ```
  A rule file without a `paths:` key applies globally to all interactions.

### Cross-Tool Compatibility

Cline also automatically detects and loads rules from other agents' config files
when a `.clinerules/` directory is not present:

- `.cursorrules` — Cursor legacy single-file format
- `.windsurfrules` — Windsurf legacy single-file format
- `AGENTS.md` — AGENTS.md cross-agent standard

### Global Rules

- `~/Documents/Cline/Rules/` (macOS/Windows) — global rules directory loaded
  for all projects. Same `.md`/`.txt` format as workspace rules.
  Linux equivalent follows XDG base dir convention.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | Primary format in `.clinerules/` |
| `.txt` | Also supported in `.clinerules/` |

## Limits

No documented line/character limits. Individual rule files should be kept
focused; Cline loads all matching files and concatenates them.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultClineLayout()` in `presets.go`.
2. Add `.clinerules/*.md` glob rules to `DefaultConfig()`.
3. Add `TestInitLayout_ClineSchema` to `init_test.go`.
4. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.
