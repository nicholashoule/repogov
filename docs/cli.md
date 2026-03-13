# repogov CLI Reference

`repogov` is a command-line tool for auditing repository file line-count limits and
directory layout conventions for AI-agent platforms (GitHub Copilot, Cursor, Windsurf,
Claude, Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant, GitLab)
and common repository root structure.

## Installation

Build from source:

```sh
go build -o repogov ./cmd/repogov/
```

Or install directly:

```sh
go install github.com/nicholashoule/repogov/cmd/repogov@latest
```

## Synopsis

```
repogov [flags] <subcommand>
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config <path>` | auto-discovered | Path to a JSON or YAML config file. Searched in repo root, then `.github/`. |
| `-root <dir>` | `.` | Repository root directory. |
| `-exts .md,.mdc` | from config `include_exts` | Comma-separated extension filter. Use `all` to scan every file type. |
| `-agent <name[,name…]>` | _(none)_ | Agent/layout preset(s): `copilot`, `cursor`, `windsurf`, `claude`, `kiro`, `gemini`, `continue`, `cline`, `roocode`, `jetbrains`, `gitlab`, `root`, or `all`. Required for `init`. Comma-separate for multiple. |
| `-quiet` | false | Suppress output; rely on exit code only. |
| `-json` | false | Output results as JSON. |

## Subcommands

### `limits`

Check file line counts against configured limits.

```sh
repogov -root . limits
repogov -root . -exts .md,.mdc limits
repogov -root . -exts all limits
repogov -root . -config .github/repogov-config.json limits
```

Walks the repository, checks every matching file against its resolved limit, and prints
a per-file summary. Exits `1` if any file exceeds its limit, `0` otherwise.

**Output columns:** `[STATUS]  path -- N/limit lines (pct% of limit)`

### `layout`

Validate the directory structure against one or more platform presets.

```sh
repogov -root . -agent copilot layout
repogov -root . -agent cursor layout
repogov -root . -agent kiro layout
repogov -root . -agent gemini layout
repogov -root . -agent gitlab layout
repogov -root . -agent root layout
repogov -root . -agent all layout
repogov -root . layout                  # defaults to all agents
```

Checks for required files, optional files, unexpected files, and naming conventions
(e.g., lowercase filenames inside `.cursor/rules/`). Exits `1` on any failure.

> **Note:** `-agent all` skips platform directories that do not exist in the
> repository (e.g., a Copilot-only repo will not fail for missing `.cursor/`).
> File-only schemas (Gemini) are similarly skipped when their required files are
> absent. The `root` preset is excluded from `all` — use `-agent root` explicitly.

### `init`

Scaffold the platform directory structure under the repository root.

```sh
repogov -root . -agent copilot init
repogov -root . -agent cursor init
repogov -root . -agent kiro init
repogov -root . -agent gemini init
repogov -root . -agent gitlab init
repogov -root . -agent copilot,windsurf init
repogov -root . -agent all init
```

Creates:
- The platform root directory (`.github/`, `.cursor/`, `.windsurf/`, `.claude/`, `.kiro/`, `.continue/`, `.clinerules/`, `.roo/`, `.aiassistant/`, `.gitlab/`)
- Required subdirectories
- Placeholder files for each required file that does not already exist

Existing files are **never overwritten**. Re-running `init` is safe.

### `validate`

Validate the configuration file and report structural/semantic issues.

```sh
repogov -root . validate
repogov -root . -config .github/repogov-config.json validate
repogov -root . -json validate
```

Exits `1` if any errors are found, `0` if the config is valid (warnings do not
cause a non-zero exit).

### `all`

Run both `limits` and `layout` checks in sequence.

```sh
repogov -root .
repogov -root . -agent copilot
repogov -root . -agent all
```

This is the default subcommand when none is specified. Exits non-zero if either
check fails.

### `version`

Print the version string and exit.

```sh
repogov version
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more checks failed |
| `2` | Usage error or configuration problem |

## Config File Discovery

When `-config` is not provided, `repogov` searches in this order:

1. `<root>/repogov-config.json`
2. `<root>/repogov-config.yaml` / `.yml`
3. `<root>/.github/repogov-config.json`
4. `<root>/.github/repogov-config.yaml` / `.yml`

If no file is found, built-in defaults are used (300-line default limit, 85% warning
threshold).

## JSON Output

Pass `-json` to receive machine-readable output. The schema for each subcommand:

**`limits -json`** — array of `Result` objects:

```json
[
  {
    "path": "README.md",
    "lines": 42,
    "limit": 300,
    "status": "PASS"
  }
]
```

**`layout -json`** — map of platform name to array of `LayoutResult` objects:

```json
{
  "copilot": [
    {
      "path": ".github/copilot-instructions.md",
      "status": "PASS",
      "message": ""
    }
  ]
}
```

**`validate -json`**:

```json
{
  "path": ".github/repogov-config.json",
  "valid": true,
  "violations": []
}
```

**`init -json`**:

```json
[
  {
    "platform": "copilot",
    "created": [
      ".github/instructions/",
      ".github/instructions/general.instructions.md"
    ]
  }
]
```

## Examples

```sh
# Run all checks for all agents from the repo root
repogov -root .

# Check only Copilot layout
repogov -root . -agent copilot layout

# Check GitLab layout
repogov -root . -agent gitlab layout

# Check root-level files (README, LICENSE, CONTRIBUTING, etc.)
repogov -root . -agent root layout

# Scaffold Copilot + Windsurf
repogov -root . -agent copilot,windsurf init

# Scaffold GitLab layout
repogov -root . -agent gitlab init

# Scan every file type, not just .md
repogov -root . -exts all limits

# Quiet mode (CI, exit code only)
repogov -root . -quiet

# JSON output for tooling
repogov -root . -json limits
```

## See Also

- [git-hooks.md](git-hooks.md) — pre-commit hook that runs these checks automatically
- [design.md](design.md) — architecture and design decisions
