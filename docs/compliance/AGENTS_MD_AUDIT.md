# AGENTS.md Cross-Agent Standard Audit

Source: https://agents.md
Verified: 2026-10-17

## Overview

`AGENTS.md` is an open cross-agent standard for providing context to AI coding agents. It is
recognized by GitHub Copilot, Cursor, OpenAI Codex, Gemini CLI, and others. The project is now
stewarded by the Agentic AI Foundation under the Linux Foundation (per agents.md). Claude Code
is not listed on agents.md and its support was not re-verified here.

## Configuration Files

- `AGENTS.md` at the repo root — loaded by all major coding agents as a "README for agents".
  Plain Markdown; no frontmatter or required fields (agents.md specifies no schema).
  Explicit user prompts in chat override any `AGENTS.md`.
- **Nested scoping**: an `AGENTS.md` in any subdirectory provides directory-scoped instructions.
  Agents walk up from the current file to the repo root and load every `AGENTS.md` found; the
  closer file takes precedence.

## Typical Sections

- Context / overview
- Dev environment tips
- Testing instructions
- PR instructions
- Links to `docs/` and platform-specific instruction directories

## Memory Configuration

`AGENTS.md` itself _is_ the cross-agent memory layer — it is the standard mechanism for
providing persistent, always-on context to any agent that supports it. There is no separate
`memory.md` concept for this format.

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Root | `AGENTS.md` (repo root) | Yes — loaded by all supporting agents |
| Nested | `AGENTS.md` (any subdirectory) | Yes — scoped to that directory and below |

Agents listed on agents.md (2026-10-17) include Codex, Jules, Factory, Aider, goose, opencode,
Zed, Warp, VS Code, Devin, Junie, Amp, Cursor, RooCode, Gemini CLI, Kilo Code, Semgrep, GitHub
Copilot coding agent, Windsurf, and Augment Code. Aider needs `read: AGENTS.md` in
`.aider.conf.yml`; Gemini CLI needs its context file name set to `AGENTS.md` in
`.gemini/settings.json`. Earlier claims for Claude Code, Kiro CLI, Continue.dev and Cline were
not re-verified (GitHub's Copilot docs also read `CLAUDE.md`/`GEMINI.md` as alternatives). Place long-lived project context here
to share it across agents without duplicating it in each agent's own rule files.

## Limits

- Root `AGENTS.md`: 200 lines (repogov governance).
- Nested `AGENTS.md` files: 300 lines (default).

## Init Behavior

`init` seeds an `AGENTS.md` at the repo root for all platforms, with links to `docs/`,
`README.md`, and the schema's instruction/rules directory. The context section labels each
platform's rules dir by name (e.g., "Cursor rule files", "Windsurf rule files", "Claude rule files").
