# Cursor Audit

Source: https://cursor.com/docs/context/rules
Verified: 2026-10-17

## Configuration Files

- `.cursor/rules/` — project rules; version-controlled.
  - Current docs state project rules MUST use `.mdc`; a plain `.md` file in `.cursor/rules`
    is "ignored by the rules system" (no frontmatter). This contradicts the prior audit and
    the preset, which accepts both. For plain markdown use `AGENTS.md` instead.
  - `.mdc` frontmatter keys: `description`, `globs`, `alwaysApply`.
  - Files can be organized into **subdirectories** within `.cursor/rules/` for grouping.
- `AGENTS.md` at project root or subdirectory — plain markdown, no metadata; nested files
  combine with parents, more specific wins.
- User Rules - global, set in Customize > Rules (Agent chat only); not filesystem files.
- `.cursorrules` (single root file) - **legacy / deprecated**. The current rules page
  (https://cursor.com/docs/context/rules, fetched 2026-10-17) no longer mentions it. Per
  maintainer-supplied information (2026-10-17, not stated on any vendor page): it may still be
  read for backward compatibility in basic completions but is silently ignored by Agent mode and
  gets no modern features. Migrate to `.cursor/rules/*.mdc` (multi-file, frontmatter-scoped).
  Do not create `.cursorrules` in new repos; repogov does not scaffold it.

Rule types: Always Apply, Apply Intelligently (description), Apply to Specific Files (globs),
Apply Manually (@-mention). Precedence: Team rules, then Project, then User.

### Team Rules (Team/Enterprise)

Cursor Team and Enterprise plans support dashboard-managed rules:
- Defined and edited in the Cursor dashboard (not local files)
- Applied to all team members automatically (can be enforced)
- **Remote rules via GitHub** — rules can also be imported and synced from any GitHub
  repository, enabling a centralized team rule source outside the project repo

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | Ignored in `.cursor/rules` per current docs (use AGENTS.md instead) |
| `.mdc` | Supports YAML frontmatter (`description`, `globs`, `alwaysApply`) |

## Seeded Files

`init -agent cursor` creates the `.cursor/rules/` directory and seeds:

The full rule set is seeded as `.mdc` (Cursor ignores plain `.md` rules). Each template's Copilot-style
`applyTo` is translated to Cursor frontmatter: `applyTo: "**"` becomes `alwaysApply: true`; any other glob
becomes `globs: "<glob>"` with `alwaysApply: false`; `description` is the document's first heading.

| File | Frontmatter | Purpose |
|------|-------------|---------|
| `general.mdc`, `memory.mdc`, `governance.mdc`, `security.mdc`, `repo.mdc`, `backend.mdc`, `frontend.mdc`, `emoji-prevention.mdc` | `description`, `alwaysApply: true` | Project conventions (apply everywhere) |
| `codereview.mdc`, `library.mdc`, `testing.mdc` | `description`, `globs: "**/*.md"`, `alwaysApply: false` | Documentation-scoped rules |

Earlier repogov versions seeded plain `.md` files here, which Cursor ignores; `layout` now warns about
them (agent pitfalls) with a rename hint. `init` never overwrites or renames existing files.

## Memory Configuration

Cursor has no dedicated memory file or runtime memory system. The `.cursor/rules/` directory
is the sole persistent context layer. There is no `memory.mdc` that Cursor writes to or reads
from at runtime.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Project | `.cursor/rules/*.mdc` (plain `.md` is ignored) | Yes — controlled by `alwaysApply` or `globs` frontmatter |
| User | Settings UI only (Customize > Rules) | — |

For always-on project context, set `alwaysApply: true` in the rule's YAML frontmatter.
There is no session-level memory mechanism.

## Limits

- Official best-practice size limit is 500 lines; repogov enforces 300.

## Preset

`DefaultCursorLayout()` in `presets.go`. Uses an empty glob for `rules/` to accept both `.md` and `.mdc`.
Follow-up: `.md` files are inert in Cursor; consider warning on them or restricting to `*.mdc`.
CLI agent name: `cursor`.
