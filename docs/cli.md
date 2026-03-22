# repogov CLI Reference

`repogov` is a command-line tool for auditing repository file line-count limits and
directory layout conventions for AI-agent platforms (GitHub Copilot, Cursor, Windsurf,
Claude, Kiro, Gemini CLI, Continue, Cline, Roo Code, JetBrains AI Assistant, Zed AI,
GitLab) and common repository root structure.

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
| `-agent <name[,name…]>` | _(none)_ | AI agent preset(s): `copilot`, `cursor`, `windsurf`, `claude`, `kiro`, `gemini`, `continue`, `cline`, `roocode`, `jetbrains`, `zed`, or `all`. Required for `init` (unless `-platform` is given). Comma-separate for multiple. |
| `-platform <name[,name…]>` | _(none)_ | Repository platform preset(s): `gitlab`, `root`, or `all`. Comma-separate for multiple. Use explicitly — not included in `-agent all`. |
| `-descriptive` | `false` | Use `*.instructions.md` naming convention for seeded files (overrides config `descriptive_names`). |
| `-seed` | `false` | Seed missing template files into existing directories without overwriting (`init` only). Maps to `init_always_create` at runtime. |
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
repogov -root . -agent windsurf layout
repogov -root . -agent claude layout
repogov -root . -agent kiro layout
repogov -root . -agent gemini layout
repogov -root . -agent continue layout
repogov -root . -agent cline layout
repogov -root . -agent roocode layout
repogov -root . -agent jetbrains layout
repogov -root . -agent zed layout
repogov -root . -platform gitlab layout
repogov -root . -platform root layout
repogov -root . -agent all layout
repogov -root . layout                  # defaults to all agents
```

Checks for required files, optional files, unexpected files, and naming conventions
(e.g., lowercase filenames inside `.cursor/rules/`). Exits `1` on any failure.

> **Note:** `-agent all` skips platform directories that do not exist in the
> repository (e.g., a Copilot-only repo will not fail for missing `.cursor/`).
> File-only schemas (Gemini) are similarly skipped when their required files are
> absent. Repository platform presets (`gitlab`, `root`) are separate from agents —
> use `-platform gitlab` or `-platform root` explicitly.

### `init`

Scaffold the platform directory structure under the repository root.

```sh
repogov -root . -agent copilot init
repogov -root . -agent cursor init
repogov -root . -agent windsurf init
repogov -root . -agent claude init
repogov -root . -agent kiro init
repogov -root . -agent gemini init
repogov -root . -agent continue init
repogov -root . -agent cline init
repogov -root . -agent roocode init
repogov -root . -agent jetbrains init
repogov -root . -agent zed init
repogov -root . -platform gitlab init
repogov -root . -agent copilot,windsurf init
repogov -root . -agent all init
repogov -root . -agent copilot -seed init       # seed missing files into existing dirs
repogov -root . -agent copilot -descriptive init # use *.instructions.md naming
```

Creates:
- The platform root directory (`.github/`, `.cursor/`, `.windsurf/`, `.claude/`, `.kiro/`, `.continue/`, `.clinerules/`, `.roo/`, `.aiassistant/` via `-agent`) or repository platform directory (`.gitlab/` via `-platform`) or a root-level file (`.rules` for Zed, `GEMINI.md` for Gemini CLI)
- Required subdirectories
- Placeholder files for each required file that does not already exist
- A curated set of scoped rule templates seeded into the platform's rules directory:
  `general`, `memory`, `codereview`, `governance`, `security`, `testing`, `library`,
  `backend`, `frontend`, `emoji-prevention`, `repo`
- `AGENTS.md` at the repository root (created or context-section updated)
- `repogov-config.json` — for Copilot, placed in `.github/`; for all other agents,
  placed at `.github/` if it already exists, otherwise at the repository root so
  `FindConfig` can discover it without an explicit `-config` flag

Existing files are **never overwritten**. Re-running `init` is safe.

> **Note:** `-root` is resolved to the nearest `.git` ancestor when one exists;
> if no `.git` directory is found, the resolved absolute `-root` path is used.
> Running `init` from inside an agent subdirectory (e.g., `.cursor/rules/`) or
> passing an explicit agent path via `-root .cursor` both correctly target the
> repository root for Git repositories, while non-git or temporary directories
> are scaffolded in-place, preventing double-nested directories.

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

If no file is found, built-in defaults are used (500-line default limit, 85% warning
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
repogov -root . -platform gitlab layout

# Check root-level files (README, LICENSE, CONTRIBUTING, etc.)
repogov -root . -platform root layout

# Scaffold Copilot + Windsurf
repogov -root . -agent copilot,windsurf init

# Scaffold GitLab layout
repogov -root . -platform gitlab init

# Scan every file type, not just .md
repogov -root . -exts all limits

# Quiet mode (CI, exit code only)
repogov -root . -quiet

# JSON output for tooling
repogov -root . -json limits
```

## Configuration Keys

The config file supports these keys (JSON and YAML):

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default` | int | `500` | Global fallback line limit when no per-file or glob rule matches. |
| `warning_threshold` | int or string | `85` / `"85%"` | Percentage of the limit at which PASS becomes WARN. Accepts bare int or percent string. |
| `include_exts` | string[] | `[".md", ".mdc"]` | File extensions to scan. Empty array scans all types. |
| `skip_dirs` | string[] | `[".git", "vendor", "workflows"]` | Directory names to skip during walks. |
| `rules` | object[] | _(agent-based defaults)_ | Glob-based limit rules (`glob`, `limit`). First match wins. |
| `files` | map[string]int | _(agent-based defaults)_ | Per-file limit overrides. Key is repo-relative path (forward slashes). Limit `0` exempts. |
| `init_always_create` | bool | `false` | Seed default template files into existing non-empty directories. CLI `-seed` sets this at runtime. |
| `descriptive_names` | bool | `false` | Use `*.instructions.md` naming convention for seeded files. CLI `-descriptive` overrides this. |
| `init_include_files` | string[] | `[]` | Allowlist of template stems to seed during init (e.g. `["general", "testing"]`). When non-empty, only listed stems are created. Takes precedence over `init_exclude_files`. |
| `init_exclude_files` | string[] | `[]` | Blocklist of template stems to skip during init (e.g. `["backend", "frontend"]`). Ignored when `init_include_files` is non-empty. |
| `skip_frontmatter` | bool | `false` | Disable YAML frontmatter validation during layout checks. When false, files in directories with `DirRule.Frontmatter` set are checked for required keys (e.g. `applyTo`). |

### Frontmatter Validation

Layout schemas can require YAML frontmatter keys in specific directories. For example,
the Copilot preset requires `applyTo` in `.github/instructions/` and `.github/rules/`
files. During layout checks, each file in a managed directory with a `Frontmatter`
requirement is validated for:

1. A YAML frontmatter opening delimiter (`---`) on the first line
2. A closing delimiter (`---`)
3. All required keys present (e.g. `applyTo`)

Set `skip_frontmatter: true` in the config to disable this validation globally.

## See Also

- [git-hooks.md](git-hooks.md) — pre-commit hook that runs these checks automatically
- [design.md](design.md) — architecture and design decisions
