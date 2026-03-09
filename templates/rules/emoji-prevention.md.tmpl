---
applyTo: "**"
---

# Emoji Prevention Instructions

## Rule

Do **not** introduce emoji or Unicode pictographic characters into
source code, documentation, comments, commit messages, or CI output.
Use plain-text equivalents instead.

## Why

- Emoji render inconsistently across terminals, editors, and fonts
- They can complicate grep, diff, and other line-oriented tools
- Screen readers may announce them unexpectedly or skip them entirely
- They inflate token counts in LLM-assisted workflows
- They add no semantic value that plain text cannot convey

## Scope

These rules apply to all files tracked by git:

- Source files like (`.go`, `.py`, `.js`, etc.)
- Markdown documentation (`.md`, `.markdown`, etc.)
- YAML and JSON configuration (`.yml`, `.yaml`, `.json`)
- Shell scripts and Makefiles (`.sh`, `Makefile`, etc.)
- Commit messages and PR descriptions

## Exceptions

- Test fixtures that explicitly exercise emoji handling
- Docs that describe or reference emoji (e.g., unicode-coverage.md)
- Third-party files vendored as-is

## Text alternatives

Use plain-text equivalents that convey the same meaning:

| Instead of | Write |
|------------|-------|
| checkmark / cross | `[PASS]` / `[FAIL]` |
| warning / info | `WARNING:` / `NOTE:` |
| fire / breaking change | `[BREAKING]` |
| lock / key | `Security:` / `Auth:` |
| bug | `BUG:` |
| construction / WIP | `[WIP]` or `Draft:` |

## Enforcement

Install once:

```sh
go install github.com/nicholashoule/demojify-sanitize/cmd/demojify@latest
```

Audit (exits `1` if emoji found):

```sh
demojify
```

Fix in place:

```sh
demojify -fix # Rewrite stripped affected files in place after reporting
demojify -sub # Substitute emoji with text tokens instead of stripping; implies -fix
```
