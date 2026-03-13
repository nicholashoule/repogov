# repogov

[![CI](https://github.com/nicholashoule/repogov/actions/workflows/ci.yml/badge.svg)](https://github.com/nicholashoule/repogov/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nicholashoule/repogov.svg)](https://pkg.go.dev/github.com/nicholashoule/repogov)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nicholashoule/repogov)](go.mod)
[![License](https://img.shields.io/github/license/nicholashoule/repogov)](LICENSE)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-none-brightgreen)](go.mod)

A dependency-free Go library and CLI for repository governance: enforce file-length limits and validate or scaffold AI-agent-ready directory layouts for Copilot, Cursor, Windsurf, Claude, GitLab, and common repository root conventions.

## Features

- **Line-count limits** -- configurable per-file, per-glob, and default
  limits with PASS/WARN/FAIL/SKIP classification
- **Layout governance** -- schema-based validation of `.github/`, `.cursor/`,
  `.windsurf/`, `.claude/`, `.gitlab/`, and repository root directory structure
- **Platform presets** -- built-in rules for Copilot, Cursor, Windsurf, Claude, GitLab, and root conventions
- **Zero dependencies** -- pure stdlib, no external imports
- **Cross-platform** -- works on Windows, Linux, and macOS

## Installation

### Library

```bash
go get github.com/nicholashoule/repogov
```

### CLI

```bash
go install github.com/nicholashoule/repogov/cmd/repogov@latest
```

```bash
go run github.com/nicholashoule/repogov/cmd/repogov@latest -agent copilot init
```

## Quick Start

### As a library

```go
package main

import (
    "fmt"
    "github.com/nicholashoule/repogov"
)

func main() {
    cfg := repogov.DefaultConfig()
    results, err := repogov.CheckDir(".", []string{".go", ".md"}, cfg)
    if err != nil {
        panic(err)
    }
    fmt.Print(repogov.Summary(results))

    schema := repogov.DefaultCopilotLayout()
    layout, err := repogov.CheckLayout(".", schema)
    if err != nil {
        panic(err)
    }
    fmt.Print(repogov.LayoutSummary(layout))
}
```

### As a CLI

```bash
# Go run (Copilot - init)
go run github.com/nicholashoule/repogov/cmd/repogov@latest -agent copilot init

# Check line limits (defaults to .md and .mdc files)
repogov -root . limits

# Include additional file types
repogov -root . -exts .md,.mdc,.yaml limits

# Check Copilot layout
repogov -root . layout

# Scaffold platform directory structure (agent flag must precede subcommand)
repogov -root . -agent copilot init

# Quiet mode (exit code only)
repogov -root . -quiet all
```

## Configuration

Create `.github/repogov-config.json`:

```json
{
  "default": 300,
  "warning_threshold": 80,
  "skip_dirs": [".git", "vendor"],
  "include_exts": [".md", ".mdc"],
  "rules": [
    {"glob": "docs/*.md", "limit": 1000}
  ],
  "files": {
    "docs/design.md": 500,
    "CHANGELOG.md": 0
  }
}
```

### Extension Filter

`include_exts` controls which file types are scanned. Defaults to `[".md", ".mdc"]`.
Set to `[]` (empty array) to scan every file type. Add `.yaml`, `.txt`, or any extension:

```json
"include_exts": [".md", ".mdc", ".yaml"]
```

The `-exts` CLI flag overrides this at runtime; pass `-exts all` to bypass the filter.

### Limit Resolution Order

1. Per-file override (`files` map, exact match)
2. First matching glob rule (`rules` array, in order)
3. `default` value (falls back to 300 if zero)
4. A limit of 0 exempts the file (status = SKIP)

## Public API

| Symbol | File | Purpose |
|--------|------|---------|
| `DefaultConfig()` | repogov.go | Sensible defaults (300 lines, 80% warn) |
| `ResolveLimit(path, cfg)` | repogov.go | Resolve effective limit for a path |
| `CountLines(path)` | check.go | Count lines in a file |
| `CheckFile(path, cfg)` | check.go | Check a single file |
| `CheckDir(root, exts, cfg)` | check.go | Walk and check a directory tree |
| `CheckDirContext(ctx, root, exts, cfg)` | check.go | Context-aware directory check |
| `LoadConfig(path)` | config.go | Load JSON/YAML config with defaults |
| `FindConfig(root)` | config.go | Auto-discover config file in standard locations |
| `SaveConfig(path, cfg)` | config.go | Write config as JSON or YAML |
| `ValidateConfig(cfg)` | config.go | Validate config and return violations |
| `InitLayout(root, schema)` | init.go | Scaffold platform directory structure |
| `InitLayoutWithConfig(root, schema, cfg)` | init.go | Scaffold with config options (always-create, filters) |
| `InitLayoutAll(root, schemas)` | init.go | Scaffold multiple platform schemas in one pass |
| `InitLayoutAllWithConfig(root, schemas, cfg)` | init.go | Multi-schema scaffold with config options |
| `CheckLayout(root, schema)` | layout.go | Validate directory structure |
| `CheckLayoutContext(ctx, root, schema)` | layout.go | Context-aware layout check |
| `DefaultCopilotLayout()` | presets.go | GitHub Copilot `.github/` preset |
| `DefaultCursorLayout()` | presets.go | Cursor AI `.cursor/` preset |
| `DefaultWindsurfLayout()` | presets.go | Windsurf `.windsurf/` preset |
| `DefaultClaudeLayout()` | presets.go | Claude Code `.claude/` preset |
| `DefaultGitLabLayout()` | presets.go | GitLab `.gitlab/` preset |
| `DefaultRootLayout()` | presets.go | Repository root layout preset |
| `Passed(results)` | format.go | Check if all results pass |
| `Summary(results)` | format.go | Human-readable limit summary |
| `LayoutPassed(results)` | format.go | Check if all layout results pass |
| `LayoutSummary(results)` | format.go | Human-readable layout summary |

## CLI Subcommands

| Subcommand | Description |
|------------|-------------|
| `limits` | Check file line counts against configured limits |
| `layout` | Validate directory structure against platform preset |
| `init` | Scaffold the platform directory structure |
| `validate` | Validate the configuration file and report issues |
| `all` | Run both limits and layout checks |
| `version` | Print version and exit |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | auto-discovered | Path to JSON or YAML config file (searched in repo root then `.github/`) |
| `-root` | `.` | Repository root directory |
| `-exts` | from config | Comma-separated extension filter override; use `all` to scan every file type (default read from `include_exts` in config) |
| `-agent` | | Agent preset(s): `copilot`, `cursor`, `windsurf`, `claude`, `gitlab`, `root`, `all`, or comma-separated list |
| `-quiet` | `false` | Suppress output; exit code only |
| `-json` | `false` | Output results as JSON |

## Standalone Usage (Zero Dependencies)

Because repogov has no external dependencies, you can embed it directly
in a pre-commit hook or a minimal CLI without pulling in any third-party
modules.

### As a pre-commit hook

Create a file such as `scripts/hooks/pre-commit.go` that imports repogov
and runs governance checks before each commit:

```go
// pre-commit.go -- standalone pre-commit hook using repogov.
```

Pair it with a shell wrapper at `.git/hooks/pre-commit`:

```bash
#!/bin/sh
droot="$(git rev-parse --show-toplevel)"
cd "${droot}/scripts/hooks" && exec go run pre-commit.go "${droot}"
```

### As a minimal CLI

Build a single-file CLI with no framework beyond `flag` and repogov:

```go
// main.go -- minimal repogov CLI, zero external dependencies.
package main

import (
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/nicholashoule/repogov"
)

func main() {
    root := flag.String("root", ".", "repository root directory")
    exts := flag.String("exts", ".md", "comma-separated extensions")
    flag.Parse()

    cfg, _ := repogov.LoadConfig(*root + "/.github/repogov-config.json")
    results, err := repogov.CheckDir(*root, strings.Split(*exts, ","), cfg)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
    fmt.Print(repogov.Summary(results))
    if !repogov.Passed(results) {
        os.Exit(1)
    }
}
```

## Status Codes

| Status | Meaning |
|--------|---------|
| PASS | File within limit or layout rule satisfied |
| WARN | File at or above warning percentage threshold |
| FAIL | File over limit or required item missing |
| SKIP | File exempt (limit=0) or extension-filtered |
| INFO | Optional layout item present |

## Development

```bash
make test       # Run all tests
make race       # Test with race detector
make coverage   # Coverage report
make fmt        # Format code
make vet        # Run go vet
make build      # Build CLI binary
```

## License

Apache License 2.0 -- see [LICENSE](LICENSE).
