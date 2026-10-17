package repogov_test

import (
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

	// AGENTS.md only (Antigravity/OpenCode style): passes, with the reason shown.
	writeFile(t, root, "AGENTS.md", "# a\n")
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

func TestInitGemini_StillCreatesGeminiMd(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", "# a\n")
	created, err := repogov.InitLayout(root, repogov.DefaultGeminiLayout())
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
