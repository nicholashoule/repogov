# GitHub Copilot Audit

Source: https://docs.github.com/en/copilot/how-tos/configure-custom-instructions/add-repository-instructions
Also: https://docs.github.com/en/copilot/concepts/prompting/response-customization
Verified: 2026-10-17

## Configuration Files

- `.github/copilot-instructions.md` — repo-wide context; 50-line limit.
- `.github/instructions/*.instructions.md` — path-specific rules; `applyTo` frontmatter glob.
  Subdirectories under `.github/instructions/` are supported for organization.
  **Legacy/optional**: `init` only creates `instructions/` if it already pre-exists in the repo.
- `.github/rules/*.md` — primary scoped-rule directory used by fresh inits. On a fresh init (no
  pre-existing `instructions/`), all default `*.instructions.md` files are seeded here. When
  `instructions/` already exists, `general.md` is seeded here as a companion. Governed at 300 lines.
- `.github/prompts/*.prompt.md` — reusable prompt templates (path confirmed on GitHub's
  customization cheat sheet, https://docs.github.com/en/copilot/reference/customization-cheat-sheet,
  fetched 2026-10-17).
- `AGENTS.md` anywhere in the repo — agent instructions; nearest file in the tree wins.
  A single `CLAUDE.md` or `GEMINI.md` in the repo root is also read by Copilot.
- Personal and organization instructions exist (GitHub UI); precedence: personal, then
  repository (path-specific, repo-wide, agent), then organization.
- `.github/skills/<skill-name>/SKILL.md` (agent skills) is confirmed on the cheat sheet, which also
  lists `.claude/skills/<name>/SKILL.md` and `.agents/skills/<name>/SKILL.md` as project locations.
- `.github/agents/AGENT-NAME.md` (custom agents; org/enterprise level uses `/agents/AGENT-NAME.md`)
  is the path the cheat sheet lists. Maintainer-supplied guidance (2026-10-17) and community guides
  say the file must be named `*.agent.md` and that plain `.md` files are ignored; the cheat sheet
  states neither, so prefer `*.agent.md` for safety. Not governed by repogov.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `copilot-instructions.md`, `rules/*.md`, `instructions/*.instructions.md`, `prompts/*.prompt.md` |

## Frontmatter Keys

Instruction files in `.github/instructions/` and `.github/rules/` support YAML frontmatter:

| Key | Values | Effect |
|-----|--------|--------|
| `applyTo` | glob string; comma-separate multiple (e.g. `"**/*.ts,**/*.tsx"`) | Scopes the rule to matching file paths |
| `excludeAgent` | `"code-review"` or `"cloud-agent"` (docs now say `cloud-agent`; older docs said `coding-agent`) | Excludes this rule from the specified Copilot feature |

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

- `.github/copilot-instructions.md`: 50 lines (repogov policy; GitHub only advises "no longer than 2 pages", not task-specific).
- `.github/rules/*.md` and `.github/instructions/*.instructions.md`: 300 lines.

## Preset

`DefaultCopilotLayout()` in `presets.go`.
CLI agent name: `copilot`.
