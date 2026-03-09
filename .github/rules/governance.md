---
applyTo: "**"
---

# Governance Instructions

## Line Limits

All files must stay within their configured line limit.
See [repogov-config.json](../repogov-config.json) for limits and rules.

- **Resolution order**: per-file override, first matching glob, default
- A limit of `0` exempts the file (status = SKIP)
- WARN at the configured `warning_threshold` percentage

## Enforcing Limits

### Minimal CLI Example

```sh
go run github.com/nicholashoule/repogov/cmd/repogov@latest -agent copilot
go run github.com/nicholashoule/repogov/cmd/repogov@latest -agent copilot init
```

Pre-commit hook (`.git/hooks/pre-commit`):

```sh
#!/bin/sh
go run github.com/nicholashoule/repogov/cmd/repogov@latest limits
```
