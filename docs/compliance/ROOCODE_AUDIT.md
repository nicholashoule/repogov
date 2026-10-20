# Roo Code Audit

Sources: https://roocodeinc.github.io/Roo-Code/features/custom-instructions
(docs.roocode.com 301-redirects here) and https://github.com/RooCodeInc/Roo-Code —
Verified: 2026-10-17.

## Deprecation Notice

- **Status: shut down and archived.** The GitHub repository states it was archived by
  its owner on May 15, 2026 (read-only) and that "The Roo Code Extension was shut down
  on May 15th." Evidence: https://github.com/RooCodeInc/Roo-Code.
  The archive notice and extension shutdown were re-confirmed from the repo page on
  2026-10-17. Whether Cloud and Router were also sunset is not stated there (secondary
  reporting only), so it is not asserted here.
- **Migration targets named by the repo:** Cline (https://cline.bot) or ZooCode, a
  community fork (https://github.com/Zoo-Code-Org/Zoo-Code/).
- repogov keeps the `roocode` preset for existing repositories and forks; expect no new
  Roo Code features. Conventions below are as last documented.

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
- `.roorules` — legacy fallback single file at workspace root, used only if
  `.roo/rules/` is missing or empty.
- `.roorules-{modeSlug}` — legacy per-mode fallback single file, used only if the
  matching directory is missing or empty.
- `AGENTS.md` (preferred) / `AGENT.md` at workspace root — loaded when the
  `roo-cline.useAgentRules` setting is enabled (default: on); loaded after mode rules
  and before the generic rules directories.
- Mode rules load before general rules; global and workspace rules are aggregated
  (workspace wins conflicts); files are read recursively, sorted by basename.

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
