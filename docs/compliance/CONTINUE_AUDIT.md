# Continue.dev Audit

Source: https://docs.continue.dev/customize/deep-dives/rules — verified 2026-03-07.

## Configuration Files

### Local (Workspace) Rules

- `.continue/rules/` — primary multi-file rules directory. All `.md` files in
  this directory are loaded automatically. Files are processed in lexicographical
  order; prefix filenames with numbers (`01-general.md`, `02-backend.md`) to
  control load order.

  Each `.md` file supports optional YAML frontmatter:

  | Field | Type | Description |
  |-------|------|-------------|
  | `name` | string | Display name shown in the rules toolbar |
  | `globs` | string or array | Apply rule only when matched files are in context |
  | `regex` | string or array | Apply rule only when matched content is in context files |
  | `alwaysApply` | boolean | `true` = always include; `false` = include only when globs/regex match |
  | `description` | string | Agent reads this to decide relevance when `alwaysApply: false` |

  Example:
  ```yaml
  ---
  name: TypeScript Best Practices
  globs: ["**/*.ts", "**/*.tsx"]
  alwaysApply: false
  ---
  ```

### Global Rules

- `~/.continue/rules/` — global rules applied to all projects. Same format as
  local workspace rules.

### Hub Rules (remote)

- Stored on Continue Mission Control (`continue.dev`). Referenced in
  `config.yaml` via `uses: username/rule-name`. No local files are created;
  rules are fetched from the Hub at runtime.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `.continue/rules/*.md` — Markdown with optional YAML frontmatter |
| `.yaml` | `.continue/config.yaml` — references Hub rules via `uses:` |

## Limits

No documented line/character limits. Applies to both local and global rules.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultContinueLayout()` in `presets.go`.
2. Add `.continue/rules/*.md` glob rules to `DefaultConfig()`.
3. Add `TestInitLayout_ContinueSchema` to `init_test.go`.
4. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.
