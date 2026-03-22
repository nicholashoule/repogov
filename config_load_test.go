package repogov_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestLoadConfig_FileNotExist(t *testing.T) {
	cfg, err := repogov.LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Should return defaults.
	if cfg.Default != 500 {
		t.Errorf("Default = %d, want 500", cfg.Default)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	data := `{
		"default": 300,
		"warning_threshold": 90,
		"skip_dirs": [".git"],
		"rules": [{"glob": "*.md", "limit": 1000}],
		"files": {"README.md": 1200}
	}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300", cfg.Default)
	}
	if cfg.WarningThreshold != 90 {
		t.Errorf("WarningThreshold = %d, want 90", cfg.WarningThreshold)
	}
	if len(cfg.Rules) != 1 {
		t.Errorf("Rules count = %d, want 1", len(cfg.Rules))
	}
	if cfg.Files["README.md"] != 1200 {
		t.Errorf("Files[README.md] = %d, want 1200", cfg.Files["README.md"])
	}
}

func TestLoadConfig_PartialFile(t *testing.T) {
	// Partial config -- missing fields should fall back to defaults.
	data := `{"default": 250}`
	path := writeTempFile(t, "partial.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != 250 {
		t.Errorf("Default = %d, want 250", cfg.Default)
	}
	if cfg.WarningThreshold != 85 {
		t.Errorf("WarningThreshold should fall back to 85, got %d", cfg.WarningThreshold)
	}
	if cfg.SkipDirs == nil {
		t.Error("SkipDirs should fall back to defaults")
	}
	if cfg.Rules == nil {
		t.Error("Rules should fall back to defaults")
	}
	if cfg.Files == nil {
		t.Error("Files should fall back to defaults")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	path := writeTempFile(t, "bad.json", "{invalid json}")

	_, err := repogov.LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfig_WarningThresholdPercent_JSON(t *testing.T) {
	data := `{
		"default": 300,
		"warning_threshold": "90%",
		"rules": [{"glob": "*.md", "limit": 1000}]
	}`
	path := writeTempFile(t, "pct.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WarningThreshold != 90 {
		t.Errorf("WarningThreshold = %d, want 90", cfg.WarningThreshold)
	}
}

func TestLoadConfig_WarningThresholdBareInt_JSON(t *testing.T) {
	data := `{
		"default": 300,
		"warning_threshold": 85
	}`
	path := writeTempFile(t, "bare.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WarningThreshold != 85 {
		t.Errorf("WarningThreshold = %d, want 85", cfg.WarningThreshold)
	}
}

func TestSaveConfig(t *testing.T) {
	cfg := repogov.DefaultConfig()
	cfg.Files["test.md"] = 100

	path := filepath.Join(t.TempDir(), "out.json")
	if err := repogov.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Read it back.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded repogov.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Default != 500 {
		t.Errorf("round-trip Default = %d, want 500", loaded.Default)
	}
	if loaded.Files["test.md"] != 100 {
		t.Errorf("round-trip Files[test.md] = %d, want 100", loaded.Files["test.md"])
	}
}

func TestLoadConfig_MergesDefaultFiles(t *testing.T) {
	// User provides only files; default file entries should be merged.
	data := `{"files": {"README.md": 1200, "CHANGELOG.md": 0}}`
	path := writeTempFile(t, "merge.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	// User entries present.
	if cfg.Files["README.md"] != 1200 {
		t.Errorf("Files[README.md] = %d, want 1200", cfg.Files["README.md"])
	}
	// Default entry merged in.
	if cfg.Files[".github/copilot-instructions.md"] != 50 {
		t.Errorf("Files[.github/copilot-instructions.md] = %d, want 50 (default)",
			cfg.Files[".github/copilot-instructions.md"])
	}
	// memory.md and memory.instructions.md per-agent entries merged in at 200.
	for _, key := range []string{
		".github/rules/memory.md",
		".github/rules/memory.instructions.md",
		".github/instructions/memory.instructions.md",
		".cursor/rules/memory.md",
		".cursor/rules/memory.instructions.md",
		".windsurf/rules/memory.md",
		".windsurf/rules/memory.instructions.md",
		".claude/rules/memory.md",
		".claude/rules/memory.instructions.md",
		".kiro/steering/memory.md",
		".kiro/steering/memory.instructions.md",
		".continue/rules/memory.md",
		".continue/rules/memory.instructions.md",
		".clinerules/memory.md",
		".clinerules/memory.instructions.md",
		".roo/rules/memory.md",
		".roo/rules/memory.instructions.md",
		".aiassistant/rules/memory.md",
		".aiassistant/rules/memory.instructions.md",
	} {
		if cfg.Files[key] != 200 {
			t.Errorf("Files[%s] = %d, want 200 (default)", key, cfg.Files[key])
		}
	}
	// Rules should fall back to defaults when not specified.
	if cfg.Rules == nil {
		t.Error("Rules should fall back to defaults")
	}
}

func TestLoadConfig_DownstreamFormat(t *testing.T) {
	// Simulates the downstream config format with _-prefixed metadata.
	data := `{
		"_about": "Per-file line-limit overrides.",
		"_format": "Keys are repo-relative paths.",
		"_built_in": {
			".github/instructions/*.md": 300,
			"default": 500
		},
		"files": {
			"CHANGELOG.md": 0,
			"README.md": 1200,
			"docs/design.md": 750
		}
	}`
	path := writeTempFile(t, "downstream.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	// Underscore fields silently ignored; core fields populated.
	if cfg.Default != 500 {
		t.Errorf("Default = %d, want 500 (underscore keys ignored, falls back to default)", cfg.Default)
	}
	if cfg.Files["README.md"] != 1200 {
		t.Errorf("Files[README.md] = %d, want 1200", cfg.Files["README.md"])
	}
	if cfg.Files["docs/design.md"] != 750 {
		t.Errorf("Files[docs/design.md] = %d, want 750", cfg.Files["docs/design.md"])
	}
	// Default file entries merged.
	if cfg.Files[".github/copilot-instructions.md"] != 50 {
		t.Errorf("default copilot-instructions.md entry should be merged")
	}
	// Rules fall back to defaults when not in config.
	if len(cfg.Rules) < 1 {
		t.Errorf("Rules count = %d, want >= 1 (defaults)", len(cfg.Rules))
	}
}

func TestFindConfig_InGitHub(t *testing.T) {
	root := t.TempDir()
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
	}
}

func TestFindConfig_InRoot(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"default":250}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
	}
}

func TestFindConfig_PrefersRoot(t *testing.T) {
	root := t.TempDir()
	// Create both locations; root should win.
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ghCfg := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(ghCfg, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rootCfg := filepath.Join(root, "repogov-config.json")
	if err := os.WriteFile(rootCfg, []byte(`{"default":250}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != rootCfg {
		t.Errorf("FindConfig should prefer root, got %q, want %q", found, rootCfg)
	}
}

func TestFindConfig_NoneExist(t *testing.T) {
	root := t.TempDir()
	found := repogov.FindConfig(root)
	if found != "" {
		t.Errorf("FindConfig = %q, want empty string", found)
	}
}

func TestFindConfig_RepogovConfigJson(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"default":400}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
	}
}

func TestFindConfig_YAMLFile(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "repogov-config.yaml")
	if err := os.WriteFile(cfgPath, []byte("default: 400\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
	}
}

func TestFindConfig_YMLFile(t *testing.T) {
	root := t.TempDir()
	// Only .yml present; should still be found.
	cfgPath := filepath.Join(root, "repogov-config.yml")
	if err := os.WriteFile(cfgPath, []byte("default: 400\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
	}
}

func TestFindConfig_JSONOverYAML(t *testing.T) {
	root := t.TempDir()
	jsonPath := filepath.Join(root, "repogov-config.json")
	yamlPath := filepath.Join(root, "repogov-config.yaml")
	if err := os.WriteFile(jsonPath, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(yamlPath, []byte("default: 400\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != jsonPath {
		t.Errorf("FindConfig should prefer JSON over YAML, got %q", found)
	}
}

func TestLoadConfig_YAML(t *testing.T) {
	data := `default: 300
warning_threshold: 90
skip_dirs:
  - .git
  - vendor
rules:
  - glob: "*.md"
    limit: 200
files:
  README.md: 1200
  CHANGELOG.md: 0
`
	path := writeTempFile(t, "config.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300", cfg.Default)
	}
	if cfg.WarningThreshold != 90 {
		t.Errorf("WarningThreshold = %d, want 90", cfg.WarningThreshold)
	}
	if len(cfg.SkipDirs) != 2 {
		t.Errorf("SkipDirs count = %d, want 2", len(cfg.SkipDirs))
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Glob != "*.md" || cfg.Rules[0].Limit == nil || *cfg.Rules[0].Limit != 200 {
		t.Errorf("Rules = %+v, want [{*.md 200}]", cfg.Rules)
	}
	if cfg.Files["README.md"] != 1200 {
		t.Errorf("Files[README.md] = %d, want 1200", cfg.Files["README.md"])
	}
	// Default file entries should be merged.
	if cfg.Files[".github/copilot-instructions.md"] != 50 {
		t.Errorf("default copilot-instructions.md should be merged")
	}
}

func TestLoadConfig_YAML_Partial(t *testing.T) {
	data := "default: 250\n"
	path := writeTempFile(t, "partial.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != 250 {
		t.Errorf("Default = %d, want 250", cfg.Default)
	}
	if cfg.WarningThreshold != 85 {
		t.Errorf("WarningThreshold should fall back to 85, got %d", cfg.WarningThreshold)
	}
	if cfg.Rules == nil {
		t.Error("Rules should fall back to defaults")
	}
}

func TestLoadConfig_YAML_Comments(t *testing.T) {
	data := `# Top comment
default: 300 # inline comment
# skip_dirs section
skip_dirs:
  - .git # standard
`
	path := writeTempFile(t, "comments.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300", cfg.Default)
	}
	if len(cfg.SkipDirs) != 1 || cfg.SkipDirs[0] != ".git" {
		t.Errorf("SkipDirs = %v, want [.git]", cfg.SkipDirs)
	}
}

func TestLoadConfig_WarningThresholdPercent_YAML(t *testing.T) {
	data := "default: 300\nwarning_threshold: 90%\n"
	path := writeTempFile(t, "pct.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WarningThreshold != 90 {
		t.Errorf("WarningThreshold = %d, want 90", cfg.WarningThreshold)
	}
}

func TestSaveConfig_YAML(t *testing.T) {
	cfg := repogov.DefaultConfig()
	cfg.Files["test.md"] = 100

	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := repogov.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Load it back.
	loaded, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Default != 500 {
		t.Errorf("round-trip Default = %d, want 500", loaded.Default)
	}
	if loaded.Files["test.md"] != 100 {
		t.Errorf("round-trip Files[test.md] = %d, want 100", loaded.Files["test.md"])
	}
	if len(loaded.Rules) != len(cfg.Rules) {
		t.Errorf("round-trip Rules count = %d, want %d", len(loaded.Rules), len(cfg.Rules))
	}
}

func TestLoadConfig_YAML_InvalidValue(t *testing.T) {
	data := "default: abc\n"
	path := writeTempFile(t, "bad.yaml", data)

	_, err := repogov.LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML value")
	}
}
