package repogov

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"en-US", "en-US", true},
		{"EN_gb", "en-GB", true},
		{"en", "en-US", true},
		{"es", "es-ES", true},
		{"es-419", "es-419", true},
		{"pt", "pt-BR", true},
		{"pt_pt", "pt-PT", true},
		{"fr", "fr-FR", true},
		{"de", "de-DE", true},
		{"zh", "zh-Hans", true},
		{"zh-CN", "zh-Hans", true},
		{"zh-tw", "zh-Hant", true},
		{"zh-HK", "zh-Hant", true},
		{"no", "nb-NO", true},
		{"ar", "ar-SA", true},
		{"ja", "ja-JP", true},
		{"fil", "fil-PH", true},
		{" ga-ie ", "ga-IE", true},
		{"en-NZ", "en-NZ", false},
		{"tlh", "tlh", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeLanguage(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeLanguage(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	// Every known tag round-trips, case-insensitively.
	for _, l := range KnownLanguages() {
		if got, ok := NormalizeLanguage(strings.ToUpper(l.Tag)); !ok || got != l.Tag {
			t.Errorf("known tag %s does not round-trip: %q %v", l.Tag, got, ok)
		}
		if l.Name == "" {
			t.Errorf("%s has no display name", l.Tag)
		}
	}
}

func TestKnownLanguages_RTLAndCopy(t *testing.T) {
	rtl := map[string]bool{"he-IL": true, "ar-SA": true, "fa-IR": true, "ur-PK": true}
	for _, l := range KnownLanguages() {
		if l.RTL != rtl[l.Tag] {
			t.Errorf("%s RTL = %v, want %v", l.Tag, l.RTL, rtl[l.Tag])
		}
	}
	ls := KnownLanguages()
	ls[0].Tag = "mutated"
	if KnownLanguages()[0].Tag == "mutated" {
		t.Error("KnownLanguages must return a copy")
	}
	if len(ls) < 60 {
		t.Errorf("expected the broad language set, got %d entries", len(ls))
	}
}

func TestLanguageDirective_NamesVariant(t *testing.T) {
	cases := map[string]string{
		"en-US":   "Prose language: American English (en-US). Write documentation (.md, .txt), code comments and commit messages in American English, with its spelling and conventions (American spelling",
		"en-GB":   "British spelling",
		"pt-BR":   "not European Portuguese",
		"pt-PT":   "not Brazilian Portuguese",
		"zh-Hans": "Simplified characters",
		"zh-Hant": "Traditional characters",
		"he":      "right-to-left",
	}
	for tag, want := range cases {
		if got := languageDirective(tag); !strings.Contains(got, want) {
			t.Errorf("directive for %s = %q; want it to contain %q", tag, got, want)
		}
	}
	if got := languageDirective("en-US"); !strings.HasSuffix(got, "keep identifiers, code and quoted text as they are.") {
		t.Errorf("directive must end with the keep-as-is clause: %q", got)
	}
	if languageDirective("") != languageDirective("en-US") {
		t.Error("an empty language must state the default, en-US")
	}
	if languageDirective("tlh") != "" {
		t.Error("an unknown language must produce no directive (validate reports it)")
	}
}

// The default prose language is en-US everywhere: DefaultConfig, LoadConfig
// (missing or empty key) and the effective tag of an empty value.
func TestDefaultLanguage_IsEnUS(t *testing.T) {
	if DefaultLanguage != "en-US" || DefaultConfig().Language != "en-US" {
		t.Errorf("default language = %q / %q; want en-US", DefaultLanguage, DefaultConfig().Language)
	}
	dir := t.TempDir()
	for name, body := range map[string]string{
		"missing.json": `{"default": 300}`,
		"empty.json":   `{"language": "", "default": 300}`,
		"missing.yaml": "default: 300\n",
		"empty.yaml":   "language: \"\"\ndefault: 300\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if cfg, err := LoadConfig(p); err != nil || cfg.Language != "en-US" {
			t.Errorf("%s: language = %q, %v; want en-US", name, cfg.Language, err)
		}
	}
	if cfg, _ := LoadConfig(filepath.Join(dir, "absent.json")); cfg.Language != "en-US" {
		t.Errorf("no config file: language = %q; want en-US", cfg.Language)
	}
}

func TestValidateConfig_UnknownLanguageWarns(t *testing.T) {
	warn := func(lang string) *Violation {
		cfg := DefaultConfig()
		cfg.Language = lang
		for _, v := range ValidateConfig(cfg) {
			if v.Field == "language" {
				return &v
			}
		}
		return nil
	}
	if v := warn("en-NZ"); v == nil || v.Severity != "warning" || !strings.Contains(v.Message, "en-GB") {
		t.Errorf("en-NZ: want a warning naming the closest tags (en-GB...), got %+v", v)
	}
	if v := warn("tlh"); v == nil || v.Severity != "warning" || !strings.Contains(v.Message, "en-US") {
		t.Errorf("tlh: want a warning explaining how to find tags, got %+v", v)
	}
	for _, ok := range []string{"", "en-gb", "zh-TW", "pt"} {
		if v := warn(ok); v != nil {
			t.Errorf("%q must be accepted, got %+v", ok, v)
		}
	}
}

func TestLoadConfig_LanguageNormalized(t *testing.T) {
	dir := t.TempDir()
	j := filepath.Join(dir, "repogov-config.json")
	y := filepath.Join(dir, "repogov-config.yaml")
	if err := os.WriteFile(j, []byte(`{"language": "en_gb"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(y, []byte("language: \"pt\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg, err := LoadConfig(j); err != nil || cfg.Language != "en-GB" {
		t.Errorf("JSON language = %q, %v; want en-GB", cfg.Language, err)
	}
	if cfg, err := LoadConfig(y); err != nil || cfg.Language != "pt-BR" {
		t.Errorf("YAML language = %q, %v; want pt-BR", cfg.Language, err)
	}
}

// agentFileRenders renders every scaffolded agent instruction file (and the
// general rule) for lang.
func agentFileRenders(lang string) map[string]string {
	claude, copilot := DefaultClaudeLayout(), DefaultCopilotLayout()
	return map[string]string{
		"AGENTS.md":               agentsMdContent(claude, lang),
		"CLAUDE.md":               claudeMdContent("claude", ".claude", lang),
		"copilot-instructions.md": copilotInstructionsContent(copilotNarrowSchema("", copilot), false, lang),
		"GEMINI.md":               geminiMdContent(lang),
		".rules":                  zedRulesContent(lang),
		"general.md":              generalInstructionsContent(".github", "rules", lang),
	}
}

func TestAgentFiles_StateLanguageWhenSet(t *testing.T) {
	directive := languageDirective("pt-BR")
	for name, content := range agentFileRenders("pt-BR") {
		if !strings.Contains(content, directive) {
			t.Errorf("%s does not state the prose language:\n%s", name, content)
		}
		if strings.Contains(content, "(en-US)") {
			t.Errorf("%s states the default language as well as pt-BR", name)
		}
	}
}

// With no language set every scaffolded agent file states the default,
// en-US: an empty language renders exactly what "en-US" renders.
func TestAgentFiles_DefaultLanguageIsEnUS(t *testing.T) {
	directive := languageDirective("en-US")
	explicit := agentFileRenders("en-US")
	for name, content := range agentFileRenders("") {
		if !strings.Contains(content, directive) {
			t.Errorf("%s does not state the default prose language (en-US):\n%s", name, content)
		}
		if content != explicit[name] {
			t.Errorf("%s: an empty language must render exactly as en-US", name)
		}
		if strings.Contains(content, "\n\n\n") {
			t.Errorf("%s has a stray blank line:\n%s", name, content)
		}
	}
}

// End to end: init states the language in every scaffolded agent file and
// writes it into the generated config right after _agent_hints; with none
// set, that language is en-US.
func TestInitLayoutAll_LanguageInScaffoldedFiles(t *testing.T) {
	schemas := []LayoutSchema{
		DefaultCopilotLayout(), DefaultClaudeLayout(), DefaultGeminiLayout(), DefaultZedLayout(), DefaultCursorLayout(),
	}
	files := []string{
		"AGENTS.md", ".claude/CLAUDE.md", ".github/copilot-instructions.md", "GEMINI.md", ".rules",
		".github/rules/general.md", ".claude/rules/general.md", ".cursor/rules/general.mdc",
	}
	for lang, want := range map[string]string{"en-GB": "en-GB", "": "en-US"} {
		root := t.TempDir()
		cfg := DefaultConfig()
		cfg.Language = lang
		if _, err := InitLayoutAllWithConfig(root, schemas, cfg); err != nil {
			t.Fatal(err)
		}
		directive := languageDirective(want)
		for _, f := range files {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
			if err != nil {
				t.Fatalf("%s not scaffolded: %v", f, err)
			}
			if !strings.Contains(string(data), directive) {
				t.Errorf("lang %q: %s does not state %s", lang, f, want)
			}
		}
		p := filepath.Join(root, "repogov-config.json")
		loaded, err := LoadConfig(p)
		if err != nil || loaded.Language != want {
			t.Errorf("generated config language = %q, %v; want %q", loaded.Language, err, want)
		}
		raw, _ := os.ReadFile(p)
		if !strings.Contains(string(raw), "  },\n  \"language\": \""+want+"\",\n") {
			t.Errorf("lang %q: the language key must follow _agent_hints:\n%s", lang, raw)
		}
	}
	// A plain InitLayout (no config) states and writes en-US too.
	root := t.TempDir()
	if _, err := InitLayout(root, DefaultClaudeLayout()); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(root, ".claude", "CLAUDE.md")); !strings.Contains(string(data), "(en-US)") {
		t.Errorf("InitLayout must state en-US by default:\n%s", data)
	}
	if cfg, err := LoadConfig(FindConfig(root)); err != nil || cfg.Language != "en-US" {
		t.Errorf("InitLayout config language = %q, %v", cfg.Language, err)
	}
	if raw, _ := os.ReadFile(FindConfig(root)); !strings.Contains(string(raw), `"language": "en-US"`) {
		t.Errorf("the generated config must write the language key:\n%s", raw)
	}
}

// update-config adds language: en-US when the key is absent, right after
// the hints, and never overwrites an existing value without -language.
func TestUpgradeConfigWithLanguage_AddsDefaultWhenAbsent(t *testing.T) {
	for _, name := range []string{"repogov-config.json", "repogov-config.yaml"} {
		dir := t.TempDir()
		p := filepath.Join(dir, name)
		body := "{\n  \"default\": 300\n}\n"
		if isYAML(name) {
			body = "# header\ndefault: 300\n"
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if changed, err := UpgradeConfigHints(p); err != nil || !changed {
			t.Fatalf("%s: %v %v", name, changed, err)
		}
		data, _ := os.ReadFile(p)
		s := string(data)
		h, l := strings.Index(s, "_agent_hints"), strings.Index(s, "language")
		if cfg, err := LoadConfig(p); err != nil || cfg.Language != "en-US" || l < 0 || h > l {
			t.Errorf("%s: want language en-US written after the hints:\n%s", name, s)
		}
		if isYAML(name) && !strings.Contains(s, "\nlanguage: en-US\ndefault: 300\n") {
			t.Errorf("%s: language must sit right after the hints block:\n%s", name, s)
		}
		if !isYAML(name) && !strings.Contains(s, "  },\n  \"language\": \"en-US\",\n  \"default\": 300") {
			t.Errorf("%s: language must sit right after the hints member:\n%s", name, s)
		}
		// Idempotent.
		if changed, err := UpgradeConfigHints(p); err != nil || changed {
			t.Errorf("%s: second run must be a no-op: %v %v", name, changed, err)
		}
	}
}

func TestUpgradeConfigWithLanguage_AddsNeverOverwritesWithoutFlag(t *testing.T) {
	for _, name := range []string{"repogov-config.json", "repogov-config.yaml"} {
		dir := t.TempDir()
		p := filepath.Join(dir, name)
		body := "{\n  \"default\": 300\n}\n"
		if isYAML(name) {
			body = "# header\ndefault: 300\n"
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		// Added when absent (an explicit tag, not the en-US default).
		if changed, err := UpgradeConfigWithLanguage(p, "en_gb"); err != nil || !changed {
			t.Fatalf("%s: add: %v %v", name, changed, err)
		}
		cfg, err := LoadConfig(p)
		if err != nil || cfg.Language != "en-GB" || cfg.Default != 300 {
			t.Fatalf("%s: after add: %+v %v", name, cfg, err)
		}
		data, _ := os.ReadFile(p)
		if !strings.Contains(string(data), `"language": "en-GB"`) && !strings.Contains(string(data), "language: en-GB") {
			t.Errorf("%s: the config must state the language as its own value:\n%s", name, data)
		}
		// Without a language the key is never touched.
		if _, err := UpgradeConfigHints(p); err != nil {
			t.Fatal(err)
		}
		if changed, err := UpgradeConfigWithLanguage(p, ""); err != nil || changed {
			t.Errorf("%s: re-run without language must be a no-op: %v %v", name, changed, err)
		}
		if cfg, _ := LoadConfig(p); cfg.Language != "en-GB" {
			t.Errorf("%s: language overwritten without the flag: %q", name, cfg.Language)
		}
		// An explicit language replaces the value.
		if _, err := UpgradeConfigWithLanguage(p, "fr"); err != nil {
			t.Fatal(err)
		}
		if cfg, _ := LoadConfig(p); cfg.Language != "fr-FR" {
			t.Errorf("%s: explicit language not applied: %q", name, cfg.Language)
		}
		// An unknown tag is refused and nothing is written.
		before, _ := os.ReadFile(p)
		if _, err := UpgradeConfigWithLanguage(p, "en-NZ"); err == nil || !strings.Contains(err.Error(), "en-GB") {
			t.Errorf("%s: unknown tag must be refused naming close tags: %v", name, err)
		}
		if after, _ := os.ReadFile(p); !bytes.Equal(after, before) {
			t.Errorf("%s: refused update modified the file", name)
		}
	}
}
