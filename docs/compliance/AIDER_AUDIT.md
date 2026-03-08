# Aider Audit

Source: https://aider.chat/docs/config/aider_conf.html and https://aider.chat/docs/usage/conventions.html
Verified: 2026-03-07

## Configuration Files

### Tool Configuration (NOT instructions)

- `.aider.conf.yml` — YAML config for model selection, API keys, linting, git behavior, and other
  tool settings. This file does **not** carry coding instructions or conventions; it is purely
  operational configuration. Looked up in order: home dir, git root, current dir (last wins).
- `.aiderignore` — git-ignore-style file for excluding paths from the repo map.
- `.env` — environment variables (API keys, etc.); scoped to the git root by default.

### Coding Instructions / Conventions

- There is no single mandated filename for coding instructions. The recommended pattern is to
  create a `CONVENTIONS.md` (or any named markdown file) and load it as read-only context.
- Loading mechanisms:
  - CLI: `aider --read CONVENTIONS.md`
  - In-chat: `/read CONVENTIONS.md`
  - Persistent via `.aider.conf.yml`: `read: CONVENTIONS.md` or a list `read: [CONVENTIONS.md, STYLE.md]`
- The `read:` key in `.aider.conf.yml` accepts a list of files, so **multiple** instruction files
  can be referenced — but they are explicitly enumerated, not scanned from a directory glob.
- Community-contributed convention files: https://github.com/Aider-AI/conventions

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.yml` | `.aider.conf.yml` (tool config) |
| `.md` | `CONVENTIONS.md` or any user-named instruction file (loaded via `--read`) |
| (any) | `--read` accepts any file as additional read-only context |

## Limits

- No structured multi-file directory pattern for instructions (no `.aider/rules/*.md` glob).
- No line-limit enforcement by repogov.

## Preset

None. No repogov preset exists for Aider.
