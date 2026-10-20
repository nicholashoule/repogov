package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// Untrusted file names must not be able to forge log lines, ANSI escapes or
// GitHub Actions workflow commands in text output.
func TestSummary_NeutralizesControlCharacters(t *testing.T) {
	bidi := string(rune(0x202e))
	evil := "a.md\n::error::forged\x1b[31m" + bidi
	out := repogov.Summary([]repogov.Result{{Path: evil, Lines: 1, Limit: 10, Status: repogov.Pass}})
	lay := repogov.LayoutSummary([]repogov.LayoutResult{{Path: evil, Status: repogov.Warn, Message: "m\nsecond line"}})
	for name, s := range map[string]string{"Summary": out, "LayoutSummary": lay} {
		if strings.ContainsAny(s, "\x1b"+bidi) {
			t.Errorf("%s leaked an escape/bidi character: %q", name, s)
		}
		for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "::error::") {
				t.Errorf("%s allowed a forged workflow-command line: %q", name, line)
			}
		}
	}
}

// A line-exempt file that fails its byte limit must say why in the summary.
func TestSummary_ShowsByteFailureForLineExemptFile(t *testing.T) {
	root := t.TempDir()
	cfg := repogov.DefaultConfig()
	cfg.Files["AGENTS.md"] = 0
	writeFile(t, root, "AGENTS.md", strings.Repeat("x", 30000)+"\n")
	res, _ := repogov.CheckDir(root, []string{".md"}, cfg)
	if s := repogov.Summary(res); !strings.Contains(s, "byte limit") {
		t.Errorf("summary hides the reason for the FAIL:\n%s", s)
	}
}

// Rule files that are symlinks are reported, never opened.
func TestPitfalls_SymlinkedRuleFileNotRead(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(secret, []byte("---\ntrigger: TOP-SECRET-VALUE\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, ".agents", "rules", "link.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, r := range repogov.CheckAgentPitfalls(root) {
		if strings.Contains(r.Message, "TOP-SECRET-VALUE") {
			t.Fatalf("content of a symlink target leaked into a finding: %+v", r)
		}
	}
	r, ok := findResult(repogov.CheckAgentPitfalls(root), ".agents/rules/link.md")
	if !ok || r.Status != repogov.Warn {
		t.Errorf("symlinked rule file should produce a warning: %+v", r)
	}
}

// Dependency/build trees are not searched for rule directories, and the
// search is depth-bounded.
func TestPitfalls_SkipsHeavyAndDeepTrees(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "dist/.agents/rules/bad.md", "no frontmatter\n")
	writeFile(t, root, "target/x/.agents/rules/bad.md", "no frontmatter\n")
	deep := strings.Repeat("d/", 12)
	writeFile(t, root, deep+".agents/rules/bad.md", "no frontmatter\n")
	if rs := repogov.CheckAgentPitfalls(root); len(rs) != 0 {
		t.Errorf("skipped or too-deep trees must not be searched: %+v", rs)
	}
}

func TestUpgradeConfigHints_AtomicAndBounded(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "repogov-config.json")
	if err := os.WriteFile(p, []byte(`{"default": 300}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := repogov.UpgradeConfigHints(p); err != nil || !changed {
		t.Fatalf("upgrade: %v %v", changed, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
	if info, _ := os.Stat(p); info.Mode().Perm() != 0o600 && os.PathSeparator == '/' {
		t.Errorf("permissions not preserved: %v", info.Mode().Perm())
	}
	// Oversized configs are refused untouched.
	big := filepath.Join(dir, "big.json")
	if err := os.WriteFile(big, []byte("{"+strings.Repeat(" ", 5<<20)+"}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repogov.UpgradeConfigHints(big); err == nil {
		t.Error("a config over the size cap must be refused")
	}
}
