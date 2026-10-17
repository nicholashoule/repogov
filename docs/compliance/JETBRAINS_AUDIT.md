# JetBrains AI Assistant Audit

Source: https://www.jetbrains.com/help/ai-assistant/configure-project-rules.html — Verified: 2026-10-17
(page dated AI Assistant 2026.2 docs, 27 May 2026). No deprecation of Project Rules
found; the rule types table below was re-confirmed unchanged. The page does not
mention `AGENTS.md`.

## Related Agent Surfaces

Per https://www.jetbrains.com/help/ai-assistant/agents.html (fetched 2026-10-17),
JetBrains agents read their own instruction files: Junie respects `AGENTS.md`, Claude
Agent uses `CLAUDE.md`, Codex uses `AGENTS.md`. That page does not mention
`.aiassistant/rules`, which applies to AI Assistant chat. Junie's own docs (https://junie.jetbrains.com/docs/guidelines-and-memory.html, fetched
2026-10-17) give this lookup order:

1. `.junie/AGENTS.md` -- preferred; per maintainer guidance, when present it is used exclusively.
2. Root `AGENTS.md`, combined with `.junie/playbook.md` and every `.junie/rules/*.md`, if present.
3. Legacy `.junie/guidelines.md` or `.junie/guidelines/` (still supported).

Global `~/.junie/AGENTS.md` is included alongside project guidelines. The page does not say
whether the search stops at the first match, so the exclusive-use rule in (1) is unconfirmed on
the vendor page. Because step 2 merges several hidden files, the result can differ between
repos; an audit that needs one source of truth should keep a single `.junie/AGENTS.md`. Older
IDE help pages (https://www.jetbrains.com/help/ai-assistant/junie-agent.html) describe a
"Guidelines path" setting (Settings | Tools | Junie | Project Settings) that overrides the
lookup and still show `.junie/guidelines.md` as the default; that setting is not on the Junie
page above. Junie is not governed by repogov.

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
