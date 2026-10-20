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
repogov [flags] <subcommand> [flags]
```

Flags may appear before or after the subcommand (`repogov limits -json`,
`repogov validate -config x.json`). Any other argument after the subcommand is an
error (exit `2`).

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config <path>` | auto-discovered | Path to a JSON or YAML config file. Searched in repo root, then `.github/`. An explicit path that does not exist is an error (exit `2`); only auto-discovery falls back to defaults. |
| `-root <dir>` | `.` | Repository root directory. When omitted, `.` is walked up to the nearest `.git` ancestor; an explicit `-root` is used exactly as given. |
| `-exts .md,.mdc` | from config `include_exts` | Comma-separated extension filter. Use `all` to scan every file type. |
| `-agent <name[,name…]>` | _(none)_ | AI agent preset(s): `copilot`, `cursor`, `windsurf`, `claude`, `kiro`, `gemini`, `continue`, `cline`, `roocode`, `jetbrains`, `zed`, or `all`. Required for `init` (unless `-platform` is given). Comma-separate for multiple. |
| `-platform <name[,name…]>` | _(none)_ | Repository platform preset(s): `github`, `gitlab`, `bitbucket`, `root`, `auto`, or `all`. Comma-separate for multiple. Use explicitly — not included in `-agent all`. |
| `-language <tag>` | _(none)_ | Prose language as a BCP 47 tag (e.g. `en-US`), for `init` and `update-config` only (exit `2` elsewhere or for an unknown tag). See [Prose Language](#prose-language). |
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

Files that must never be parsed are reported as SKIP:

| SKIP reason | Detection | Notes |
|-------------|-----------|-------|
| exempt | resolved limit `0` | a byte limit (`max_bytes`) still applies |
| binary | NUL byte within the first 512 bytes | only the sniff prefix is read; a byte limit still applies (from the file size) |
| sensitive | `.crt`, `.pem`, `.key`, `.p12`, `.pfx`, `.jks` | never opened |
| symlink or special file | `Lstat` type | never opened or followed |

Text saved as UTF-16 or UTF-32 with a byte order mark is not binary: its lines are
counted in the decoded text. There is no maximum line length; minified or generated
files with very long lines are counted normally. A file that cannot be read is a FAIL,
never silently dropped.

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

For the `claude` agent, the layout run also performs a **read-only settings
audit** of `.claude/settings.json` and `.claude/settings.local.json`: an Info
summary (line and permission-rule counts) plus advisory findings for exact
duplicate rules, rules shadowed by a broader `:*` prefix rule, allow rules
overridden by deny, and consolidation suggestions when five or more literal
rules share a command prefix. Malformed JSON is a Fail (Claude Code cannot
load it). Every finding carries a `-- FIX:` hint so agents can act on it
directly. repogov **never creates or modifies settings files**, and
credential files (`.claude/.credentials.json`) are detected by filename only
-- their content is never opened or parsed.

> **Note:** `-agent all` skips platform directories that do not exist in the
> repository (e.g., a Copilot-only repo will not fail for missing `.cursor/`).
> File-only schemas (Gemini) are similarly skipped when their required files are
> absent. Repository platform presets (`gitlab`, `root`) are separate from agents —
> use `-platform gitlab` or `-platform root` explicitly.


**Agent pitfalls.** With `-agent all` (the default), `cursor`, or `gemini`, `layout`
also prints an "agent pitfalls" section (JSON key `pitfalls`) for configuration an agent
silently ignores:

| Finding | Meaning |
|---------|---------|
| WARN plain `.md` in `.cursor/rules/` | Cursor ignores it; project rules must be `.mdc` |
| FAIL `.agents/rules/*.md` without frontmatter or `trigger` | Antigravity silently discards the rule |
| FAIL unrecognized `trigger` (e.g. `alwaysOn`) | Must be `always_on`, `model_decision`, `glob`, or `manual` |
| WARN `model_decision` without `description`; FAIL `glob` without `globs` | Rule cannot work as written |
| WARN nested `.agents/rules/<dir>/*.md` | Scanned flat; ignored unless registered in `.agents/rules.json` `entries` (`path` relative to `.agents/`, `include_only`, `exclude`; exact paths). A registered nested rule is validated like a top-level one |
| WARN `.agents/rules.json` | Malformed, holds an unknown key, or not a regular file: nothing in it counts as registered |

`README.md` files are never flagged. Each finding carries a `-- FIX:` hint.

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
repogov -root . -agent claude init -language en-GB # state British English in agent files
```

Creates:
- The platform root directory (`.github/`, `.cursor/`, `.windsurf/`, `.claude/`, `.kiro/`, `.continue/`, `.clinerules/`, `.roo/`, `.aiassistant/` via `-agent`) or repository platform directory (`.gitlab/` via `-platform`) or a root-level file (`.rules` for Zed, `GEMINI.md` for Gemini CLI)
- Required subdirectories
- Placeholder files for each required file that does not already exist
- A curated set of scoped rule templates seeded into the platform's rules directory:
  `general`, `memory`, `codereview`, `governance`, `security`, `testing`, `library`,
  `backend`, `frontend`, `emoji-prevention`, `repo`
- `AGENTS.md` at the repository root (created, or only its marked Context section updated)
- `repogov-config.json`, only when no config exists yet at the repository root or in
  `.github/` (so init never adds a second, shadowed config) — for Copilot, placed in
  `.github/`; for all other agents, placed at `.github/` if that directory exists,
  otherwise at the repository root. Its `language` key (right after `_agent_hints`) is
  `en-US` unless `-language` gives another tag.

Existing files are **never overwritten**, with one exception: in an existing root
`AGENTS.md`, init replaces only the text between the marker comments
`<!-- repogov:context:begin -->` and `<!-- repogov:context:end -->` (a generated
`AGENTS.md` wraps its `## Context` section in them) and preserves every other byte.
Section bounds are never guessed from Markdown headings. When the file does not hold
exactly one begin marker followed by exactly one end marker -- markers missing,
duplicated or out of order -- init leaves it byte-for-byte unchanged and prints a
`note:` that the Context section was not refreshed, with the reason (a `notice`
field with `-json`); init still exits `0`, and `layout` lists it as INFO. To opt in, put the begin marker on the line before `## Context` and the
end marker after the section's last line, then re-run init. Init writes atomically,
skips the write when nothing changed, and refuses a symlinked or special
`AGENTS.md`. Re-running `init` is safe.

Every scaffolded agent file states the prose language: `-language <tag>`, else the
active config's `language` key, else `en-US`; see [Prose Language](#prose-language). An
existing config is not modified by init -- use `update-config -language` for that.

> **Note:** When `-root` is omitted, `.` is resolved to the nearest `.git` ancestor,
> so running `init` from inside an agent subdirectory (e.g., `.cursor/rules/`)
> targets the repository root instead of double-nesting directories. An explicit
> `-root` is used exactly as given (made absolute), never walked up; with no `.git`
> ancestor the working directory is used.

### `validate`

Validate the configuration file and report structural/semantic issues.

```sh
repogov -root . validate
repogov -root . -config .github/repogov-config.json validate
repogov -root . -json validate
```

Exits `1` if any errors are found, `0` if the config is valid (warnings do not
cause a non-zero exit).

### `update-config`

Add or refresh the `_agent_hints` block in an existing JSON or YAML config, so configs
created before hints existed (or with older hint wording) pick up the current
guidance for AI agents.

```sh
repogov -root . update-config
repogov -root . -config .github/repogov-config.json update-config
repogov -root . update-config -language en-US   # add or replace the language key
```

With `-language <tag>` the top-level `language` key is added (or its value replaced)
by the same kind of textual edit. Without `-language`, an absent key is added as
`"language": "en-US"` (the default) and an existing value is never touched. A newly
added key goes right after `_agent_hints`. The hints are the same for every config:
the `language` key's own one-line hint tells an agent to write in it.

The edit is textual: all other content (key order, compact arrays, indentation,
line endings) is preserved byte-for-byte, and the effective configuration is
verified unchanged before anything is written. Re-running is a no-op. In YAML
configs the hints are a top-level `_agent_hints:` mapping of double-quoted
strings, inserted after any leading comment header (and `---`); comments
elsewhere are preserved. A UTF-8 BOM and CRLF line endings are kept. Exits `2`
if no config is found or the file cannot be parsed.

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
| `2` | Usage error or configuration problem (unknown flag, subcommand or extra argument; unreadable or invalid config; explicit `-config` not found; unknown `-language`) |

## Config File Discovery

When `-config` is not provided, `repogov` searches in this order:

1. `<root>/repogov-config.json`
2. `<root>/repogov-config.yaml` / `.yml`
3. `<root>/.github/repogov-config.json`
4. `<root>/.github/repogov-config.yaml` / `.yml`

If no file is found, built-in defaults are used (500-line default limit, 85% warning
threshold). An explicit `-config` path must exist: a missing one exits `2` with the
path in the message.

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

The config file supports these keys (JSON and YAML). Parsing is strict: an unknown
key, tab indentation, or inline YAML flow syntax (other than an empty `[]`, or `{}`
for `files`) is an error naming the key or line. Top-level keys starting with `_`
(e.g. `_agent_hints`) are metadata and are ignored, as is a top-level `$schema` key
in JSON. YAML keys may be quoted (`"default": 300`).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `language` | string | `en-US` | Prose language (BCP 47 tag) stated in scaffolded agent files. A missing or empty value means `en-US`; set another tag to change it. Unknown tags are a `validate` warning. See [Prose Language](#prose-language). |
| `max_bytes` | object[] | `AGENTS.md`, `GEMINI.md`, `.agents/rules/*.md` at 24000 | Glob-based byte limits (`glob`, `limit`), first match wins; checked in addition to line limits. `[]` disables. |
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

Set `skip_frontmatter: true` in the config to disable this validation globally. A
UTF-8 byte order mark before the opening delimiter is accepted. A rule file that is a
symlink or special file is a SKIP (as in the limits check) and is never followed or
opened.

### Prose Language

`language` is the natural language of the repository's prose (Markdown and text
files, code comments, commit messages) -- not a programming language. The default is
`en-US`: a missing or empty `language` key means `en-US`, and there is no opt-out --
choosing another language means setting it. The scaffolded `AGENTS.md`,
`.claude/CLAUDE.md`, `.github/copilot-instructions.md`, `GEMINI.md`, Zed `.rules` and
the `general` rule state it, for example: "Prose language: British English (en-GB).
Write documentation (.md, .txt), code comments and commit messages in British
English, with its spelling and conventions (British spelling: colour, behaviour,
-ise); keep identifiers, code and quoted text as they are." Every config `init`
generates writes the key right after `_agent_hints`. Existing repos are not
re-scaffolded, so `init` also writes the line into the marked Context section of an
existing `AGENTS.md`, and `layout` WARNs when `AGENTS.md` does not contain the
language's tag in parentheses (e.g. `(en-US)`). There is no detection: set another
tag with `init -language` or `update-config -language`, or edit the config. Tags
match case-insensitively, accept `_`, and bare tags resolve to a default (`en` ->
`en-US`, `pt` -> `pt-BR`, `zh` -> `zh-Hans`). The full tag list is in the README
[Prose Language](../README.md#prose-language) table and `repogov.KnownLanguages()`.

## See Also

- [git-hooks.md](git-hooks.md) — pre-commit hook that runs these checks automatically
- [design.md](design.md) — architecture and design decisions
