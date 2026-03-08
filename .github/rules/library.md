---
applyTo: "**/*.md"
---

# Library Documentation Instructions

## API Reference Documentation

- Document every public symbol in a reference table or section
- Include the function signature, a one-line summary, and the source file
- Group symbols by concern (types, functions, configuration)
- Keep the public API table up to date when symbols are added or removed
- Link to doc comments or pkg.go.dev for full details

## README Structure

- Start with a one-paragraph summary of what the library does
- Include install and quick-start sections early
- Provide at least one runnable example
- List configuration options with defaults and valid ranges
- Add a public API table linking symbols to source files
- End with status codes, development instructions, and license

## Design Documents

- Place architecture and design docs in `docs/`
- Start with purpose, then architecture, then details
- Document constraints and trade-offs, not just decisions
- Include diagrams or tables when they clarify structure
- Keep design docs under their configured line limit

## Changelog

- Maintain a CHANGELOG.md following Keep a Changelog format
- Group entries under Added, Changed, Deprecated, Removed, Fixed, Security
- Reference issue or PR numbers for traceability
- Write entries from the user's perspective, not the developer's

## Cross-References

- Link between README, docs/, and instruction files -- do not duplicate
- Use relative links for all in-repo references
- Reference specific sections with anchor links
- Keep the link graph shallow: two hops maximum to find any topic

## Versioning Documentation

- Document breaking changes prominently in CHANGELOG and README
- Mark deprecated features with a timeline for removal
- Update install instructions when the minimum version changes
- Tag documentation updates alongside code releases
