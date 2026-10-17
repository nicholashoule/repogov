# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.9.0] - 2026-10-17

### Security

- `CheckDir`/`CheckFile` no longer open symlinks or special files (reported as SKIP); unreadable files now FAIL instead of vanishing
- Scaffolding creates files with `O_EXCL` (`writeNewFile`), so it never writes through a symlink or races a creator
- Sensitive-extension skip list gains `.p12`, `.pfx`, `.jks`
- CI: `upload-artifact` pinned to a SHA; unused `pull-requests: write` removed
- Text output neutralizes control characters, ANSI escapes and Unicode bidi overrides in file names and messages (`Summary`, `LayoutSummary`), so a crafted file name cannot forge log lines or GitHub Actions workflow commands (`::error::`); JSON output was already escaped
- Agent-pitfall checks never open symlinks or special files, cap reads at 1 MiB, skip dependency/build trees (`node_modules`, `dist`, `build`, `target`, `.venv`, ...) and stop at depth 8
- `update-config` writes atomically (temp file + rename, permissions preserved) and refuses configs over 4 MiB
- Line-exempt files that fail a byte limit now show the reason in the text summary

### Changed

- Minimum Go version raised from 1.22 to 1.24 (`go.mod`, CI matrix, `docs/design.md`)
- CI matrix now covers Go 1.24, 1.25, and stable
- `scripts/hooks/pre-commit` — bumped the self-governance pin from
  `repogov@v0.7.2` to the released `repogov@v0.8.0` (post-release pin
  sweep per the PIN POLICY comment at the pin site)

### Fixed

- **`init -agent cursor` now seeds `.mdc` rules with Cursor frontmatter** (`description`, `globs`, `alwaysApply`) instead of plain `.md` files with Copilot `applyTo` frontmatter, which Cursor ignores. `descriptive_names` no longer affects Cursor. Existing `.md` rules are not renamed (init never overwrites); `layout` warns about them with a rename hint. `.cursor/rules/memory.mdc` gets the 200-line limit
- `-agent zed` layout accepts any of Zed's documented rules files (`.cursorrules`, `.windsurfrules`, `.clinerules`, `.github/copilot-instructions.md`, `AGENT.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) in place of `.rules`
- Root layout no longer reports files nested inside managed directories (`.claude/rules/a.md`, `.github/workflows/ci.yml`, `docs/sub/b.md`, `.opencode/agents/a.md`) as unexpected: a `NoCreate` directory with no glob now covers its whole subtree. Files outside managed directories are still flagged
- Root layout recognizes `opencode.json`/`opencode.jsonc` (optional) and `.gemini/`, `.opencode/`, `.agents/`, `.codex/` (managed directories)
- `-agent gemini` layout accepts `AGENTS.md` in place of `GEMINI.md` (new `LayoutSchema.Alternatives`); read by Antigravity and OpenCode. `init` still creates `GEMINI.md`
- `LoadConfig` now accepts configs saved with a UTF-8 BOM (common from Windows editors/PowerShell); previously JSON/YAML parsing failed with "invalid character"

### Added

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
