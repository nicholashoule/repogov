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

Cline imposes no documented per-file line/character limit. repogov enforces:

- `.clinerules/*.md`: 300 lines (enforced via `DefaultConfig().Rules` glob).

## Memory Configuration

Cline has no built-in project-level `memory.md` file. A community convention known
as "Memory Bank" instructs the agent (via `.clinerules/` rule files) to maintain
a set of Markdown files (e.g. `memory-bank/projectbrief.md`, `memory-bank/activeContext.md`)
that track project state across sessions. These files are user-managed content;
Cline does not load them automatically — they are referenced from rule files.

repogov does not govern `memory-bank/` files by default. Teams using the Memory Bank
pattern should add explicit glob rules to `repogov-config.json` if they want
line-limit enforcement on those files.

## Seeded Files

`init -agent cline` creates the `.clinerules/` directory and seeds the full
default instruction set directly into `.clinerules/` (no `rules/` subdirectory):

| File | `applyTo` | Purpose |
|------|-----------|----------|
| `general.md` | `"**"` | General coding conventions |
| `backend.md` | `"**"` | API design, error handling, auth, data, testing |
| `frontend.md` | `"**"` | Component structure, state, API integration, a11y |
| `codereview.md` | `"**/*.md"` | Code review guidelines |
| `governance.md` | `"**"` | Line limits, layout, zero dependencies |
| `library.md` | `"**/*.md"` | Library authoring conventions |
| `testing.md` | `"**/*.md"` | Test structure and coverage |
| `emoji-prevention.md` | `"**"` | No emoji in docs |
| `security.md` | `"**"` | Security guidelines |
| `repo.md` | `"**"` | Repository conventions |

## Preset

`DefaultClineLayout()` in `presets.go`. Cline loads rule files directly from `.clinerules/`;
the schema uses `Root: ".clinerules"` with a `"."` DirRule (matches all `*.md` files in the root
of `.clinerules/` directly). CLI agent name: `cline`.
