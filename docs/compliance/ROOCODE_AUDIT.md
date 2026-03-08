# Roo Code Audit

Source: https://docs.roocode.com/features/custom-instructions — verified 2026-03-07.

## Configuration Files

Roo Code uses a hierarchical rules system with workspace-level, mode-specific,
and global scopes.

### Workspace Rules

- `.roo/rules/` — primary workspace rules directory. Loads `.md` and `.txt`
  files recursively. Replaces the legacy `.roorules` single file.
- `.roo/rules-{modeSlug}/` — mode-specific rules directory applied only when
  the named mode is active.
  Common mode slugs: `code`, `architect`, `ask`, `debug`, `test`.
  Example: `.roo/rules-code/` applies only in Code mode.
- `.roorules` — legacy fallback single file at workspace root (still supported).
- `.roorules-{modeSlug}` — legacy per-mode fallback single file.
- `AGENTS.md` / `AGENT.md` at workspace root — loaded when the
  `roo-cline.useAgentRules` setting is enabled (default: on).

### Global Rules

- `~/.roo/rules/` — global rules applied to all projects.
- `~/.roo/rules-{modeSlug}/` — global per-mode rules.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | Primary format in `.roo/rules/` and mode-specific dirs |
| `.txt` | Also supported in rules directories |

## Limits

No documented line/character limits. Files are loaded recursively and
concatenated, so keep individual rule files focused.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultRooCodeLayout()` in `presets.go`.
2. Add `.roo/rules/*.md` glob rules to `DefaultConfig()`.
3. Consider mode-specific dirs (`.roo/rules-code/`, etc.) as optional extras.
4. Add `TestInitLayout_RooCodeSchema` to `init_test.go`.
5. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.
