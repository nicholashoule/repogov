# Windsurf (Codeium) Audit

Source: https://docs.devin.ai/desktop/cascade/memories (redirected from
https://docs.windsurf.com/windsurf/cascade/memories) — verified 2026-10-17.

## Deprecation Notice

Date: 2026-10-17. The official Windsurf docs URL now returns a 307 redirect to
`docs.devin.ai/desktop/...` (Devin Desktop docs). That page states memories apply to the
"legacy Cascade agent only", that the Devin Local agent does not persist memories, and points
to a "Devin: Open Cascade Migration Wizard" for moving memories to skills. "Windsurf" appears
only as the legacy name for paths. The same page says the product is now called **Devin Desktop** and its default
agent is **Devin Local** (it still calls the assistant UI "Cascade"), and gives no end-of-life
date for legacy Cascade or the `.windsurf` paths. Third-party sources (not vendor-verified) date
the rename to 2026-06-02 and Cascade's retirement to 2026-07-01. Migration target: `.devin/rules/` (preferred), with `.windsurf/rules/` still
read as fallback.

## Configuration Files

### Workspace Rules

- `.devin/rules/*.md` — now the preferred workspace location per current docs; repogov
  has no preset for it (see Preset).
- `.windsurf/rules/*.md` — fallback (legacy-named) multi-file rules directory. Each `.md` file
  requires a `trigger:` field in YAML frontmatter that controls when it is applied.
  Windsurf searches `.windsurf/rules/` in subdirectories and walks up to the git
  root, so nested rule files are also loaded.

  **Trigger modes** (set via `trigger:` frontmatter field):

  | Trigger | Behaviour |
  |---------|-----------|
  | `always_on` | Injected into every Cascade request automatically |
  | `model_decision` | Model decides whether the rule is relevant |
  | `glob` | Applied when the active file matches the `globs:` pattern |
  | `manual` | Injected only when the user explicitly invokes the rule |

  Example frontmatter:
  ```yaml
  ---
  trigger: glob
  globs: "**/*.go"
  ---
  ```

- `AGENTS.md` — supported natively by Windsurf's rules engine. A root-level
  `AGENTS.md` is treated as `always_on`; a subdirectory `AGENTS.md` is treated
  as an auto-glob rule scoped to that directory.

- `.windsurfrules` — legacy single-file format at workspace root; current docs say it is
  still read.

### Global Rules

- `~/.codeium/windsurf/memories/global_rules.md` — single Markdown file applied
  automatically across all projects. Maximum 6,000 characters.

### Enterprise / System Rules

- macOS `/Library/Application Support/Devin/rules/`, Linux/WSL `/etc/devin/rules/`,
  Windows `C:\ProgramData\Devin\rules\` (legacy `Windsurf` equivalents, e.g.
  `/etc/windsurf/rules/`). Managed by IT/security teams.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | `.windsurf/rules/*.md` and global `global_rules.md` |
| (none) | Legacy `.windsurfrules` has no extension |

## Limits

| Scope | Limit |
|-------|-------|
| Workspace rule file | 12,000 characters per `.md` file |
| Global rules file | 6,000 characters (`global_rules.md`) |
## Memory Configuration

Windsurf includes two separate memory mechanisms:

### User-managed rules / global rules

These are files the user writes manually:

| Scope | File | Auto-loaded |
|-------|------|-------------|
| Project | `.windsurf/rules/*.md` (any `trigger:`) | Yes — per trigger mode |
| Global | `~/.codeium/windsurf/memories/global_rules.md` | Yes — injected into every Cascade request |
| System | OS-specific system-level directory (IT-managed) | Yes — read-only for users |

`global_rules.md` is a user-written cross-project instruction file: plain Markdown, max 6,000
characters, managed directly by the user (Cascade does **not** update it automatically).

### Auto-generated memories (Cascade)

Cascade can automatically generate and retrieve per-workspace memories as it works:

| Item | Detail |
|------|---------|
| Storage location | `~/.codeium/windsurf/memories/` (per workspace; machine-local, outside repo) |
| Content | Auto-generated facts Cascade believes are useful for future sessions |
| Loading | Retrieved automatically when Cascade judges them relevant (no credits consumed) |
| User control | Memories panel in Cascade UI; individual memories can be deleted |
| Distinction | NOT the same as `global_rules.md`; auto-memories are ephemeral agent-generated facts |

Repogov does not govern auto-memories (they are outside the repo). Prefer
`.windsurf/rules/` with `trigger: always_on` for durable, team-shared project context.

### Workflows and Skills

- **Workflows** (`~/.codeium/windsurf/workflows/`) — slash-command prompt templates
  (`/workflow-name`). User-written; not project files.
- **Skills** — invoked by the model or @mention; the docs page gives no paths
  (third-party sources mention `.windsurf/skills/`, unverified).

## Seeded Files

`init -agent windsurf` creates the `.windsurf/rules/` directory and seeds:

| File | Frontmatter | Purpose |
|------|-------------|---------|
| `general.md` | `trigger: always_on` | General project conventions |

## Preset

`DefaultWindsurfLayout()` in `presets.go`. Validates `.windsurf/rules/` with `*.md` glob.
Follow-up: add `.devin/rules/` support (new preferred path).
CLI agent name: `windsurf`.
