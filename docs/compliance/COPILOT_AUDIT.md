# GitHub Copilot Audit

Source: https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot
Verified: 2026-03-13

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

## Frontmatter Keys

Instruction files in `.github/instructions/` and `.github/rules/` support YAML frontmatter:

| Key | Values | Effect |
|-----|--------|--------|
| `applyTo` | glob string (e.g. `"**"`, `"**/*.go"`) | Scopes the rule to matching file paths |
| `excludeAgent` | `"code-review"` or `"coding-agent"` | Excludes this rule from the specified Copilot feature |

Example — a rule that applies everywhere but is excluded from code-review runs:
```yaml
---
applyTo: "**"
excludeAgent: "code-review"
---
```

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

## Memory Configuration

GitHub Copilot has no dedicated memory file or runtime memory system. All persistent
context is provided through instruction files checked into the repository. There is no
equivalent to a `memory.md` that Copilot writes to or reads from at runtime.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Repo-wide | `.github/copilot-instructions.md` | Yes — always injected |
| Path-scoped | `.github/instructions/*.instructions.md` or `.github/rules/*.md` | Yes — matched by `applyTo` glob |
| Global | None (VS Code user settings for model/UI prefs only) | — |

To persist rules across sessions, add them to `.github/copilot-instructions.md` or a scoped
rules file. There is no session-level or agent-runtime memory mechanism.

## Limits

- `.github/copilot-instructions.md`: 50 lines (enforced via `DefaultConfig().Files`).
- `.github/rules/*.md` and `.github/instructions/*.instructions.md`: 300 lines.

## Preset

`DefaultCopilotLayout()` in `presets.go`.
CLI agent name: `copilot`.
