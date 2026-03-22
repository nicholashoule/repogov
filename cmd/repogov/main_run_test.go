package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

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
	runInit(root, "", "copilot", "", true, false, false, false, new(bytes.Buffer), new(bytes.Buffer))
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
	runInit(root, "", "all", "", true, false, false, false, new(bytes.Buffer), new(bytes.Buffer))
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
	if code := runLayout(t.TempDir(), "", "copilot", "", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunLayout_Pass(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml":        "name: CI\non: push\n",
		".github/copilot-instructions.md": "# Copilot\n",
	})
	stdout, stderr := bufs()
	if code := runLayout(root, "", "copilot", "", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "", "bitbucket", "", true, false, stdout, stderr); code != 2 {
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
	if code := runLayout(root, "", "copilot", "", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results map[string][]repogov.LayoutResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

func TestRunLayout_JSON_Fail(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "", "copilot", "", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var results map[string][]repogov.LayoutResult
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
	if code := runLayout(root, "", "", "gitlab", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunLayout_Cursor(t *testing.T) {
	root := writeTempDir(t, map[string]string{".cursor/rules/.gitkeep": ""})
	stdout, stderr := bufs()
	if code := runLayout(root, "", "cursor", "", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLayout_Windsurf(t *testing.T) {
	root := writeTempDir(t, map[string]string{".windsurf/rules/.gitkeep": ""})
	stdout, stderr := bufs()
	if code := runLayout(root, "", "windsurf", "", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "claude", "", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "", "root", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "", "root", true, false, stdout, stderr); code == 0 {
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
	if code := runLayout(root, "", "", "root", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "", "root", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "all", "", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "all", "", false, true, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "all", "", true, false, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "all", "", false, true, stdout, stderr); code != 0 {
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
	if code := runLayout(root, "", "copilot", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Layout:") {
		t.Fatalf("expected Layout summary, got: %s", stdout.String())
	}
}
