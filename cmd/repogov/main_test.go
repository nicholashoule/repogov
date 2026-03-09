package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// writeTempDir creates a temporary directory tree from the given file map.
func writeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// nlines returns n newline-terminated lines.
func nlines(n int) string { return strings.Repeat("line\n", n) }

// bufs returns a fresh pair of output/error buffers.
func bufs() (*bytes.Buffer, *bytes.Buffer) {
	return &bytes.Buffer{}, &bytes.Buffer{}
}

// --- run() dispatcher covers main() ---

func TestRun_Version(t *testing.T) {
	stdout, stderr := bufs()
	code := run([]string{"version"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "repogov") {
		t.Fatalf("expected version in output, got: %s", stdout.String())
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	stdout, stderr := bufs()
	code := run([]string{"bogus"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown subcommand") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRun_BadFlag(t *testing.T) {
	stdout, stderr := bufs()
	code := run([]string{"-notaflag"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRun_DefaultAll(t *testing.T) {
	root := t.TempDir()
	// Pre-init github so limits has a config and layout has its dir.
	runInit(root, "", "copilot", true, false, false, new(bytes.Buffer), new(bytes.Buffer))
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-agent", "copilot", "-quiet"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_Limits(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-exts", ".md", "-quiet", "limits"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_Layout(t *testing.T) {
	root := t.TempDir()
	// Pre-init all platforms so every layout check has its dir.
	runInit(root, "", "all", true, false, false, new(bytes.Buffer), new(bytes.Buffer))
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-agent", "all", "-quiet", "layout"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_Init(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-agent", "all", "-quiet", "init"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_Validate(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-quiet", "validate"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_All_Fail(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.md":                      nlines(400),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	code := run([]string{"-root", root, "-quiet", "all"}, stdout, stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
}

// --- runLimits tests ---

func TestRunLimits_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"hello.md":                    nlines(10),
		".github/repogov-config.json": `{"default": 300, "warning_threshold": 80, "skip_dirs": [".git"]}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLimits_Fail(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.md":                      nlines(400),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunLimits_ExtWithoutDot(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", "md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLimits_BadConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": "not-a-number"}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunLimits_NoConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{"a.md": nlines(5)})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLimits_CustomConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":        nlines(5),
		"custom.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, filepath.Join(root, "custom.json"), ".md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLimits_JSON_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []repogov.Result
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

func TestRunLimits_JSON_Fail(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.md":                      nlines(400),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var results []repogov.Result
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestRunLimits_Verbose(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "[PASS]") {
		t.Fatalf("expected PASS, got: %s", stdout.String())
	}
}

// --- runLayout tests ---

func TestRunLayout_MissingDir(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "copilot", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunLayout_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\non: push\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "copilot", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "bitbucket", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRunLayout_JSON_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\non: push\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "copilot", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []repogov.LayoutResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

func TestRunLayout_JSON_Fail(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "copilot", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var results []repogov.LayoutResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestRunLayout_GitLab(t *testing.T) {
	// .gitlab/ exists with optional CODEOWNERS and template dirs.
	root := writeTempDir(t, map[string]string{
		".gitlab/issue_templates/.gitkeep":         "",
		".gitlab/merge_request_templates/.gitkeep": "",
		".gitlab/CODEOWNERS":                       "* @team\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "gitlab", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_Cursor(t *testing.T) {
	root := writeTempDir(t, map[string]string{".cursor/rules/.gitkeep": ""})
	stdout, stderr := bufs()
	if code := runLayout(root, "cursor", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_Windsurf(t *testing.T) {
	root := writeTempDir(t, map[string]string{".windsurf/rules/.gitkeep": ""})
	stdout, stderr := bufs()
	if code := runLayout(root, "windsurf", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_Claude(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/CLAUDE.md":       "# CLAUDE.md\n",
		".claude/rules/.gitkeep":  "",
		".claude/agents/.gitkeep": "",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "claude", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_Root_Pass(t *testing.T) {
	// A typical repo root containing only recognized optional files and a
	// managed subdirectory should pass with no unexpected-file warnings.
	root := writeTempDir(t, map[string]string{
		"README.md":     "# Project\n",
		"LICENSE":       "MIT\n",
		".gitignore":    "*.o\n",
		"docs/index.md": "# Docs\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "root", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_Root_NamingViolation(t *testing.T) {
	// A root-level file that violates the lowercase naming convention must
	// cause a Fail result and a non-zero exit code.
	root := writeTempDir(t, map[string]string{
		"README.md":   "# Project\n",
		"MyScript.sh": "#!/bin/sh\n", // mixed-case: violates "lowercase" naming rule
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "root", true, false, stdout, stderr); code == 0 {
		t.Fatalf("expected non-zero exit for naming violation (stdout: %s stderr: %s)", stdout.String(), stderr.String())
	}
}

func TestRunLayout_Root_GitDirNotFlagged(t *testing.T) {
	// .git as a directory (normal clone) must never appear as an unexpected entry.
	root := writeTempDir(t, map[string]string{
		"README.md": "# Project\n",
		".git/HEAD": "ref: refs/heads/main\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "root", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_Root_GitFileNotFlagged(t *testing.T) {
	// .git as a plain file (Git worktree gitdir pointer) must not be flagged
	// as an unexpected file by the root layout checker.
	root := writeTempDir(t, map[string]string{
		"README.md": "# Project\n",
		".git":      "gitdir: ../.git/worktrees/feature\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "root", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_All_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\n",
		".github/copilot-instructions.md": "# Copilot\n",
		".cursor/rules/.gitkeep":          "",
		".windsurf/rules/.gitkeep":        "",
		".claude/CLAUDE.md":               "# CLAUDE.md\n",
		".claude/rules/.gitkeep":          "",
		".claude/agents/.gitkeep":         "",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "all", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_All_JSON(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\n",
		".github/copilot-instructions.md": "# Copilot\n",
		".cursor/rules/.gitkeep":          "",
		".windsurf/rules/.gitkeep":        "",
		".claude/CLAUDE.md":               "# CLAUDE.md\n",
		".claude/rules/.gitkeep":          "",
		".claude/agents/.gitkeep":         "",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "all", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	for _, p := range []string{"copilot", "cursor", "windsurf", "claude"} {
		if _, ok := result[p]; !ok {
			t.Errorf("expected key %q in JSON output", p)
		}
	}
}

func TestRunLayout_All_SkipsAbsentPlatforms(t *testing.T) {
	// A repo that only has .github/ should pass for -agent all because
	// platforms whose root directories are absent are silently skipped.
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "all", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_All_SkipsAbsentPlatforms_JSON(t *testing.T) {
	// JSON output should only include platforms whose root directories are
	// present; absent platforms must not appear as keys at all.
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "all", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if _, ok := result["copilot"]; !ok {
		t.Error("expected key \"copilot\" in JSON output (platform root present)")
	}
	for _, absent := range []string{"cursor", "windsurf", "claude", "gitlab"} {
		if _, ok := result[absent]; ok {
			t.Errorf("unexpected key %q in JSON output (platform root absent, should be skipped)", absent)
		}
	}
}

func TestRunLayout_Verbose(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\non: push\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "copilot", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Layout:") {
		t.Fatalf("expected Layout summary, got: %s", stdout.String())
	}
}

// --- runInit tests ---

func TestRunInit_RequiresPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "", true, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 when no platform given, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-agent is required") {
		t.Fatalf("expected usage error, got: %s", stderr.String())
	}
}

// TestRunInit_EachPlatformToTemp inits each platform into an isolated temp
// directory (named "temp" within the test root) and then validates the layout.
func TestRunInit_EachPlatformToTemp(t *testing.T) {
	tests := []struct {
		platform string
		dir      string
	}{
		{"copilot", ".github"},
		{"cursor", ".cursor"},
		{"windsurf", ".windsurf"},
		{"claude", ".claude"},
	}
	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			temp := filepath.Join(t.TempDir(), "temp")
			if err := os.MkdirAll(temp, 0o755); err != nil {
				t.Fatal(err)
			}
			stdout, stderr := bufs()

			// Init the platform layout.
			if code := runInit(temp, "", tc.platform, true, false, false, stdout, stderr); code != 0 {
				t.Fatalf("runInit %s: exit %d (stderr: %s)", tc.platform, code, stderr.String())
			}

			// Verify the platform root directory was created.
			if _, err := os.Stat(filepath.Join(temp, tc.dir)); err != nil {
				t.Fatalf("%s: %s not created: %v", tc.platform, tc.dir, err)
			}

			// Validate the layout passes.
			if code := runLayout(temp, tc.platform, false, false, stdout, stderr); code != 0 {
				t.Fatalf("runLayout %s: exit %d (stdout: %s)", tc.platform, code, stdout.String())
			}
		})
	}
}

// TestRunInit_AllToTemp inits all platforms into a single temp directory,
// validates all layouts, and confirms idempotency.
func TestRunInit_AllToTemp(t *testing.T) {
	temp := filepath.Join(t.TempDir(), "temp")
	if err := os.MkdirAll(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := bufs()

	// First init: create all platforms.
	if code := runInit(temp, "", "all", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit all: exit %d (stderr: %s)", code, stderr.String())
	}

	// Validate all layouts pass.
	if code := runLayout(temp, "all", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout all: exit %d (stdout: %s)", code, stdout.String())
	}

	// Second init: must be idempotent (nothing new created).
	stdout.Reset()
	if code := runInit(temp, "", "all", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("second runInit all: exit %d", code)
	}
	if strings.Contains(stdout.String(), "Scaffolded") {
		t.Fatalf("second init should have created nothing; got: %s", stdout.String())
	}
}

func TestRunInit_CreatesLayout(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err != nil {
		t.Fatalf(".github not created: %v", err)
	}
	// On a fresh init, rules/ is created (instructions/ only when pre-existing).
	if _, err := os.Stat(filepath.Join(root, ".github", "rules")); err != nil {
		t.Fatalf(".github/rules not created: %v", err)
	}
}

func TestRunInit_GitLab(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "gitlab", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".gitlab")); err != nil {
		t.Fatalf(".gitlab not created: %v", err)
	}
}

func TestRunInit_Cursor(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "cursor", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "rules")); err != nil {
		t.Fatalf(".cursor/rules not created: %v", err)
	}
}

func TestRunInit_Windsurf(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "windsurf", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".windsurf", "rules")); err != nil {
		t.Fatalf(".windsurf/rules not created: %v", err)
	}
}

func TestRunInit_Claude(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "claude", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	for _, dir := range []string{".claude/rules", ".claude/agents"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil {
			t.Fatalf("%s not created: %v", dir, err)
		}
	}
}

func TestRunInit_All(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "all", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
	for _, dir := range []string{".github", ".cursor/rules", ".windsurf/rules", ".claude/rules", ".claude/agents"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil {
			t.Fatalf("%s not created: %v", dir, err)
		}
	}
}

func TestRunInit_All_JSON(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "all", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 combined entry for all platforms, got %d", len(results))
	}
	if results[0].Platform != "all" {
		t.Errorf("expected platform \"all\", got %q", results[0].Platform)
	}
	if len(results[0].Created) == 0 {
		t.Error("expected non-empty created list")
	}
}

func TestRunInit_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunInit_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "bitbucket", true, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRunInit_JSON(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Platform != "copilot" {
		t.Fatalf("expected github, got %s", result.Platform)
	}
	if len(result.Created) == 0 {
		t.Fatal("expected at least one created path")
	}
}

func TestRunInit_JSON_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Created []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Created) != 0 {
		t.Fatalf("expected 0 created, got %d", len(result.Created))
	}
}

func TestRunInit_Verbose(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Scaffolded") {
		t.Fatalf("expected 'Scaffolded', got: %s", stdout.String())
	}
}

func TestRunInit_Verbose_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "nothing to create") {
		t.Fatalf("expected 'nothing to create', got: %s", stdout.String())
	}
}

// --- runValidate tests ---

func TestRunValidate_ValidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": 80}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_InvalidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1, "warning_threshold": 200}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunValidate_NoConfig_Quiet(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunValidate_NoConfig_Verbose(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "No config file found") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRunValidate_BadConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": "not-a-number"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunValidate_JSON_Valid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.Valid {
		t.Fatal("expected valid=true")
	}
}

func TestRunValidate_JSON_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var result struct {
		Valid      bool                `json:"valid"`
		Violations []repogov.Violation `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Valid || len(result.Violations) == 0 {
		t.Fatal("expected violations")
	}
}

func TestRunValidate_WarningsOnly(t *testing.T) {
	// A backslash in a file path triggers a warning (severity="warning", not "error").
	// runValidate should print WARNING and return 0.
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": "{\"default\": 300, \"files\": {\"a\\\\b.md\": 100}}",
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0 (warnings only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARNING") {
		t.Fatalf("expected WARNING in output, got: %s", stdout.String())
	}
}

func TestRunValidate_CustomConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{"myconfig.json": `{"default": 300}`})
	stdout, stderr := bufs()
	if code := runValidate(root, filepath.Join(root, "myconfig.json"), true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_RelativeConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{"myconfig.json": `{"default": 300}`})
	stdout, stderr := bufs()
	if code := runValidate(root, "myconfig.json", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_Verbose_Valid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Fatalf("expected 'is valid', got: %s", stdout.String())
	}
}

func TestRunValidate_Verbose_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), "[FAIL]") {
		t.Fatalf("expected '[FAIL]', got: %s", stdout.String())
	}
}

// --- isAbsolute tests ---

func TestIsAbsolute(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", false},
		{"relative/path", false},
		{"file.json", false},
		{"/absolute/path", true},
		{"\\absolute\\path", true},
		{"C:\\Windows\\path", true},
		{"C:/unix/style", true},
		{"D:\\other", true}, //nolint:misspell // false positive: "ther" is a substring of "other", not a word
	}
	for _, tt := range tests {
		got := isAbsolute(tt.path)
		if got != tt.want {
			t.Errorf("isAbsolute(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFindGitRoot_Found(t *testing.T) {
	// Create a temp dir with a .git sub-directory and a nested sub-directory.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, repoRoot)
	}
}

func TestFindGitRoot_FoundAtSelf(t *testing.T) {
	// findGitRoot should return the directory itself when .git lives there.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(repoRoot)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", repoRoot, got, repoRoot)
	}
}

func TestFindGitRoot_GitFile(t *testing.T) {
	// .git as a plain file (Git worktree) should also be recognized.
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: ../.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, repoRoot)
	}
}

func TestFindGitRoot_NotFound(t *testing.T) {
	// A temp directory with no ancestor .git should return "".
	dir := t.TempDir()
	// Walk past the temp dir to a sub with no .git anywhere.
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	// We can't guarantee the test runner itself isn't inside a git repo, so
	// only verify that findGitRoot on a path with no .git returns some value
	// (may or may not be "" depending on the environment).  Instead, test the
	// behavior we care about: the returned path, if non-empty, must contain .git.
	got := findGitRoot(sub)
	if got != "" {
		if _, err := os.Stat(filepath.Join(got, ".git")); err != nil {
			t.Errorf("findGitRoot returned %q which has no .git: %v", got, err)
		}
	}
}

func TestResolveRoot_ExplicitPath(t *testing.T) {
	// When root is not ".", it should be returned unchanged.
	dir := t.TempDir()
	got := resolveRoot(dir)
	if got != dir {
		t.Errorf("resolveRoot(%q) = %q, want %q (explicit path should be unchanged)", dir, got, dir)
	}
}

func TestResolveRoot_AutoDetect(t *testing.T) {
	// When root is "." and the CWD is inside a git repo, resolveRoot should
	// return the git root.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	got := resolveRoot(".")
	// On macOS, t.TempDir() returns a symlinked path (/var/folders/…) while
	// os.Getwd() returns the real path (/private/var/folders/…). Use
	// os.SameFile for an inode-level comparison that is symlink-agnostic.
	gotInfo, err1 := os.Stat(got)
	wantInfo, err2 := os.Stat(repoRoot)
	if err1 != nil {
		t.Fatalf("resolveRoot returned unstat-able path %q: %v", got, err1)
	}
	if err2 != nil {
		t.Fatalf("stat repoRoot %q: %v", repoRoot, err2)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("resolveRoot(\".\") = %q, want same directory as %q", got, repoRoot)
	}
}

func TestResolveRoot_DotNonDefault(t *testing.T) {
	// An explicit non-"." path must not be processed by git-root detection.
	got := resolveRoot("/nonexistent/path")
	if got != "/nonexistent/path" {
		t.Errorf("resolveRoot returned %q, want /nonexistent/path", got)
	}
}

// TestRun_Init_NestedDir verifies that running init with root="." from a
// nested subdirectory (e.g. .github/) creates files in the git repo root,
// not in the subdirectory.
func TestRun_Init_NestedDir(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := bufs()
	// No -root flag: should auto-detect repoRoot via git root resolution.
	code := run([]string{"-agent", "copilot", "init"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}

	// Files must be created under repoRoot, not under nested.
	if _, err := os.Stat(filepath.Join(repoRoot, ".github", "copilot-instructions.md")); err != nil {
		t.Errorf(".github/copilot-instructions.md not found under repo root: %v", err)
	}
	// Nothing should have been created inside the nested directory itself
	// (other than the .github/instructions dir that we pre-created).
	entries, err := os.ReadDir(nested)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("unexpected files created inside nested dir %s: %v", nested, names)
	}
}
