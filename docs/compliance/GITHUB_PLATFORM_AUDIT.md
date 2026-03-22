# GitHub Platform Audit

Source: https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions
Verified: 2026-03-22

## Overview

GitHub uses a `.github/` directory for repository-level configuration including
community health files, issue/PR templates, Actions workflows, and Dependabot
configuration. This audit covers the **platform** structure; see `COPILOT_AUDIT.md`
for Copilot-specific AI instruction files.

## Detection Markers

repogov auto-detects GitHub when:
- `.github/` directory exists at the repository root.

## Configuration Directory

| Path | Purpose |
|------|---------|
| `.github/` | Repository-level configuration root |

## Community Health Files

| File | Purpose |
|------|---------|
| `CODE_OF_CONDUCT.md` | Community code of conduct |
| `CODEOWNERS` | Automatic reviewer assignment |
| `CONTRIBUTING.md` | Contribution guidelines |
| `FUNDING.yml` | Sponsor button configuration |
| `SECURITY.md` | Security policy and vulnerability reporting |
| `SUPPORT.md` | Support resources |

## Templates

| Path | Purpose |
|------|---------|
| `.github/ISSUE_TEMPLATE/` | Issue template directory (YAML or Markdown) |
| `.github/ISSUE_TEMPLATE.md` | Single-file issue template |
| `.github/PULL_REQUEST_TEMPLATE/` | PR template directory |
| `.github/PULL_REQUEST_TEMPLATE.md` | Single-file PR template |
| `.github/pull_request_template.md` | Lowercase alternative PR template |

## CI/CD

| Path | Purpose |
|------|---------|
| `.github/workflows/` | GitHub Actions workflow definitions (`.yml`/`.yaml`) |
| `.github/dependabot.yml` | Dependabot dependency update configuration |

## Naming Conventions

- Directory and file names are generally lowercase.
- Exceptions: `CODEOWNERS`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, `FUNDING.yml`,
  `ISSUE_TEMPLATE`, `PULL_REQUEST_TEMPLATE`, `SECURITY.md`, `SUPPORT.md`.

## Preset

- Function: `DefaultGitHubLayout()` in `presets.go`
- CLI: `repogov -platform github layout`
