package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestGeminiLayout_AgentsMdSatisfiesRequirement(t *testing.T) {
	// Neither file: still a FAIL.
	root := t.TempDir()
	rs, err := repogov.CheckLayout(root, repogov.DefaultGeminiLayout())
	if err != nil {
		t.Fatal(err)
	}
	if repogov.LayoutPassed(rs) {
		t.Fatalf("no GEMINI.md and no AGENTS.md must fail: %+v", rs)
	}

	// AGENTS.md only, with Gemini CLI configured to read it: passes, with
	// the reason shown.
	writeFile(t, root, "AGENTS.md", "# a\n")
	writeFile(t, root, ".gemini/settings.json", `{"context": {"fileName": ["AGENTS.md", "GEMINI.md"]}}`)
	rs, _ = repogov.CheckLayout(root, repogov.DefaultGeminiLayout())
	if !repogov.LayoutPassed(rs) || len(rs) != 1 || !strings.Contains(rs[0].Message, "satisfied by AGENTS.md") {
		t.Fatalf("AGENTS.md should satisfy GEMINI.md: %+v", rs)
	}

	// GEMINI.md present: the normal "required file present" path.
	writeFile(t, root, "GEMINI.md", "# g\n")
	rs, _ = repogov.CheckLayout(root, repogov.DefaultGeminiLayout())
	if len(rs) != 1 || rs[0].Message != "required file present" {
		t.Fatalf("unexpected result: %+v", rs)
	}
}

// Gemini CLI reads AGENTS.md only when .gemini/settings.json lists it in
// context.fileName (a string or a list); without that, or with a settings
// file that cannot be used, AGENTS.md does not stand in for GEMINI.md and
// the failure says why.
func TestGeminiLayout_AgentsMdNeedsContextFileName(t *testing.T) {
	cases := []struct {
		name, settings string // settings "" = no .gemini/settings.json
		pass           bool
		why            string
	}{
		{"no settings", "", false, "no .gemini/"},
		{"string", `{"context": {"fileName": "AGENTS.md"}}`, true, ""},
		{"list", "\ufeff" + `{"context": {"fileName": ["CONTEXT.md", "AGENTS.md"]}}`, true, ""},
		{"not listed", `{"context": {"fileName": ["GEMINI.md"]}}`, false, "does not list it"},
		{"unset", `{"ui": {"theme": "x"}}`, false, "is not set"},
		{"case differs", `{"context": {"fileName": "agents.md"}}`, false, "does not list it"},
		{"wrong type", `{"context": {"fileName": 3}}`, false, "neither a string nor a list"},
		{"malformed", `{"context": {"fileName": ["AGENTS.md"]`, false, "not valid JSON"},
		{"comments", "// c\n{\"context\": {\"fileName\": \"AGENTS.md\"}}", false, "not valid JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "AGENTS.md", "# a\n")
			if tc.settings != "" {
				writeFile(t, root, ".gemini/settings.json", tc.settings)
			}
			rs, err := repogov.CheckLayout(root, repogov.DefaultGeminiLayout())
			if err != nil {
				t.Fatal(err)
			}
			if got := repogov.LayoutPassed(rs); got != tc.pass {
				t.Fatalf("pass = %v, want %v: %+v", got, tc.pass, rs)
			}
			if !tc.pass && (!strings.Contains(rs[0].Message, tc.why) || !strings.Contains(rs[0].Message, "context.fileName")) {
				t.Errorf("failure must say why (%q): %q", tc.why, rs[0].Message)
			}
		})
	}
}

// An alternative satisfies a requirement only as a regular file: a
// directory (Cline's .clinerules/ directory, an AGENTS.md directory) or a
// symlink with the alternative's name is not a rules file the agent reads.
func TestLayoutAlternative_RequiresRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".clinerules/a.md", "# cline\n")
	if rs, _ := repogov.CheckLayout(root, repogov.DefaultZedLayout()); repogov.LayoutPassed(rs) {
		t.Errorf("a .clinerules/ directory must not satisfy Zed's .rules: %+v", rs)
	}

	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".gemini/settings.json", `{"context": {"fileName": "AGENTS.md"}}`)
	if rs, _ := repogov.CheckLayout(root, repogov.DefaultGeminiLayout()); repogov.LayoutPassed(rs) {
		t.Errorf("an AGENTS.md directory must not satisfy GEMINI.md: %+v", rs)
	}

	root = t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(target, []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if rs, _ := repogov.CheckLayout(root, repogov.DefaultZedLayout()); repogov.LayoutPassed(rs) {
		t.Errorf("a CLAUDE.md symlink must not satisfy Zed's .rules: %+v", rs)
	}
}

func TestInitGemini_StillCreatesGeminiMd(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", "# a\n")
	created, err := repogov.InitLayout(root, repogov.DefaultGeminiLayout())
	// The unmarked AGENTS.md is a notice, not an error: init completes.
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range created {
		if c == "GEMINI.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("init must still create GEMINI.md even when AGENTS.md exists: %v", created)
	}
}

func TestRootLayout_AgentConfigIsNotUnexpected(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "opencode.json", "{}")
	writeFile(t, root, "opencode.jsonc", "{}")
	writeFile(t, root, ".opencode/agents/a.md", "x")
	writeFile(t, root, ".gemini/settings.json", "{}")
	writeFile(t, root, ".agents/rules/r.md", "x")
	writeFile(t, root, ".codex/config.toml", "x")
	writeFile(t, root, ".claude/rules/a.md", "x")
	writeFile(t, root, ".github/workflows/ci.yml", "x")
	rs, err := repogov.CheckLayout(root, repogov.DefaultRootLayout())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		if r.Status == repogov.Warn || r.Status == repogov.Fail {
			t.Errorf("agent config must not be flagged: %+v", r)
		}
	}
}

func TestRootLayout_UnmanagedNestedFileStillFlagged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "random/deep/file.md", "x")
	rs, _ := repogov.CheckLayout(root, repogov.DefaultRootLayout())
	flagged := false
	for _, r := range rs {
		if r.Path == "random/deep/file.md" && r.Status == repogov.Warn {
			flagged = true
		}
	}
	if !flagged {
		t.Errorf("files outside managed dirs must still be flagged: %+v", rs)
	}
}

func TestZedLayout_AnyDocumentedRulesFileSatisfies(t *testing.T) {
	for _, alt := range []string{".cursorrules", "AGENTS.md", "CLAUDE.md", "AGENT.md", ".github/copilot-instructions.md"} {
		root := t.TempDir()
		writeFile(t, root, alt, "x")
		rs, _ := repogov.CheckLayout(root, repogov.DefaultZedLayout())
		if !repogov.LayoutPassed(rs) {
			t.Errorf("%s should satisfy the Zed requirement: %+v", alt, rs)
		}
	}
	rs, _ := repogov.CheckLayout(t.TempDir(), repogov.DefaultZedLayout())
	if repogov.LayoutPassed(rs) {
		t.Error("no rules file at all must still fail")
	}
}
