package repogov

import (
	"encoding/json"
	"os"
	"reflect"
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
	var own map[string]json.RawMessage
	if err := json.Unmarshal(data, &own); err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(own["_agent_hints"], &a)
	_ = json.Unmarshal(generated["_agent_hints"], &b)
	if !reflect.DeepEqual(a, b) {
		t.Error(".github/repogov-config.json _agent_hints differs from defaultAgentHints; regenerate or sync")
	}
}
