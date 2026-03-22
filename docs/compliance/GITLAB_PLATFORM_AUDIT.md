# GitLab Platform Audit

Source: https://docs.gitlab.com/ee/user/project/description_templates.html
Verified: 2026-03-22

## Overview

GitLab uses a `.gitlab/` directory for repository-level configuration including
issue and merge request templates. CI/CD is configured via `.gitlab-ci.yml` at the
repository root (outside the `.gitlab/` directory scope).

## Detection Markers

repogov auto-detects GitLab when either:
- `.gitlab/` directory exists at the repository root, **or**
- `.gitlab-ci.*` file exists at the repository root (e.g. `.gitlab-ci.yml`).

## Configuration Directory

| Path | Purpose |
|------|---------|
| `.gitlab/` | Repository-level configuration root |

## Templates

| Path | Purpose |
|------|---------|
| `.gitlab/issue_templates/` | Issue description templates (Markdown) |
| `.gitlab/merge_request_templates/` | Merge request description templates (Markdown) |

## Community Health Files

| File | Purpose |
|------|---------|
| `.gitlab/CODEOWNERS` | Code ownership and automatic reviewer assignment |

## CI/CD

| Path | Purpose |
|------|---------|
| `.gitlab-ci.yml` | GitLab CI/CD pipeline definition (repo root) |

Note: `.gitlab-ci.yml` lives at the repository root and is outside the scope of the
`.gitlab/` layout schema. Enforce its size via the limits config if needed.

## Naming Conventions

- Directory and file names are generally lowercase.
- Exceptions: `CODEOWNERS`.

## Preset

- Function: `DefaultGitLabLayout()` in `presets.go`
- CLI: `repogov -platform gitlab layout`
