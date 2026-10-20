# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.9.0] - 2026-10-20

### Breaking

- **Config parsing is strict.** v0.8.0 silently ignored what now fails with the offending key or line named: unknown keys in JSON (top-level or inside a rule) and YAML (e.g. a `warning_treshold` typo, a `limt` rule key); YAML tab indentation; YAML inline flow collections (`skip_dirs: [vendor, dist]`, `files: {README.md: 10}`; an empty `[]`, or `{}` for `files`, is still allowed); a scalar where a list belongs (`skip_dirs: vendor`); and stray indented lines. To fix: correct the key name (`repogov validate` names it), put metadata under a key starting with `_` (a top-level `$schema` key in JSON is also accepted), indent with spaces, and write lists as one `- value` per line. YAML `skip_dirs: []` / `rules: []` now mean an explicit empty list (as in JSON) rather than "use the defaults", and indentless block lists (`skip_dirs:` followed by `- vendor` at column 0) are read instead of ignored. YAML keys may be quoted (`"default": 300`)
- **An explicit `-root` is used as given**; only an omitted `-root` (`.`) is walked up to the nearest git root. v0.8.0 walked an explicit subdirectory such as `-root .github` up to the repository root; pass the repository root itself instead
- **`init` refreshes only a marked Context section in an existing `AGENTS.md`.** Section bounds are no longer guessed from Markdown headings (setext, indented or tab-separated headings, backtick fences whose info string contains a backtick, and `## Context` inside an HTML comment all made the old splice delete user text). Only the text between `<!-- repogov:context:begin -->` and `<!-- repogov:context:end -->` is replaced; generated `AGENTS.md` files carry the markers. A file without exactly one begin marker followed by exactly one end marker -- every hand-written `AGENTS.md` until the markers are added -- is left byte-for-byte unchanged: `init` prints a `note:` that its Context section was not refreshed (a `notice` field with `-json`; exit `0`), and `layout` lists it as INFO. To opt in, put the begin marker on the line before `## Context` and the end marker after the section, then re-run `init`. Library calls still return `nil` in that case; the new `AgentsMdContextStatus(root)` reports whether the section is refreshable and, if not, why
- **The prose language defaults to `en-US`** (`DefaultLanguage`): `DefaultConfig().Language` is `en-US`, `LoadConfig` fills a missing or empty `language` with it, and every scaffolded agent file states it; there is no opt-out. To use another language, set `language` (or pass `-language <tag>` to `init` or `update-config`)

### Security

- `CheckDir`/`CheckFile` no longer open symlinks or special files (reported as SKIP); unreadable files now FAIL instead of vanishing
- Scaffolding creates files with `O_EXCL` (`writeNewFile`), so it never writes through a symlink or races a creator
- `init` no longer writes through a symlinked `AGENTS.md`: the Context update `Lstat`s the file, refuses symlinks and special files with a reason, caps it at 4 MiB, and writes atomically (temp file + `fsync` + rename); `SaveConfig` likewise refuses symlinks and writes atomically, keeping an existing file's permissions
- Frontmatter checks `Lstat` each rule file and report a symlink or special file as SKIP, never followed or opened, consistent with the limits check (a FIFO previously hung the check; a symlink was followed)
- Sensitive-extension skip list gains `.p12`, `.pfx`, `.jks`
- CI: `upload-artifact` pinned to a SHA; the unused workflow-level `pull-requests: write` permission removed (only `contents: read` remains); every `actions/checkout` step sets `persist-credentials: false`
- Text output neutralizes control characters, ANSI escapes and Unicode bidi overrides in file names and messages (`Summary`, `LayoutSummary`), so a crafted file name cannot forge log lines or GitHub Actions workflow commands (`::error::`); JSON output was already escaped
- Agent-pitfall checks never open symlinks or special files, cap reads at 1 MiB, skip dependency/build trees (`node_modules`, `dist`, `build`, `target`, `.venv`, ...) and stop at depth 8
- `update-config` writes atomically (temp file + rename, permissions preserved) and refuses configs over 4 MiB
- Line-exempt files that fail a byte limit now show the reason in the text summary

### Changed

- `_agent_hints` is redesigned as short, agent-neutral decision hints: one actionable line per situation (`on_fail`, `on_warn`, `on_pitfall`, `on_settings`, `exempt`, `never`, `verify`) and one per config key, the `language` key included. The block is the same for every config and names no specific agent. `never` holds two agent rules (edit no files outside the repo; never read or paste secrets), and a settings finding is reported to the user without widening permissions. `update-config` refreshes an existing config to the new block.
- Minimum Go version raised from 1.22 to 1.24 (`go.mod`, CI matrix, `docs/design.md`)
- CI matrix now covers Go 1.24, 1.25, and stable
- `scripts/hooks/pre-commit` — bumped the self-governance pin from
  `repogov@v0.7.2` to the released `repogov@v0.8.0` (post-release pin
  sweep per the PIN POLICY comment at the pin site)
- Every config `init` generates writes `"language"` right after `_agent_hints` (`en-US` unless `-language` gives another tag), and `update-config` adds `"language": "en-US"` there when the key is absent; an existing value is never overwritten without `-language`
- `LayoutSummary` counts SKIP results (shown as `| N skip` only when present)
- CLI: flags are accepted after the subcommand (`repogov limits -json`, `repogov validate -config x.json`); a stray argument after the subcommand exits `2` with a reason instead of being ignored
- `init` creates no config when one already exists at the repo root or in `.github/` (it previously added a second, shadowed or shadowing, config)

### Fixed

- **`init -agent cursor` now seeds `.mdc` rules with Cursor frontmatter** (`description`, `globs`, `alwaysApply`) instead of plain `.md` files with Copilot `applyTo` frontmatter, which Cursor ignores. `descriptive_names` no longer affects Cursor. Existing `.md` rules are not renamed (init never overwrites); `layout` warns about them with a rename hint. `.cursor/rules/memory.mdc` gets the 200-line limit
- `-agent zed` layout accepts any of Zed's documented rules files (`.cursorrules`, `.windsurfrules`, `.clinerules`, `.github/copilot-instructions.md`, `AGENT.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) in place of `.rules`
- Root layout no longer reports files nested inside managed directories (`.claude/rules/a.md`, `.github/workflows/ci.yml`, `docs/sub/b.md`, `.opencode/agents/a.md`) as unexpected: a `NoCreate` directory with no glob now covers its whole subtree. Files outside managed directories are still flagged
- Root layout recognizes `opencode.json`/`opencode.jsonc` (optional) and `.gemini/`, `.opencode/`, `.agents/`, `.codex/` (managed directories)
- `-agent gemini` layout accepts `AGENTS.md` in place of `GEMINI.md` (new `LayoutSchema.Alternatives`); read by Antigravity and OpenCode. `init` still creates `GEMINI.md`
- `LoadConfig` now accepts configs saved with a UTF-8 BOM (common from Windows editors/PowerShell); previously JSON/YAML parsing failed with "invalid character"
- **`init` no longer corrupts an existing `AGENTS.md`.** The `## Context` update matched `## Context` anywhere (including `### Context` and mid-line) and, without a following `## ` heading, replaced everything to end of file. It now replaces only the text between the Context markers (see Breaking), keeps CRLF line endings, and skips the write when nothing changed. Docs no longer claim init never modifies existing files
- Atomic writes (`SaveConfig`, `update-config`, the AGENTS.md refresh) failed on Windows with "Access is denied" when an editor, indexer or antivirus scanner held the file open (`os.WriteFile` had succeeded). The rename is now retried for `ERROR_ACCESS_DENIED`/`ERROR_SHARING_VIOLATION` with a doubling backoff bounded at 1 s, then fails naming the cause and the next step; the temp file is `fsync`ed before the rename, so a crash can never leave a truncated config
- `scripts/hooks/pre-commit` lists staged and changed files with `git -c core.quotePath=false`, so files with non-ASCII names are re-staged after an emoji substitution (they were printed as quoted octal escapes and never matched)
- `SaveConfig` YAML round-trips an empty string (`yamlQuote("")` is `""`, so `skip_dirs: [""]` no longer writes a bare `-` that fails to load) and writes `rules: []` for an explicit empty rule list (it was dropped, so the defaults came back on load)
- A file starting `FF FE 00 00` is read as UTF-32LE only when its length is a multiple of 4, and otherwise as UTF-16LE (a UTF-16LE BOM followed by U+0000)
- **An explicit `-config` that does not exist is an error** (exit `2`, "config file ... not found") for every subcommand; previously all checks silently ran against defaults. Auto-discovery still falls back to defaults
- **Flags after the subcommand were silently dropped** (`repogov limits -config x` used the auto-discovered config; `repogov validate -json` printed text)
- **UTF-16/UTF-32 files were skipped as binary.** A UTF-16 or UTF-32 byte order mark is recognized before the NUL-byte sniff and lines are counted in the decoded text; binary files remain unparsed but are now held to their `max_bytes` limit (from the file size). A failed `Stat` during a byte check is now an error rather than a pass
- Frontmatter validation accepts a UTF-8 BOM before the opening `---`
- `init` linked `governance.md` to a shadowed config and checked `.github/` before the repo root; the link now follows `FindConfig` precedence (root, then `.github/`; JSON, then YAML)
- `_agent_hints` carried a duplicate, mis-indented `max_bytes` key; removed. The pinning test now compares the repository config's hints byte-for-byte and rejects duplicate keys
- YAML `_agent_hints` values now escape backslashes, newlines and control characters (the backslash replacement was a no-op)
- Agent-pitfall checks run for `-agent Cursor` / `-agent GEMINI` (agent names are now matched case-insensitively)
- `scripts/hooks/pre-commit` no longer runs `git add -u` after demojify substitutions (which staged every modified tracked file and broke partial staging); it re-stages only the staged files the substitution changed and lists partially staged ones for manual review
- CLI help: `update-config` described as editing the JSON or YAML config; subcommand list, `-platform all`, flag placement and exit codes documented

### Added

- **Prose language key** (`language`, `Config.Language`): the BCP 47 tag of the natural language the repository's prose (Markdown, text files, code comments, commit messages) is written in -- not a programming language. A closed set of 63 tags covers English, Spanish and Portuguese regional variants, major European, Middle Eastern, South Asian, East and Southeast Asian and African languages (`KnownLanguages`, with display name, spelling/script convention and right-to-left flag for `he`, `ar`, `fa`, `ur`); `NormalizeLanguage`/`LookupLanguage` match case-insensitively, accept `_`, and resolve aliases and bare tags to documented defaults (`en` -> `en-US`, `pt` -> `pt-BR`, `zh`/`zh-CN` -> `zh-Hans`, `zh-TW`/`zh-HK` -> `zh-Hant`). Every scaffolded agent instruction file (`AGENTS.md`, `CLAUDE.md`, `copilot-instructions.md`, `GEMINI.md`, Zed `.rules`, the `general` rule) carries a directive telling agents to write in that language and spelling (`Language.Directive`); the default is `en-US` (see Breaking). `validate` warns on an unknown tag and names the closest known ones. No detection: `init -language <tag>` writes it into a new config, and `update-config -language <tag>` (`UpgradeConfigWithLanguage`) adds or replaces it in an existing JSON/YAML config; without the flag an existing value is never touched. `.github/repogov-config.json` sets `en-US`
- The language reaches existing repos, which `init` never re-scaffolds: the marked `AGENTS.md` Context section carries the language line, and `layout` WARNs (`CheckAgentsMdLanguage`) when `AGENTS.md` does not state the configured language as its tag in parentheses, e.g. `(en-US)`, with a FIX hint (add the Context markers and re-run `init`, or state it by hand)

- **Byte limits** (`max_bytes`, `ResolveByteLimit`; `Result.Bytes`/`ByteLimit`): size limits checked in addition to line limits, because some agents truncate by bytes (Antigravity: 24,000 per rule file). Defaults cover `AGENTS.md`, `GEMINI.md`, `.agents/rules/*.md`; supported in JSON and YAML, validated, and written into generated configs. A line-exempt file is still byte-checked
- **Agent pitfall checks** (`CheckAgentPitfalls`, `pitfalls.go`), reported by `layout`/`all` (text and JSON `pitfalls` key): Cursor plain `.md` rules (ignored by Cursor) and Antigravity `.agents/rules` files with missing/invalid frontmatter or trigger (silently discarded), `model_decision` without description, `glob` without globs, and unregistered nested rule files
- `_agent_hints` block (ignored by the loader) guiding AI agents: written by the config generator (`config_hints.go`) into every new `repogov-config.json` and mirrored in `.github/repogov-config.json`, pinned by a test
- `make lint` target
- **`repogov update-config`** (`UpgradeConfigHints`, `config_upgrade.go`) -- adds or refreshes the `_agent_hints` block in an existing JSON config as a textual edit: all other bytes (key order, compact arrays, indentation, CRLF) are preserved, the effective config is verified unchanged before writing, and re-running is a no-op. Works on JSON and YAML configs (YAML hints are a top-level `_agent_hints:` mapping of double-quoted strings, generated from the same source as the JSON block and skipped by repogov's YAML reader); a UTF-8 BOM and CRLF endings are preserved. Configs written before hints existed keep working unchanged

### Documentation

- `docs/compliance/*`: every audit re-verified against current vendor docs (2026-10-17). New deprecation notices: Roo Code (archived 2026-05-15), Windsurf (moving to Devin docs), Gemini CLI (partly superseded by Antigravity CLI), Zed (Rules Library replaced by Skills), Devin Knowledge, Plandex Cloud, Sourcegraph Cody Free/Pro. Cursor now documented as `.mdc`-only; Claude audit gains skills, `CLAUDE.local.md`, permission precedence and a `CheckClaudeSettings` section. `AI_AGENTS_AUDIT.md` gains a Status Review table with preset follow-ups
- Dropped `PLANDEX_AUDIT.md` and `SOURCEGRAPH_AUDIT.md` (Plandex Cloud wound down; Cody Free/Pro discontinued; neither has a file-based instruction path) -- see "Dropped Agents" in `AI_AGENTS_AUDIT.md`
- Cursor: `.cursorrules` documented as legacy/deprecated (ignored by Agent mode)
- Copilot: `.github/prompts/`, `.github/skills/` and `.github/agents/` paths confirmed against GitHub's customization cheat sheet
- JetBrains: Junie guideline lookup order documented (`.junie/AGENTS.md`, root `AGENTS.md` + `playbook.md` + `rules/*.md`, legacy `guidelines.md`)
- New `OPENCODE_AUDIT.md`: OpenCode qualifies for a Tier 2 preset (not built); registered in `AI_AGENTS_AUDIT.md`
- Gemini/Antigravity: `.agents/rules/*.md` rules, `trigger:` frontmatter, flat scanning and `rules.json` documented from antigravity.google/docs/rules; includes (`@[label](path)`, `@filename`) and limits (24 KB per file after expansion, 20k-token always-on budget) added
- Codex: `.codex/rules/*.rules` (Starlark command-execution policy, trusted projects only) documented from the vendor rules page
- Primary-source follow-up: Bitbucket `.bitbucket/CODEOWNERS` confirmed; Codex hooks (`.codex/hooks.json`) and custom agents (`.codex/agents/*.toml`) paths confirmed; Windsurf rename to Devin Desktop confirmed on vendor docs

[Unreleased]: https://github.com/nicholashoule/repogov/compare/v0.9.0...HEAD
[v0.9.0]: https://github.com/nicholashoule/repogov/compare/v0.8.0...v0.9.0
[v0.8.0]: https://github.com/nicholashoule/repogov/compare/v0.7.2...v0.8.0
[v0.7.2]: https://github.com/nicholashoule/repogov/compare/v0.7.1...v0.7.2
[v0.7.1]: https://github.com/nicholashoule/repogov/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/nicholashoule/repogov/compare/v0.6.1...v0.7.0
[v0.6.1]: https://github.com/nicholashoule/repogov/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/nicholashoule/repogov/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/nicholashoule/repogov/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/nicholashoule/repogov/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/nicholashoule/repogov/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/nicholashoule/repogov/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/nicholashoule/repogov/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/nicholashoule/repogov/releases/tag/v0.1.0
