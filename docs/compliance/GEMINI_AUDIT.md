# Gemini CLI Audit

Sources:
- https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/gemini-md.md — verified 2026-10-17
- https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/tutorials/memory-management.md — verified 2026-10-17
- https://github.com/google-gemini/gemini-cli — verified 2026-10-17 (README replacement notice)
- https://antigravity.google/docs/cli/gcli-migration — verified 2026-10-17
- https://antigravity.google/docs/rules — verified 2026-10-17

## Deprecation Notice (verified 2026-10-17)

> **Gemini CLI has been partly superseded by Antigravity CLI (`agy`).** The
> repo README states that for unpaid-tier and Google One users "Gemini CLI was
> replaced by Antigravity CLI on June 18th, 2026", and that Gemini CLI "has not
> been optimized for Gemini 4 Argon". The repo is not archived and still has
> issues/PRs/releases. The README (re-read 2026-10-17) does not say whether paid/Code Assist
> users are affected and still lists API-key and Vertex AI sign-in; no official
> shutdown date exists for them, so none is asserted here.
>
> Migration target: Antigravity CLI. Per the official migration page,
> workspace `GEMINI.md` and `AGENTS.md`, and global `~/.gemini/GEMINI.md`, are
> read by both CLIs, so no rename is needed. Other surfaces move:
>
> | Item | Gemini CLI | Antigravity CLI |
> |---|---|---|
> | Global skills | `~/.gemini/skills/` | `~/.gemini/antigravity-cli/skills/` |
> | Workspace skills | `.gemini/skills/` | `.agents/skills/` (manual move) |
> | Global MCP | inline in `~/.gemini/settings.json` | `~/.gemini/config/mcp_config.json` |
> | Workspace MCP | inline in settings | `.agents/mcp_config.json` |
>
> `agy plugin import gemini` converts extensions. Not verified: any official
> Gemini CLI shutdown date (the migration page gives no dates).

## Antigravity Rules (successor surface, verified 2026-10-17)

Source: https://antigravity.google/docs/rules. Rules are cumulative: all discovered rules from
global, workspace and directory scopes are combined; on conflict, more specific directory rules win.

**Directory scope** -- walking up from each file read or edited to the workspace root, at each level:
- `<dir>/AGENTS.md` or `<dir>/GEMINI.md`
- `<dir>/.agents/AGENTS.md` or `<dir>/.agents/GEMINI.md`
- `<dir>/.agents/rules/*.md` (legacy `<dir>/.agent/rules/*.md` still read)

**Global scope** -- always active, no frontmatter: `~/.gemini/AGENTS.md`, `~/.gemini/GEMINI.md`,
`~/.gemini/config/AGENTS.md`, `~/.gemini/config/GEMINI.md`. Modular global rules (frontmatter required):
`~/.gemini/config/rules/*.md`. The CLI also reads `~/.gemini/antigravity-cli/rules/*.md` and plugin
rules under `~/.gemini/antigravity-cli/plugins/<name>/rules/`.

**Frontmatter.** `AGENTS.md`/`GEMINI.md` take none and are always on. Every `.md` in `rules/` must start
with YAML frontmatter:

| Key | Notes |
|-----|-------|
| `trigger` | Required: `always_on`, `model_decision`, `glob`, or `manual` (snake_case only) |
| `description` | Required for `model_decision`; recommended for all |
| `globs` | Required for `glob` (comma-separated; singular `glob` also accepted) |

**Silent-failure traps** (worth a repogov check): a rule file with no frontmatter, or an unrecognized
trigger such as camelCase `alwaysOn`/`modelDecision`, is **silently discarded**; `.agents/rules/` is scanned
flat, so files in subdirectories (`rules/frontend/react.md`) are ignored unless registered in
`.agents/rules.json` (`inherits` and `entries` arrays with `path`, `exclude`, `include_only`).

**File includes** (in `AGENTS.md`, `GEMINI.md` or `.agents/rules/*.md`):
- `@[label](path)` inlines the target's contents into the rule before prompt evaluation and size checks.
  Relative paths resolve from the rule file's directory; `~/` expands to the home directory; YAML
  frontmatter in the included file is stripped and only the Markdown body is spliced in.
- `@filename` does not inline anything: it resolves the path (relative to the rule file, or
  workspace-relative/absolute) and rewrites it to a canonical `@/workspace/path` reference.
- The page states no include depth or recursion limit.

**Limits.** Per file: **24,000 bytes**, measured *after* `@[label](path)` expansion; larger files are
truncated (where the cut falls, and whether a notice is added, is not stated). Aggregate: a
**20,000-token** budget for all active global and `always_on` rules, separate from the skills/MCP
budget; on overflow the largest rule files are replaced by `- <path>: <description>` pointers the agent
reads on demand. No character limit is stated.

**repogov support:** `-agent gemini` accepts `AGENTS.md` when `GEMINI.md` is absent only if the
project's `.gemini/settings.json` lists `"AGENTS.md"` in `context.fileName` (`LayoutSchema.Alternatives`;
`init` still creates `GEMINI.md`). Antigravity reads `AGENTS.md` by default, but Gemini CLI's default
context file is `GEMINI.md` only (see Configurable Filename), and repogov cannot tell which CLI a
contributor runs; a user-level `~/.gemini/settings.json` is invisible to other contributors and CI, so it
is not consulted. A missing, non-regular, unreadable or malformed settings file (JSON with comments
included) means no fallback, and the FAIL names the reason. `.agents/rules/*.md` is covered by the byte
limit and the agent-pitfall checks; nested rule files are matched exactly against the parsed
`rules.json` `entries` (`path` relative to `.agents/`, `include_only`, `exclude`; `inherits` is not
followed), and registered ones are validated like top-level rules. There is no `.agents/` layout preset
yet.

**repogov implications (not implemented):** repogov's line limits do not bound these byte and token
limits -- a 300-line rule can exceed 24 KB, and includes expand *after* repogov counts lines, so a short
file with large `@[...](...)` includes can still be truncated. A Gemini/Antigravity check would need a
byte-size warning near 24,000 and awareness of includes. `DefaultGeminiLayout()` covers `GEMINI.md` only. A
`.agents/` layout (`rules/*.md` with the trigger vocabulary above, `AGENTS.md`, `skills/`) would also
serve Codex, Cline, Devin and Copilot skills, which share `.agents/skills/`.

## Configuration Files (Gemini CLI)

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
Subdirectory files (e.g. `./src/GEMINI.md`) are also supported per the memory
tutorial.

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

Current docs list only `show` and `reload`; `add` was not re-confirmed on
2026-10-17. The docs also describe agent-saved memories (location
unspecified) and an experimental Auto Memory feature.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `GEMINI.md` (default filename); configurable via `settings.json` |
| `.json` | `~/.gemini/settings.json` — MCP server config and filename overrides |

## Limits

Gemini CLI imposes no documented per-file line/character limit. repogov enforces:

- `GEMINI.md`: 200 lines (enforced via `DefaultConfig().Files`).

## Memory Configuration

Gemini CLI manages memory through the `GEMINI.md` hierarchy and in-session commands.
These commands operate on the global `~/.gemini/GEMINI.md` file, not on project files.

| Command | Effect |
|---------|--------|
| `/memory show` | Display full concatenated context |
| `/memory reload` | Re-scan and reload all `GEMINI.md` files |
| `/memory add <text>` | Append text to `~/.gemini/GEMINI.md` |

No workspace-level `memory.md` file exists. Persistent per-project memory is achieved
by editing `GEMINI.md` at the repository root.

## Seeded Files

`init -agent gemini` seeds a `GEMINI.md` at the repository root.

| File | Notes |
|------|-------|
| `GEMINI.md` | Primary instruction file (Required). Generated by `createGeminiMd`; templated from `templates/agents/GEMINI.md.tmpl`. |

## Preset

`DefaultGeminiLayout()` in `presets.go`. The schema has `Root: "."` and `Required: ["GEMINI.md"]`;
`Naming.Exceptions` includes `GEMINI.md` to allow the uppercase filename.
The init short-circuits the generic Required-file loop (skips `GEMINI.md`) so that
`createGeminiMd` writes the templated content instead of a bare placeholder.
CLI agent name: `gemini`.
