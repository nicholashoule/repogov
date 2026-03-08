# Zed AI Audit

Source: https://zed.dev/docs/ai/rules — verified 2026-03-07.

## Configuration Files

### Project Rules File

Zed supports a single project-level rules file at the project root. The first
file found from the following list (in priority order) is used:

| Priority | Filename |
|----------|----------|
| 1 | `.rules` |
| 2 | `.cursorrules` |
| 3 | `.windsurfrules` |
| 4 | `.clinerules` |
| 5 | `.github/copilot-instructions.md` |
| 6 | `AGENT.md` |
| 7 | `AGENTS.md` |
| 8 | `CLAUDE.md` |
| 9 | `GEMINI.md` |

The matched file is auto-included in all Agent Panel interactions (always-on).
Only one file is loaded — the first match wins.

### Rules Library (editor-managed)

Zed provides an in-IDE Rules Library for creating and managing reusable rules:

- Rules are stored locally by Zed (not in the project directory).
- Any rule in the library can be set as a **default rule** (auto-inserted into
  every new Agent Panel interaction) by clicking the paper-clip icon.
- Rules can be @-mentioned inline during a conversation.
- Access via: Agent Panel -> `...` menu -> `Rules...`, or `agent: open rules library`.

There is no multi-file directory scan for project rules — only one project root
file is supported.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.rules` | Primary Zed-native filename (root of project) |
| `.md` | Supported via `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `AGENT.md`, `.github/copilot-instructions.md` |
| (none) | `.cursorrules`, `.windsurfrules`, `.clinerules` have no extension |

## Limits

No documented line/character limits for project rules files.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultZedLayout()` in `presets.go`.
2. Seed a `.rules` file at workspace root (Zed's preferred filename).
3. `AGENTS.md` is also detected by Zed — the existing cross-agent seed covers
   this automatically.
4. Add `TestInitLayout_ZedSchema` to `init_test.go`.
5. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.

Note: Because only a single file is used, the repogov preset needs only a root
file rather than a multi-file directory.
