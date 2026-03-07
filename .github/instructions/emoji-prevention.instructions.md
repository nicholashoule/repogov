---
applyTo: "**/*.md"
---

# Emoji Prevention Instructions

## Rule

Do **not** introduce emoji or Unicode pictographic characters into
source code, documentation, comments, commit messages, or CI output.
Use plain-text equivalents instead.

## Why

- Emoji render inconsistently across terminals, editors, and fonts
- They break grep, diff, and other line-oriented tools
- Screen readers may announce them unexpectedly or skip them entirely
- They inflate token counts in LLM-assisted workflows
- They add no semantic value that plain text cannot convey

## Common Substitutions

| Instead of | Write |
|------------|-------|
| U+2705 / U+2714 | `[PASS]`, `OK`, `yes` |
| U+274C / U+2716 | `[FAIL]`, `ERROR`, `no` |
| U+26A0 | `[WARN]`, `WARNING` |
| U+2139 | `[INFO]`, `NOTE` |
| U+1F680 | `deploy`, `launch`, `ship` |
| U+1F41B | `bug`, `defect`, `issue` |
| U+1F527 | `fix`, `patch`, `repair` |
| U+2728 | `new`, `feature`, `add` |
| U+1F4DD | `docs`, `note`, `document` |
| U+1F525 | `hot`, `critical`, `urgent` |
| U+27A1 / U+2192 | `->`, `-->`, `==>` |
| U+1F504 | `refactor`, `rework`, `restructure` |

## Text alternatives

| Instead of | Write |
|------------|-------|
| checkmark emoji | `[PASS]` or `[OK]` |
| cross/X emoji | `[FAIL]` or `[ERROR]` |
| warning emoji | `WARNING:` |
| info/note emoji | `[INFO]` or `NOTE:` |
| lightbulb emoji | `TIP:` |
| rocket emoji | `Deployment` or `Released` |
| chart emoji | `Report` or `Metrics` |
| star emoji | `[FEATURED]` |
| lock emoji | `Security` |

## Enforcement

Use the `demojify-sanitize` tool to detect emoji:

```sh
go install github.com/nicholashoule/demojify-sanitize/cmd/demojify@latest
```

- **Pre-commit hook**: install `demojify` as a pre-commit hook to prevent new emoji from being committed.
- **Manual audit**: `go run github.com/nicholashoule/demojify-sanitize/cmd/demojify -root .`
- **Auto-fix with removal**: `demojify -root . -fix`

## Scope

This rule applies to all files tracked by git:

- Go source files (`.go`)
- Markdown documentation (`.md`)
- YAML and JSON configuration (`.yml`, `.yaml`, `.json`)
- Shell scripts and Makefiles
- Commit messages and PR descriptions

## Exceptions

- Test fixtures that explicitly exercise emoji handling
- Docs that describe or reference emoji (e.g., unicode-coverage.md)
- Third-party files vendored as-is
