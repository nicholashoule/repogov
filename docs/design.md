# Design: repogov

## Purpose

repogov is a dependency-free Go library and CLI for auditing repository
file lengths and directory layout conventions. It helps teams enforce
consistent structure across GitHub Copilot, Cursor, Windsurf, Claude,
Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant,
GitLab, and standard repository root conventions.

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
repogov.go      -- Core types: Status, Config, Rule, Result, ResolveLimit, DefaultConfig
check.go        -- CountLines, CheckFile, CheckDir, CheckDirContext
config.go       -- LoadConfig, FindConfig, SaveConfig, ValidateConfig
format.go       -- Summary, Passed, LayoutSummary, LayoutPassed
init.go         -- InitLayout, InitLayoutAll, InitLayoutWithConfig, InitLayoutAllWithConfig
layout.go       -- LayoutSchema, LayoutResult, CheckLayout, CheckLayoutContext
presets.go      -- DefaultCopilotLayout, DefaultCursorLayout, DefaultWindsurfLayout,
                   DefaultClaudeLayout, DefaultKiroLayout, DefaultGeminiLayout,
                   DefaultContinueLayout, DefaultClineLayout, DefaultRooCodeLayout,
                   DefaultJetBrainsLayout, DefaultZedLayout, DefaultGitLabLayout, DefaultRootLayout
scaffold.go     -- File creators, AGENTS.md management, config scaffolding, instruction seeding
template.go     -- Embedded template FS, mustReadTemplate, mustRenderTemplate
yaml.go         -- Minimal YAML parser and serializer (stdlib only)
cmd/repogov/    -- CLI with limits/layout/init/validate/all/version subcommands
```

## Configuration

Configuration uses a JSON or YAML file (default: `repogov-config.json`, searched
in repo root then `.github/`):

```json
{
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
  "files": {
    "README.md": 1200,
    "CHANGELOG.md": 0
  }
}
```

`warning_threshold` accepts a bare integer (80) or a percent string ("80%")
via the `PercentInt` type. `descriptive_names` controls `*.instructions.md`
vs `*.md` naming. `skip_frontmatter` disables YAML frontmatter validation in
layout checks. `init_always_create` seeds missing files into existing
directories. `init_include_files` / `init_exclude_files` filter which template
stems are created during init.

### Limit Resolution Order

1. Per-file override (`files` map, exact match)
2. First matching glob rule (`rules` array, in order)
3. `default` value (falls back to 300 if zero)

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
repogov [flags] <limits|layout|init|validate|all|version>

Flags:
  -config       Path to JSON/YAML config (default: auto-discovered)
  -root         Repository root (default: .; resolved to git root when available, otherwise to the absolute path)
  -exts         Extension filter (e.g., .go,.md)
  -agent        AI agent preset: copilot, cursor, windsurf, claude, kiro, gemini,
                continue, cline, roocode, jetbrains, zed, or all
  -platform     Repository platform preset: gitlab, root, or all
  -descriptive  Use *.instructions.md naming (overrides config descriptive_names)
  -seed         Seed missing files into existing dirs (init only)
  -quiet        Exit code only
  -json         JSON output
```

## Constraints

- Zero external dependencies (stdlib only)
- Go 1.22+
- Cross-platform (Windows, Linux, macOS)
- Auditing is read-only; `init` scaffolds new files but never overwrites existing ones
