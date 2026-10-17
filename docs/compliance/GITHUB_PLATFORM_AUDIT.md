# GitHub Platform Audit

Source: https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions
Source: https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file
Verified: 2026-10-17

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
| `ACCESSIBILITY.md` | Accessibility goals and reporting process (newly supported; not in preset) |
| `CODEOWNERS` | Automatic reviewer assignment (`.github/`, root, or `docs/`; first found wins; under 3 MB) |
| `CONTRIBUTING.md` | Contribution guidelines |
| `FUNDING.yml` | Sponsor button configuration |
| `SECURITY.md` | Security policy and vulnerability reporting |
| `SUPPORT.md` | Support resources |
| `VULNERABILITY_REPORT.yml`/`.yaml` | Private vulnerability report form (`.github/` only; not in preset) |

Most files may live in `.github/`, the repo root, or `docs/` (checked in that order);
`FUNDING.yml` and `VULNERABILITY_REPORT.yml` are `.github/` only. Org/user defaults come from a
public `.github` repository; a repo's own `ISSUE_TEMPLATE/` fully replaces the defaults.

## Templates

| Path | Purpose |
|------|---------|
| `.github/ISSUE_TEMPLATE/` | Issue template directory (YAML or Markdown) |
| `.github/ISSUE_TEMPLATE.md` | Single-file issue template |
| `.github/PULL_REQUEST_TEMPLATE/` | PR template directory |
| `.github/PULL_REQUEST_TEMPLATE.md` | Single-file PR template |
| `.github/pull_request_template.md` | Lowercase alternative PR template |
| `.github/ISSUE_TEMPLATE/config.yml` | Issue chooser configuration |
| `.github/DISCUSSION_TEMPLATE/` | Discussion category forms (not in preset) |

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
