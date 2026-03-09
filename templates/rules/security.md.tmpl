---
applyTo: "**"
---

# Security Instructions

## Secrets and Credentials

- Never commit secrets, tokens, API keys, or private keys to version control
- Store secrets in environment variables or a secrets manager, not in source files
- Provide a `.env.example` with placeholder values to document required variables
- Rotate credentials immediately on suspected or confirmed exposure

## Input Validation

- Validate and sanitize all external input at trust boundaries
- Use parameterized queries or ORM methods -- never interpolate user data into SQL or shell commands
- Sanitize file paths to prevent directory traversal
- Reject unexpected input shapes at the API boundary; do not silently discard fields

## Authentication and Authorization

- Enforce authorization checks before loading or mutating data, not after
- Apply least-privilege to service accounts, API keys, and IAM roles
- Use short-lived tokens with refresh and revocation support
- Do not mix authentication schemes within the same service

## Error Handling

- Never return stack traces, internal paths, or system details in API responses
- Return structured error objects with a code and user-safe message
- Log the full error server-side with request context (request ID, user ID where safe)

## Dependencies

- Pin dependency versions and verify checksums
- Review security advisories before upgrading third-party packages
- Remove unused dependencies; run `go mod tidy` (or equivalent) regularly
- Do not implement custom cryptography -- use audited standard libraries

## Vulnerability Disclosure

- Follow the process in `SECURITY.md` for reporting vulnerabilities
- Do not discuss unpatched vulnerabilities in public issues or PRs
- Update `CHANGELOG.md` under a `Security` section when a fix ships
