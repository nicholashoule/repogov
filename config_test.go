package repogov_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestLoadConfig_FileNotExist(t *testing.T) {
	cfg, err := repogov.LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Should return defaults.
	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300", cfg.Default)
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
	if loaded.Default != 300 {
		t.Errorf("round-trip Default = %d, want 300", loaded.Default)
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
	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300 (underscore keys ignored, falls back to default)", cfg.Default)
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
	cfgPath := filepath.Join(ghDir, "repogov.json")
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
	cfgPath := filepath.Join(root, "repogov.json")
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
	ghCfg := filepath.Join(ghDir, "repogov.json")
	if err := os.WriteFile(ghCfg, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rootCfg := filepath.Join(root, "repogov.json")
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

func TestFindConfig_DotRepogov(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, ".repogov.json")
	if err := os.WriteFile(cfgPath, []byte(`{"default":350}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != cfgPath {
		t.Errorf("FindConfig = %q, want %q", found, cfgPath)
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
	cfgPath := filepath.Join(root, "repogov.yaml")
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
	cfgPath := filepath.Join(root, "repogov.yml")
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

func TestFindConfig_PrefersRepogovConfig(t *testing.T) {
	root := t.TempDir()
	repogovCfg := filepath.Join(root, "repogov-config.json")
	repogovPlain := filepath.Join(root, "repogov.json")
	if err := os.WriteFile(repogovCfg, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repogovPlain, []byte(`{"default":250}`), 0o644); err != nil {
		t.Fatal(err)
	}

	found := repogov.FindConfig(root)
	if found != repogovCfg {
		t.Errorf("FindConfig should prefer repogov-config.json, got %q", found)
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
	if loaded.Default != 300 {
		t.Errorf("round-trip Default = %d, want 300", loaded.Default)
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

func TestValidateConfig_Valid(t *testing.T) {
	cfg := repogov.DefaultConfig()
	vs := repogov.ValidateConfig(cfg)
	if len(vs) != 0 {
		t.Errorf("DefaultConfig should be valid, got %d violations", len(vs))
		for _, v := range vs {
			t.Logf("  [%s] %s: %s", v.Severity, v.Field, v.Message)
		}
	}
}

func TestValidateConfig_NegativeDefault(t *testing.T) {
	cfg := repogov.Config{Default: -1, WarningThreshold: 80}
	vs := repogov.ValidateConfig(cfg)
	hasError := false
	for _, v := range vs {
		if v.Field == "default" && v.Severity == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("negative default should produce an error violation")
	}
}

func TestValidateConfig_WarningThresholdOutOfRange(t *testing.T) {
	cfg := repogov.Config{Default: 500, WarningThreshold: 150}
	vs := repogov.ValidateConfig(cfg)
	hasError := false
	for _, v := range vs {
		if v.Field == "warning_threshold" && v.Severity == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("warning_threshold > 100 should produce an error violation")
	}
}

func TestValidateConfig_EmptyGlob(t *testing.T) {
	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 80,
		Rules:            []repogov.Rule{{Glob: "", Limit: repogov.RuleLimit(100)}},
	}
	vs := repogov.ValidateConfig(cfg)
	hasError := false
	for _, v := range vs {
		if v.Field == "rules[0].glob" && v.Severity == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("empty glob should produce an error violation")
	}
}

func TestValidateConfig_DuplicateGlob(t *testing.T) {
	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 80,
		Rules: []repogov.Rule{
			{Glob: "*.md", Limit: repogov.RuleLimit(300)},
			{Glob: "*.md", Limit: repogov.RuleLimit(200)},
		},
	}
	vs := repogov.ValidateConfig(cfg)
	hasWarning := false
	for _, v := range vs {
		if v.Severity == "warning" && v.Field == "rules[1].glob" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("duplicate globs should produce a warning")
	}
}

func TestValidateConfig_BackslashPath(t *testing.T) {
	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 80,
		Files:            map[string]int{"docs\\file.md": 100},
	}
	vs := repogov.ValidateConfig(cfg)
	hasWarning := false
	for _, v := range vs {
		if v.Severity == "warning" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("backslash in file path should produce a warning")
	}
}

func TestValidateConfig_FileOverridesRule(t *testing.T) {
	// A per-file limit higher than the matching rule should warn.
	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 80,
		Rules:            []repogov.Rule{{Glob: "*.md", Limit: repogov.RuleLimit(300)}},
		Files:            map[string]int{"README.md": 1200},
	}
	vs := repogov.ValidateConfig(cfg)
	hasWarning := false
	for _, v := range vs {
		if v.Severity == "warning" && v.Field == `files["README.md"]` {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("file with limit exceeding its rule should produce a warning")
	}
}

func TestValidateConfig_FileStricterThanRule(t *testing.T) {
	// A per-file limit stricter than the matching rule should NOT warn.
	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 80,
		Rules:            []repogov.Rule{{Glob: "*.md", Limit: repogov.RuleLimit(300)}},
		Files:            map[string]int{"README.md": 50},
	}
	vs := repogov.ValidateConfig(cfg)
	for _, v := range vs {
		if v.Field == `files["README.md"]` {
			t.Errorf("stricter per-file limit should not produce a violation, got: %s", v.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// isSafeFileSegment – tested indirectly through ValidateConfig and Init APIs.
// ---------------------------------------------------------------------------

func TestValidateConfig_UnsafeFilesKeySegment(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{"space in segment", "docs/my file.md"},
		{"path traversal", "../hack.md"},
		{"double-dot segment", ".github/../secret.md"},
		{"colon (Windows reserved)", "docs/file:name.md"},
		{"asterisk", "docs/*.md"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := repogov.Config{
				Default:          300,
				WarningThreshold: 80,
				Files:            map[string]int{tc.path: 200},
			}
			vs := repogov.ValidateConfig(cfg)
			hasError := false
			for _, v := range vs {
				if v.Severity == "error" && v.Field == `files["`+tc.path+`"]` {
					hasError = true
				}
			}
			if !hasError {
				t.Errorf("path %q should produce an error violation", tc.path)
			}
		})
	}
}

func TestValidateConfig_SafeFilesKey_NoFalsePositive(t *testing.T) {
	// DefaultConfig paths must not trigger the segment-safety check.
	cfg := repogov.DefaultConfig()
	vs := repogov.ValidateConfig(cfg)
	for _, v := range vs {
		if v.Severity == "error" && strings.Contains(v.Field, "files[") {
			t.Errorf("DefaultConfig files key raised unexpected error: [%s] %s", v.Field, v.Message)
		}
	}
}

func TestValidateConfig_UnsafeIncludeFiles(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stem  string
		field string
	}{
		{"path traversal", "../hack", "init_include_files[0]"},
		{"space in name", "my template", "init_include_files[0]"},
		{"double-dot", "..", "init_include_files[0]"},
		{"with extension and traversal", "../../etc/passwd.md", "init_include_files[0]"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := repogov.Config{
				Default:          300,
				WarningThreshold: 80,
				InitIncludeFiles: []string{tc.stem},
			}
			vs := repogov.ValidateConfig(cfg)
			hasError := false
			for _, v := range vs {
				if v.Severity == "error" && v.Field == tc.field {
					hasError = true
				}
			}
			if !hasError {
				t.Errorf("stem %q should produce an error on field %s", tc.stem, tc.field)
			}
		})
	}
}

func TestValidateConfig_UnsafeExcludeFiles(t *testing.T) {
	cfg := repogov.Config{
		Default:          300,
		WarningThreshold: 80,
		InitExcludeFiles: []string{"safe-stem", "bad stem!"},
	}
	vs := repogov.ValidateConfig(cfg)
	hasError := false
	for _, v := range vs {
		if v.Severity == "error" && v.Field == "init_exclude_files[1]" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("unsafe exclude stem should produce an error violation")
	}
	// The safe stem at index 0 must not produce an error.
	for _, v := range vs {
		if v.Field == "init_exclude_files[0]" {
			t.Errorf("safe stem raised unexpected violation: %s", v.Message)
		}
	}
}

func TestValidateConfig_SafeIncludeExcludeFiles_NoFalsePositive(t *testing.T) {
	cfg := repogov.Config{
		Default:          300,
		WarningThreshold: 80,
		InitIncludeFiles: []string{"general", "testing", "repo.md", "emoji-prevention"},
		InitExcludeFiles: []string{"backend", "frontend"},
	}
	vs := repogov.ValidateConfig(cfg)
	for _, v := range vs {
		if v.Severity == "error" &&
			(strings.HasPrefix(v.Field, "init_include_files") || strings.HasPrefix(v.Field, "init_exclude_files")) {
			t.Errorf("safe stem raised unexpected error: [%s] %s", v.Field, v.Message)
		}
	}
}

func TestFindAllConfigs_None(t *testing.T) {
	root := t.TempDir()
	all := repogov.FindAllConfigs(root)
	if len(all) != 0 {
		t.Errorf("expected no configs, got %v", all)
	}
}

func TestFindAllConfigs_OnlyGitHub(t *testing.T) {
	root := t.TempDir()
	ghCfg := filepath.Join(root, ".github", "repogov-config.json")
	if err := os.MkdirAll(filepath.Dir(ghCfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ghCfg, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	all := repogov.FindAllConfigs(root)
	if len(all) != 1 || all[0] != ghCfg {
		t.Errorf("expected [%s], got %v", ghCfg, all)
	}
}

func TestLoadConfig_InitAlwaysCreate(t *testing.T) {
	data := `{"default": 300, "init_always_create": true}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InitAlwaysCreate {
		t.Error("expected InitAlwaysCreate=true")
	}
}

func TestLoadConfig_InitAlwaysCreate_DefaultFalse(t *testing.T) {
	// When the field is absent it must default to false.
	data := `{"default": 300}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitAlwaysCreate {
		t.Error("expected InitAlwaysCreate=false when not specified")
	}
}

func TestLoadConfig_Descriptive(t *testing.T) {
	data := `{"default": 300, "descriptive_names": true}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DescriptiveNames {
		t.Error("expected Descriptive=true")
	}
}

func TestLoadConfig_Descriptive_DefaultFalse(t *testing.T) {
	// When the field is absent it must default to false.
	data := `{"default": 300}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DescriptiveNames {
		t.Error("expected Descriptive=false when not specified")
	}
}

func TestFindAllConfigs_BothPresent(t *testing.T) {
	root := t.TempDir()
	rootCfg := filepath.Join(root, "repogov-config.json")
	ghCfg := filepath.Join(root, ".github", "repogov-config.json")
	if err := os.MkdirAll(filepath.Dir(ghCfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootCfg, []byte(`{"default":400}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ghCfg, []byte(`{"default":300}`), 0o644); err != nil {
		t.Fatal(err)
	}
	all := repogov.FindAllConfigs(root)
	if len(all) != 2 {
		t.Fatalf("expected 2 configs, got %d: %v", len(all), all)
	}
	if all[0] != rootCfg {
		t.Errorf("first (active) config should be root: got %s", all[0])
	}
	if all[1] != ghCfg {
		t.Errorf("second (overridden) config should be .github: got %s", all[1])
	}
}

// TestLoadConfig_YAML_NewFields verifies that the YAML parser correctly handles
// init_always_create, descriptive_names, init_include_files, and init_exclude_files.
func TestLoadConfig_YAML_NewFields(t *testing.T) {
	data := `default: 300
init_always_create: true
descriptive_names: true
init_include_files:
  - general
  - testing
init_exclude_files:
  - backend
  - frontend
`
	path := writeTempFile(t, "newfields.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InitAlwaysCreate {
		t.Error("expected InitAlwaysCreate=true from YAML")
	}
	if !cfg.DescriptiveNames {
		t.Error("expected DescriptiveNames=true from YAML")
	}
	if len(cfg.InitIncludeFiles) != 2 || cfg.InitIncludeFiles[0] != "general" || cfg.InitIncludeFiles[1] != "testing" {
		t.Errorf("InitIncludeFiles = %v, want [general testing]", cfg.InitIncludeFiles)
	}
	if len(cfg.InitExcludeFiles) != 2 || cfg.InitExcludeFiles[0] != "backend" || cfg.InitExcludeFiles[1] != "frontend" {
		t.Errorf("InitExcludeFiles = %v, want [backend frontend]", cfg.InitExcludeFiles)
	}
}

// TestLoadConfig_YAML_NewFields_Defaults verifies that the YAML parser leaves
// new fields at their zero values when absent.
func TestLoadConfig_YAML_NewFields_Defaults(t *testing.T) {
	data := "default: 300\n"
	path := writeTempFile(t, "nofields.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitAlwaysCreate {
		t.Error("expected InitAlwaysCreate=false when absent")
	}
	if cfg.DescriptiveNames {
		t.Error("expected DescriptiveNames=false when absent")
	}
	if len(cfg.InitIncludeFiles) != 0 {
		t.Errorf("expected empty InitIncludeFiles, got %v", cfg.InitIncludeFiles)
	}
	if len(cfg.InitExcludeFiles) != 0 {
		t.Errorf("expected empty InitExcludeFiles, got %v", cfg.InitExcludeFiles)
	}
}

// TestSaveAndLoadConfig_YAML_NewFields verifies that new fields round-trip
// correctly through YAML marshaling and unmarshaling.
func TestSaveAndLoadConfig_YAML_NewFields(t *testing.T) {
	cfg := repogov.DefaultConfig()
	cfg.InitAlwaysCreate = true
	cfg.DescriptiveNames = true
	cfg.InitIncludeFiles = []string{"general", "security"}
	cfg.InitExcludeFiles = []string{"emoji-prevention"}

	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := repogov.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.InitAlwaysCreate {
		t.Error("round-trip: expected InitAlwaysCreate=true")
	}
	if !loaded.DescriptiveNames {
		t.Error("round-trip: expected DescriptiveNames=true")
	}
	if len(loaded.InitIncludeFiles) != 2 || loaded.InitIncludeFiles[0] != "general" {
		t.Errorf("round-trip InitIncludeFiles = %v, want [general security]", loaded.InitIncludeFiles)
	}
	if len(loaded.InitExcludeFiles) != 1 || loaded.InitExcludeFiles[0] != "emoji-prevention" {
		t.Errorf("round-trip InitExcludeFiles = %v, want [emoji-prevention]", loaded.InitExcludeFiles)
	}
}

// TestLoadConfig_YAML_InvalidBool verifies that invalid boolean values in YAML
// produce a meaningful error.
func TestLoadConfig_YAML_InvalidBool(t *testing.T) {
	for _, field := range []string{"init_always_create", "descriptive_names"} {
		field := field
		t.Run(field, func(t *testing.T) {
			data := field + ": notabool\n"
			path := writeTempFile(t, field+".yaml", data)
			_, err := repogov.LoadConfig(path)
			if err == nil {
				t.Errorf("expected error for invalid bool in field %s", field)
			}
		})
	}
}
