---
applyTo: "**/*.md"
---

# General Instructions

## Writing Style

- Use clear, concise language
- Prefer active voice over passive voice
- Write in complete sentences
- Keep paragraphs focused on a single idea
- Avoid jargon unless the audience expects it

## Document Structure

- Start every document with a level-1 heading (`# Title`)
- Use heading levels sequentially -- do not skip levels
- Keep headings descriptive and concise
- Add a blank line before and after headings
- Use lists for three or more related items

## Formatting

- Wrap inline code, commands, file paths, and symbols in backticks
- Use fenced code blocks with a language identifier for multi-line code
- Use bold for emphasis on key terms, not for entire sentences
- Use tables when presenting structured comparisons or reference data
- Keep lines under 120 characters where possible for diff readability

## File Organization

- Use consistent file naming conventions (see copilot-instructions.md)
- Keep files focused and within configured line limits
- Place detailed content in `docs/` and link from top-level files
- Do not duplicate content -- link to the canonical source instead

## Links and References

- Use relative links for in-repo references
- Verify links are valid before committing
- Use descriptive link text -- avoid "click here" or bare URLs
- Reference specific sections with anchor links when useful

## Maintenance

- Keep documentation in sync with the feature it describes
- Remove or update stale content promptly
- Date-stamp time-sensitive information
- Review docs as part of the pull request process
