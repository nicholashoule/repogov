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

Continue.dev imposes no documented per-file line/character limit. repogov enforces:

- `.continue/rules/*.md`: 300 lines (enforced via `DefaultConfig().Rules` glob).

## Memory Configuration

Continue.dev has no dedicated `memory.md` or project-level memory file. Persistent
context is provided by rules files in `.continue/rules/` that are always-included
(set `alwaysApply: true` in frontmatter). Global rules in `~/.continue/rules/` apply
to all projects but are not file-based within the repository.

## Seeded Files

`init -agent continue` creates the `.continue/rules/` directory and seeds:

| File | Frontmatter | Purpose |
|------|-------------|---------|
| `general.md` | `applyTo: "**"` | General project conventions |

## Preset

`DefaultContinueLayout()` in `presets.go`. Validates `.continue/rules/` with `*.md` glob.
CLI agent name: `continue`.
