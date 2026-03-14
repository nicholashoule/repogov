# JetBrains AI Assistant Audit

Source: https://www.jetbrains.com/help/ai-assistant/configure-project-rules.html — verified 2026-03-07.

## Configuration Files

### Project Rules

- `.aiassistant/rules/` — primary multi-file rules directory. Each `.md` file
  is a rule that can be configured with a **Rule type** controlling when it is
  applied. Files can be created via Settings -> Tools -> AI Assistant -> Rules, or
  by manually placing `.md` files in the directory.

  **Rule types** (set in the rule file's settings panel):

  | Type | Behaviour |
  |------|-----------|
  | `Always` | Automatically added to all chat sessions |
  | `Manually` | Applied only when invoked via `@rule:` or `#rule:` in chat, or added through the attachment action |
  | `By model decision` | Model decides whether the rule is relevant; requires an `Instruction` description |
  | `By file patterns` | Applied when a referenced file matches the specified glob pattern (e.g. `*.kt`, `src/**`) |
  | `Off` | Rule is inactive |

  Rule type metadata is stored by the IDE, not as YAML frontmatter inside the `.md` file itself.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `.aiassistant/rules/*.md` — Markdown content for rule body |

## Limits

JetBrains AI Assistant imposes no documented per-file line/character limit.
repogov enforces:

- `.aiassistant/rules/*.md`: 300 lines (enforced via `DefaultConfig().Rules` glob).

## Memory Configuration

JetBrains AI Assistant has no project-level `memory.md` file. Rule type metadata
(Always / Manually / By model decision / By file patterns) is stored in the IDE's
local settings, not in the Markdown rule files themselves. Persistent context is
provided by rules with type `Always`. There is no runtime memory file that the
agent writes to within the repository.

## Seeded Files

`init -agent jetbrains` creates the `.aiassistant/rules/` directory and seeds:

| File | Purpose |
|------|----------|
| `general.md` | General project conventions |

**Note:** Rule type (Always / Manually / etc.) is IDE-managed metadata, not stored
in the `.md` file itself. repogov governs Markdown content and file placement;
rule-type settings remain with each developer's IDE configuration.

## Preset

`DefaultJetBrainsLayout()` in `presets.go`. Validates `.aiassistant/rules/` with `*.md` glob.
CLI agent name: `jetbrains`.
