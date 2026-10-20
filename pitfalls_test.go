package repogov_test

import (
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func findResult(rs []repogov.LayoutResult, path string) (repogov.LayoutResult, bool) {
	for _, r := range rs {
		if r.Path == path {
			return r, true
		}
	}
	return repogov.LayoutResult{}, false
}

func TestPitfalls_CursorPlainMarkdownIgnored(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".cursor/rules/style.md", "# style\n")
	writeFile(t, root, ".cursor/rules/ok.mdc", "---\ndescription: x\n---\n")
	writeFile(t, root, ".cursor/rules/README.md", "# docs\n")
	writeFile(t, root, ".cursor/rules/sub/deep.md", "# deep\n")

	rs := repogov.CheckAgentPitfalls(root)
	if r, ok := findResult(rs, ".cursor/rules/style.md"); !ok || r.Status != repogov.Warn || !strings.Contains(r.Message, "FIX: rename to style.mdc") {
		t.Errorf("style.md should warn with a rename hint: %+v", rs)
	}
	if _, ok := findResult(rs, ".cursor/rules/sub/deep.md"); !ok {
		t.Error("nested plain .md should also be flagged (Cursor supports subdirectories)")
	}
	if _, ok := findResult(rs, ".cursor/rules/ok.mdc"); ok {
		t.Error(".mdc must not be flagged")
	}
	if _, ok := findResult(rs, ".cursor/rules/README.md"); ok {
		t.Error("README.md is documentation, not a rule")
	}
}

func TestPitfalls_AntigravityFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".agents/rules/good.md", "---\ntrigger: always_on\n---\nbody\n")
	writeFile(t, root, ".agents/rules/none.md", "just text\n")
	writeFile(t, root, ".agents/rules/camel.md", "---\ntrigger: alwaysOn\n---\n")
	writeFile(t, root, ".agents/rules/bogus.md", "---\ntrigger: sometimes\n---\n")
	writeFile(t, root, ".agents/rules/notrigger.md", "---\ndescription: x\n---\n")
	writeFile(t, root, ".agents/rules/md.md", "---\ntrigger: model_decision\n---\n")
	writeFile(t, root, ".agents/rules/glob.md", "---\ntrigger: glob\n---\n")
	writeFile(t, root, ".agents/rules/globok.md", "---\ntrigger: \"glob\"   # comment\nglobs: \"*.ts\"\n---\n")
	writeFile(t, root, ".agents/rules/unclosed.md", "---\ntrigger: always_on\n")
	writeFile(t, root, ".agents/rules/README.md", "no frontmatter, but it is docs\n")

	rs := repogov.CheckAgentPitfalls(root)
	want := map[string]repogov.Status{
		".agents/rules/none.md":      repogov.Fail,
		".agents/rules/camel.md":     repogov.Fail,
		".agents/rules/bogus.md":     repogov.Fail,
		".agents/rules/notrigger.md": repogov.Fail,
		".agents/rules/md.md":        repogov.Warn,
		".agents/rules/glob.md":      repogov.Fail,
		".agents/rules/unclosed.md":  repogov.Fail,
	}
	for path, st := range want {
		r, ok := findResult(rs, path)
		if !ok || r.Status != st {
			t.Errorf("%s: got %+v, want %v", path, r, st)
		}
	}
	for _, clean := range []string{".agents/rules/good.md", ".agents/rules/globok.md", ".agents/rules/README.md"} {
		if _, ok := findResult(rs, clean); ok {
			t.Errorf("%s should produce no finding", clean)
		}
	}
	if r, _ := findResult(rs, ".agents/rules/camel.md"); !strings.Contains(r.Message, "use always_on") {
		t.Errorf("camelCase should suggest the snake_case value: %q", r.Message)
	}
	if info, ok := findResult(rs, ".agents/rules"); !ok || info.Status != repogov.Info || !strings.Contains(info.Message, "9 ") {
		t.Errorf("summary info missing or wrong count: %+v", info)
	}
	if repogov.LayoutPassed(rs) {
		t.Error("Fail findings must fail the layout run")
	}
}

func TestPitfalls_AntigravityNestedFilesAndRegistry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".agents/rules/frontend/react.md", "---\ntrigger: always_on\n---\n")
	rs := repogov.CheckAgentPitfalls(root)
	if r, ok := findResult(rs, ".agents/rules/frontend/react.md"); !ok || r.Status != repogov.Warn || !strings.Contains(r.Message, "rules.json") {
		t.Errorf("unregistered nested rule should warn: %+v", rs)
	}
	// Registered in rules.json (the documented entries form): no warning,
	// and it is counted as checked.
	writeFile(t, root, ".agents/rules.json", `{"entries":[{"path":"rules","include_only":["frontend/react.md"]}]}`)
	rs = repogov.CheckAgentPitfalls(root)
	if _, ok := findResult(rs, ".agents/rules/frontend/react.md"); ok {
		t.Errorf("registered nested rule must not warn: %+v", rs)
	}
	if info, _ := findResult(rs, ".agents/rules"); !strings.Contains(info.Message, "1 ") {
		t.Errorf("a registered nested rule is checked: %+v", info)
	}
}

// Registration is decided by parsing rules.json and matching entry paths
// exactly, never by a substring of its text: a lookalike path, a key whose
// meaning is unknown, or malformed JSON registers nothing (and says so).
func TestPitfalls_AntigravityRegistryParsedExactly(t *testing.T) {
	const nested = ".agents/rules/frontend/react.md"
	cases := []struct {
		name, registry string
		registered     bool
		registryWarn   string // expected message fragment on .agents/rules.json
	}{
		{"documented include_only", `{"entries":[{"path":"rules","include_only":["frontend/react.md"]}]}`, true, ""},
		{"whole directory", `{"entries":[{"path":"rules/frontend"}]}`, true, ""},
		{"include_only directory", `{"entries":[{"path":"rules","include_only":["frontend/"]}]}`, true, ""},
		{"lookalike suffix", `{"entries":[{"path":"rules","include_only":["frontend/react.md.disabled"]}]}`, false, ""},
		{"lookalike file entry", `{"entries":[{"path":"rules/frontend/react.md.disabled"}]}`, false, ""},
		{"text only in exclude", `{"entries":[{"path":"rules","exclude":["frontend/react.md"]}]}`, false, ""},
		{"excluded by name", `{"entries":[{"path":"rules","exclude":["react.md"]}]}`, false, ""},
		{"excluded dir", `{"entries":[{"path":"rules","exclude":["frontend/"]}]}`, false, ""},
		{"file path entry (undocumented)", `{"entries":[{"path":"rules/frontend/react.md"}]}`, false, ""},
		{"other directory", `{"entries":[{"path":"../shared-rules"}]}`, false, ""},
		{"text in another key", `{"inherits":[{"path":"rules/frontend/react.md"}]}`, false, ""},
		{"malformed", `{"entries":[{"path":"rules/frontend/react.md"}`, false, "not a valid registry"},
		{"unknown key", `{"entries":[{"path":"rules","include_glob":["frontend/react.md"]}]}`, false, "not a valid registry"},
		{"trailing data", `{"entries":[{"path":"rules"}]} {}`, false, "not a valid registry"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, nested, "---\ntrigger: always_on\n---\n")
			writeFile(t, root, ".agents/rules.json", tc.registry)
			rs := repogov.CheckAgentPitfalls(root)
			r, warned := findResult(rs, nested)
			if tc.registered && warned {
				t.Errorf("registered nested rule must not warn: %+v", r)
			}
			if !tc.registered && (!warned || r.Status != repogov.Warn || !strings.Contains(r.Message, "rules.json")) {
				t.Errorf("unregistered nested rule must warn: %+v", rs)
			}
			reg, ok := findResult(rs, ".agents/rules.json")
			if tc.registryWarn == "" && ok {
				t.Errorf("unexpected registry finding: %+v", reg)
			}
			if tc.registryWarn != "" && (!ok || reg.Status != repogov.Warn || !strings.Contains(reg.Message, tc.registryWarn)) {
				t.Errorf("a broken registry must say why: %+v", rs)
			}
		})
	}
}

// Registration waives only the location warning: a registered nested rule
// that Antigravity would discard (no frontmatter, bad trigger) still fails.
func TestPitfalls_AntigravityRegisteredNestedRuleStillValidated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".agents/rules.json", `{"entries":[{"path":"rules"}]}`)
	writeFile(t, root, ".agents/rules/frontend/none.md", "just text\n")
	writeFile(t, root, ".agents/rules/frontend/camel.md", "---\ntrigger: alwaysOn\n---\n")
	writeFile(t, root, ".agents/rules/frontend/ok.md", "---\ntrigger: always_on\n---\n")
	rs := repogov.CheckAgentPitfalls(root)
	for _, p := range []string{".agents/rules/frontend/none.md", ".agents/rules/frontend/camel.md"} {
		if r, ok := findResult(rs, p); !ok || r.Status != repogov.Fail || !strings.Contains(r.Message, "silently discards") {
			t.Errorf("%s: registered rule must still be validated: %+v", p, rs)
		}
	}
	if r, ok := findResult(rs, ".agents/rules/frontend/ok.md"); ok {
		t.Errorf("a valid registered rule needs no finding: %+v", r)
	}
	if repogov.LayoutPassed(rs) {
		t.Error("an invalid registered rule must fail the layout run")
	}
}

// Quoted frontmatter keys are the same keys: "trigger": always_on is a
// valid rule, not one missing its trigger.
func TestPitfalls_AntigravityQuotedFrontmatterKeys(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".agents/rules/dq.md", "---\n\"trigger\": always_on\n---\n")
	writeFile(t, root, ".agents/rules/sq.md", "---\n'trigger': glob\n'globs': \"*.ts\"\n---\n")
	writeFile(t, root, ".agents/rules/bad.md", "---\n\"trigger\": alwaysOn\n---\n")
	rs := repogov.CheckAgentPitfalls(root)
	for _, p := range []string{".agents/rules/dq.md", ".agents/rules/sq.md"} {
		if r, ok := findResult(rs, p); ok {
			t.Errorf("%s: quoted keys must be read as their unquoted names: %+v", p, r)
		}
	}
	if r, ok := findResult(rs, ".agents/rules/bad.md"); !ok || !strings.Contains(r.Message, "unrecognized trigger") {
		t.Errorf("a quoted key's value must still be validated: %+v", rs)
	}
}

func TestPitfalls_NothingWhenAbsentAndLegacyDir(t *testing.T) {
	root := t.TempDir()
	if rs := repogov.CheckAgentPitfalls(root); len(rs) != 0 {
		t.Errorf("empty repo should produce nothing: %+v", rs)
	}
	writeFile(t, root, "node_modules/x/.agents/rules/bad.md", "no frontmatter\n")
	if rs := repogov.CheckAgentPitfalls(root); len(rs) != 0 {
		t.Errorf("node_modules must be skipped: %+v", rs)
	}
	writeFile(t, root, "pkg/.agent/rules/legacy.md", "no frontmatter\n")
	if _, ok := findResult(repogov.CheckAgentPitfalls(root), "pkg/.agent/rules/legacy.md"); !ok {
		t.Error("legacy .agent/rules in a subdirectory should be checked")
	}
}
