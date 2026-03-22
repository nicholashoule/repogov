# Bitbucket Platform Audit

Source: https://support.atlassian.com/bitbucket-cloud/docs/get-started-with-bitbucket-pipelines/
Verified: 2026-03-22

## Overview

Bitbucket does not use a dedicated configuration directory. CI/CD is configured via
`bitbucket-pipelines.yml` at the repository root. Community health files (README,
LICENSE, CONTRIBUTING) are placed at the repository root following standard conventions.

## Detection Markers

repogov auto-detects Bitbucket when:
- `bitbucket-pipelines.*` file exists at the repository root (e.g. `bitbucket-pipelines.yml`).

## Configuration Files

| Path | Purpose |
|------|---------|
| `bitbucket-pipelines.yml` | Bitbucket Pipelines CI/CD configuration |

## Community Health Files

| File | Purpose |
|------|---------|
| `README.md` | Repository overview |
| `LICENSE` | License file |
| `CONTRIBUTING.md` | Contribution guidelines |

## CI/CD

| Path | Purpose |
|------|---------|
| `bitbucket-pipelines.yml` | Pipeline definitions, steps, and deployment targets |

Bitbucket Pipelines uses a single YAML file at the repo root. There is no equivalent
of a `.bitbucket/` configuration directory.

## Naming Conventions

- File names are generally lowercase.
- Exceptions: `CONTRIBUTING.md`, `LICENSE`, `README.md`.

## Preset

- Function: `DefaultBitbucketLayout()` in `presets.go`
- CLI: `repogov -platform bitbucket layout`
