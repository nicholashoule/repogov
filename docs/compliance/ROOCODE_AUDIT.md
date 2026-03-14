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

Roo Code imposes no documented per-file line/character limit. repogov enforces:

- `.roo/rules/*.md`: 300 lines (enforced via `DefaultConfig().Rules` glob).

**Note:** Mode-specific directories (`.roo/rules-code/`, `.roo/rules-architect/`, etc.)
are not included in the default glob rule. Teams using mode-specific dirs should
add additional glob rules to `repogov-config.json`.

## Memory Configuration

Roo Code has no built-in project-level `memory.md` file. The community "Memory Bank"
pattern instructs the agent (via `.roo/rules/` files) to maintain structured Markdown
files (e.g. `memory-bank/projectbrief.md`, `memory-bank/activeContext.md`) that
track project state across sessions. Roo Code does not load these files automatically;
they are referenced from rule files.

repogov does not govern `memory-bank/` files by default. Teams using the Memory Bank
pattern should add explicit glob rules to `repogov-config.json` if they want
line-limit enforcement on those files.

## Seeded Files

`init -agent roocode` creates the `.roo/rules/` directory and seeds:

| File | Frontmatter | Purpose |
|------|-------------|----------|
| `general.md` | `applyTo: "**"` | General project conventions |

## Preset

`DefaultRooCodeLayout()` in `presets.go`. Validates `.roo/rules/` with `*.md` glob.
Mode-specific directories (`.roo/rules-{modeSlug}/`) are recognized in `DefaultRootLayout()`
as part of the `.roo/` managed directory but are not enforced by the Roo Code preset.
CLI agent name: `roocode`.
