// Command xplat is a tiny cross-platform shim for the filesystem operations
// Makefiles need, so recipes contain ZERO shell-specific syntax.
//
// Vendored per-repo cross-platform Makefile helper (no shared module by
// design: a ~120-line build shim should not add a build-time dependency).
//
// Make invokes `go run ./tools/xplat <cmd> ...` directly (never a shell
// builtin), so the same recipe works identically under cmd.exe, PowerShell,
// and POSIX sh. This replaces the fragile, recurring `ifeq ($(OS)...)` /
// `$(findstring sh,$(SHELL))` cmd-vs-sh dual recipes (mkdir -p vs
// "if not exist", rm -rf vs "rmdir /s /q", cp vs "copy /Y", test -f vs
// "if not exist") that broke on Windows ("The syntax of the command is
// incorrect.").
//
// Subcommands:
//
//	mkdir <dir>              os.MkdirAll (idempotent; silent if it exists)
//	rmrf <path>...           os.RemoveAll each (idempotent; silent if absent)
//	cp <src> <dst>           copy a file, dst mode 0o755 (carries the
//	                         executable bit POSIX git hooks need; a no-op
//	                         distinction on Windows)
//	require <path> <message> exit 1 with <message> if <path> is missing,
//	                         else exit 0 (build-precondition guard)
//	help <makefile>...       render the `## ` help screen (replaces the
//	                         awk-vs-powershell HELP_CMD branch, which
//	                         broke under sh-on-Windows); `##@ group`
//	                         headers supported, plain text (no ANSI)
//
// Exit codes: 0 success, 1 operation failure, 2 usage error.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// emit writes a line to w, discarding the write error: this is a build
// helper writing diagnostics to stderr, where a failed write is both
// impossible in practice and not actionable (satisfies errcheck without
// noise at every call site).
func emit(w io.Writer, a ...any) { _, _ = fmt.Fprintln(w, a...) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		emit(stderr, "usage: xplat <mkdir|rmrf|cp|require|help> ...")
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "mkdir":
		if len(rest) != 1 || rest[0] == "" {
			emit(stderr, "usage: xplat mkdir <dir>")
			return 2
		}
		if err := os.MkdirAll(rest[0], 0o755); err != nil {
			emit(stderr, "xplat mkdir:", err)
			return 1
		}
		return 0

	case "rmrf":
		if len(rest) == 0 {
			emit(stderr, "usage: xplat rmrf <path>...")
			return 2
		}
		for _, p := range rest {
			if err := os.RemoveAll(p); err != nil {
				emit(stderr, "xplat rmrf:", err)
				return 1
			}
		}
		return 0

	case "cp":
		if len(rest) != 2 || rest[0] == "" || rest[1] == "" {
			emit(stderr, "usage: xplat cp <src> <dst>")
			return 2
		}
		if err := copyFile(rest[0], rest[1]); err != nil {
			emit(stderr, "xplat cp:", err)
			return 1
		}
		return 0

	case "require":
		if len(rest) != 2 {
			emit(stderr, "usage: xplat require <path> <message>")
			return 2
		}
		if _, err := os.Stat(rest[0]); err != nil {
			emit(stderr, rest[1])
			return 1
		}
		return 0

	case "help":
		if len(rest) == 0 {
			emit(stderr, "usage: xplat help <makefile>...")
			return 2
		}
		return printHelp(stdout, stderr, rest)

	default:
		emit(stderr, "xplat: unknown subcommand", cmd)
		return 2
	}
}

// copyFile copies src to dst, creating dst's parent directory if needed and
// marking dst executable (0o755) so POSIX git hooks are runnable; on Windows
// the mode is largely cosmetic but harmless.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// targetRe matches a documented target line: "<name>: ... ## <description>".
// Name excludes '.' so .PHONY/.SUFFIXES lines are skipped, matching the
// historical awk pattern ^[a-zA-Z_-]+ (digits added for targets like
// build-core / migrate-staged).
var targetRe = regexp.MustCompile(`^([A-Za-z0-9_-]+):.*?##\s+(.+)$`)

// printHelp renders the make help screen from the given makefile(s),
// replacing the awk (Unix) / powershell (Windows) HELP_CMD branch that
// broke when $(OS)=Windows_NT picked the powershell form but recipes ran
// under sh. "##@ group" lines become section headers; documented targets
// are listed aligned. Plain text (no ANSI) for terminal-agnostic output.
func printHelp(stdout, stderr io.Writer, files []string) int {
	for _, mf := range files {
		b, err := os.ReadFile(mf)
		if err != nil {
			emit(stderr, "xplat help:", err)
			return 1
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimRight(line, "\r")
			if g, ok := strings.CutPrefix(line, "##@ "); ok {
				emit(stdout, "")
				emit(stdout, strings.TrimSpace(g))
				continue
			}
			if m := targetRe.FindStringSubmatch(line); m != nil {
				emit(stdout, fmt.Sprintf("  %-28s %s", m[1], strings.TrimSpace(m[2])))
			}
		}
	}
	return 0
}
