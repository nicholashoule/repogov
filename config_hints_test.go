package repogov

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDefaultConfigJSON_AgentHints(t *testing.T) {
	out := defaultConfigJSON(DefaultConfig())

	var generated map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &generated); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}
	if _, ok := generated["_agent_hints"]; !ok {
		t.Fatal("generated config is missing _agent_hints")
	}

	// Hints must be ignored by the loader and not disturb real settings.
	path := t.TempDir() + "/repogov-config.json"
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig rejected generated config: %v", err)
	}
	if cfg.Default != DefaultConfig().Default {
		t.Errorf("Default = %d, want %d", cfg.Default, DefaultConfig().Default)
	}

	// The repo's own config must carry byte-for-byte the generator's hints.
	data, err := os.ReadFile("../../.github/repogov-config.json")
	if err != nil {
		data, err = os.ReadFile(".github/repogov-config.json")
	}
	if err != nil {
		t.Skipf("repo config not found: %v", err)
	}
	data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
	var own struct {
		Hints json.RawMessage `json:"_agent_hints"`
	}
	if err := json.Unmarshal(data, &own); err != nil {
		t.Fatal(err)
	}
	block := agentHints()
	want := block[strings.Index(block, "{") : strings.LastIndex(block, "}")+1]
	if string(own.Hints) != want {
		t.Errorf(".github/repogov-config.json _agent_hints differs from the generator; run 'repogov update-config' to sync\n got: %s\nwant: %s", own.Hints, want)
	}
}

// Regression: the hints block carried a duplicate, mis-indented "max_bytes"
// key. encoding/json accepts duplicates silently, so walk the token stream.
func TestAgentHints_NoDuplicateKeys(t *testing.T) {
	body := strings.TrimSuffix(strings.TrimSpace(agentHints()), ",")
	dec := json.NewDecoder(strings.NewReader("{" + body + "}"))
	if err := checkNoDuplicateKeys(dec); err != "" {
		t.Error(err)
	}
	if !json.Valid([]byte("{" + body + "}")) {
		t.Error("hints block is not valid JSON")
	}
}

// checkNoDuplicateKeys consumes one JSON value from dec and reports the first
// duplicate object key, or "".
func checkNoDuplicateKeys(dec *json.Decoder) string {
	tok, err := dec.Token()
	if err != nil {
		return err.Error()
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return ""
	}
	switch d {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				return err.Error()
			}
			key, _ := k.(string)
			if seen[key] {
				return "duplicate key " + key
			}
			seen[key] = true
			if msg := checkNoDuplicateKeys(dec); msg != "" {
				return msg
			}
		}
	case '[':
		for dec.More() {
			if msg := checkNoDuplicateKeys(dec); msg != "" {
				return msg
			}
		}
	}
	if _, err := dec.Token(); err != nil { // closing delimiter
		return err.Error()
	}
	return ""
}

// The hints document the language key in one line, in both forms, and
// every hint stays a one-liner an agent can act on: no entry grows into an
// essay (the reason the block was redesigned).
func TestAgentHints_LightweightAndDocumentLanguage(t *testing.T) {
	if !strings.Contains(defaultAgentHints, `"language": "Prose language (BCP 47, e.g. en-US)`) {
		t.Error("_agent_hints keys must document the language key")
	}
	if y := agentHintsYAML(); !strings.Contains(y, `language: "Prose language (BCP 47, e.g. en-US)`) {
		t.Errorf("YAML hints missing the language key:\n%s", y)
	}
	const maxHint = 100
	for _, line := range strings.Split(defaultAgentHints, "\n") {
		if _, v, ok := strings.Cut(strings.TrimSpace(line), `": "`); ok && len(v) > maxHint {
			t.Errorf("hint over %d characters (keep hints to one actionable line): %s", maxHint, strings.TrimSpace(line))
		} else if strings.HasPrefix(strings.TrimSpace(line), `"`) && !strings.Contains(line, `": `) && len(strings.TrimSpace(line)) > maxHint {
			t.Errorf("list hint over %d characters: %s", maxHint, strings.TrimSpace(line))
		}
	}
}

// Regression: yamlDoubleQuote did not escape backslashes or newlines, so a
// value could break out of its scalar.
func TestYAMLDoubleQuote_Escapes(t *testing.T) {
	cases := map[string]string{
		`a\b`:       `"a\\b"`,
		`say "hi"`:  `"say \"hi\""`,
		"one\ntwo":  `"one\ntwo"`,
		"tab\there": `"tab\there"`,
		"bell\x07":  `"bell\x07"`,
		"Bokmål":    `"Bokmål"`,
	}
	for in, want := range cases {
		if got := yamlDoubleQuote(in); got != want {
			t.Errorf("yamlDoubleQuote(%q) = %s, want %s", in, got, want)
		}
	}
}
