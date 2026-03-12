---
applyTo: "**"
---

# General Instructions

## Writing Style

- Do **not** introduce emoji or Unicode pictographic characters
- Keep language concise and direct; prefer active voice
- Use US English spelling (e.g., "behavior", "summarize")
- Avoid jargon unless the audience expects it

Emoji prevention rules are detailed in .github/rules/emoji-prevention.md.

## Document Structure

- Start every document with a level-1 heading (`# Title`)
- Use heading levels sequentially -- do not skip levels
- Use lists for three or more related items

## Formatting

- Wrap inline code, commands, file paths, and symbols in backticks
- Use fenced code blocks with a language identifier for multi-line code
- Use tables when presenting structured comparisons or reference data
- Keep lines under 120 characters where possible for diff readability

## File Organization

- Keep files focused and within configured line limits
- Place detailed content in `docs/` and link from top-level files
- Do not duplicate content -- link to the canonical source instead

## File Naming Conventions

**Prefer lowercase filenames** in `docs/` and `.github/` directories:

- Use `kebab-case` or `snake_case` for multi-word filenames
- Exception: `*_AUDIT.md` files in `docs/` may use uppercase
- Exception: GitHub-mandated filenames remain uppercase (e.g., `CODEOWNERS`)

## Links and References

- Use relative links for in-repo references
- Use descriptive link text -- avoid "click here" or bare URLs

## Repository Conventions

- Follow existing code style and patterns
- Keep files within configured line limits
- Write tests for new functionality
