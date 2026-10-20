# Design: repogov

## Purpose

repogov is a dependency-free Go library and CLI for auditing repository
file lengths and directory layout conventions. It helps teams enforce
consistent structure across GitHub Copilot, Cursor, Windsurf, Claude,
Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant,
Zed AI, GitLab, and standard repository root conventions.

## Architecture

The library is split into two orthogonal concerns:

1. **Line-count limits** -- configurable per-file, per-glob, and default
   limits that classify files as PASS, WARN, FAIL, or SKIP.

2. **Layout governance** -- schema-based validation of directory structure
   using preset rules for GitHub (`.github/`), Cursor (`.cursor/`), Windsurf
   (`.windsurf/`), Claude (`.claude/`), Kiro (`.kiro/`), Gemini CLI (`GEMINI.md`
   at root), Continue (`.continue/`), Cline (`.clinerules/`), Roo Code (`.roo/`),
   JetBrains AI Assistant (`.aiassistant/`), GitLab (`.gitlab/`), and the
   repository root (`.`).

Both concerns produce structured results that can be consumed
programmatically or formatted for human-readable output.

## Package Structure

```
repogov.go        -- Core types: Status, Config, Rule, Result, ResolveLimit, DefaultConfig
check.go          -- CountLines, CheckFile, CheckDir, CheckDirContext (UTF-16/32 aware)
config.go         -- LoadConfig (strict), FindConfig, FindAllConfigs, SaveConfig, ValidateConfig
config_hints.go   -- The _agent_hints block every generated config carries
config_upgrade.go -- UpgradeConfigHints, UpgradeConfigWithLanguage (update-config); atomic writes
language.go       -- Prose language: KnownLanguages, NormalizeLanguage, LookupLanguage, DefaultLanguage
format.go         -- Summary, Passed, LayoutSummary, LayoutPassed
init.go           -- InitLayout, InitLayoutAll, InitLayoutWithConfig, InitLayoutAllWithConfig
layout.go         -- LayoutSchema, LayoutResult, CheckLayout, CheckLayoutContext
pitfalls.go       -- CheckAgentPitfalls: files an agent would silently ignore
presets.go        -- DefaultCopilotLayout, DefaultCursorLayout, DefaultWindsurfLayout,
                     DefaultClaudeLayout, DefaultKiroLayout, DefaultGeminiLayout,
                     DefaultContinueLayout, DefaultClineLayout, DefaultRooCodeLayout,
                     DefaultJetBrainsLayout, DefaultZedLayout, DefaultGitLabLayout, DefaultRootLayout
scaffold.go       -- File creators, AGENTS.md (marked Context section, AgentsMdContextStatus),
                     config scaffolding, instruction seeding
settings.go       -- Read-only audit of agent settings files
template.go       -- Embedded template FS, mustReadTemplate, mustRenderTemplate
yaml.go           -- Minimal strict YAML parser and serializer (stdlib only)
rename_*.go       -- Platform rename retry for atomic writes (Windows sharing violations)
cmd/repogov/      -- CLI: limits, layout, init, update-config, validate, all, version
```

## Configuration

Configuration uses a JSON or YAML file (default: `repogov-config.json`, searched
in repo root then `.github/`):

```json
{
  "_agent_hints": { "about": "Keys starting with _ guide AI agents; repogov ignores them." },
  "language": "en-US",
  "default": 500,
  "warning_threshold": "85%",
  "skip_dirs": [".git", "vendor", "workflows"],
  "include_exts": [".md", ".mdc"],
  "descriptive_names": false,
  "skip_frontmatter": false,
  "init_always_create": false,
  "init_include_files": [],
  "init_exclude_files": [],
  "rules": [
    {"glob": "docs/*.md", "limit": 1000}
  ],
  "max_bytes": [
    {"glob": ".agents/rules/*.md", "limit": 24000}
  ],
  "files": {
    "README.md": 1200,
    "CHANGELOG.md": 0
  }
}
```

`_agent_hints` (abridged above) is a block of one-line hints for AI agents
that open the file; keys starting with `_` (and a top-level `$schema`) are
ignored, and `update-config` refreshes the block in place. `language` is the
BCP 47 tag of the repository's prose (docs, comments, commits), `en-US` by
default; scaffolded agent files and AGENTS.md's marked Context section state
it. Parsing is strict: an unknown key, YAML flow syntax or tab indentation is
an error, so a typo never silently falls back to a default.

`warning_threshold` accepts a bare integer (80) or a percent string ("80%")
via the `PercentInt` type. `descriptive_names` controls `*.instructions.md`
vs `*.md` naming. `skip_frontmatter` disables YAML frontmatter validation in
layout checks. `init_always_create` seeds missing files into existing
directories. `init_include_files` / `init_exclude_files` filter which template
stems are created during init.

### Limit Resolution Order

1. Per-file override (`files` map, exact match)
2. First matching glob rule (`rules` array, in order)
3. `default` value (falls back to the built-in 500 if zero)

A limit of 0 exempts the file from checking (status = SKIP).

## Layout Schemas

A `LayoutSchema` defines:
- Required files (must exist; FAIL if missing)
- Optional files (INFO if present, silent if absent)
- Directory rules (glob patterns, minimum file counts, frontmatter requirements)
- Naming conventions (case rules with exceptions)

The `DirRule.Frontmatter` field lists YAML frontmatter keys that must be present
in every file within the directory (e.g. `applyTo` for Copilot instructions).
`Config.SkipFrontmatter` disables this validation globally; `StripFrontmatter()`
returns a copy of a schema with all frontmatter requirements cleared.

Presets are provided for Copilot, Cursor, Windsurf, Claude, Kiro, Gemini CLI,
Continue, Cline, Roo Code, JetBrains AI Assistant, GitLab, and the repository
root; custom schemas can be constructed programmatically.

File-only schemas (e.g., Gemini, where `Root == "."` and `Dirs` is empty)
use a short-circuit path in `CheckLayoutContext`: only Required/Optional file
existence is checked; directory walking is skipped entirely to avoid false
unexpected-file results at the repository root.

`root` is intentionally excluded from the `all` expansion — root layout
is project-structure-specific and would generate unexpected-file noise for
projects with non-standard top-level directories.

## Status Classification

| Status | Meaning |
|--------|---------|
| PASS   | File within limit or layout rule satisfied |
| WARN   | File at/above warning percentage threshold |
| FAIL   | File over limit or required item missing |
| SKIP   | File exempt (limit=0) or extension-filtered |
| INFO   | Optional layout item present |

## CLI

```
repogov [flags] <limits|layout|init|update-config|validate|all|version> [flags]

Flags (accepted before or after the subcommand):
  -config       Path to JSON/YAML config (default: auto-discovered; a given path must exist)
  -root         Repository root (default: . resolved to the git root; a given -root is used as is)
  -exts         Extension filter (e.g., .go,.md)
  -agent        AI agent preset: copilot, cursor, windsurf, claude, kiro, gemini,
                continue, cline, roocode, jetbrains, zed, or all
  -platform     Repository platform preset: github, gitlab, bitbucket, root, all, or auto
  -descriptive  Use *.instructions.md naming (overrides config descriptive_names)
  -seed         Seed missing files into existing dirs (init only)
  -language     Prose language tag (init and update-config only; default en-US)
  -quiet        Exit code only
  -json         JSON output

Exit codes: 0 pass, 1 a check failed, 2 usage or config error.
```

## Constraints

- Zero external dependencies (stdlib only)
- Go 1.24+
- Cross-platform (Windows, Linux, macOS)
- Auditing is read-only. `init` creates missing files and never overwrites
  existing ones, with one opt-in exception: in an AGENTS.md that carries the
  `<!-- repogov:context:begin -->` / `<!-- repogov:context:end -->` markers, it
  refreshes only the text between them. An unmarked AGENTS.md is left untouched
  (a note, not an error; `AgentsMdContextStatus` says why).
- `update-config` rewrites only the `_agent_hints` block and a missing
  `language` key; every write is atomic and refuses symlinks.
