package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -------------------------------------------------------------------
// Error paths: missing or unknown agent
// -------------------------------------------------------------------

func TestScaffold_Init_NoAgent(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "", "", true, false, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-agent") {
		t.Errorf("expected -agent hint in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Init_UnknownAgent(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "notion", "", true, false, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Errorf("expected 'unknown agent' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Init_OutputReportsCreatedPaths(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit: exit %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	// Every reported path must actually exist.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+ ") {
			rel := strings.TrimPrefix(line, "+ ")
			abs := filepath.Join(root, filepath.FromSlash(rel))
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				t.Errorf("reported created path %q does not exist", rel)
			}
		}
	}
}

func TestScaffold_Init_JSON_ReportsCreatedPaths(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", "", false, true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit JSON: exit %d, stderr: %s", code, stderr.String())
	}
	var result struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout.String())
	}
	if result.Platform != "copilot" {
		t.Errorf("expected platform=copilot, got %q", result.Platform)
	}
	if len(result.Created) == 0 {
		t.Error("expected non-empty created list in JSON output")
	}
	// Every JSON-reported path must actually exist on disk.
	for _, rel := range result.Created {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			t.Errorf("JSON reported created %q but file does not exist", rel)
		}
	}
}

// -------------------------------------------------------------------
// limits subcommand
// -------------------------------------------------------------------

func TestScaffold_Limits_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	// Init first so a valid config and .md files are present.
	if code := runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit: exit %d", code)
	}
	stdout.Reset()
	// Scan .md and .mdc files; all generated files must be within limits.
	if code := runLimits(root, "", ".md,.mdc", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLimits after init: exit %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "[PASS]") {
		t.Errorf("expected [PASS] in limits output, got: %s", stdout.String())
	}
}

func TestScaffold_Limits_ExceedLimit(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.md":                      nlines(500),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (limit exceeded), got %d", code)
	}
}

func TestScaffold_Limits_WarnBeforeExceed(t *testing.T) {
	// File at 90% of the warning-threshold limit should produce a WARN, not FAIL.
	root := writeTempDir(t, map[string]string{
		"near.md":                     nlines(90),
		".github/repogov-config.json": `{"default": 100, "warning_threshold": "80%"}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0 (warning only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARN") {
		t.Errorf("expected WARN in output, got: %s", stdout.String())
	}
}

func TestScaffold_Limits_NoConfig_UsesDefaults(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"tiny.md": nlines(5),
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestScaffold_Limits_AllSentinelIncludesGo(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.go":                      nlines(500),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	// "all" sentinel removes extension filter so .go exceeds limit.
	if code := runLimits(root, "", "all", false, true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (all sentinel includes big.go), got %d", code)
	}
}

func TestScaffold_Limits_BadConfigJSON(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": "not-a-number"}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (bad config), got %d", code)
	}
}

func TestScaffold_Limits_JSONOutput(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []interface{}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

// -------------------------------------------------------------------
// layout subcommand
// -------------------------------------------------------------------

func TestScaffold_Layout_PassAfterInit(t *testing.T) {
	tests := []struct {
		agent    string
		platform string
	}{
		{"copilot", "copilot"},
		{"cursor", "cursor"},
		{"windsurf", "windsurf"},
		{"claude", "claude"},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()
			if code := runInit(root, "", tc.agent, "", true, false, false, false, stdout, stderr); code != 0 {
				t.Fatalf("runInit %s: exit %d", tc.agent, code)
			}
			if code := runLayout(root, "", tc.platform, "", true, false, stdout, stderr); code != 0 {
				t.Fatalf("runLayout %s after init: exit %d", tc.platform, code)
			}
		})
	}
}

func TestScaffold_Layout_MissingCopilotInstructions(t *testing.T) {
	// Init copilot, then delete the required file.
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)

	required := filepath.Join(root, ".github", "copilot-instructions.md")
	if err := os.Remove(required); err != nil {
		t.Fatalf("cannot remove required file: %v", err)
	}

	if code := runLayout(root, "", "copilot", "", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected layout to fail without copilot-instructions.md, got exit %d", code)
	}
}

func TestScaffold_Layout_MissingRootDir(t *testing.T) {
	// Fresh empty directory: every platform dir is absent.
	tests := []string{"copilot", "cursor", "windsurf", "claude"}
	for _, platform := range tests {
		t.Run(platform, func(t *testing.T) {
			stdout, stderr := bufs()
			if code := runLayout(t.TempDir(), "", platform, "", true, false, stdout, stderr); code != 1 {
				t.Fatalf("expected exit 1 (missing layout dir), got %d", code)
			}
		})
	}
}

func TestScaffold_Layout_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "", "jira", "", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Errorf("expected 'unknown agent' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Layout_All_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "all", "", true, false, false, false, stdout, stderr)

	if code := runLayout(root, "", "all", "", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected layout all to pass after init, got exit %d\nstderr: %s", code, stderr.String())
	}
}

func TestScaffold_Layout_JSONOutput(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)

	stdout.Reset()
	if code := runLayout(root, "", "copilot", "", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

// -------------------------------------------------------------------
// validate subcommand
// -------------------------------------------------------------------

func TestScaffold_Validate_ValidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": "80%"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("expected 'is valid' in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_InvalidNegativeDefault(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (negative default), got %d", code)
	}
	if !strings.Contains(stdout.String(), "[FAIL]") {
		t.Errorf("expected [FAIL] in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_InvalidThresholdOver100(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": "200%"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (threshold > 100), got %d", code)
	}
}

func TestScaffold_Validate_MissingConfig(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (no config), got %d", code)
	}
	if !strings.Contains(stderr.String(), "No config file found") {
		t.Errorf("expected 'No config file found' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Validate_MalformedJSON(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{broken json`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (malformed JSON), got %d", code)
	}
}

func TestScaffold_Validate_BackslashPathWarning(t *testing.T) {
	// A backslash in a file key is a warning-level violation; exit must be 0.
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": "{\"default\": 300, \"files\": {\"a\\\\b.md\": 100}}",
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0 (warning only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARNING") {
		t.Errorf("expected WARNING in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_JSONOutput_Valid(t *testing.T) {
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
		t.Error("expected valid=true in JSON output")
	}
}

func TestScaffold_Validate_JSONOutput_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -5}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var result struct {
		Valid      bool        `json:"valid"`
		Violations interface{} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false in JSON output")
	}
}

func TestScaffold_Validate_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)

	stdout.Reset()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected validate to pass on init-generated config, exit %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("expected 'is valid' in output, got: %s", stdout.String())
	}
}

// -------------------------------------------------------------------
// run() dispatcher integration
// -------------------------------------------------------------------

func TestScaffold_Run_EachSubcommand(t *testing.T) {
	root := t.TempDir()
	// Pre-init so all subcommands have a config and layout dirs.
	bufs1, bufs2 := bufs()
	runInit(root, "", "all", "", true, false, false, false, bufs1, bufs2)

	tests := []struct {
		args     []string
		wantCode int
		desc     string
	}{
		{
			args:     []string{"-root", root, "-agent", "copilot", "-quiet", "init"},
			wantCode: 0,
			desc:     "init copilot (already exists)",
		},
		{
			args:     []string{"-root", root, "-agent", "all", "-quiet", "layout"},
			wantCode: 0,
			desc:     "layout all after init",
		},
		{
			args:     []string{"-root", root, "-quiet", "limits"},
			wantCode: 0,
			desc:     "limits with default config",
		},
		{
			args:     []string{"-root", root, "-quiet", "validate"},
			wantCode: 0,
			desc:     "validate with init-generated config",
		},
		{
			args:     []string{"-root", root, "-agent", "all", "-quiet"},
			wantCode: 0,
			desc:     "default (all) subcommand",
		},
		{
			args:     []string{"version"},
			wantCode: 0,
			desc:     "version prints",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			stdout, stderr := bufs()
			if code := run(tc.args, stdout, stderr); code != tc.wantCode {
				t.Errorf("%s: expected exit %d, got %d\nstdout: %s\nstderr: %s",
					tc.desc, tc.wantCode, code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestScaffold_Run_UnknownSubcommand(t *testing.T) {
	stdout, stderr := bufs()
	if code := run([]string{"bogus-command"}, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Run_NoArgsShowsHelp(t *testing.T) {
	stdout, stderr := bufs()
	// No -root provided: run uses cwd; config likely absent so limits might
	// return 0 or 1, but the process must not panic.
	run([]string{"-h"}, stdout, stderr)
	// Help exits 0 or 2 depending on flag implementation -- just confirm
	// it outputs usage text.
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "repogov") {
		t.Errorf("expected repogov in help output, got: %s", combined)
	}
}

// -------------------------------------------------------------------
// AGENTS.md context section per-agent accuracy
// -------------------------------------------------------------------

func TestScaffold_AgentsMd_ContextLinks(t *testing.T) {
	tests := []struct {
		agent       string
		mustHave    []string
		mustNotHave []string
	}{
		{
			agent:       "copilot",
			mustHave:    []string{"README.md", "docs/", ".github/rules/", "copilot-instructions.md"},
			mustNotHave: []string{".cursor/", ".windsurf/", ".claude/"},
		},
		{
			agent:       "cursor",
			mustHave:    []string{"README.md", "docs/", ".cursor/rules/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".windsurf/", ".claude/"},
		},
		{
			agent:       "windsurf",
			mustHave:    []string{"README.md", "docs/", ".windsurf/rules/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".cursor/", ".claude/"},
		},
		{
			agent:       "claude",
			mustHave:    []string{"README.md", "docs/", ".claude/rules/", ".claude/agents/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".cursor/", ".windsurf/"},
		},
		{
			agent:    "kiro",
			mustHave: []string{"README.md", "docs/", ".kiro/steering/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "gemini",
			mustHave: []string{"README.md", "docs/", "GEMINI.md"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "continue",
			mustHave: []string{"README.md", "docs/", ".continue/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "cline",
			mustHave: []string{"README.md", "docs/", ".clinerules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "roocode",
			mustHave: []string{"README.md", "docs/", ".roo/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "jetbrains",
			mustHave: []string{"README.md", "docs/", ".aiassistant/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "zed",
			mustHave: []string{"README.md", "docs/", ".rules"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/", "GEMINI.md",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()
			runInit(root, "", tc.agent, "", true, false, false, false, stdout, stderr)

			agPath := filepath.Join(root, "AGENTS.md")
			for _, link := range tc.mustHave {
				assertFileContains(t, agPath, link)
			}
			for _, link := range tc.mustNotHave {
				assertFileNotContains(t, agPath, link)
			}
		})
	}
}
