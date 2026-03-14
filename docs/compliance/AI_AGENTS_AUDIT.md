# AI Agent Setup Audit

Reference file for tracking supported AI agent configuration patterns.
Consult and update this file when adding or modifying agent support in repogov.

## Support Criteria

The following requirements determine whether an AI agent qualifies for a
repogov preset (`Default<Agent>Layout()`) rather than being tracked-only.
An agent must satisfy **all four** criteria to justify the implementation cost.

### 1. File-based, version-controlled instructions

The agent must load its instructions from files that live inside the repository
(or a well-known global config directory such as `~/.config/` or `~/.agent/`).
Instructions managed exclusively through a web platform or IDE settings UI
cannot be governed by repogov.

*Disqualifies:* Devin (Knowledge/Repo Setup via app.devin.ai); any agent
where the only instruction path is a platform-side database.

### 2. Officially documented, stable config paths

A canonical config path must exist in the agent's public, official documentation
and must have remained stable across at least one major release. Undocumented or
community-reported paths create maintenance risk — if the agent changes the path,
the preset silently breaks.

*Disqualifies (currently):* Amazon Q Developer (docs inaccessible); Sourcegraph
Cody (rules docs return 404). Both should be re-evaluated when docs become
available.

### 3. Markdown-compatible instruction files

Instruction files must be plain text — typically Markdown — so that repogov's
line-limit checks are meaningful. Pure YAML/TOML operational config files
(e.g. `.aider.conf.yml`) carry model and API settings, not natural-language
coding instructions; governing their line count provides no value.

*Applies to:* any agent whose instruction path is a YAML/TOML/binary config
rather than a Markdown narrative file.

### 4. Scannable directory structure or predictable filename

repogov's `LayoutSchema` is built around a `Root` directory with `Dirs` glob
rules or named `Required`/`Optional` files. The agent must "own" a directory
(e.g. `.cursor/rules/`, `.windsurf/rules/`, `.claude/rules/`) or have a
well-known filename (e.g. `CLAUDE.md`, `GEMINI.md`) that repogov can target
with a glob. Agents that rely entirely on explicit per-invocation file
enumeration (Aider's `read:` list; Plandex's manual `plandex load`) have no
stable layout to validate.

*Disqualifies as preset (tracked-only):* Aider (instructions enumerated in
`.aider.conf.yml`); Plandex (no auto-loaded instruction files confirmed).

---

## Criteria Validation — All Tracked Agents

Validation of the seven currently tracked agents against the four support criteria.
Legend: Pass / Warn (partial) / Fail

| Agent | C1: File-based | C2: Documented path | C3: Markdown content | C4: Scannable layout | Qualifies | Has Preset |
|-------|:--------------:|:-------------------:|:--------------------:|:--------------------:|:---------:|:----------:|
| GitHub Copilot | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Cursor | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Windsurf | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Claude Code | Pass | Pass | Pass | Pass | **Yes** | Yes |
| AGENTS.md (cross-agent) | Pass | Pass | Pass | Pass | **Yes** | N/A — cross-platform seed |
| OpenAI Codex CLI | Pass | Pass | Pass | Pass | **Yes** | Pending — not yet built |
| Aider | Pass | Pass | Warn | Fail | **No** | No (correct) |
| Amazon Q Developer CLI | Pass | Fail | Pass | Warn | **No** | No — deprecated; Kiro CLI supersedes |
| Kiro CLI | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Gemini CLI | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Continue.dev | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Cline | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Roo Code | Pass | Pass | Pass | Pass | **Yes** | Yes |
| JetBrains AI Assistant | Pass | Pass | Pass | Pass | **Yes** | Yes |
| Zed AI | Pass | Pass | Pass | Pass | **Yes** | Yes |

### Finding: OpenAI Codex CLI has no preset despite qualifying

Codex CLI passes all four criteria — `AGENTS.md` directory walkup and `.agents/skills/`
are both file-based, officially documented, Markdown-native, and glob-scannable. A
`DefaultCodexLayout()` preset should be created. See [CODEX_AUDIT.md](CODEX_AUDIT.md)
action items.

### Finding: Kiro CLI, Gemini CLI, Continue.dev, Cline, Roo Code, JetBrains AI Assistant — presets built (Tier 1)

All six Tier 1 agents pass all four criteria and now have `Default<Agent>Layout()` presets,
glob rules in `DefaultConfig()`, and full `init` support. See the Per-Agent Files section
and individual audit files for details.

### Finding: Amazon Q Developer CLI is deprecated — superseded by Kiro CLI

**C2 (fail):** Official docs were inaccessible at every verification attempt and the
product is now deprecated. The Amazon Q Developer CLI GitHub repo (2026-03-07)
carries a maintenance-mode notice directing users to Kiro CLI. The `.amazonq/rules/`
path was never verified and is now moot.

No `DefaultAmazonQLayout()` should be built. Teams using Amazon Q Developer CLI
should migrate to Kiro CLI. See [AMAZONQ_AUDIT.md](AMAZONQ_AUDIT.md) for migration notes
and [KIRO_AUDIT.md](KIRO_AUDIT.md) for the Kiro CLI configuration model.

### Finding: Aider correctly has no preset

**C3 (partial):** Aider's instruction files can be Markdown, but they must be
discovered by parsing `.aider.conf.yml`'s `read:` key — repogov cannot scan for them
independently without understanding the YAML config.

**C4 (fail):** There is no auto-scanned directory. Files are explicitly enumerated
per invocation. No `Root` + `Dirs` glob exists for a `LayoutSchema`.

Aider's tracked-only status is correct and should be maintained. The audit file
documents the pattern for teams that manually load instruction files via `--read`.

---

## Per-Agent Files

### Supported (repogov preset exists)

Agents with a `Default<Agent>Layout()` preset and full `init` support:

| Agent | Audit File |
|-------|-----------|
| GitHub Copilot | [COPILOT_AUDIT.md](COPILOT_AUDIT.md) |
| Cursor | [CURSOR_AUDIT.md](CURSOR_AUDIT.md) |
| Windsurf (Codeium) | [WINDSURF_AUDIT.md](WINDSURF_AUDIT.md) |
| Claude Code (Anthropic) | [CLAUDE_AUDIT.md](CLAUDE_AUDIT.md) |
| AGENTS.md (cross-agent standard) | [AGENTS_MD_AUDIT.md](AGENTS_MD_AUDIT.md) |
| Kiro CLI | [KIRO_AUDIT.md](KIRO_AUDIT.md) |
| Gemini CLI | [GEMINI_AUDIT.md](GEMINI_AUDIT.md) |
| Continue.dev | [CONTINUE_AUDIT.md](CONTINUE_AUDIT.md) |
| Cline | [CLINE_AUDIT.md](CLINE_AUDIT.md) |
| Roo Code | [ROOCODE_AUDIT.md](ROOCODE_AUDIT.md) |
| JetBrains AI Assistant | [JETBRAINS_AUDIT.md](JETBRAINS_AUDIT.md) |
| Zed AI | [ZED_AUDIT.md](ZED_AUDIT.md) |

### Tracked (no preset — criteria not met or preset not yet built)

Agents documented for awareness but not implemented in repogov:

| Agent | Audit File | Status |
|-------|-----------|--------|
| Aider | [AIDER_AUDIT.md](AIDER_AUDIT.md) | Does not qualify — fails C3 and C4; no preset planned |
| OpenAI Codex CLI | [CODEX_AUDIT.md](CODEX_AUDIT.md) | Qualifies (Tier 2) — preset not yet built; see action items in audit file |
| Amazon Q Developer CLI | [AMAZONQ_AUDIT.md](AMAZONQ_AUDIT.md) | Deprecated — superseded by Kiro CLI; no preset planned |

## Planned Support Backlog

Agents not yet supported by repogov, tracked here for future implementation.
Each agent has its own audit file below. When work begins on an agent, move its
row to the Per-Agent Files table above.

| # | Agent | Audit File | Verified Config Path(s) | Notes |
|---|-------|-----------|-------------------------|-------|
| 1 | Amazon Q Developer CLI | [AMAZONQ_AUDIT.md](AMAZONQ_AUDIT.md) | **Deprecated** — superseded by Kiro CLI. `.amazonq/rules/*.md` was unverified and is now moot. | CLI is maintenance-only; no preset planned; see AMAZONQ_AUDIT.md |
| 2 | Sourcegraph Cody | [SOURCEGRAPH_AUDIT.md](SOURCEGRAPH_AUDIT.md) | Unverified — no file-based instructions found in official docs | Rules docs pages return 404; see audit file for action items |
| 3 | Devin | [DEVIN_AUDIT.md](DEVIN_AUDIT.md) | None (web-platform managed) | Instructions via Knowledge/Repo Setup in app.devin.ai; not file-based |
| 4 | Plandex | [PLANDEX_AUDIT.md](PLANDEX_AUDIT.md) | `.plandex/` (state dir, not instructions); explicit `plandex load` for context | No auto-loaded instruction file confirmed; docs unavailable |


## Implementation Priority Tiers

Priority ranking for agents in the Planned Support Backlog above. Tiers are based on documentation quality, pattern stability, adoption, and implementation effort.

### Tier 1 — Fully documented, implement now

All four C1–C4 criteria are met; docs verified; patterns are stable and widely adopted.

**All Tier 1 agents have been implemented.** See the Supported section above.

| Agent | Root / Single File | Multi-file Dir | Status |
|-------|-------------------|----------------|--------|
| Kiro CLI | — | `.kiro/steering/*.md` | **Built** — `DefaultKiroLayout()` |
| Gemini CLI | `GEMINI.md` | — | **Built** — `DefaultGeminiLayout()` |
| Continue.dev | — | `.continue/rules/*.md` | **Built** — `DefaultContinueLayout()` |
| Cline | — | `.clinerules/*.md` | **Built** — `DefaultClineLayout()` |
| Roo Code | — | `.roo/rules/*.md` + `.roo/rules-{mode}/*.md` | **Built** — `DefaultRooCodeLayout()` |
| JetBrains AI Assistant | — | `.aiassistant/rules/*.md` | **Built** — `DefaultJetBrainsLayout()` |
| Zed AI | `.rules` | — | **Built** — `DefaultZedLayout()` |

### Tier 2 — Viable with minor caveats

Criteria met but implementation is more complex or docs have minor gaps.

| Agent | Root / Single File | Multi-file Dir | Caveat |
|-------|-------------------|----------------|--------|
| OpenAI Codex CLI | `AGENTS.md` (directory walkup) | `.agents/skills/` (per-skill `SKILL.md`) | Skills sub-dir structure is more complex than a flat glob |

### Tier 3 — Tracked only, no preset planned

| Agent | Reason |
|-------|--------|
| Amazon Q Developer CLI | Deprecated; maintenance-only; superseded by Kiro CLI |
| Devin | Web-platform managed; no repo files (fails C1) |
| Plandex | No auto-loaded instruction files confirmed (fails C4) |
| Sourcegraph Cody | No confirmed file-based instruction pattern (fails C2 and C4) |
| Aider | No auto-scanned directory; explicit `--read` enumeration (fails C4) |

## Support Matrix

Agents with a `repogov Preset` value of `none` are tracked for reference only and are not supported.

| Agent | Root Config | Multi-file Dir | repogov Preset | Verified |
|-------|-------------|----------------|----------------|:---------:|
| GitHub Copilot | `.github/copilot-instructions.md` | `.github/instructions/*.instructions.md` | `DefaultCopilotLayout()` | Yes |
| Cursor | none (use rules dir) | `.cursor/rules/*.md` or `*.mdc` | `DefaultCursorLayout()` | Yes |
| Windsurf | `.windsurf/rules/*.md` (`trigger: always_on`); `~/.codeium/windsurf/memories/global_rules.md` (global) | `.windsurf/rules/*.md` with `trigger:` frontmatter | `DefaultWindsurfLayout()` | Yes |
| Claude Code | `CLAUDE.md` or `.claude/CLAUDE.md` | `.claude/rules/*.md`, `.claude/agents/*.md` | `DefaultClaudeLayout()` | Yes |
| AGENTS.md (all agents) | `AGENTS.md` (root or nested) | none | cross-platform, seeded by `init` | Yes |
| Aider | `.aider.conf.yml` (tool config); `CONVENTIONS.md` via `--read` (instructions) | no directory pattern; multi-file via `read:` list in `.aider.conf.yml` | none | n/a |
| OpenAI Codex CLI | `AGENTS.md` (directory walkup + `AGENTS.override.md`) | `.agents/skills/` (per-skill `SKILL.md`); nested `AGENTS.md` tree | none | Yes |
| Amazon Q Developer CLI | `.amazonq/rules/*.md` (unverified; CLI now deprecated) | `.amazonq/rules/*.md` | none — deprecated | No |
| Kiro CLI | `AGENTS.md` at workspace root (also detected) | `.kiro/steering/*.md` with `inclusion:` frontmatter controlling load mode (`always`, `fileMatch`, `manual`, `auto`) | `DefaultKiroLayout()` | Yes |
| Gemini CLI | `GEMINI.md` (hierarchical walkup); `~/.gemini/GEMINI.md` global | — | `DefaultGeminiLayout()` | Yes |
| Continue.dev | — | `.continue/rules/*.md`; `~/.continue/rules/*.md` global | `DefaultContinueLayout()` | Yes |
| Cline | — | `.clinerules/*.md`; `~/Documents/Cline/Rules/` global | `DefaultClineLayout()` | Yes |
| Roo Code | — | `.roo/rules/*.md`; `~/.roo/rules/` global | `DefaultRooCodeLayout()` | Yes |
| JetBrains AI Assistant | — | `.aiassistant/rules/*.md` | `DefaultJetBrainsLayout()` | Yes |
| Zed AI | `.rules` (root, first match wins) | — | `DefaultZedLayout()` | Yes |

All preset agents enforce a 300-line limit on scoped instruction files.
`AGENTS.md` is governed at 200 lines (root) and the default (300) for nested files.
`GEMINI.md` is governed at 200 lines (enforced via `DefaultConfig().Files`).
`.claude/CLAUDE.md` is governed at 200 lines (enforced via `DefaultConfig().Files`).
`.github/copilot-instructions.md` is governed at 50 lines (enforced via `DefaultConfig().Files`).

## Supported File Extensions

Extensions recognized by each agent for instruction/config files in a git repo.

| Agent | Extension(s) | Notes |
|-------|-------------|-------|
| GitHub Copilot | `.md` | `copilot-instructions.md`, `rules/*.md`, `instructions/*.instructions.md`, `prompts/*.prompt.md`; `excludeAgent:` frontmatter key excludes rule from specific Copilot features |
| Cursor | `.md`, `.mdc` | `.mdc` supports YAML frontmatter (`description`, `globs`, `alwaysApply`); subdirectories within `.cursor/rules/` supported; Team/Enterprise: dashboard-managed Team Rules and remote GitHub-imported rules |
| Windsurf | `.md` | `.windsurf/rules/*.md` (trigger frontmatter: `always_on`, `model_decision`, `glob`, `manual`); global `global_rules.md`; legacy `.windsurfrules` (no extension, not in current docs) |
| Claude Code | `.md`, `.json` | `CLAUDE.md`, `rules/*.md`, `agents/*.md`; `settings.json` / `settings.local.json` for permissions and hooks |
| AGENTS.md (cross-agent) | `.md` | Plain Markdown; no frontmatter required |
| Kiro CLI | `.md` | `.kiro/steering/*.md`; optional `inclusion:` frontmatter controls load mode (`always`/`fileMatch`/`manual`/`auto`); file refs via `#[[file:path]]` |
| Gemini CLI | `.md` | `GEMINI.md`; hierarchical walkup from CWD to `$HOME` |
| Continue.dev | `.md` | `.continue/rules/*.md`; YAML frontmatter: `globs`, `alwaysApply`, `description` |
| Cline | `.md` | `.clinerules/*.md`; also accepts `.cursorrules`, `.windsurfrules`; YAML `paths:` frontmatter for conditional rules |
| Roo Code | `.md` | `.roo/rules/*.md`; fallback `.roorules`; YAML frontmatter supported |
| JetBrains AI Assistant | `.md` | `.aiassistant/rules/*.md`; rule types: Always / Manually / By model decision / By file patterns |
| Aider | `.yml`, `.md` (any) | `.aider.conf.yml` (tool config); `--read` / `read:` accepts any file as instruction context |
| OpenAI Codex CLI | `.md`, `.toml`, `.yaml` | `AGENTS.md` (directory walkup); `.agents/skills/<name>/SKILL.md`; `~/.codex/config.toml` (global config); `agents/openai.yaml` (skill UI metadata) |

## Maintenance Notes

When adding a new agent:

1. Create a `<AGENT>_AUDIT.md` file in this directory following the existing per-agent file structure.
2. Add a `Default<Agent>Layout()` function to `presets.go`.
3. Add applicable glob rules to `DefaultConfig()` in `repogov.go`.
4. Update `.github/repogov-config.json` rules to match.
5. Add `TestInitLayout_<Agent>Schema` to `init_test.go`.
6. Add the agent to the support matrix and file extensions table above.
7. Add a row to the Per-Agent Files table above linking to the new audit file.

GitHub vs GitLab: note if anything needs to be different depending on the SaaS platform.

When an existing agent deprecates or changes its config format:

1. Update the relevant per-agent audit file with source URL and date verified.
2. Adjust the preset `Optional` list or `Dirs` map as needed.
3. Add `Naming.Exceptions` for any mandated uppercase filenames.
4. Update the support matrix above.

The CLI agent names map to presets as follows: `copilot` -> `DefaultCopilotLayout()`, `cursor` -> `DefaultCursorLayout()`, `windsurf` -> `DefaultWindsurfLayout()`, `claude` -> `DefaultClaudeLayout()`, `kiro` -> `DefaultKiroLayout()`, `gemini` -> `DefaultGeminiLayout()`, `continue` -> `DefaultContinueLayout()`, `cline` -> `DefaultClineLayout()`, `roocode` -> `DefaultRooCodeLayout()`, `jetbrains` -> `DefaultJetBrainsLayout()`.
