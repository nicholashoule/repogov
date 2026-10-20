# GitLab Platform Audit

Source: https://docs.gitlab.com/user/project/description_templates/
Source: https://docs.gitlab.com/user/project/codeowners/
Verified: 2026-10-17

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
| `.gitlab/merge_request_templates/` | Merge request description templates (`.md` only) |

Templates must be on the default branch. A `Default.md` (case-insensitive) in either directory
is the default; project settings override it. Group/instance template repositories also exist.

## Community Health Files

| File | Purpose |
|------|---------|
| `CODEOWNERS` | Code ownership. GitLab checks `./CODEOWNERS`, `./docs/CODEOWNERS`, then `./.gitlab/CODEOWNERS`; first found is used |

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
