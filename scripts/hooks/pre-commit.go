// pre-commit.go is the cross-platform implementation of the pre-commit Git
// hook. It is invoked by scripts/hooks/pre-commit via `go run` so the logic
// runs natively on Linux, macOS, and Windows without relying on POSIX
// shell utilities beyond the minimal shebang wrapper.
//
// Install once with:
//
//	make hooks
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	demojify "github.com/nicholashoule/demojify-sanitize"
)

func main() {
	// When invoked from the pre-commit shell wrapper the repo root is
	// passed as the first positional argument so the hook can chdir back
	// after `go run` resolves the module in scripts/hooks/.
	if len(os.Args) > 1 {
		if err := os.Chdir(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "cannot chdir to repo root %s: %v\n", os.Args[1], err)
			os.Exit(1)
		}
	}

	if !checkFmt() || !checkVet() || !checkDeps() || !checkMarkdownLimits() || !checkEmoji() {
		os.Exit(1)
	}
}

// checkFmt runs gofmt -s -l . to find unformatted files, then auto-fixes
// only those files with gofmt -s -w and re-stages them with git add.
// This mirrors what `make fmt` does, so the commit proceeds with clean formatting.
func checkFmt() bool {
	// 1. List files that need formatting.
	out, err := exec.Command("gofmt", "-s", "-l", ".").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FAIL] gofmt: %v\n", err)
		return false
	}
	files := strings.TrimSpace(string(out))
	if files == "" {
		fmt.Fprintln(os.Stderr, "[PASS] gofmt")
		return true
	}

	// 2. Collect the specific files that need formatting.
	var unformatted []string
	for _, f := range strings.Split(files, "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			unformatted = append(unformatted, f)
		}
	}

	// 3. Auto-fix formatting only on the files that need it.
	args := append([]string{"-s", "-w"}, unformatted...)
	fix := exec.Command("gofmt", args...)
	fix.Stderr = os.Stderr
	if err := fix.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[FAIL] gofmt -w: %v\n", err)
		return false
	}

	// 4. Re-stage the files that were reformatted.
	for _, f := range unformatted {
		add := exec.Command("git", "add", f)
		if err := add.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: could not re-stage %s: %v\n", f, err)
		}
	}

	fmt.Fprintln(os.Stderr, "[AUTO] gofmt: reformatted and re-staged the following files:")
	for _, f := range unformatted {
		fmt.Fprintf(os.Stderr, "  %s\n", f)
	}
	return true
}

// defaultMarkdownLimit is the fallback line limit for any .md file not covered
// by a built-in rule or a per-file override in .github/line-limits.json.
const defaultMarkdownLimit = 500

// mdLimitConfig is the structure of .github/line-limits.json.
//   - Default overrides the built-in defaultMarkdownLimit for all files.
//   - Files maps repo-relative forward-slash paths to per-file limits.
//     A limit of 0 exempts the file from checking entirely.
type mdLimitConfig struct {
	Default int            `json:"default"`
	Files   map[string]int `json:"files"`
}

// loadMDLimitConfig reads .github/line-limits.json if it exists.
// Missing file is not an error; all fields default to zero values.
func loadMDLimitConfig() (mdLimitConfig, error) {
	cfg := mdLimitConfig{Files: make(map[string]int)}
	data, err := os.ReadFile(filepath.Join(".github", "line-limits.json"))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("reading .github/line-limits.json: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing .github/line-limits.json: %w", err)
	}
	if cfg.Files == nil {
		cfg.Files = make(map[string]int)
	}
	return cfg, nil
}

// resolveLimit returns the effective line limit for a repo-relative path
// (forward slashes). Returns 0 when the file is exempt from checking.
// Priority: per-file override > built-in rule > config default > built-in default.
func resolveLimit(relPath string, cfg mdLimitConfig) int {
	// 1. Per-file override from .github/line-limits.json.
	if v, ok := cfg.Files[relPath]; ok {
		return v
	}
	// 2. Built-in rules (most specific first).
	if relPath == ".github/copilot-instructions.md" {
		return 50
	}
	// filepath.Match uses OS path separators, so convert both sides.
	if ok, _ := filepath.Match(
		filepath.Join(".github", "instructions", "*.md"),
		filepath.FromSlash(relPath),
	); ok {
		return 300
	}
	// 3. Config-level default, then built-in default.
	if cfg.Default > 0 {
		return cfg.Default
	}
	return defaultMarkdownLimit
}

// checkMarkdownLimits walks every .md file in the repo and enforces line-count
// limits according to the following priority:
//
//	.github/line-limits.json "files" entry  (per-file override; 0 = exempt)
//	Built-in rule for .github/instructions/*.md     (300 lines)
//	Built-in rule for .github/copilot-instructions.md (50 lines)
//	.github/line-limits.json "default" entry (global override)
//	defaultMarkdownLimit constant             (500 lines)
//
// Output levels:
//
//	[PASS]  within limit
//	[WARN]  >= 80% of limit -- approaching, consider refactoring
//	[FAIL]  over limit -- commit blocked
//	[SKIP]  exempt (limit = 0) or non-.md file
func checkMarkdownLimits() bool {
	cfg, err := loadMDLimitConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  WARNING: %v\n", err)
	}

	configNote := ""
	if cfg.Default > 0 {
		configNote = fmt.Sprintf(" (default overridden to %d via .github/line-limits.json)", cfg.Default)
	}
	fmt.Fprintf(os.Stderr, "check: markdown line-count limits%s\n", configNote)

	passed := true
	checked, skipped, failures := 0, 0, 0

	err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden/vendor directories.
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Normalise to forward-slash repo-relative path for config lookups.
		relPath := filepath.ToSlash(path)
		relPath = strings.TrimPrefix(relPath, "./")

		limit := resolveLimit(relPath, cfg)
		if limit == 0 {
			fmt.Fprintf(os.Stderr, "  [SKIP]  %s (exempt)\n", relPath)
			skipped++
			return nil
		}

		count, err := countLines(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  WARNING: could not read %s: %v\n", relPath, err)
			return nil
		}
		checked++
		pct := 100 * count / limit

		switch {
		case count > limit:
			fmt.Fprintf(os.Stderr, "  [FAIL]  %s -- %d/%d lines (%d%% of limit, %d over)\n",
				relPath, count, limit, pct, count-limit)
			fmt.Fprintf(os.Stderr, "          FIX: shorten content, or add an override to .github/line-limits.json\n")
			passed = false
			failures++
		case pct >= 80:
			fmt.Fprintf(os.Stderr, "  [WARN]  %s -- %d/%d lines (%d%% of limit, %d remaining)\n",
				relPath, count, limit, pct, limit-count)
		default:
			fmt.Fprintf(os.Stderr, "  [PASS]  %s -- %d/%d lines (%d%% of limit)\n",
				relPath, count, limit, pct)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  WARNING: error walking repo: %v\n", err)
	}

	switch {
	case checked == 0 && skipped == 0:
		fmt.Fprintln(os.Stderr, "  [SKIP]  no .md files found")
	case failures > 0:
		fmt.Fprintf(os.Stderr, "[FAIL] markdown limits: %d of %d file(s) over limit (%d exempt)\n",
			failures, checked, skipped)
	default:
		fmt.Fprintf(os.Stderr, "[PASS] markdown limits (%d file(s) checked, %d exempt)\n",
			checked, skipped)
	}
	return passed
}

// countLines returns the number of lines in the file at path.
func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// checkVet runs go vet ./... and reports any issues.
func checkVet() bool {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "[FAIL] go vet (run: make vet)")
		return false
	}
	fmt.Fprintln(os.Stderr, "[PASS] go vet")
	return true
}

// checkEmoji uses the demojify library (github.com/nicholashoule/demojify-sanitize)
// to scan the repository for emoji violations.
func checkEmoji() bool {
	cfg := demojify.DefaultScanConfig()
	cfg.Root = "."
	cfg.Options = demojify.Options{RemoveEmojis: true}

	findings, err := demojify.ScanDir(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FAIL] emoji check: %v\n", err)
		return false
	}
	if len(findings) == 0 {
		fmt.Fprintln(os.Stderr, "[PASS] no emoji violations")
		return true
	}
	fmt.Fprintf(os.Stderr, "[FAIL] emoji violations found in %d file(s):\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s\n", f.Path)
	}
	fmt.Fprintln(os.Stderr, "       FIX: run `demojify -root . -sub` to substitute emoji with text")
	return false
}

// checkDeps reads go.mod and fails if any "require" directive is found.
// This enforces the zero-dependency contract: external tools must be
// invoked via os/exec, never imported as Go library dependencies.
func checkDeps() bool {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FAIL] cannot read go.mod: %v\n", err)
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" || strings.HasPrefix(trimmed, "require ") {
			fmt.Fprintf(os.Stderr, "[FAIL] go.mod contains require directive -- zero-dependency contract violated\n")
			fmt.Fprintf(os.Stderr, "       Use os/exec to shell out to external tools; do not import them.\n")
			return false
		}
	}
	fmt.Fprintln(os.Stderr, "[PASS] zero dependencies")
	return true
}
