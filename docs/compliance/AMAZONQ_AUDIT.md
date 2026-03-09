# Amazon Q Developer CLI Audit

Sources:
- https://github.com/aws/amazon-q-developer-cli — verified 2026-03-07 (maintenance-mode notice added)
- https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/q-chat-custom-instructions.html — blocked/no content (multiple attempts)
- https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/customizing-q.html — blocked/no content (multiple attempts)

## [WARNING] Deprecation Notice (verified 2026-03-07)

> **Amazon Q Developer CLI is no longer actively maintained** and will only
> receive critical security fixes. Amazon Q Developer CLI is transitioning to
> **Kiro CLI** (`kiro.dev`), a closed-source product by AWS. For the latest
> features and updates, teams should migrate to Kiro CLI.
>
> Source: `https://github.com/aws/amazon-q-developer-cli` README — commit
> `e14ea18` by `shankara-n`, message "Add maintenance status and Kiro CLI
> transition notice (#3524)", merged ~3 months before 2026-03-07.

See [KIRO_AUDIT.md](KIRO_AUDIT.md) for the Kiro CLI configuration model,
which supersedes this file for all new work.

## Previously Reported Configuration (unverified, now moot)

**Status: Unverified and superseded.** Official Amazon Q Developer docs
returned no accessible content from multiple attempts. The following pattern
was widely reported by the community but was never verified from official
documentation. Given the product's deprecation, verification is no longer
worthwhile.

### Reported Workspace Rules

- `.amazonq/rules/` — reported multi-file rules directory for workspace-level
  instructions. Files reportedly Markdown (`.md`).

### Reported Global Rules

- `~/.aws/amazonq/rules/` — reported global rules directory.

## File Extensions

| Extension | Notes |
|-----------|-------|
| `.md` | Reported format for rules in `.amazonq/rules/` (unverified) |

## repogov Support Status

**No preset planned.** Amazon Q Developer CLI is deprecated/maintenance-mode.
No `DefaultAmazonQLayout()` will be built. Teams should migrate to Kiro CLI.

See [KIRO_AUDIT.md](KIRO_AUDIT.md) and the action items there for implementing
`DefaultKiroLayout()`.

## Action Items

- ~~Re-verify against official Amazon Q Developer docs~~ — moot given deprecation.
- Track Kiro CLI migration progress in [KIRO_AUDIT.md](KIRO_AUDIT.md).
- Consider communicating the migration path in project-level `AGENTS.md` or
  `docs/` for teams that currently use Amazon Q Developer.
