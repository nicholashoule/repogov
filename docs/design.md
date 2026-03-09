# Design: repogov

## Purpose

repogov is a dependency-free Go library and CLI for auditing repository
file lengths and directory layout conventions. It helps teams enforce
consistent structure across Copilot, Cursor, Windsurf, Claude, GitLab,
and standard repository root conventions.

## Architecture

The library is split into two orthogonal concerns:

1. **Line-count limits** -- configurable per-file, per-glob, and default
   limits that classify files as PASS, WARN, FAIL, or SKIP.

2. **Layout governance** -- schema-based validation of directory structure
   using preset rules for GitHub (`.github/`), Cursor (`.cursor/`), Windsurf
   (`.windsurf/`), Claude (`.claude/`), GitLab (`.gitlab/`), and the repository
   root (`.`).

Both concerns produce structured results that can be consumed
programmatically or formatted for human-readable output.

## Package Structure

```
repogov.go      -- Core types: Status, Config, Rule, Result
check.go        -- CountLines, CheckFile, CheckDir, CheckDirContext
layout.go       -- LayoutSchema, CheckLayout, CheckLayoutContext
presets.go       -- DefaultCopilotLayout, DefaultCursorLayout, DefaultWindsurfLayout, DefaultClaudeLayout, DefaultGitLabLayout, DefaultRootLayout
config.go        -- LoadConfig, SaveConfig (JSON)
format.go        -- Summary, Passed, LayoutSummary, LayoutPassed
cmd/repogov/     -- CLI with limits/layout/all/version subcommands
```

## Configuration

Configuration uses a JSON file (default: `.github/repogov-config.json`):

```json
{
  "default": 300,
  "warning_threshold": 80,
  "skip_dirs": [".git", "vendor"],
  "rules": [
    {"glob": "docs/*.md", "limit": 1000}
  ],
  "files": {
    "README.md": 1200,
    "CHANGELOG.md": 0
  }
}
```

### Limit Resolution Order

1. Per-file override (`files` map, exact match)
2. First matching glob rule (`rules` array, in order)
3. `default` value (falls back to 300 if zero)

A limit of 0 exempts the file from checking (status = SKIP).

## Layout Schemas

A `LayoutSchema` defines:
- Required files (must exist; FAIL if missing)
- Optional files (INFO if present, silent if absent)
- Directory rules (glob patterns and minimum file counts)
- Naming conventions (case rules with exceptions)

Presets are provided for Copilot, Cursor, Windsurf, Claude, GitLab, and the
repository root; custom schemas can be constructed programmatically.

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
  -config    Path to JSON/YAML config (default: auto-discovered)
  -root      Repository root (default: .)
  -exts      Extension filter (e.g., .go,.md)
  -agent     Agent preset: copilot, cursor, windsurf, claude, gitlab, root, or all
  -quiet     Exit code only
  -json      JSON output
```

## Constraints

- Zero external dependencies (stdlib only)
- Go 1.22+
- Cross-platform (Windows, Linux, macOS)
- Auditing is read-only; `init` scaffolds new files but never overwrites existing ones
