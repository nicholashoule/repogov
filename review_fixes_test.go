package repogov

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
)

// --- spliceContextSection (AGENTS.md marked Context section) ---

const (
	ctxBegin = "<!-- repogov:context:begin -->"
	ctxEnd   = "<!-- repogov:context:end -->"
)

// spliceFile writes content to a temp AGENTS.md, splices "## Context\n\n-
// new\n" into it, and returns the result and the splice error.
func spliceFile(t *testing.T, content string) (string, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	spliceErr := spliceContextSection(p, "## Context\n\n- new\n")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data), spliceErr
}

// Markdown the old heading-based splice misread as a section end (or start):
// each one deleted user text. The tail after the Context section must survive.
var spliceTrapTails = map[string]string{
	"setext heading":         "Appendix\n========\n\nkeep me\n",
	"indented heading":       "  ## Indented\n\nkeep me\n",
	"tab after hashes":       "##\tTabbed\n\nkeep me\n",
	"backtick-info fence":    "```a`b\n## Not a fence\n\nkeep me\n",
	"heading in a comment":   "<!--\n## Context\n-->\n\nkeep me\n",
	"setext heading level 2": "Notes\n-----\n\nkeep me\n",
}

func TestSpliceContextSection_UnmarkedLeavesUserTextIntact(t *testing.T) {
	for name, tail := range spliceTrapTails {
		for _, in := range []string{
			"# A\n\n## Context\n\n- old\n\n" + tail,
			"# A\n\n<!--\n## Context\n-->\n\n" + tail,
		} {
			got, err := spliceFile(t, in)
			if err != nil || got != in {
				t.Errorf("%s: unmarked file must be left unchanged with no error (err %v):\n%q", name, err, got)
			}
		}
	}
}

func TestSpliceContextSection_MarkedRefreshesOnlyBetweenMarkers(t *testing.T) {
	for name, tail := range spliceTrapTails {
		head := "# A\n\n" + tail + "\n"
		in := head + ctxBegin + "\n## Context\n\n- old\n\n" + tail + ctxEnd + "\n\n" + tail
		want := head + ctxBegin + "\n## Context\n\n- new\n" + ctxEnd + "\n\n" + tail
		got, err := spliceFile(t, in)
		if err != nil || got != want {
			t.Errorf("%s: err %v\ngot:\n%q\nwant:\n%q", name, err, got, want)
		}
	}
}

func TestSpliceContextSection_BadMarkersLeaveFileUnchanged(t *testing.T) {
	section := "## Context\n\n- old\n"
	bad := map[string]string{
		"no markers":      "# A\n\n" + section,
		"begin only":      ctxBegin + "\n" + section,
		"end only":        section + ctxEnd + "\n",
		"duplicate begin": ctxBegin + "\n" + ctxBegin + "\n" + section + ctxEnd + "\n",
		"duplicate end":   ctxBegin + "\n" + section + ctxEnd + "\n" + ctxEnd + "\n",
		"two pairs":       ctxBegin + "\n" + section + ctxEnd + "\n" + ctxBegin + "\n" + section + ctxEnd + "\n",
		"out of order":    ctxEnd + "\n" + section + ctxBegin + "\n",
	}
	for name, in := range bad {
		p := filepath.Join(t.TempDir(), "AGENTS.md")
		if err := os.WriteFile(p, []byte(in), 0o644); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-time.Hour).Truncate(time.Second)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
		if err := spliceContextSection(p, "## Context\n\n- new\n"); err != nil {
			t.Errorf("%s: bad markers are a notice, not an error: %v", name, err)
		}
		ok, reason := AgentsMdContextStatus(filepath.Dir(p))
		if ok || !strings.Contains(reason, "left unchanged") || !strings.Contains(reason, ctxBegin) {
			t.Errorf("%s: status must say why and how to opt in: %v %q", name, ok, reason)
		}
		data, _ := os.ReadFile(p)
		info, _ := os.Stat(p)
		if string(data) != in || !info.ModTime().Equal(old) {
			t.Errorf("%s: file must be left byte-for-byte unchanged and unwritten:\n%q", name, data)
		}
	}
}

func TestAgentsMdContextStatus(t *testing.T) {
	root := t.TempDir()
	if ok, reason := AgentsMdContextStatus(root); ok || !strings.Contains(reason, "does not exist") {
		t.Errorf("no AGENTS.md: %v %q", ok, reason)
	}
	p := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(p, []byte("# A\n\n"+ctxBegin+"\n## Context\n"+ctxEnd+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, reason := AgentsMdContextStatus(root); !ok || reason != "" {
		t.Errorf("one ordered marker pair must be refreshable: %v %q", ok, reason)
	}
	if err := os.WriteFile(p, []byte("# A\n\n## Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, reason := AgentsMdContextStatus(root); ok || !strings.Contains(reason, "no repogov:context markers") || !strings.Contains(reason, "re-run 'repogov init'") {
		t.Errorf("unmarked: want the cause and how to opt in: %v %q", ok, reason)
	}
}

func TestSpliceContextSection_CRLF(t *testing.T) {
	in := "# A\r\n\r\n" + ctxBegin + "\r\n## Context\r\n\r\n- old\r\n" + ctxEnd + "\r\n\r\n## Next\r\n"
	want := "# A\r\n\r\n" + ctxBegin + "\r\n## Context\r\n\r\n- new\r\n" + ctxEnd + "\r\n\r\n## Next\r\n"
	if got, err := spliceFile(t, in); err != nil || got != want {
		t.Errorf("err %v\ngot:\n%q\nwant:\n%q", err, got, want)
	}
}

func TestSpliceContextSection_UnchangedNotRewritten(t *testing.T) {
	p := filepath.Join(t.TempDir(), "AGENTS.md")
	content := "# A\n\n" + ctxBegin + "\n## Context\n\n- new\n" + ctxEnd + "\n\n## Next\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	if err := spliceContextSection(p, "## Context\n\n- new\n"); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(p); !info.ModTime().Equal(old) {
		t.Error("an unchanged section must not be rewritten")
	}
}

func TestSpliceContextSection_RefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(target, []byte(ctxBegin+"\n## Context\n\n- secret\n"+ctxEnd+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "AGENTS.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := spliceContextSection(link, "## Context\n\n- new\n"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("symlinked AGENTS.md must be refused with a reason, got %v", err)
	}
	if data, _ := os.ReadFile(target); !strings.Contains(string(data), "secret") {
		t.Error("symlink target was modified")
	}
}

// A freshly scaffolded AGENTS.md carries the markers, and re-running init
// with the same schema is a no-op (template and generated section agree).
func TestInit_AgentsMdMarkedAndStable(t *testing.T) {
	for _, lang := range []string{"", "en-GB"} {
		root := t.TempDir()
		cfg := DefaultConfig()
		cfg.Language = lang
		if _, err := InitLayoutWithConfig(root, DefaultClaudeLayout(), cfg); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, "AGENTS.md")
		first, _ := os.ReadFile(p)
		if strings.Count(string(first), ctxBegin) != 1 || strings.Count(string(first), ctxEnd) != 1 {
			t.Fatalf("lang %q: scaffolded AGENTS.md lacks one marker pair:\n%s", lang, first)
		}
		if _, err := InitLayoutWithConfig(root, DefaultClaudeLayout(), cfg); err != nil {
			t.Fatal(err)
		}
		if second, _ := os.ReadFile(p); !bytes.Equal(second, first) {
			t.Errorf("lang %q: re-init changed AGENTS.md:\n%s", lang, second)
		}
	}
}

// The marked section carries the language line: en-US by default, and an
// existing repo that sets another language gets that line on its next init,
// inside the section repogov refreshes. An empty language is en-US again.
func TestInit_MarkedSectionGainsLanguageLine(t *testing.T) {
	root := t.TempDir()
	if _, err := InitLayout(root, DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "AGENTS.md")
	before, _ := os.ReadFile(p)
	inSection := func(s, directive string) bool {
		b, e, d := strings.Index(s, ctxBegin), strings.Index(s, ctxEnd), strings.Index(s, directive)
		return d > b && d < e && strings.Count(s, "Prose language") == 1
	}
	if !inSection(string(before), languageDirective("en-US")) {
		t.Fatalf("default AGENTS.md must state en-US once, inside the markers:\n%s", before)
	}
	cfg := DefaultConfig()
	cfg.Language = "en-GB"
	if _, err := InitLayoutWithConfig(root, DefaultCopilotLayout(), cfg); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(p); !inSection(string(after), languageDirective("en-GB")) {
		t.Errorf("en-GB directive must replace en-US between the markers:\n%s", after)
	}
	cfg.Language = ""
	if _, err := InitLayoutWithConfig(root, DefaultCopilotLayout(), cfg); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); !bytes.Equal(got, before) {
		t.Errorf("an empty language must give the en-US section byte-for-byte:\n%s", got)
	}
	// The multi-schema path carries the language too.
	cfg.Language = "en-GB"
	if _, err := InitLayoutAllWithConfig(root, []LayoutSchema{DefaultCopilotLayout(), DefaultClaudeLayout()}, cfg); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); !strings.Contains(string(got), languageDirective("en-GB")) {
		t.Errorf("all-agent init must state the language in the section:\n%s", got)
	}
}

func TestCheckAgentsMdLanguage(t *testing.T) {
	root := t.TempDir()
	if rs := CheckAgentsMdLanguage(root, "en-US"); rs != nil {
		t.Errorf("no AGENTS.md: want no finding, got %+v", rs)
	}
	p := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(p, []byte("# A\n\nWrite in English.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rs := CheckAgentsMdLanguage(root, "tlh"); rs != nil {
		t.Errorf("unknown language (a validate warning): want no finding, got %+v", rs)
	}
	// An empty language is the default, en-US: there is no opt-out.
	for _, lang := range []string{"en", ""} {
		rs := CheckAgentsMdLanguage(root, lang)
		if len(rs) != 1 || rs[0].Status != Warn || !strings.Contains(rs[0].Message, "(en-US)") || !strings.Contains(rs[0].Message, "FIX:") || !strings.Contains(rs[0].Message, ctxBegin) {
			t.Errorf("lang %q not stated: want one WARN with a FIX hint, got %+v", lang, rs)
		}
	}
	if err := os.WriteFile(p, []byte("# A\n\n"+languageDirective("en-US")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rs := CheckAgentsMdLanguage(root, "en-US"); len(rs) != 1 || rs[0].Status != Pass {
		t.Errorf("tag present: want PASS, got %+v", rs)
	}
	if rs := CheckAgentsMdLanguage(root, "en-GB"); len(rs) != 1 || rs[0].Status != Warn {
		t.Errorf("a different tag must not satisfy the check, got %+v", rs)
	}
}

// --- strict config parsing ---

func TestUnmarshalYAML_RejectsUnsupportedSyntax(t *testing.T) {
	bad := map[string]string{
		"flow list":        "skip_dirs: [vendor, node_modules]\n",
		"flow map":         "files: {README.md: 10}\n",
		"flow scalar":      "default: [1]\n",
		"tab indent":       "skip_dirs:\n\t- vendor\n",
		"unknown key":      "warning_treshold: 80\n",
		"unknown rule key": "rules:\n  - glob: \"*.md\"\n    limt: 5\n",
		"scalar list":      "skip_dirs: vendor\n",
		"stray indent":     "default: 300\n  - x\n",
		"not a key":        "just text\n",
	}
	for name, doc := range bad {
		var cfg Config
		if err := unmarshalYAML([]byte(doc), &cfg); err == nil {
			t.Errorf("%s: accepted %q; want an error", name, doc)
		} else if !strings.Contains(err.Error(), "line ") {
			t.Errorf("%s: error should name the line: %v", name, err)
		}
	}
	good := map[string]string{
		"empty list":        "skip_dirs: []\ninclude_exts: []\n",
		"empty files map":   "files: {}\n",
		"metadata key":      "_agent_hints:\n  about: \"x\"\n  keys:\n    default: \"y\"\ndefault: 300\n",
		"indentless list":   "skip_dirs:\n- vendor\n- dist\n",
		"indentless rules":  "rules:\n- glob: \"*.md\"\n  limit: 10\n",
		"language and more": "language: en-GB\ndefault: 300\n",
	}
	for name, doc := range good {
		var cfg Config
		if err := unmarshalYAML([]byte(doc), &cfg); err != nil {
			t.Errorf("%s: rejected valid config: %v", name, err)
		}
	}
	var cfg Config
	_ = unmarshalYAML([]byte("skip_dirs:\n- vendor\n- dist\n"), &cfg)
	if strings.Join(cfg.SkipDirs, ",") != "vendor,dist" {
		t.Errorf("indentless list parsed as %v", cfg.SkipDirs)
	}
	cfg = Config{}
	_ = unmarshalYAML([]byte("skip_dirs: []\n"), &cfg)
	if cfg.SkipDirs == nil || len(cfg.SkipDirs) != 0 {
		t.Errorf("skip_dirs: [] must be an explicit empty list, got %#v", cfg.SkipDirs)
	}
}

func TestLoadConfig_JSONRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := LoadConfig(write("typo.json", `{"warning_treshold": 80}`)); err == nil || !strings.Contains(err.Error(), "warning_treshold") {
		t.Errorf("top-level typo must fail naming the key, got %v", err)
	}
	if _, err := LoadConfig(write("nested.json", `{"rules": [{"glob": "*.md", "limt": 5}]}`)); err == nil {
		t.Error("unknown key inside a rule must fail")
	}
	cfg, err := LoadConfig(write("meta.json", `{"_comment": "x", "_agent_hints": {"a": 1}, "default": 250}`))
	if err != nil || cfg.Default != 250 {
		t.Errorf("_-prefixed keys must be ignored: %+v %v", cfg, err)
	}
}

// --- UTF-16/32 text and binary byte limits ---

func utf16File(t *testing.T, text string, bigEndian bool) string {
	t.Helper()
	units := utf16.Encode([]rune(text))
	var b []byte
	if bigEndian {
		b = append(b, 0xFE, 0xFF)
	} else {
		b = append(b, 0xFF, 0xFE)
	}
	for _, u := range units {
		if bigEndian {
			b = append(b, byte(u>>8), byte(u))
		} else {
			b = append(b, byte(u), byte(u>>8))
		}
	}
	p := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckFile_UTF16And32AreCounted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Default = 2
	for _, be := range []bool{false, true} {
		p := utf16File(t, "one\ntwo\nthree\n", be)
		res, err := checkFileAt(p, "doc.md", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != Fail || res.Lines != 3 {
			t.Errorf("UTF-16 (big-endian=%v) must be line-counted: %+v", be, res)
		}
	}
	// UTF-32LE: BOM FF FE 00 00, then 4-byte units; final line unterminated.
	b := []byte{0xFF, 0xFE, 0x00, 0x00}
	for _, r := range "a\nb\nc" {
		b = append(b, byte(r), 0, 0, 0)
	}
	p := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := checkFileAt(p, "doc.md", cfg)
	if err != nil || res.Status != Fail || res.Lines != 3 {
		t.Errorf("UTF-32 must be line-counted: %+v %v", res, err)
	}
}

func TestCheckFile_BinaryStillHeldToByteLimit(t *testing.T) {
	cfg := DefaultConfig()
	p := filepath.Join(t.TempDir(), "AGENTS.md")
	data := append([]byte("text\x00"), make([]byte, 30000)...)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := checkFileAt(p, "AGENTS.md", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != Fail || !strings.Contains(res.Action, "byte limit") {
		t.Errorf("binary file over max_bytes must FAIL, got %+v", res)
	}
	// A small binary file is still skipped.
	if err := os.WriteFile(p, []byte("a\x00b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, _ := checkFileAt(p, "AGENTS.md", cfg); res.Status != Skip {
		t.Errorf("small binary file must SKIP, got %+v", res)
	}
}

// --- frontmatter ---

func TestCheckFrontmatter_BOMAndNonRegular(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	if err := os.WriteFile(p, []byte("\xEF\xBB\xBF---\napplyTo: \"**\"\n---\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, r := range checkFrontmatter(p, "a.md", []string{"applyTo"}) {
		if r.Status != Pass {
			t.Errorf("BOM-prefixed frontmatter must pass: %+v", r)
		}
	}
	link := filepath.Join(dir, "link.md")
	if err := os.Symlink(p, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	rs := checkFrontmatter(link, "link.md", []string{"applyTo"})
	if len(rs) != 1 || rs[0].Status != Skip || !strings.Contains(rs[0].Message, "not followed") {
		t.Errorf("symlinked rule file must SKIP unopened with a reason (as CheckDir does): %+v", rs)
	}
}

// --- init config precedence ---

func TestInit_DoesNotShadowRootConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "repogov-config.json"), []byte(`{"default": 300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitLayout(root, DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github", "repogov-config.json")); !os.IsNotExist(err) {
		t.Error("init created .github/repogov-config.json although a root config is active")
	}
	gov, err := os.ReadFile(filepath.Join(root, ".github", "rules", "governance.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gov), "](../../repogov-config.json)") {
		t.Errorf("governance.md must link the active root config:\n%s", gov)
	}
}

func TestInitAll_DoesNotShadowGitHubConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "repogov-config.json"), []byte(`{"default": 300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitLayoutAll(root, []LayoutSchema{DefaultClaudeLayout(), DefaultCursorLayout()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "repogov-config.json")); !os.IsNotExist(err) {
		t.Error("init created a root config that shadows .github/repogov-config.json")
	}
}

func TestDetectConfigRelFrom_FollowsFindConfigPrecedence(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"repogov-config.yaml", ".github/repogov-config.json"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte("default: 300\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	name, rel := detectConfigRelFrom(root, filepath.Join(root, ".github", "rules"))
	if name != "repogov-config.yaml" || rel != "../../repogov-config.yaml" {
		t.Errorf("got %s %s; want the root YAML config (FindConfig precedence)", name, rel)
	}
}

// --- SaveConfig ---

func TestSaveConfig_AtomicAndRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "repogov-config.json")
	cfg := DefaultConfig()
	cfg.Language = "en-GB"
	if err := SaveConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
	if got, err := LoadConfig(p); err != nil || got.Language != "en-GB" {
		t.Errorf("round trip: %+v %v", got.Language, err)
	}
	yp := filepath.Join(dir, "repogov-config.yaml")
	if err := SaveConfig(yp, cfg); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadConfig(yp); err != nil || got.Language != "en-GB" {
		t.Errorf("YAML round trip: %+v %v", got.Language, err)
	}
	target := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := SaveConfig(link, cfg); err == nil {
		t.Error("SaveConfig must refuse to write through a symlink")
	}
	if data, _ := os.ReadFile(target); string(data) != "{}" {
		t.Error("symlink target was modified")
	}
}

// --- repository hygiene: CI permissions and the pre-commit hook ---

func TestCIWorkflow_LeastPrivilege(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Skipf("workflow not found: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "pull-requests: write") {
		t.Error("ci.yml must not grant pull-requests: write")
	}
	if c, p := strings.Count(s, "uses: actions/checkout@"), strings.Count(s, "persist-credentials: false"); c == 0 || c != p {
		t.Errorf("every actions/checkout step needs persist-credentials: false (%d checkouts, %d set)", c, p)
	}
}

func TestPreCommitHook_DoesNotRestageEverything(t *testing.T) {
	data, err := os.ReadFile("scripts/hooks/pre-commit")
	if err != nil {
		t.Skipf("hook not found: %v", err)
	}
	if strings.Contains(string(data), "git add -u") {
		t.Error("pre-commit must restage only files demojify changed, not every modified tracked file (git add -u)")
	}
	// Non-ASCII paths must print verbatim, or the re-stage never matches them.
	if n := strings.Count(string(data), "git diff"); n != 0 {
		t.Errorf("%d git diff call(s) without -c core.quotePath=false", n)
	}
	if n := strings.Count(string(data), "git -c core.quotePath=false diff"); n < 3 {
		t.Errorf("want the staged/unstaged path listings to use core.quotePath=false, found %d", n)
	}
}

// --- atomic write: Windows rename retry ---

func TestRenameRetrying_RetriesTransientThenSucceeds(t *testing.T) {
	transientErr := errors.New("access is denied")
	transient := func(err error) bool { return errors.Is(err, transientErr) }
	calls := 0
	var slept time.Duration
	rename := func(_, _ string) error {
		calls++
		if calls < 4 {
			return transientErr
		}
		return nil
	}
	if err := renameRetrying(rename, transient, func(d time.Duration) { slept += d }, "a", "b"); err != nil {
		t.Fatalf("want success after transient failures, got %v", err)
	}
	if calls != 4 || slept != 70*time.Millisecond {
		t.Errorf("calls %d, slept %v; want 4 calls and 10+20+40 ms of backoff", calls, slept)
	}
}

func TestRenameRetrying_BoundedAndSaysWhy(t *testing.T) {
	transientErr := errors.New("access is denied")
	calls := 0
	var slept time.Duration
	err := renameRetrying(func(_, _ string) error { calls++; return transientErr },
		func(err error) bool { return errors.Is(err, transientErr) },
		func(d time.Duration) { slept += d }, "a", "b")
	if !errors.Is(err, transientErr) || !strings.Contains(err.Error(), "holding the file open") || !strings.Contains(err.Error(), "close it") {
		t.Errorf("want the cause and next step, got %v", err)
	}
	if slept > renameRetryBudget || calls < 2 {
		t.Errorf("slept %v over %d calls; want bounded retries within %v", slept, calls, renameRetryBudget)
	}
	// A non-transient failure is returned at once, unretried.
	calls = 0
	permanent := errors.New("no such file")
	if err := renameRetrying(func(_, _ string) error { calls++; return permanent },
		func(err error) bool { return errors.Is(err, transientErr) },
		func(time.Duration) { t.Error("slept on a permanent failure") }, "a", "b"); !errors.Is(err, permanent) || calls != 1 {
		t.Errorf("permanent failure: err %v after %d calls", err, calls)
	}
}

func TestIsTransientRenameError_PlatformErrnos(t *testing.T) {
	denied := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(5)}   // ERROR_ACCESS_DENIED
	sharing := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.Errno(32)} // ERROR_SHARING_VIOLATION
	want := runtime.GOOS == "windows"
	if isTransientRenameError(denied) != want || isTransientRenameError(sharing) != want {
		t.Errorf("transient detection on %s: want %v", runtime.GOOS, want)
	}
	if isTransientRenameError(errors.New("other")) {
		t.Error("an unrelated error must not be retried")
	}
}

// --- YAML round trip of empty values ---

func TestSaveConfigYAML_EmptyValuesRoundTrip(t *testing.T) {
	if got := yamlQuote(""); got != `""` {
		t.Errorf(`yamlQuote("") = %q; want ""`, got)
	}
	p := filepath.Join(t.TempDir(), "repogov-config.yaml")
	cfg := DefaultConfig()
	cfg.SkipDirs = []string{""}
	cfg.Rules = []Rule{}
	if err := SaveConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(p)
	if err != nil {
		data, _ := os.ReadFile(p)
		t.Fatalf("saved YAML does not load: %v\n%s", err, data)
	}
	if len(got.SkipDirs) != 1 || got.SkipDirs[0] != "" {
		t.Errorf("SkipDirs = %#v; want []string{\"\"}", got.SkipDirs)
	}
	if got.Rules == nil || len(got.Rules) != 0 {
		t.Errorf("Rules = %#v; want an explicit empty list", got.Rules)
	}
}

// --- strict decoding: $schema and quoted YAML keys ---

func TestStrictDecoding_SchemaKeyAndQuotedYAMLKeys(t *testing.T) {
	dir := t.TempDir()
	j := filepath.Join(dir, "repogov-config.json")
	if err := os.WriteFile(j, []byte(`{"$schema": "https://example.com/repogov.schema.json", "default": 250}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg, err := LoadConfig(j); err != nil || cfg.Default != 250 {
		t.Errorf("top-level $schema must be ignored: %+v %v", cfg.Default, err)
	}
	var cfg Config
	doc := "\"default\": 300\n'warning_threshold': 70\n\"_agent_hints\":\n  about: \"x\"\nrules:\n  - \"glob\": \"*.md\"\n    limit: 5\n"
	if err := unmarshalYAML([]byte(doc), &cfg); err != nil {
		t.Fatalf("quoted keys must be accepted: %v", err)
	}
	if cfg.Default != 300 || cfg.WarningThreshold != 70 || len(cfg.Rules) != 1 || cfg.Rules[0].Glob != "*.md" {
		t.Errorf("quoted keys parsed as %+v", cfg)
	}
	if err := unmarshalYAML([]byte("\"defaults\": 300\n"), &cfg); err == nil {
		t.Error("a quoted unknown key must still be rejected")
	}
	// A quoted hints key is refreshed in place, not duplicated.
	out, err := upgradeHintsInYAML([]byte("\"_agent_hints\":\n  about: \"old\"\ndefault: 300\n"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "_agent_hints"); n != 1 {
		t.Errorf("hints block duplicated (%d occurrences):\n%s", n, out)
	}
}

// --- FF FE 00 00: UTF-32LE or UTF-16LE + U+0000 ---

func TestCheckFile_FFFE0000Ambiguity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Default = 100
	write := func(b []byte) string {
		p := filepath.Join(t.TempDir(), "doc.md")
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// UTF-16LE: BOM, U+0000, then "a\nb\nc" -- 14 bytes, not a multiple of 4.
	b := []byte{0xFF, 0xFE, 0x00, 0x00}
	for _, r := range "a\nb\nc" {
		b = append(b, byte(r), 0)
	}
	if res, err := checkFileAt(write(b), "doc.md", cfg); err != nil || res.Lines != 3 {
		t.Errorf("UTF-16LE with a leading U+0000 must count 3 lines: %+v %v", res, err)
	}
	// UTF-32LE: 4-byte units, total a multiple of 4.
	b = []byte{0xFF, 0xFE, 0x00, 0x00}
	for _, r := range "a\nb\nc" {
		b = append(b, byte(r), 0, 0, 0)
	}
	if res, err := checkFileAt(write(b), "doc.md", cfg); err != nil || res.Lines != 3 {
		t.Errorf("UTF-32LE must count 3 lines: %+v %v", res, err)
	}
	if u, _ := wideEncoding([]byte{0xFF, 0xFE, 0x00, 0x00}, 14); u != 2 {
		t.Errorf("FF FE 00 00 with a 14-byte file: unit %d; want UTF-16 (2)", u)
	}
	if u, _ := wideEncoding([]byte{0xFF, 0xFE, 0x00, 0x00}, 24); u != 4 {
		t.Errorf("FF FE 00 00 with a 24-byte file: unit %d; want UTF-32 (4)", u)
	}
}
