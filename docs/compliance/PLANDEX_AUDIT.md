# Plandex Audit

Source: https://github.com/plandex-ai/plandex (README) — verified 2026-03-07.
Note: https://docs.plandex.ai returned 404 for all configuration/notes pages.

## Configuration Model

**Status: Partially verified** — Plandex's official documentation site returned
404 for configuration-related pages. The following is based on the GitHub README
and widely-reported community information.

### Project State Directory

Plandex stores plan state (not user instructions) in a `.plandex/` directory
at the project root. This is analogous to a `.git/` directory — it tracks plan
versions, branch history, and diffs — rather than a rules or instructions store.

### Context Loading

Plandex loads context explicitly via CLI commands (`plandex load <file>`,
`pdx load <dir>`). There is no auto-loaded instruction file (no analog to
`AGENTS.md`). Project-specific instructions must be loaded into context
explicitly at the start of a plan.

### Reported Notes Feature

Some community references mention a `plandex notes` command for storing
persistent project notes, but this was not verified from official documentation.
If verified:

- Notes may be stored in `.plandex/` or a related config directory.
- Notes would be included automatically in all plan contexts.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.plandex/` | Project state directory (plan versions, diffs, branches) — not instruction files |

## Action Items

- Re-verify against official Plandex docs once available:
  - `https://docs.plandex.ai/notes`
  - `https://docs.plandex.ai/context`
  - `https://docs.plandex.ai/configuration`
- Confirm whether `plandex notes` stores instruction content that repogov
  should govern.
- Check if auto-loaded context files are planned for a future version.

## repogov Support Status

Not yet supported due to unverified file-based instruction format. Plandex's
explicit-load model may not require a repogov preset. Revisit once notes/context
documentation is available.
