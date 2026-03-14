# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.6.0] - 2026-03-14

### Changed

- **Code decomposition**: split `init.go` (1252 lines) into three focused files:
  - `init.go` (334 lines) — public API and orchestration (`InitLayout`, `InitLayoutWithConfig`, `InitLayoutAll`, `InitLayoutAllWithConfig`, `initLayoutSingle`, `initLayoutAllWithOptions`), plus `initOptions`, `templateStem`, and `shouldSeedFile`.
  - `template.go` (33 lines) — embedded template filesystem (`templateFS`), `mustReadTemplate`, and `mustRenderTemplate`.
  - `scaffold.go` (889 lines) — all scaffolding content generators and file-writing helpers: agent-specific file creators (`createCopilotInstructions`, `createClaudeMd`, `createGeminiMd`, `createZedRules`), AGENTS.md management (`createAgentsMd`, `updateAgentsMdContext`, `UpdateAgentsMdContextAll`, context section builders), instruction template seeding (`createDefaultInstructions`, `defaultInstructionFilesFor`, all `*InstructionsContent` functions), config scaffolding (`createDefaultConfig`, `createDefaultConfigAll`, `schemaConfig`, `defaultConfigJSON`), filesystem helpers (`dirIsNew`, `isDirEmpty`, `dirHasGlobFiles`, `copilotNarrowSchema`), and mapping utilities (`schemaRootToAgent`, `rulesLabel`, `placeholderContent`).
- **Deduplicated `intToStr`/`intStr`**: removed `intToStr` from `layout.go` (identical to `intStr` in `scaffold.go`); `formatDirMinMessage` and `formatDirPassMessage` now call the single `intStr` utility.
- **Extracted `spliceContextSection` helper** in `scaffold.go`: the read→find-marker→splice→write logic was duplicated between `updateAgentsMdContext` and `UpdateAgentsMdContextAll`; both now delegate to the shared helper.
- **Simplified `agentsMdContextSection`**: was a 35-line near-duplicate of `agentsMdMergedContextSection`; now a one-liner that delegates via `agentsMdMergedContextSection([]LayoutSchema{schema})`.
- **Added `writeLine` closure** in `agentsMdMergedContextSection`: replaced 10 repetitions of the dedup-and-write pattern with a single closure.
- **Fixed misplaced doc comment**: the `agentsMdContextSection` godoc was orphaned above `UpdateAgentsMdContextAll`; moved to its correct function.
- **Updated `docs/design.md` Package Structure**: added missing files (`scaffold.go`, `template.go`, `yaml.go`), corrected function-to-file attributions, and expanded descriptions for `config.go` and `init.go`.

### Fixed

- `DefaultConfig().Files` — added `.github/instructions/memory.instructions.md: 200` entry. Repos where Copilot init targeted `instructions/` instead of `rules/` were missing the per-file limit for the memory template, falling back to the 300-line glob instead of the intended 200-line limit.

## [v0.5.1] - 2026-03-13

### Added

- `DefaultZedLayout()` preset in `presets.go` — supports Zed AI's single-file `.rules` format at repo root.
- `-agent zed` CLI preset for `layout`, `init`, and `all` subcommands.
- `templates/agents/zed-rules.tmpl` — template for Zed's root-level `.rules` project rules file.
- `createZedRules()` and `zedRulesContent()` in `init.go`; `IsZed` template data field and context section logic for `AGENTS.md`.
- `".rules": 200` added to `DefaultConfig()` (`repogov.go`).
- `docs/compliance/ZED_AUDIT.md` — Preset, Seeded Files, Memory Configuration, and Limits sections added; "Not yet supported" section removed.
- `docs/compliance/AI_AGENTS_AUDIT.md` — Zed AI added to criteria table (all Pass), Supported table, Tier 1 table, and Support Matrix.
- **Memory Configuration** sections added to all agent audit files: `COPILOT_AUDIT.md`, `CURSOR_AUDIT.md`, `WINDSURF_AUDIT.md`, `CLAUDE_AUDIT.md`, `KIRO_AUDIT.md`, `CLINE_AUDIT.md`, `ROOCODE_AUDIT.md`, `CONTINUE_AUDIT.md`, `JETBRAINS_AUDIT.md`, `GEMINI_AUDIT.md`, `AGENTS_MD_AUDIT.md`, `ZED_AUDIT.md`.
- `docs/compliance/KIRO_AUDIT.md` — "Not yet supported" section replaced with Preset, Seeded Files, Memory Configuration, and Limits sections.
- `templates/rules/memory.md.tmpl` — dedicated project memory seed template with sections for Architecture Decisions, Known Issues, Conventions, Dependency Notes, and Session Notes. Kept separate from `general.md` which serves a different purpose.
- `memoryInstructionsContent()` in `init.go`; `memory.instructions.md` / `memory.md` registered in `defaultInstructionFilesFor()` and seeded for all 9 rule-seeding agents (copilot, cursor, windsurf, claude, kiro, continue, cline, roocode, jetbrains).
- Nine `memory.md` per-file entries added to `DefaultConfig().Files` at 200 lines each (`.github/rules/memory.md`, `.cursor/rules/memory.md`, `.windsurf/rules/memory.md`, `.claude/rules/memory.md`, `.kiro/steering/memory.md`, `.continue/rules/memory.md`, `.clinerules/memory.md`, `.roo/rules/memory.md`, `.aiassistant/rules/memory.md`).
- `*.mdc` and `scripts/hooks/pre-commit` added to `.gitattributes` LF normalization rules.
- `scripts/hooks/pre-commit` — bumped `demojify-sanitize` from `v0.6.0` to `v0.7.0`.

### Fixed

- `DefaultCopilotLayout()` `workflows` `DirRule` — `NoCreate: true` was missing, causing an empty `workflows/` directory to be created on every `copilot init` even when GitHub Actions was not in use.
- `schemaConfig()` in `init.go` — changed signature from `(cfg Config, root string)` to `(cfg Config, schema LayoutSchema)` and replaced the `root == "." && !strings.Contains(k, "/")` case with `root == "." && isInRequired(k, schema.Required)`. Added `isInRequired` helper. Gemini and Zed both use `Root == "."` but own different root-level files (`GEMINI.md` and `.rules` respectively); the old code included both in each agent's generated config.
- `initLayoutSingle` config placement — when no `.github/` directory exists (all non-copilot single-agent inits), the generated `repogov-config.json` now lands at the repo root instead of inside the agent's own directory (e.g., `.cursor/repogov-config.json`). `FindConfig` searches the repo root first, so the config is now auto-discoverable without requiring an explicit `-config` flag.
- `resolveRoot()` in `cmd/repogov/main.go` — generalized from "only git-walk when root is `.`" to "always resolve to an absolute path, then walk up to the nearest git root". An explicit `-root .cursor` (or any agent subdir path) is now correctly resolved to the repo root, preventing double-nested scaffolding (e.g., `.cursor/.cursor/rules/`).
- `DefaultConfig().Files` — added `memory.instructions.md` variants alongside each `memory.md` entry (18 entries total, 9 agents × 2 naming conventions). When `descriptive_names` is enabled the scaffolded file is `memory.instructions.md`; without the paired entry it fell back to the 300-line `*.md` glob instead of the intended 200-line limit.

## [v0.5.0] - 2026-03-12

### Added

- `DefaultKiroLayout()`, `DefaultGeminiLayout()`, `DefaultContinueLayout()`, `DefaultClineLayout()`, `DefaultRooCodeLayout()`, `DefaultJetBrainsLayout()` layout presets in `presets.go`.
- `-agent kiro`, `-agent gemini`, `-agent continue`, `-agent cline`, `-agent roocode`, `-agent jetbrains` CLI presets for `layout`, `init`, and `all` subcommands.
- `templates/agents/GEMINI.md.tmpl` — template for Gemini CLI's root-level `GEMINI.md` instruction file.
- `crossAgentRootFile()` helper in `init.go` — explicit allowlist for root-level files that belong in every agent's generated `repogov-config.json`; currently only `AGENTS.md` qualifies. Prevents agent-specific files from being injected into unrelated platform configs.
- `anyRequiredFileExists()` helper in `cmd/repogov/main.go` — enables `runLayout all` to correctly skip file-only schemas (e.g., Gemini) whose required files are absent, matching existing absent-directory skip behavior for all other platforms.
- Glob rules for `.kiro/steering/*.md`, `.continue/rules/*.md`, `.clinerules/*.md`, `.roo/rules/*.md`, and `.aiassistant/rules/*.md` added to `DefaultConfig()` (`repogov.go`) and `.github/repogov-config.json`.
- `docs/compliance/AI_AGENTS_AUDIT.md` — Kiro, Gemini CLI, Continue, Cline, Roo Code, and JetBrains AI Assistant moved from backlog to Supported.

### Changed

- `defaultLimit` constant changed from `300` to `500` (`repogov.go`); raises the built-in fallback limit for all unconfigured files.
- `schemaConfig()` in `init.go` replaced the blanket "no-slash key = include everywhere" rule with an explicit `crossAgentRootFile()` allowlist check; agent-specific root files (e.g., `GEMINI.md`) no longer appear in unrelated agents' generated `repogov-config.json`.
- `CheckLayoutContext()` in `layout.go` short-circuits for file-only schemas (`Root == "."`, empty `Dirs`): validates only Required/Optional file existence without walking the directory, preventing false unexpected-file failures on Gemini layout checks.
- `DefaultRootLayout()` updated with `NoCreate: true` entries for `.kiro/`, `.continue/`, `.clinerules/`, `.roo/`, `.aiassistant/`, and `GEMINI.md` added to the naming exception list.
- `DefaultConfig()` extended: `GEMINI.md: 200` and corrected `.claude/CLAUDE.md: 200` (was `50`).

### Fixed

- `.claude/CLAUDE.md` default limit corrected from `50` to `200` in `DefaultConfig()` (`repogov.go`).

## [v0.4.0] - 2026-03-09

### Added

- Test assertions in `TestInitLayout_ClaudeSchema` and the claude scaffold test verifying that generated `.claude/CLAUDE.md` contains `-agent claude` and does not contain the literal `{{.Agent}}` string. Catches regressions where `claudeMdContent` returns raw template text instead of rendering through `text/template`.

### Changed

- `scripts/hooks/pre-commit` — bumped `demojify-sanitize` from `v0.5.0` to `v0.6.0`.

## [v0.3.0] - 2026-03-09

### Added

- `mustRenderTemplate` helper in `init.go` — single chokepoint that parses and executes every embedded template through `text/template`. All content functions now route through it, so adding a `{{.Field}}` placeholder to any template in the future will never silently emit literal text.
- `templates/rules/security.md.tmpl` — new scoped rule template covering secrets hygiene, input validation, authentication/authorization, dependency pinning, and vulnerability disclosure.
- Embedded template system: `mustReadTemplate` + `embed.FS` in `init.go` replace all inline string-concatenation content builders. Template files live under `templates/` and are embedded at compile time, keeping generated file content fully auditable and diff-friendly.
- `templates/agents/` subdirectory for agent root files: `AGENTS.md.tmpl`, `CLAUDE.md.tmpl`, `copilot-instructions.md.tmpl`.
- `templates/rules/` subdirectory for scoped rule templates: `backend.md.tmpl`, `codereview.md.tmpl`, `emoji-prevention.md.tmpl`, `frontend.md.tmpl`, `general.md.tmpl`, `governance.md.tmpl`, `library.md.tmpl`, `repo.md.tmpl`, `testing.md.tmpl`.
- `agentsMdTemplateData` struct — typed data passed to `agents/AGENTS.md.tmpl`, enabling conditional sections (`HasRules`, `HasInstructions`, `IsCopilot`, `IsClaude`, etc.) without Go code changes.
- All template files use the `.md.tmpl` extension consistently, including previously static `.md` files, so any file is ready for `{{...}}` directives without a rename.
- Fix LICENSE, APPENDIX: How to apply the Apache License to your work.

### Changed

- All `*Content()` helper functions in `init.go` (`agentsMdContent`, `claudeMdContent`, `copilotInstructionsContent`, `governanceInstructionsContent`, and all rule-file helpers) now route through `mustRenderTemplate` rather than building output with `strings.Builder` or calling `mustReadTemplate` directly.
- Template files organized into `templates/agents/` (one-per-agent root files) and `templates/rules/` (granular rule files seeded into platform `rules/` directories). Adding a new agent or rule requires only a new template file, not a Go code change.
- `templates/rules/general.md.tmpl` — removed `Maintenance` section and generic writing platitudes that provide no agent-specific signal.
- `templates/rules/emoji-prevention.md.tmpl` — condensed 30-row alternatives table to 6 representative rows; table length was inflating the token count the file itself warns against.
- `templates/rules/repo.md.tmpl` — replaced three bulleted layout lists (`.github`, `.gitlab`, root) with concise prose lines.
- `templates/rules/codereview.md.tmpl` — removed `Review Etiquette` section; interpersonal guidance for human reviewers, not actionable agent behavior.
- `templates/rules/testing.md.tmpl` — removed `Development Section` section; duplicated guidance already covered by `library.md.tmpl`.

### Fixed

- `init.go` `claudeMdContent` — was returning raw `mustReadTemplate` output instead of rendering through `text/template`, causing `{{.Agent}}` to appear verbatim in generated `.claude/CLAUDE.md` files. Now rendered correctly via `mustRenderTemplate`.
- `templates/agents/CLAUDE.md.tmpl` — replaced hardcoded `claude` in `go run` commands with `{{.Agent}}` so the template renders correctly for any agent target.
- `scripts/hooks/pre-commit.go` — `go test` failure hint now points to `make test` instead of `go test ./...`, consistent with all other hook failure hints and picking up any flags added to the make target.
- `*.tmpl` added to `.gitattributes` text/LF normalization rules so template line endings are consistent across platforms.
- Removed duplicate `temp*` entry from `.gitignore`.

## [v0.2.0] - 2026-03-08

### Added

- `findGitRoot` / `resolveRoot` helpers in `cmd/repogov`: when `-root` is the default `.`, the CLI now walks up from the working directory to find the nearest `.git` ancestor and uses that as the repository root. Running `repogov init` from inside a subdirectory (e.g. `.github/instructions/`) no longer scaffolds files into that subdirectory.
- Trailing-slash directory glob support in `ResolveLimit`: a rule glob ending with `/` (e.g. `"docs/"`) is now treated as a recursive directory prefix, matching any file under that directory at any depth. Previously, such globs were passed to `filepath.Match` and never matched any file.

### Fixed

- `ResolveLimit` now correctly applies directory-scoped rules written as `"docs/"` — the limit was silently ignored and the global default was used instead.


## [v0.1.0] - 2026-03-08

### Added

- Core types: `Status`, `PercentInt`, `Config`, `Rule`, `Result` (`repogov.go`)
- Line counting: `CountLines`, `CheckFile`, `CheckDir`, `CheckDirContext` (`check.go`)
- Layout governance: `LayoutSchema`, `DirRule`, `NamingRule`, `LayoutResult` (`layout.go`)
- Layout validation: `CheckLayout`, `CheckLayoutContext` (`layout.go`)
- Layout scaffolding: `InitLayout` with default instruction file seeding (`init.go`)
- Platform presets: `DefaultCopilotLayout`, `DefaultCursorLayout`, `DefaultWindsurfLayout`, `DefaultClaudeLayout`, `DefaultGitLabLayout`, `DefaultRootLayout` (`presets.go`)
- `DirRule.NoCreate` field — prevents `repogov init` from creating recognized-but-optional directories (`layout.go`)
- Configuration: `LoadConfig`, `FindConfig`, `SaveConfig`, `ValidateConfig` (`config.go`)
- Configuration types: `Violation` for structured config validation (`config.go`)
- YAML config support: load and save `.yaml`/`.yml` config files (`yaml.go`)
- Output helpers: `Summary`, `Passed`, `LayoutSummary`, `LayoutPassed` (`format.go`)
- CLI tool: `cmd/repogov` with `limits`, `layout`, `init`, `validate`, `all`, `version` subcommands
- `-agent root` support for validating repository root layout
- `-agent all` skips platforms whose root directories are absent; repos adopting a subset of platforms exit 0
- Table-driven tests for all packages
- `docs/design.md` architecture documentation
- `docs/cli.md` CLI reference documentation
- `docs/git-hooks.md` Git hooks integration guide
- `docs/compliance/` audit docs for all major AI-agent platforms

### Fixed

- Pre-commit hook: check `command -v demojify` before running; print install instructions and exit 1 if not found (`scripts/hooks/pre-commit`)
- Layout walker: skip `.git` before the `d.IsDir()` check so worktree gitdir pointer files are also silently skipped, not flagged as unexpected (`layout.go`)
- `DefaultRootLayout` `Dirs` entries all set `NoCreate: true` so `repogov root init` does not scaffold common project directories (`presets.go`)
- Sorted keys in default config JSON for deterministic output (`init.go`)

[Unreleased]: https://github.com/nicholashoule/repogov/compare/v0.6.0...HEAD
[v0.6.0]: https://github.com/nicholashoule/repogov/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/nicholashoule/repogov/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/nicholashoule/repogov/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/nicholashoule/repogov/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/nicholashoule/repogov/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/nicholashoule/repogov/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/nicholashoule/repogov/releases/tag/v0.1.0
