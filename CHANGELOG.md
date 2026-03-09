# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.3.1] - 2026-03-09

### Added

- `templates/rules/security.md.tmpl` — new scoped rule template covering secrets hygiene, input validation, authentication/authorization, dependency pinning, and vulnerability disclosure.

### Changed

- `templates/rules/general.md.tmpl` — removed `Maintenance` section and generic writing platitudes that provide no agent-specific signal.
- `templates/rules/emoji-prevention.md.tmpl` — condensed 30-row alternatives table to 6 representative rows; table length was inflating the token count the file itself warns against.
- `templates/rules/repo.md.tmpl` — replaced three bulleted layout lists (`.github`, `.gitlab`, root) with concise prose lines.
- `templates/rules/codereview.md.tmpl` — removed `Review Etiquette` section; interpersonal guidance for human reviewers, not actionable agent behavior.
- `templates/rules/testing.md.tmpl` — removed `Development Section` section; duplicated guidance already covered by `library.md.tmpl`.

### Fixed

- `templates/agents/CLAUDE.md.tmpl` — replaced hardcoded `claude` in `go run` commands with `{{.Agent}}` so the template renders correctly for any agent target.

## [v0.3.0] - 2026-03-08

### Added

- Fix LICENSE, APPENDIX: How to apply the Apache License to your work.
- Embedded template system: `mustReadTemplate` + `embed.FS` in `init.go` replace all inline string-concatenation content builders. Template files live under `templates/` and are embedded at compile time, keeping generated file content fully auditable and diff-friendly.
- `templates/agents/` subdirectory for agent root files: `AGENTS.md.tmpl`, `CLAUDE.md.tmpl`, `copilot-instructions.md.tmpl`.
- `templates/rules/` subdirectory for scoped rule templates: `backend.md.tmpl`, `codereview.md.tmpl`, `emoji-prevention.md.tmpl`, `frontend.md.tmpl`, `general.md.tmpl`, `governance.md.tmpl`, `library.md.tmpl`, `repo.md.tmpl`, `testing.md.tmpl`.
- `agentsMdTemplateData` struct — typed data passed to `agents/AGENTS.md.tmpl`, enabling conditional sections (`HasRules`, `HasInstructions`, `IsCopilot`, `IsClaude`, etc.) without Go code changes.
- All template files use the `.md.tmpl` extension consistently, including previously static `.md` files, so any file is ready for `{{...}}` directives without a rename.

### Changed

- All `*Content()` helper functions in `init.go` (`agentsMdContent`, `claudeMdContent`, `copilotInstructionsContent`, `governanceInstructionsContent`, and all rule-file helpers) now delegate to `mustReadTemplate` rather than building output with `strings.Builder`.
- Template files organized into `templates/agents/` (one-per-agent root files) and `templates/rules/` (granular rule files seeded into platform `rules/` directories). Adding a new agent or rule requires only a new template file, not a Go code change.

### Fixed

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

[Unreleased]: https://github.com/nicholashoule/repogov/compare/v0.3.1...HEAD
[v0.3.1]: https://github.com/nicholashoule/repogov/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/nicholashoule/repogov/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/nicholashoule/repogov/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/nicholashoule/repogov/releases/tag/v0.1.0
