package repogov

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// downstreamConfig is a real-world config written before _agent_hints existed.
const downstreamConfig = `{
  "default": 500,
  "descriptive_names": true,
  "warning_threshold": "85%",
  "skip_dirs": [".git", "vendor", "workflows"],
  "include_exts": [".md", ".mdc"],
  "rules": [
    {"glob": ".github/rules/*.md", "limit": 300}
  ],
  "files": {
    ".github/copilot-instructions.md": 50,
    ".github/instructions/memory.instructions.md": 200,
    "AGENTS.md": 200,
    "README.md": 1200,
    "USAGE.md": 700,
    "docs/design.md": 700,
    "docs/console-template-layout.md": 700
  }
}
`

func loadFrom(t *testing.T, data string) Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "repogov-config.json")
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestUpgradeConfigHints_DownstreamConfig(t *testing.T) {
	// Supported as-is, before any upgrade.
	orig := loadFrom(t, downstreamConfig)
	if orig.Files["USAGE.md"] != 700 || !orig.DescriptiveNames {
		t.Fatalf("downstream config not loaded correctly: %+v", orig)
	}

	out, err := upgradeHintsInJSON([]byte(downstreamConfig))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"_agent_hints"`) {
		t.Fatal("hints not added")
	}
	// Everything after the inserted block is byte-identical to the original.
	rest := downstreamConfig[strings.Index(downstreamConfig, "\n")+1:]
	if !strings.HasSuffix(s, rest) {
		t.Errorf("original content was not preserved verbatim:\n%s", s)
	}
	if got := loadFrom(t, s); !reflect.DeepEqual(got, orig) {
		t.Errorf("effective config changed after upgrade")
	}
	// Valid JSON and equal to what the generator would emit for the hints.
	var gen, up map[string]json.RawMessage
	_ = json.Unmarshal([]byte(defaultConfigJSON(DefaultConfig())), &gen)
	_ = json.Unmarshal(out, &up)
	var a, b any
	_ = json.Unmarshal(gen[agentHintsKey], &a)
	_ = json.Unmarshal(up[agentHintsKey], &b)
	if !reflect.DeepEqual(a, b) {
		t.Error("upgraded hints differ from generator hints")
	}
}

func TestUpgradeConfigHints_IdempotentAndRefresh(t *testing.T) {
	first, err := upgradeHintsInJSON([]byte(downstreamConfig))
	if err != nil {
		t.Fatal(err)
	}
	second, err := upgradeHintsInJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("second upgrade changed the file; must be idempotent")
	}

	// A stale hints block (older wording, in the middle of the file) is refreshed.
	stale := strings.Replace(downstreamConfig, `"default": 500,`,
		`"_agent_hints": {"about": "old"},`+"\n  "+`"default": 500,`, 1)
	refreshed, err := upgradeHintsInJSON([]byte(stale))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(refreshed, []byte(`"old"`)) || !bytes.Contains(refreshed, []byte("on_limit_fail")) {
		t.Errorf("stale hints not refreshed:\n%s", refreshed)
	}
	if !bytes.Equal(refreshed, first) {
		t.Errorf("refreshed output should equal fresh upgrade:\n%s", refreshed)
	}
}

func TestUpgradeConfigHints_EdgeCases(t *testing.T) {
	// Empty object: no trailing comma.
	out, err := upgradeHintsInJSON([]byte("{}"))
	if err != nil || !json.Valid(out) {
		t.Fatalf("empty object: %v %s", err, out)
	}
	// CRLF is preserved.
	crlf := strings.ReplaceAll(downstreamConfig, "\n", "\r\n")
	out, err = upgradeHintsInJSON([]byte(crlf))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bytes.ReplaceAll(out, []byte("\r\n"), nil), []byte("\n")) {
		t.Error("mixed line endings introduced")
	}
	// Invalid JSON is rejected untouched.
	if _, err := upgradeHintsInJSON([]byte("{ nope")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUpgradeConfigHints_File(t *testing.T) {
	p := filepath.Join(t.TempDir(), "repogov-config.json")
	if err := os.WriteFile(p, []byte(downstreamConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := UpgradeConfigHints(p)
	if err != nil || !changed {
		t.Fatalf("first run: changed=%v err=%v", changed, err)
	}
	changed, err = UpgradeConfigHints(p)
	if err != nil || changed {
		t.Fatalf("second run: changed=%v err=%v", changed, err)
	}
}

func TestConfigBOM(t *testing.T) {
	withBOM := "\xef\xbb\xbf" + downstreamConfig
	// The loader tolerates a BOM.
	if cfg := loadFrom(t, withBOM); cfg.Files["USAGE.md"] != 700 {
		t.Fatalf("BOM config not loaded: %+v", cfg)
	}
	// The upgrade keeps the BOM and the rest of the file verbatim.
	out, err := upgradeHintsInJSON([]byte(withBOM))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("\xef\xbb\xbf{")) || !bytes.Contains(out, []byte(`"_agent_hints"`)) {
		t.Errorf("BOM lost or hints missing:\n%q", out[:40])
	}
	if !strings.HasSuffix(string(out), downstreamConfig[strings.Index(downstreamConfig, "\n")+1:]) {
		t.Error("body not preserved")
	}
	again, err := upgradeHintsInJSON(out)
	if err != nil || !bytes.Equal(again, out) {
		t.Error("BOM upgrade not idempotent")
	}
}

const downstreamYAML = `# repogov config (hand-maintained)
default: 500
warning_threshold: 85%
skip_dirs:
  - .git
  - vendor
rules:
  - glob: ".github/rules/*.md"
    limit: 300
files:
  README.md: 1200   # big on purpose
  AGENTS.md: 200
`

func loadYAMLFrom(t *testing.T, data string) Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "repogov-config.yaml")
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestUpgradeConfigHints_YAML(t *testing.T) {
	orig := loadYAMLFrom(t, downstreamYAML)
	out, err := upgradeHintsInYAML([]byte(downstreamYAML))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "_agent_hints:\n  about: \"") {
		t.Fatalf("hints block missing:\n%s", s)
	}
	// Header comment stays first; every original line is still present and in order.
	if !strings.HasPrefix(s, "# repogov config (hand-maintained)\n_agent_hints:") {
		t.Errorf("hints should follow the leading comment header:\n%s", s[:80])
	}
	if !strings.Contains(s, "  README.md: 1200   # big on purpose\n") {
		t.Error("inline comment lost")
	}
	if got := loadYAMLFrom(t, s); !reflect.DeepEqual(got, orig) {
		t.Errorf("effective config changed:\n got %+v\nwant %+v", got, orig)
	}
	// Idempotent.
	again, err := upgradeHintsInYAML(out)
	if err != nil || !bytes.Equal(again, out) {
		t.Error("YAML upgrade not idempotent")
	}
	// Refresh a stale block that sits mid-file, with a following key.
	stale := strings.Replace(downstreamYAML, "default: 500\n", "_agent_hints:\n  about: \"old\"\n  never_do:\n    - \"x\"\n\ndefault: 500\n", 1)
	refreshed, err := upgradeHintsInYAML([]byte(stale))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(refreshed), "\"old\"") || strings.Count(string(refreshed), "_agent_hints:") != 1 {
		t.Errorf("stale hints not replaced cleanly:\n%s", refreshed)
	}
	if got := loadYAMLFrom(t, string(refreshed)); !reflect.DeepEqual(got, orig) {
		t.Error("effective config changed after refresh")
	}
	// CRLF and BOM are preserved.
	crlf := strings.ReplaceAll(downstreamYAML, "\n", "\r\n")
	co, err := upgradeHintsInYAML([]byte("\xef\xbb\xbf" + crlf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(co, []byte("\xef\xbb\xbf")) || bytes.Contains(bytes.ReplaceAll(co, []byte("\r\n"), nil), []byte("\n")) {
		t.Error("BOM or CRLF not preserved")
	}
	// With a leading --- document marker.
	doc, err := upgradeHintsInYAML([]byte("---\n" + downstreamYAML))
	if err != nil || !strings.HasPrefix(string(doc), "---\n_agent_hints:") {
		t.Errorf("--- handling: %v\n%s", err, doc)
	}
}

// The YAML hints must carry exactly the strings in the JSON hints.
func TestAgentHintsYAML_MatchesJSON(t *testing.T) {
	var want map[string]any
	var wrapper map[string]any
	if err := json.Unmarshal([]byte("{"+strings.TrimSuffix(strings.TrimSpace(defaultAgentHints), ",")+"}"), &wrapper); err != nil {
		t.Fatal(err)
	}
	want = wrapper[agentHintsKey].(map[string]any)

	got := map[string]string{}
	for _, l := range strings.Split(agentHintsYAML(), "\n") {
		t2 := strings.TrimSpace(l)
		if i := strings.Index(t2, ": \""); i > 0 && !strings.HasPrefix(t2, "- ") {
			var v string
			if err := json.Unmarshal([]byte(t2[i+2:]), &v); err != nil {
				t.Fatalf("not a valid quoted scalar: %q: %v", t2, err)
			}
			got[t2[:i]] = v
		}
	}
	if got["about"] != want["about"] || got["on_limit_fail"] != want["on_limit_fail"] || got["verify"] != want["verify"] {
		t.Errorf("YAML scalar hints differ from JSON hints: %v", got)
	}
	keys := want["keys"].(map[string]any)
	if got["warning_threshold"] != keys["warning_threshold"] {
		t.Errorf("escaped quotes round trip failed: %q vs %q", got["warning_threshold"], keys["warning_threshold"])
	}
	nd := want["never_do"].([]any)
	if c := strings.Count(agentHintsYAML(), "\n    - \""); c != len(nd) {
		t.Errorf("never_do items = %d, want %d", c, len(nd))
	}
}
