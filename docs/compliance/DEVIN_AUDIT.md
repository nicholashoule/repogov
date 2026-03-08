# Devin (Cognition AI) Audit

Source: https://docs.devin.ai/product-guides/knowledge — verified 2026-03-07.
Source: https://docs.devin.ai/onboard-devin/repo-setup — verified 2026-03-07.

## Configuration Model

Devin does **not** use file-based instruction patterns like `AGENTS.md` or a
rules directory. All persistent context is managed through Devin's web platform
rather than checked-in project files.

### Knowledge (web-managed)

Devin's Knowledge system is configured through the Settings & Library page at
`app.devin.ai`. Each Knowledge item has:

- **Trigger description** — a phrase or sentence Devin uses to recall the item
  when it is relevant to the current task.
- **Content** — a handful of sentences with instructions or context.
- **Scope** — can be pinned to no repo, a specific repo, or all repos.

Organization-level and Enterprise-level Knowledge is shared across team members.
Devin automatically suggests new Knowledge items based on session feedback.

### Repository Setup Notes (web-managed)

The Repo Setup wizard (Settings -> Devin's Machine) includes an **Additional
Notes** step where per-repository instructions can be entered. These notes are
stored in Devin's environment snapshot, not in the repository itself.

### No File-Based Instructions

There is no official documentation for file-based instruction formats (e.g.
`.devin/playbooks/*.md` or `AGENTS.md`) being loaded by Devin automatically. Any
`AGENTS.md` present in a cloned repository may be read by Devin as part of
general codebase context, but there is no guaranteed loading mechanism.

## File Extensions

Not applicable — Devin's instruction system is entirely web-platform managed.

## repogov Support Status

Not applicable in the same way as other agents. Devin's instructions are managed
through its web platform, not through repository files. There is no repogov
preset to create.

Consider documenting Devin integration notes in a project `README.md` or
`docs/` section pointing teams to the Devin knowledge configuration.
