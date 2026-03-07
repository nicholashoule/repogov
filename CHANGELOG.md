# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Core types: `Status`, `PercentInt`, `Config`, `Rule`, `Result` (`repogov.go`)
- Line counting: `CountLines`, `CheckFile`, `CheckDir`, `CheckDirContext` (`check.go`)
- Layout governance: `LayoutSchema`, `DirRule`, `NamingRule`, `LayoutResult` (`layout.go`)
- Layout validation: `CheckLayout`, `CheckLayoutContext` (`layout.go`)
- Layout scaffolding: `InitLayout` with default instruction file seeding (`init.go`)
- Platform presets: `DefaultGitHubLayout`, `DefaultGitLabLayout` (`presets.go`)
- Configuration: `LoadConfig`, `FindConfig`, `SaveConfig`, `ValidateConfig` (`config.go`)
- Configuration types: `Violation` for structured config validation (`config.go`)
- YAML config support: load and save `.yaml`/`.yml` config files (`yaml.go`)
- Output helpers: `Summary`, `Passed`, `LayoutSummary`, `LayoutPassed` (`format.go`)
- CLI tool: `cmd/repogov` with `limits`, `layout`, `init`, `validate`, `all`, `version` subcommands
- Table-driven tests for all packages
- `docs/design.md` architecture documentation
