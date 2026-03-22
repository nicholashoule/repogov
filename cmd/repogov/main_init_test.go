package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// --- runInit tests ---

func TestRunInit_RequiresPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "", "", true, false, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 when no platform given, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-agent or -platform is required") {
		t.Fatalf("expected usage error, got: %s", stderr.String())
	}
}

// TestRunInit_EachPlatformToTemp inits each platform into an isolated temp
// directory (named "temp" within the test root) and then validates the layout.
func TestRunInit_EachPlatformToTemp(t *testing.T) {
	tests := []struct {
		platform string
		dir      string
	}{
		{"copilot", ".github"},
		{"cursor", ".cursor"},
		{"windsurf", ".windsurf"},
		{"claude", ".claude"},
	}
	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			temp := filepath.Join(t.TempDir(), "temp")
			if err := os.MkdirAll(temp, 0o755); err != nil {
				t.Fatal(err)
			}
			stdout, stderr := bufs()

			// Init the platform layout.
			if code := runInit(temp, "", tc.platform, "", true, false, false, false, stdout, stderr); code != 0 {
				t.Fatalf("runInit %s: exit %d (stderr: %s)", tc.platform, code, stderr.String())
			}

			// Verify the platform root directory was created.
			if _, err := os.Stat(filepath.Join(temp, tc.dir)); err != nil {
				t.Fatalf("%s: %s not created: %v", tc.platform, tc.dir, err)
			}

			// Validate the layout passes.
			if code := runLayout(temp, "", tc.platform, "", false, false, stdout, stderr); code != 0 {
				t.Fatalf("runLayout %s: exit %d (stdout: %s)", tc.platform, code, stdout.String())
			}
		})
	}
}

// TestRunInit_AllToTemp inits all platforms into a single temp directory,
// validates all layouts, and confirms idempotency.
func TestRunInit_AllToTemp(t *testing.T) {
	temp := filepath.Join(t.TempDir(), "temp")
	if err := os.MkdirAll(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := bufs()

	// First init: create all platforms.
	if code := runInit(temp, "", "all", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit all: exit %d (stderr: %s)", code, stderr.String())
	}

	// Validate all layouts pass.
	if code := runLayout(temp, "", "all", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout all: exit %d (stdout: %s)", code, stdout.String())
	}

	// Second init: must be idempotent (nothing new created).
	stdout.Reset()
	if code := runInit(temp, "", "all", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("second runInit all: exit %d", code)
	}
	if strings.Contains(stdout.String(), "Scaffolded") {
		t.Fatalf("second init should have created nothing; got: %s", stdout.String())
	}
}

func TestRunInit_CreatesLayout(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".github")); err != nil {
		t.Fatalf(".github not created: %v", err)
	}
	// On a fresh init, rules/ is created (instructions/ only when pre-existing).
	if _, err := os.Stat(filepath.Join(root, ".github", "rules")); err != nil {
		t.Fatalf(".github/rules not created: %v", err)
	}
}

func TestRunInit_GitLab(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "", "gitlab", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".gitlab")); err != nil {
		t.Fatalf(".gitlab not created: %v", err)
	}
}

func TestRunInit_Cursor(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "cursor", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "rules")); err != nil {
		t.Fatalf(".cursor/rules not created: %v", err)
	}
}

func TestRunInit_Windsurf(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "windsurf", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".windsurf", "rules")); err != nil {
		t.Fatalf(".windsurf/rules not created: %v", err)
	}
}

func TestRunInit_Claude(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "claude", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	for _, dir := range []string{".claude/rules", ".claude/agents"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil {
			t.Fatalf("%s not created: %v", dir, err)
		}
	}
}

func TestRunInit_All(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "all", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
	for _, dir := range []string{".github", ".cursor/rules", ".windsurf/rules", ".claude/rules", ".claude/agents"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil {
			t.Fatalf("%s not created: %v", dir, err)
		}
	}
}

func TestRunInit_All_JSON(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "all", "", false, true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 combined entry for all platforms, got %d", len(results))
	}
	if results[0].Platform != "all" {
		t.Errorf("expected platform \"all\", got %q", results[0].Platform)
	}
	if len(results[0].Created) == 0 {
		t.Error("expected non-empty created list")
	}
}

func TestRunInit_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunInit_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "unknownplatform", "", true, false, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRunInit_JSON(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", "", false, true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Platform != "copilot" {
		t.Fatalf("expected github, got %s", result.Platform)
	}
	if len(result.Created) == 0 {
		t.Fatal("expected at least one created path")
	}
}

func TestRunInit_JSON_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", "", false, true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Created []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Created) != 0 {
		t.Fatalf("expected 0 created, got %d", len(result.Created))
	}
}

func TestRunInit_Verbose(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := runInit(root, "", "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Scaffolded") {
		t.Fatalf("expected 'Scaffolded', got: %s", stdout.String())
	}
}

func TestRunInit_Verbose_AlreadyExists(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", "", true, false, false, false, stdout, stderr)
	stdout.Reset()
	if code := runInit(root, "", "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "nothing to create") {
		t.Fatalf("expected 'nothing to create', got: %s", stdout.String())
	}
}

// --- runValidate tests ---

func TestRunValidate_ValidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": 80}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_InvalidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1, "warning_threshold": 200}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunValidate_NoConfig_Quiet(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunValidate_NoConfig_Verbose(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "No config file found") {
		t.Fatalf("expected error message, got: %s", stderr.String())
	}
}

func TestRunValidate_BadConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": "not-a-number"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunValidate_JSON_Valid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.Valid {
		t.Fatal("expected valid=true")
	}
}

func TestRunValidate_JSON_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var result struct {
		Valid      bool                `json:"valid"`
		Violations []repogov.Violation `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Valid || len(result.Violations) == 0 {
		t.Fatal("expected violations")
	}
}

func TestRunValidate_WarningsOnly(t *testing.T) {
	// A backslash in a file path triggers a warning (severity="warning", not "error").
	// runValidate should print WARNING and return 0.
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": "{\"default\": 300, \"files\": {\"a\\\\b.md\": 100}}",
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0 (warnings only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARNING") {
		t.Fatalf("expected WARNING in output, got: %s", stdout.String())
	}
}

func TestRunValidate_CustomConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{"myconfig.json": `{"default": 300}`})
	stdout, stderr := bufs()
	if code := runValidate(root, filepath.Join(root, "myconfig.json"), true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_RelativeConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{"myconfig.json": `{"default": 300}`})
	stdout, stderr := bufs()
	if code := runValidate(root, "myconfig.json", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunValidate_Verbose_Valid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Fatalf("expected 'is valid', got: %s", stdout.String())
	}
}

func TestRunValidate_Verbose_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), "[FAIL]") {
		t.Fatalf("expected '[FAIL]', got: %s", stdout.String())
	}
}
