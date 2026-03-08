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

No documented line/character limits per rule file.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultJetBrainsLayout()` in `presets.go`.
2. Add `.aiassistant/rules/*.md` glob rules to `DefaultConfig()`.
3. Add `TestInitLayout_JetBrainsSchema` to `init_test.go`.
4. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.

Note: Rule type (Always / Manually / etc.) is IDE-managed metadata, not stored
in the `.md` file itself. repogov can only govern the Markdown content and file
placement; rule-type settings remain with the developer's IDE configuration.
