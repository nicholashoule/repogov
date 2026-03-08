# Gemini CLI Audit

Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/gemini-md.md — verified 2026-03-07.

## Configuration Files

Gemini CLI uses a hierarchical `GEMINI.md` system. All found files are
concatenated and sent to the model with every prompt.

### Context Hierarchy (load order)

1. **Global** — `~/.gemini/GEMINI.md` — applies to all projects.
2. **Workspace** — `GEMINI.md` in the workspace root (and parent directories
   walked up to the filesystem root).
3. **JIT (Just-In-Time)** — when a tool accesses a file or directory, Gemini
   CLI scans for `GEMINI.md` in that directory and its ancestors up to a
   trusted root. This lets per-component instructions be loaded on demand.

The CLI footer shows the number of currently loaded context files.

### Configurable Filename

The default filename `GEMINI.md` can be changed or extended via
`~/.gemini/settings.json`:

```json
{
  "context": {
    "fileName": ["AGENTS.md", "CONTEXT.md", "GEMINI.md"]
  }
}
```

When multiple names are listed, all matching files are loaded.

### Imports

Large `GEMINI.md` files can modularise context using `@file.md` import syntax
(relative or absolute paths):

```markdown
@./components/instructions.md
@../shared/style-guide.md
```

### Memory Commands

| Command | Effect |
|---------|--------|
| `/memory show` | Display full concatenated context |
| `/memory reload` | Re-scan and reload all `GEMINI.md` files |
| `/memory add <text>` | Append text to `~/.gemini/GEMINI.md` |

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `GEMINI.md` (default filename); configurable via `settings.json` |
| `.json` | `~/.gemini/settings.json` — MCP server config and filename overrides |

## Limits

No documented line/character limits per file.

## repogov Support Status

Not yet supported. To add support:

1. Create `DefaultGeminiLayout()` in `presets.go`.
2. Seed a `GEMINI.md` at the workspace root (uppercase, no extension aside from `.md`).
3. Add `Naming.Exceptions` entry for the mandated uppercase filename.
4. Add `TestInitLayout_GeminiSchema` to `init_test.go`.
5. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.
