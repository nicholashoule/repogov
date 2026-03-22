# Git Hooks

`repogov` ships a cross-platform pre-commit hook implemented in Go. The hook
runs automatically before every commit and blocks the commit if any check fails.

## What the Hook Checks

The pre-commit hook (`scripts/hooks/pre-commit`) runs three phases in order.
All must pass for the commit to proceed.

### Phase 1: repogov governance

Runs `repogov -agent copilot` via `go run` to check line limits and layout.

### Phase 2: demojify-sanitize

Runs `demojify` to detect emoji characters. If emoji are found, automatically
substitutes them, re-stages changes, and aborts so the user can review.

### Phase 3: Go checks (pre-commit.go)

`scripts/hooks/pre-commit.go` runs four checks sequentially:

| Check | Tool | What it enforces |
|-------|------|-----------------|
| `gofmt` | `gofmt -s` | Go source formatting. Auto-fixes and re-stages unformatted files. |
| `go vet` | `go vet ./...` | Go static analysis. |
| `go test` | `go test ./...` | All tests must pass. |
| `golangci-lint` | `golangci-lint run ./...` | Linter checks. Must be installed separately. |

### gofmt (auto-fix)

Unlike most hooks, the formatting check **auto-fixes** problems instead of just
failing. When unformatted files are found:

1. `gofmt -s -w` is run on only the affected files.
2. The reformatted files are re-staged with `git add`.
3. The commit proceeds.

### Markdown line limits

Limits are resolved from `repogov-config.json` (searched via standard discovery),
with built-in rules for common paths:

| Path | Limit |
|------|-------|
| `.github/copilot-instructions.md` | 50 lines |
| `.github/instructions/*.md` | 300 lines |
| Everything else | 300 lines (default) |

Per-file overrides can be placed in `.github/repogov-config.json`:

```json
{
  "files": {
    "docs/design.md": 600,
    "CHANGELOG.md": 0
  }
}
```

A limit of `0` exempts the file from checking.

## Installation

Install once per clone using `make`:

```sh
make hooks
```

This copies `scripts/hooks/pre-commit` to `.git/hooks/pre-commit` and marks it
executable. The shell wrapper delegates to the Go implementation via `go run`,
so no compiled binary is required.

### Manual installation

```sh
cp scripts/hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### Uninstallation

```sh
rm .git/hooks/pre-commit
```

## How it works

The hook consists of two files:

| File | Purpose |
|------|---------|
| `scripts/hooks/pre-commit` | POSIX shell wrapper; runs repogov, demojify-sanitize, and delegates to pre-commit.go |
| `scripts/hooks/pre-commit.go` | Cross-platform Go implementation of gofmt, go vet, go test, and golangci-lint checks |

Because the implementation is `go run`-based, it works identically on Linux, macOS,
and Windows (via Git for Windows' bundled `sh.exe`) without any compiled artifacts to
commit or manage.

The repo root is passed as the first positional argument so the Go implementation
can `chdir` back after `go run` resolves the module in `scripts/hooks/`.

## Output format

Each check prints one of:

```
[PASS]  gofmt
[AUTO]  gofmt: reformatted and re-staged the following files:
          repogov.go
[FAIL]  gofmt: ...
[PASS]  go vet
[FAIL]  go vet (run: make vet)
[PASS]  zero dependencies
[FAIL]  go.mod contains require directive -- zero-dependency contract violated
[PASS]  no emoji violations
[FAIL]  emoji violations found in N file(s):
          path/to/file.md
[PASS]  markdown limits (N file(s) checked, N exempt)
[FAIL]  markdown limits: N of N file(s) over limit (N exempt)
  [PASS]  README.md -- 42/300 lines (14% of limit)
  [WARN]  docs/design.md -- 260/300 lines (86% of limit, 40 remaining)
  [FAIL]  some/big-file.md -- 312/300 lines (104% of limit, 12 over)
          FIX: shorten content, or add an override to .github/repogov-config.json
```

## Skipping the hook

To bypass the hook for a single commit (use sparingly):

```sh
git commit --no-verify -m "message"
```

## See Also

- [cli.md](cli.md) — full CLI reference for running the same checks manually
- [design.md](design.md) — architecture and design decisions
