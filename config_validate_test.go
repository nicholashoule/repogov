package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// hasViolation reports whether vs contains a violation matching field and severity.
func hasViolation(vs []repogov.Violation, field, severity string) bool {
	for _, v := range vs {
		if v.Field == field && v.Severity == severity {
			return true
		}
	}
	return false
}

// hasAnySeverity reports whether vs contains any violation with the given severity.
func hasAnySeverity(vs []repogov.Violation, severity string) bool {
	for _, v := range vs {
		if v.Severity == severity {
			return true
		}
	}
	return false
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
	if !hasViolation(vs, "default", "error") {
		t.Error("negative default should produce an error violation")
	}
}

func TestValidateConfig_WarningThresholdOutOfRange(t *testing.T) {
	cfg := repogov.Config{Default: 500, WarningThreshold: 150}
	vs := repogov.ValidateConfig(cfg)
	if !hasViolation(vs, "warning_threshold", "error") {
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
	if !hasViolation(vs, "rules[0].glob", "error") {
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
	if !hasViolation(vs, "rules[1].glob", "warning") {
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
	if !hasAnySeverity(vs, "warning") {
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
	if !hasViolation(vs, `files["README.md"]`, "warning") {
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
			if !hasViolation(vs, `files["`+tc.path+`"]`, "error") {
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
			if !hasViolation(vs, tc.field, "error") {
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
	if !hasViolation(vs, "init_exclude_files[1]", "error") {
		t.Error("unsafe exclude stem should produce an error violation")
	}
	// The safe stem at index 0 must not produce an error.
	if hasViolation(vs, "init_exclude_files[0]", "error") {
		t.Error("safe stem raised unexpected violation")
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
	for _, field := range []string{"init_always_create", "descriptive_names", "skip_frontmatter"} {
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

func TestLoadConfig_SkipFrontmatter_JSON(t *testing.T) {
	data := `{"default": 300, "skip_frontmatter": true}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SkipFrontmatter {
		t.Error("expected SkipFrontmatter=true")
	}
}

func TestLoadConfig_SkipFrontmatter_DefaultFalse(t *testing.T) {
	data := `{"default": 300}`
	path := writeTempFile(t, "config.json", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SkipFrontmatter {
		t.Error("expected SkipFrontmatter=false when not specified")
	}
}

func TestLoadConfig_SkipFrontmatter_YAML(t *testing.T) {
	data := "default: 300\nskip_frontmatter: true\n"
	path := writeTempFile(t, "skip.yaml", data)

	cfg, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SkipFrontmatter {
		t.Error("expected SkipFrontmatter=true from YAML")
	}
}

func TestSaveAndLoadConfig_SkipFrontmatter_YAML(t *testing.T) {
	cfg := repogov.DefaultConfig()
	cfg.SkipFrontmatter = true

	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := repogov.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := repogov.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.SkipFrontmatter {
		t.Error("round-trip: expected SkipFrontmatter=true")
	}
}
