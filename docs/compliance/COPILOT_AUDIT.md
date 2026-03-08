# GitHub Copilot Audit

Source: https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot
Verified: 2026-03-07

## Configuration Files

- `.github/copilot-instructions.md` — repo-wide context; 50-line limit.
- `.github/instructions/*.instructions.md` — path-specific rules; `applyTo` frontmatter glob.
  Subdirectories under `.github/instructions/` are supported for organization.
  **Legacy/optional**: `init` only creates `instructions/` if it already pre-exists in the repo.
- `.github/rules/*.md` — primary scoped-rule directory used by fresh inits. On a fresh init (no
  pre-existing `instructions/`), all default `*.instructions.md` files are seeded here. When
  `instructions/` already exists, `general.md` is seeded here as a companion. Governed at 300 lines.
- `.github/prompts/*.prompt.md` — reusable prompt templates.
- `AGENTS.md` anywhere in the repo — agent-mode instructions (shared standard with Codex/Cursor).

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `copilot-instructions.md`, `rules/*.md`, `instructions/*.instructions.md`, `prompts/*.prompt.md` |

## Seeded Files

`init -agent copilot` seeds `copilot-instructions.md` and the following default files
(placed in `rules/` on fresh inits, `instructions/` if that dir pre-exists):

| File | `applyTo` | Purpose |
|------|-----------|---------|
| `general.instructions.md` | `"**"` | General coding conventions including US English spelling |
| `backend.instructions.md` | `"**"` | API design, error handling, auth, data, testing |
| `frontend.instructions.md` | `"**"` | Component structure, state, API integration, a11y |
| `codereview.instructions.md` | `"**/*.md"` | Code review guidelines |
| `governance.instructions.md` | `"**"` | Line limits, layout, zero dependencies |
| `library.instructions.md` | `"**/*.md"` | Library authoring conventions |
| `testing.instructions.md` | `"**/*.md"` | Test structure and coverage |
| `emoji-prevention.instructions.md` | `"**"` | No emoji in docs |

## Limits

- `.github/copilot-instructions.md`: 50 lines (enforced via `DefaultConfig().Files`).
- `.github/rules/*.md` and `.github/instructions/*.instructions.md`: 300 lines.

## Preset

`DefaultCopilotLayout()` in `presets.go`.
CLI agent name: `copilot`.
