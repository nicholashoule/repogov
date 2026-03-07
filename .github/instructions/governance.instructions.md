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

Run `repogov -root . limits` or use the library before committing.
Refactor or split files that approach their limit -- do not raise limits
without justification.

## Layout

The `.github/` directory must satisfy the GitHub layout preset.
Run `repogov -root . layout` to validate structure.

## AI Agent Support

When adding or modifying agent support, consult [docs/ai-agents-audit.md](../../docs/ai-agents-audit.md)
for the current support matrix, per-agent config patterns, and maintenance steps.

## Zero Dependencies

This repository has no external Go module dependencies.
Do not add `require` directives to the root `go.mod`.

## Pre-commit Hook Example

Use repogov as a dependency-free pre-commit hook:

```go
package main

import (
    "fmt"
    "os"
    "github.com/nicholashoule/repogov"
)

func main() {
    cfg := repogov.DefaultConfig()
    results, _ := repogov.CheckDir(".", []string{".md"}, cfg)
    fmt.Fprint(os.Stderr, repogov.Summary(results))
    layout, _ := repogov.CheckLayout(".", repogov.DefaultGitHubLayout())
    fmt.Fprint(os.Stderr, repogov.LayoutSummary(layout))
    if !repogov.Passed(results) || !repogov.LayoutPassed(layout) {
        os.Exit(1)
    }
}
```

## Minimal CLI Example

```bash
go install github.com/nicholashoule/repogov/cmd/repogov@latest
repogov -root . -exts .md all
```
