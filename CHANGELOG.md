# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/nicholashoule/repogov/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/nicholashoule/repogov/releases/tag/v0.1.0
