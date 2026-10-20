# repogov

[![CI](https://github.com/nicholashoule/repogov/actions/workflows/ci.yml/badge.svg)](https://github.com/nicholashoule/repogov/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nicholashoule/repogov.svg)](https://pkg.go.dev/github.com/nicholashoule/repogov)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nicholashoule/repogov)](go.mod)
[![License](https://img.shields.io/github/license/nicholashoule/repogov)](LICENSE)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-none-brightgreen)](go.mod)

A dependency-free Go library and CLI for repository governance: enforce file-length limits and validate or scaffold AI-agent-ready directory layouts for Copilot, Cursor, Windsurf, Claude, Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant, Zed AI, GitLab, and common repository root conventions.

## Features

- **Line-count limits** -- configurable per-file, per-glob, and default
  limits with PASS/WARN/FAIL/SKIP classification
- **Layout governance** -- schema-based validation of `.github/`, `.cursor/`,
  `.windsurf/`, `.claude/`, `.kiro/`, `.continue/`, `.clinerules/`, `.roo/`,
  `.aiassistant/`, `.gitlab/`, and repository root directory structure
- **Platform presets** -- built-in rules for Copilot, Cursor, Windsurf, Claude,
  Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant, Zed AI,
  GitLab, and root conventions
- **Seeded rule files** -- `init` scaffolds a curated set of scoped rule templates
  (general, memory, codereview, governance, security, testing, and more) into each
  platform's rules directory; existing files are never overwritten (the one in-place
  edit is the text between `<!-- repogov:context:begin -->` and
  `<!-- repogov:context:end -->` in an existing root `AGENTS.md`; without exactly one
  ordered marker pair the file is left unchanged and `init` prints a note saying so --
  add the markers around the `## Context` section to opt in)
- **Prose language** -- a `language` key (BCP 47 tag such as `en-US`) that every
  scaffolded agent instruction file states, so AI agents write docs and comments in it
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

# Scaffold platform directory structure (flags may precede or follow the subcommand)
repogov -root . -agent copilot init

# Scaffold and record the prose language (American English)
repogov -root . -agent copilot init -language en-US

# Quiet mode (exit code only)
repogov -root . -quiet all
```

## Configuration

Create `.github/repogov-config.json` (or `repogov-config.json` at the repo root):

```json
{
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
    {"glob": "AGENTS.md", "limit": 24000}
  ],
  "files": {
    "docs/design.md": 500,
    "CHANGELOG.md": 0
  }
}
```

Config parsing is strict: an unknown key (for example a `warning_treshold` typo), tab
indentation, or inline YAML flow syntax (`[a, b]`, `{k: v}`; an empty `[]` is fine) is
an error, never a silent fall-back to defaults. Keys starting with `_` (such as the
generated `_agent_hints` block) are metadata and are ignored, as is a top-level
`$schema` key in JSON. YAML keys may be quoted (`"default": 300`).

| Key | Default | Purpose |
|-----|---------|---------|
| `language` | `en-US` | Prose language as a BCP 47 tag (missing or empty means `en-US`); see [Prose Language](#prose-language) |
| `default` | `500` | Fallback line limit |
| `warning_threshold` | `85` | Percent of a limit at which WARN begins (`85` or `"85%"`) |
| `skip_dirs` | `.git`, `vendor`, `workflows` | Directory names never entered |
| `include_exts` | `.md`, `.mdc` | Extensions scanned; `[]` scans everything |
| `rules` | agent rule globs | Glob line limits, first match wins |
| `max_bytes` | `AGENTS.md`, `GEMINI.md`, `.agents/rules/*.md` | Glob byte limits |
| `files` | agent instruction files | Exact-path line limits; `0` exempts |
| `descriptive_names`, `skip_frontmatter`, `init_always_create`, `init_include_files`, `init_exclude_files` | off / empty | Init and layout options; see [docs/cli.md](docs/cli.md#configuration-keys) |

### Prose Language

`language` names the natural language the repository's prose is written in --
Markdown and text files, code comments, commit messages -- as a BCP 47 tag. It is
not a programming language. The default is `en-US`: a missing or empty key means
`en-US`, and there is no opt-out -- choosing another language means setting it. Every
file `init` scaffolds for AI agents (`AGENTS.md`, `.claude/CLAUDE.md`,
`.github/copilot-instructions.md`, `GEMINI.md`, Zed `.rules`, the `general` rule)
carries a directive such as:

> Prose language: American English (en-US). Write documentation (.md, .txt), code
> comments and commit messages in American English, with its spelling and conventions
> (American spelling: color, behavior, -ize); keep identifiers, code and quoted text as
> they are.

Agents that read those files then write in that language and spelling. Regional
variants name their convention (American vs British spelling, Brazilian vs European
Portuguese, Simplified vs Traditional Chinese), and right-to-left scripts are marked.
Every config `init` generates writes the key, right after `_agent_hints`, and
`update-config` adds `"language": "en-US"` when it is absent. In an existing
`AGENTS.md` the line goes inside the marked Context section `init` refreshes, and
`layout` WARNs when `AGENTS.md` does not state the language (its tag in parentheses,
e.g. `(en-US)`). repogov does not guess the language: pass `-language <tag>` to `init`
(written into a new config) or `update-config` (added to, or replaced in, an existing
config; without the flag an existing value is never touched). `validate` warns on an
unknown tag and names the closest known ones.

Tags match case-insensitively and accept `_` for `-`. Bare tags resolve to a default:
`en` -> `en-US`, `es` -> `es-ES`, `pt` -> `pt-BR`, `fr` -> `fr-FR`, `de` -> `de-DE`,
`zh` (and `zh-CN`, `zh-SG`) -> `zh-Hans`, `zh-TW`/`zh-HK`/`zh-MO` -> `zh-Hant`,
`no`/`nb` -> `nb-NO`; any other bare tag resolves to its only variant (`ja` -> `ja-JP`,
`ar` -> `ar-SA`). The full set is exported as `KnownLanguages()`:

| Group | Tags |
|-------|------|
| English | `en-US` American, `en-GB` British, `en-AU` Australian, `en-CA` Canadian, `en-IN` Indian |
| Spanish, Portuguese | `es-ES` European, `es-MX` Mexican, `es-419` Latin American; `pt-BR` Brazilian, `pt-PT` European |
| French, German, Italian, Dutch | `fr-FR`, `fr-CA` Canadian; `de-DE`, `de-AT` Austrian, `de-CH` Swiss Standard; `it-IT`; `nl-NL` |
| Nordic | `sv-SE`, `nb-NO` Bokmål, `da-DK`, `fi-FI`, `is-IS` |
| Central and Eastern Europe | `pl-PL`, `cs-CZ`, `sk-SK`, `hu-HU`, `ro-RO`, `bg-BG`, `hr-HR`, `sr-RS` (Cyrillic), `sl-SI`, `uk-UA`, `ru-RU`, `el-GR`, `tr-TR`, `lt-LT`, `lv-LV`, `et-EE` |
| Right-to-left | `he-IL` Hebrew, `ar-SA` Arabic, `fa-IR` Persian, `ur-PK` Urdu |
| South Asia | `hi-IN`, `bn-BD`, `ta-IN`, `te-IN`, `mr-IN` |
| East and Southeast Asia | `ja-JP`, `ko-KR`, `zh-Hans` Simplified Chinese, `zh-Hant` Traditional Chinese, `th-TH`, `vi-VN`, `id-ID`, `ms-MY`, `fil-PH` |
| Africa and regional Europe | `sw-KE`, `af-ZA`, `ca-ES` Catalan, `eu-ES` Basque, `gl-ES` Galician, `cy-GB` Welsh, `ga-IE` Irish |

### Extension Filter

`include_exts` controls which file types are scanned. Defaults to `[".md", ".mdc"]`.
Set to `[]` (empty array) to scan every file type. Add `.yaml`, `.txt`, or any extension:

```json
"include_exts": [".md", ".mdc", ".yaml"]
```

The `-exts` CLI flag overrides this at runtime; pass `-exts all` to bypass the filter.

### Byte Limits

`max_bytes` adds size limits (bytes, first matching glob wins) on top of the line
limits. Some agents truncate or drop instruction files by size, not lines, so a file
with few very long lines can pass the line check and still be cut off. Defaults cover
`AGENTS.md`, `GEMINI.md` and `.agents/rules/*.md` at 24,000 bytes (Antigravity's
per-file limit, applied after include expansion). A file over the limit is FAIL;
at or above the warning threshold is WARN. Set `"max_bytes": []` to disable.

### Agent Pitfalls

`repogov layout` also reports configuration that an agent silently ignores: plain
`.md` files in `.cursor/rules/` (Cursor requires `.mdc`), and `.agents/rules/*.md`
files with missing or invalid frontmatter or nested below the flat rules directory
(Antigravity discards them without any error).

### Limit Resolution Order

1. Per-file override (`files` map, exact match)
2. First matching glob rule (`rules` array, in order)
3. `default` value (falls back to the built-in 500 if zero)
4. A limit of 0 exempts the file (status = SKIP)

## Public API

| Symbol | File | Purpose |
|--------|------|---------|
| `DefaultConfig()` | repogov.go | Sensible defaults (500 lines, 85% warn) |
| `DetectPlatforms(root)` | detect.go | Detect repository platforms from markers (.github/, .gitlab/, bitbucket-pipelines.*) |
| `ResolveLimit(path, cfg)` | repogov.go | Resolve effective limit for a path |
| `CountLines(path)` | check.go | Count lines in a file |
| `CheckFile(path, cfg)` | check.go | Check a single file |
| `CheckDir(root, exts, cfg)` | check.go | Walk and check a directory tree |
| `CheckDirContext(ctx, root, exts, cfg)` | check.go | Context-aware directory check |
| `LoadConfig(path)` | config.go | Load JSON/YAML config with defaults |
| `FindConfig(root)` | config.go | Auto-discover config file in standard locations |
| `FindAllConfigs(root)` | config.go | Find all config files, ordered by precedence |
| `SaveConfig(path, cfg)` | config.go | Write config as JSON or YAML |
| `ValidateConfig(cfg)` | config.go | Validate config and return violations |
| `UpgradeConfigHints(path)` | config_upgrade.go | Add or refresh the `_agent_hints` block in a JSON/YAML config (textual edit) |
| `UpgradeConfigWithLanguage(path, tag)` | config_upgrade.go | Like `UpgradeConfigHints`, and set the `language` key |
| `KnownLanguages()` | language.go | The closed set of prose languages (`Language`: tag, name, convention, RTL) |
| `LookupLanguage(tag)` / `NormalizeLanguage(tag)` | language.go | Resolve a tag, alias, or bare subtag to its canonical language |
| `Language.Directive()` | language.go | The prose-language line written into scaffolded agent files |
| `InitLayout(root, schema)` | init.go | Scaffold platform directory structure |
| `InitLayoutWithConfig(root, schema, cfg)` | init.go | Scaffold with config options (always-create, filters) |
| `InitLayoutAll(root, schemas)` | init.go | Scaffold multiple platform schemas in one pass |
| `InitLayoutAllWithConfig(root, schemas, cfg)` | init.go | Multi-schema scaffold with config options |
| `CheckLayout(root, schema)` | layout.go | Validate directory structure |
| `CheckLayoutContext(ctx, root, schema)` | layout.go | Context-aware layout check |
| `StripFrontmatter(schema)` | layout.go | Return schema copy with frontmatter requirements cleared |
| `DefaultCopilotLayout()` | presets.go | GitHub Copilot `.github/` preset |
| `DefaultCursorLayout()` | presets.go | Cursor AI `.cursor/` preset |
| `DefaultWindsurfLayout()` | presets.go | Windsurf `.windsurf/` preset |
| `DefaultClaudeLayout()` | presets.go | Claude Code `.claude/` preset |
| `DefaultKiroLayout()` | presets.go | Kiro `.kiro/steering/` preset |
| `DefaultGeminiLayout()` | presets.go | Gemini CLI root-level `GEMINI.md` preset |
| `DefaultContinueLayout()` | presets.go | Continue `.continue/rules/` preset |
| `DefaultClineLayout()` | presets.go | Cline `.clinerules/` preset |
| `DefaultRooCodeLayout()` | presets.go | Roo Code `.roo/rules/` preset |
| `DefaultJetBrainsLayout()` | presets.go | JetBrains AI Assistant `.aiassistant/rules/` preset |
| `DefaultZedLayout()` | presets.go | Zed AI root-level `.rules` preset |
| `DefaultGitHubLayout()` | presets.go | GitHub platform `.github/` preset |
| `DefaultGitLabLayout()` | presets.go | GitLab `.gitlab/` preset |
| `DefaultBitbucketLayout()` | presets.go | Bitbucket root-level preset |
| `DefaultRootLayout()` | presets.go | Repository root layout preset |
| `ResolveByteLimit(path, cfg)` | repogov.go | Resolve the byte-size limit for a path (0 = none) |
| `CheckAgentPitfalls(root)` | pitfalls.go | Read-only detection of config an agent silently ignores (Cursor plain `.md` rules; Antigravity `.agents/rules` frontmatter and nesting) |
| `CheckClaudeSettings(root)` | settings.go | Read-only audit of `.claude/settings*.json` permission rules (duplicates, redundancy, conflicts, consolidation); never modifies settings, never parses credential files |
| `UpdateAgentsMdContextAll(root, schemas)` | scaffold.go | Refresh the marked AGENTS.md Context section with merged platform links (an unmarked file is left unchanged) |
| `AgentsMdContextStatus(root)` | scaffold.go | Whether AGENTS.md has the one ordered marker pair init refreshes, and if not, why and how to opt in |
| `CheckAgentsMdLanguage(root, lang)` | language.go | WARN when AGENTS.md does not state the configured prose language |
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
| `update-config` | Add or refresh the `_agent_hints` block in the JSON or YAML config; with `-language`, also set the `language` key |
| `all` | Run both limits and layout checks (default) |
| `version` | Print version and exit |

## CLI Flags

Flags may appear before or after the subcommand (`repogov limits -json`); any other
argument after the subcommand is an error.

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | auto-discovered | Path to JSON or YAML config file (searched in repo root then `.github/`). An explicit path that does not exist is an error (exit 2), never a silent fall-back to defaults |
| `-root` | `.` | Repository root directory. When omitted, `.` is walked up to the nearest git root; an explicit `-root` is used as given |
| `-exts` | from config | Comma-separated extension filter override; use `all` to scan every file type (default read from `include_exts` in config) |
| `-agent` | | AI agent preset(s): `copilot`, `cursor`, `windsurf`, `claude`, `kiro`, `gemini`, `continue`, `cline`, `roocode`, `jetbrains`, `zed`, `all`, or comma-separated list |
| `-platform` | | Repository platform preset(s): `github`, `gitlab`, `bitbucket`, `root`, `auto`, `all`, or comma-separated list |
| `-language` | | Prose language tag (e.g. `en-US`) for `init` and `update-config`; see [Prose Language](#prose-language) |
| `-descriptive` | `false` | Use `*.instructions.md` naming convention for seeded files (overrides config `descriptive_names`) |
| `-seed` | `false` | Seed missing template files into existing directories without overwriting (`init` only) |
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

    cfg, err := repogov.LoadConfig(*root + "/.github/repogov-config.json")
    if err != nil {
        // A malformed config must fail the hook, not fall back to defaults.
        fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
        os.Exit(2)
    }
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
| SKIP | File not checked: exempt (limit 0); binary (NUL byte in the first 512 bytes, no UTF-16/32 byte order mark -- still held to `max_bytes`); sensitive (`.crt`, `.pem`, `.key`, `.p12`, `.pfx`, `.jks`, never opened); or a symlink or special file (never opened) |
| INFO | Optional layout item present |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed (WARN does not fail) |
| `1` | One or more checks failed (FAIL), or `validate` found an error (with `-json` or `-quiet`, any violation) |
| `2` | Usage or configuration error: unknown flag, subcommand or argument; unreadable or invalid config; explicit `-config` not found; unknown `-language` |

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
