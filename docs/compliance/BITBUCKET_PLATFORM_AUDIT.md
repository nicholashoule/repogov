# Bitbucket Platform Audit

Sources:
- https://support.atlassian.com/bitbucket-cloud/docs/get-started-with-bitbucket-pipelines/ — verified 2026-10-17
- https://support.atlassian.com/bitbucket-cloud/docs/set-up-and-use-code-owners/ — verified 2026-10-17
Verified: 2026-10-17

## Overview

Bitbucket CI/CD is configured via `bitbucket-pipelines.yml` at the repository root; the only
dedicated directory is `.bitbucket/`, which holds Code Owners configuration (below). Community health files (README,
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

Bitbucket Pipelines uses a single YAML file at the repo root (confirmed 2026-10-17).

## Code Owners (verified, Bitbucket Cloud)

The Cloud guide states: "Create the `CODEOWNERS` file in a directory called `.bitbucket` at the root
of your repository." The configuration "is always read from the destination branch of a pull
request". An optional `teams.yaml` goes adjacent to `CODEOWNERS` in `.bitbucket/`. Rules are read
bottom to top (last matching rule wins). Data Center instead uses a server-side
`code.owners.file.path` property, so this path applies to Cloud only.

repogov's `DefaultBitbucketLayout` does not yet list `.bitbucket/CODEOWNERS`; adding it as
Optional is safe now that the path is confirmed.

## Naming Conventions

- File names are generally lowercase.
- Exceptions: `CONTRIBUTING.md`, `LICENSE`, `README.md`.

## Preset

- Function: `DefaultBitbucketLayout()` in `presets.go`
- CLI: `repogov -platform bitbucket layout`
