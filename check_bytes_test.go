package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestByteLimit_FailsWhileLinesPass(t *testing.T) {
	root := t.TempDir()
	// One huge line: 1 line (well inside 200) but 30,000 bytes (> 24,000).
	writeFile(t, root, "AGENTS.md", strings.Repeat("x", 30000)+"\n")
	res, err := repogov.CheckDir(root, []string{".md"}, repogov.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Status != repogov.Fail {
		t.Fatalf("want one FAIL, got %+v", res)
	}
	r := res[0]
	if r.Lines != 1 || r.Bytes != 30001 || r.ByteLimit != 24000 {
		t.Errorf("unexpected measurements: %+v", r)
	}
	if !strings.Contains(r.Action, "byte limit") {
		t.Errorf("action should explain the byte overage: %q", r.Action)
	}
}

func TestByteLimit_WarnsNearLimitAndPassesBelow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", strings.Repeat("x", 21000)+"\n") // 87% of 24000
	writeFile(t, root, "GEMINI.md", strings.Repeat("x", 1000)+"\n")  // fine
	res, _ := repogov.CheckDir(root, []string{".md"}, repogov.DefaultConfig())
	got := map[string]repogov.Status{}
	for _, r := range res {
		got[r.Path] = r.Status
	}
	if got["AGENTS.md"] != repogov.Warn || got["GEMINI.md"] != repogov.Pass {
		t.Errorf("statuses = %v", got)
	}
}

func TestByteLimit_LineExemptStillByteChecked(t *testing.T) {
	root := t.TempDir()
	cfg := repogov.DefaultConfig()
	cfg.Files["AGENTS.md"] = 0 // exempt from the line check only
	writeFile(t, root, "AGENTS.md", strings.Repeat("x", 30000)+"\n")
	res, _ := repogov.CheckDir(root, []string{".md"}, cfg)
	if len(res) != 1 || res[0].Status != repogov.Fail {
		t.Fatalf("line-exempt file must still fail the byte limit: %+v", res)
	}
	// Within the byte limit and line-exempt: skipped.
	writeFile(t, root, "AGENTS.md", "small\n")
	res, _ = repogov.CheckDir(root, []string{".md"}, cfg)
	if res[0].Status != repogov.Skip {
		t.Errorf("want SKIP, got %v", res[0].Status)
	}
}

func TestByteLimit_ConfigRoundTripAndValidation(t *testing.T) {
	// A config without max_bytes inherits the defaults.
	p := filepath.Join(t.TempDir(), "repogov-config.json")
	if err := os.WriteFile(p, []byte(`{"default": 300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _ := repogov.LoadConfig(p)
	if repogov.ResolveByteLimit("AGENTS.md", cfg) != 24000 || repogov.ResolveByteLimit(".agents/rules/a.md", cfg) != 24000 {
		t.Errorf("defaults not inherited: %+v", cfg.MaxBytes)
	}
	if repogov.ResolveByteLimit("README.md", cfg) != 0 {
		t.Error("unmatched file must have no byte limit")
	}
	// Explicit empty list disables byte limits.
	if err := os.WriteFile(p, []byte(`{"max_bytes": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _ = repogov.LoadConfig(p)
	if repogov.ResolveByteLimit("AGENTS.md", cfg) != 0 {
		t.Error("explicit empty max_bytes should disable byte limits")
	}
	// YAML round trip.
	y := filepath.Join(t.TempDir(), "c.yaml")
	cfg = repogov.DefaultConfig()
	if err := repogov.SaveConfig(y, cfg); err != nil {
		t.Fatal(err)
	}
	back, err := repogov.LoadConfig(y)
	if err != nil || repogov.ResolveByteLimit("GEMINI.md", back) != 24000 {
		t.Fatalf("YAML round trip lost max_bytes: %v %+v", err, back.MaxBytes)
	}
	// Validation.
	bad := repogov.Config{Default: 500, MaxBytes: []repogov.Rule{{Glob: ""}, {Glob: "[", Limit: repogov.RuleLimit(1)}, {Glob: "a", Limit: repogov.RuleLimit(-1)}}}
	if n := len(repogov.ValidateConfig(bad)); n != 3 {
		t.Errorf("want 3 violations, got %d", n)
	}
}

func TestByteLimit_GeneratedConfigIncludesMaxBytes(t *testing.T) {
	root := t.TempDir()
	if _, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".github", "repogov-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := repogov.LoadConfig(filepath.Join(root, ".github", "repogov-config.json"))
	if err != nil || !strings.Contains(string(data), `"max_bytes"`) {
		t.Fatalf("generated config lacks max_bytes: %v\n%s", err, data)
	}
	if repogov.ResolveByteLimit("AGENTS.md", cfg) != 24000 {
		t.Error("cross-agent AGENTS.md byte rule should be generated")
	}
}
