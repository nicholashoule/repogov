package repogov

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// hostileConfig holds values a config may carry that break naive quoting: a
// quote, a trailing backslash (which escapes a closing quote), a newline and
// a key-injection attempt.
func hostileConfig() Config {
	cfg := DefaultConfig()
	cfg.SkipDirs = []string{`a"b`, `c\`, "d\ne", `x", "injected": "1`}
	cfg.IncludeExts = []string{".md", `.q"`}
	cfg.Rules = []Rule{{Glob: `docs/"*.md`, Limit: RuleLimit(10)}, {Glob: `tail\`}}
	cfg.MaxBytes = []Rule{{Glob: `z\`, Limit: RuleLimit(24000)}}
	cfg.Files = map[string]int{`p"q.md`: 5, `back\slash.md`: 7}
	return cfg
}

// Regression (CodeQL go/unsafe-quoting on PR #19): defaultConfigJSON wrote
// every string by concatenation, so a value holding a quote or ending in a
// backslash closed its string early and could inject keys. Every value now
// goes through jsonString: the output parses, carries no injected key, and
// reads back exactly what went in.
func TestDefaultConfigJSON_QuotesEveryValue(t *testing.T) {
	cfg := hostileConfig()
	out := defaultConfigJSON(cfg)
	if !json.Valid([]byte(out)) {
		t.Fatalf("generated config is not valid JSON:\n%s", out)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &top); err != nil {
		t.Fatal(err)
	}
	if _, injected := top["injected"]; injected {
		t.Fatal("a skip_dirs value injected a top-level key")
	}
	var got struct {
		SkipDirs    []string       `json:"skip_dirs"`
		IncludeExts []string       `json:"include_exts"`
		Rules       []Rule         `json:"rules"`
		MaxBytes    []Rule         `json:"max_bytes"`
		Files       map[string]int `json:"files"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.SkipDirs, cfg.SkipDirs) || !slices.Equal(got.IncludeExts, cfg.IncludeExts) {
		t.Errorf("lists did not round-trip: %q %q", got.SkipDirs, got.IncludeExts)
	}
	if got.Rules[0].Glob != cfg.Rules[0].Glob || got.Rules[1].Glob != cfg.Rules[1].Glob || got.MaxBytes[0].Glob != cfg.MaxBytes[0].Glob {
		t.Errorf("globs did not round-trip: %+v %+v", got.Rules, got.MaxBytes)
	}
	if !reflect.DeepEqual(got.Files, cfg.Files) {
		t.Errorf("files did not round-trip: %v", got.Files)
	}
}

// The YAML writer escaped only quotes, so a value ending in a backslash
// escaped its own closing quote, and the reader never reversed the escapes.
// SaveConfig to YAML and LoadConfig back must return exactly the values.
func TestYAMLConfig_QuotesEveryValueAndRoundTrips(t *testing.T) {
	cfg := hostileConfig()
	cfg.SkipDirs = cfg.SkipDirs[:3] // a newline-free injection attempt is the JSON test's
	p := filepath.Join(t.TempDir(), "repogov-config.yaml")
	if err := SaveConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("a saved YAML config does not load back: %v", err)
	}
	if !slices.Equal(got.SkipDirs, cfg.SkipDirs) || !slices.Equal(got.IncludeExts, cfg.IncludeExts) {
		t.Errorf("lists did not round-trip: %q %q", got.SkipDirs, got.IncludeExts)
	}
	globs := func(rs []Rule) []string {
		var out []string
		for _, r := range rs {
			out = append(out, r.Glob)
		}
		return out
	}
	if !slices.Equal(globs(got.Rules), globs(cfg.Rules)) || !slices.Equal(globs(got.MaxBytes), globs(cfg.MaxBytes)) {
		t.Errorf("globs did not round-trip: %q %q", globs(got.Rules), globs(got.MaxBytes))
	}
	// LoadConfig merges the default files entries in; every written key
	// must read back with its own value.
	written := make([]string, 0, len(cfg.Files))
	for k := range cfg.Files {
		written = append(written, k)
	}
	sort.Strings(written)
	for _, k := range written {
		if v, ok := got.Files[k]; !ok || v != cfg.Files[k] {
			t.Errorf("files[%q] = %d, %v; want %d", k, v, ok, cfg.Files[k])
		}
	}
}

// yamlUnquote reverses exactly what yamlQuote writes, and single-quoted
// scalars unescape their doubled quote.
func TestYAMLQuoteUnquoteInverse(t *testing.T) {
	for _, s := range []string{"", "plain", `a"b`, `c\`, `\\`, "d\ne", "t\tab", "ctl\x01", `x": "y`, "it's"} {
		if got := yamlUnquote(yamlQuote(s)); got != s {
			t.Errorf("yamlUnquote(yamlQuote(%q)) = %q", s, got)
		}
	}
	if got := yamlUnquote(`'it''s'`); got != "it's" {
		t.Errorf("single-quoted unescape = %q", got)
	}
}
