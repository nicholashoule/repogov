# Sourcegraph Cody Audit

Source: https://sourcegraph.com/docs/cody — verified 2026-03-07.
Note: Custom instructions docs page returned 404; no file-based rules pattern confirmed.

## Configuration Model

**Status: Unverified for file-based instructions** — Sourcegraph Cody's
official documentation does not document a file-based instruction pattern (e.g.
`.cody/rules/*.md`) as of 2026-03-07. Both
`/docs/cody/capabilities/rules` and `/docs/cody/capabilities/custom-instructions`
returned 404.

### How Cody Uses Context

Cody retrieves context through:

- **@-mentions** — users explicitly reference files, symbols, or remote repos
  in chat.
- **Sourcegraph Search API** — Cody queries the connected Sourcegraph instance
  for relevant code automatically.
- **Prompts** (Prompt Library) — reusable prompt templates created in the
  Sourcegraph web UI and shared across a team.

There is no confirmed mechanism for a committed rules file (analogous to
`AGENTS.md` or `.cursorrules`) that Cody automatically injects as instructions.

## File Extensions

Not applicable — no confirmed file-based instruction pattern.

## Action Items

- Re-verify against official Cody docs once rules/custom-instructions pages
  become available.
- Check the Sourcegraph Cody GitHub repository for any `rules` or
  `custom-instructions` feature implementations:
  `https://github.com/sourcegraph/cody`
- Confirm whether `AGENTS.md` is detected and used by Cody agents.

## repogov Support Status

Not yet supported. Verification needed before implementing. Once a file-based
instruction format is confirmed:

1. Create `DefaultCodyLayout()` in `presets.go`.
2. Add applicable glob rules to `DefaultConfig()`.
3. Add `TestInitLayout_CodySchema` to `init_test.go`.
4. Move this file to the Per-Agent Files table in `AI_AGENTS_AUDIT.md`.
