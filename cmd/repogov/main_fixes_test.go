package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// Regression: an explicit -config that does not exist was silently ignored
// and every check ran against the built-in defaults.
func TestRun_ExplicitConfigMissing_Exit2(t *testing.T) {
	for _, args := range [][]string{
		{"limits"}, {"layout"}, {"validate"}, {"all"}, {"update-config"},
		{"-agent", "claude", "init"},
	} {
		root := writeTempDir(t, map[string]string{"README.md": "# r\n"})
		stdout, stderr := bufs()
		full := append([]string{"-root", root, "-config", "missing.json"}, args...)
		if code := run(full, stdout, stderr); code != 2 {
			t.Errorf("%v: exit %d, want 2 (stderr: %s)", args, code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "not found") || !strings.Contains(stderr.String(), "-config") {
			t.Errorf("%v: stderr must say the -config file was not found: %s", args, stderr.String())
		}
		if strings.Contains(stdout.String(), "[PASS]") {
			t.Errorf("%v: nothing may pass against defaults: %s", args, stdout.String())
		}
	}
	// Auto-discovery with no config still falls back to defaults.
	root := writeTempDir(t, map[string]string{"README.md": "# r\n"})
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "limits"}, stdout, stderr); code != 0 {
		t.Errorf("auto-discovered missing config must use defaults: exit %d, %s", code, stderr.String())
	}
}

// Regression: flags after the subcommand were silently dropped.
func TestRun_FlagsAfterSubcommand(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"doc.md":      nlines(10),
		"custom.json": `{"default": 5}`,
	})
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "limits", "-config", "custom.json"}, stdout, stderr); code != 1 {
		t.Errorf("limits -config after the subcommand must apply (10 > 5 lines): exit %d\n%s%s", code, stdout.String(), stderr.String())
	}
	stdout, stderr = bufs()
	if code := run([]string{"limits", "-json", "-root", root}, stdout, stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "[") {
		t.Errorf("limits -json must print JSON: %s", stdout.String())
	}
	stdout, stderr = bufs()
	if code := run([]string{"-root", root, "validate", "extra"}, stdout, stderr); code != 2 || !strings.Contains(stderr.String(), "unexpected argument") {
		t.Errorf("stray argument must be refused with exit 2 and a reason: %d %s", code, stderr.String())
	}
}

// Regression: an explicit -root was walked up to the git root.
func TestRun_ExplicitRootNotWalkedUp(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "pkg", "svc")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := bufs()
	if code := run([]string{"-root", sub, "-agent", "claude", "init"}, stdout, stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(sub, ".claude", "CLAUDE.md")); err != nil {
		t.Errorf("explicit -root must be used as given: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude")); !os.IsNotExist(err) {
		t.Error("explicit -root was walked up to the git root")
	}
}

func TestWantPitfalls_CaseInsensitive(t *testing.T) {
	for _, a := range []string{"Cursor", "GEMINI", "claude, Cursor"} {
		if !wantPitfalls(a, false) {
			t.Errorf("wantPitfalls(%q) = false, want true", a)
		}
	}
	if wantPitfalls("claude", false) {
		t.Error("wantPitfalls(claude) = true, want false")
	}
}

func TestRun_InitLanguage(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "-agent", "claude", "init", "-language", "en_gb"}, stdout, stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	for _, f := range []string{".claude/CLAUDE.md", "AGENTS.md", ".claude/rules/general.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil || !strings.Contains(string(data), "Prose language: British English (en-GB)") {
			t.Errorf("%s must state the prose language (%v)", f, err)
		}
	}
	cfg, err := repogov.LoadConfig(filepath.Join(root, "repogov-config.json"))
	if err != nil || cfg.Language != "en-GB" {
		t.Errorf("generated config language = %q (%v), want en-GB", cfg.Language, err)
	}

	stdout, stderr = bufs()
	if code := run([]string{"-root", t.TempDir(), "-agent", "claude", "-language", "klingon", "init"}, stdout, stderr); code != 2 || !strings.Contains(stderr.String(), "en-US") {
		t.Errorf("unknown -language must exit 2 naming valid tags: %d %s", code, stderr.String())
	}
	stdout, stderr = bufs()
	if code := run([]string{"-root", root, "-language", "en-US", "limits"}, stdout, stderr); code != 2 || !strings.Contains(stderr.String(), "only to init and update-config") {
		t.Errorf("-language with limits must be refused: %d %s", code, stderr.String())
	}
}

func TestRun_UpdateConfigLanguage(t *testing.T) {
	root := writeTempDir(t, map[string]string{"repogov-config.json": "{\n  \"default\": 300\n}\n"})
	cfgPath := filepath.Join(root, "repogov-config.json")
	lang := func() string {
		cfg, err := repogov.LoadConfig(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		return cfg.Language
	}
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "update-config"}, stdout, stderr); code != 0 || lang() != "en-US" {
		t.Fatalf("update-config without -language must add the default en-US: %d %q %s", code, lang(), stderr.String())
	}
	if raw, _ := os.ReadFile(cfgPath); !strings.Contains(string(raw), `"language": "en-US"`) {
		t.Fatalf("the key must be written out:\n%s", raw)
	}
	if code := run([]string{"-root", root, "update-config", "-language", "pt"}, stdout, stderr); code != 0 || lang() != "pt-BR" {
		t.Fatalf("update-config -language pt: %d %q %s", code, lang(), stderr.String())
	}
	if code := run([]string{"-root", root, "update-config"}, stdout, stderr); code != 0 || lang() != "pt-BR" {
		t.Errorf("update-config without -language must never overwrite: %d %q", code, lang())
	}
}

// An existing AGENTS.md without the Context markers is left unchanged and
// init reports it as a note naming how to opt in; init still succeeds.
func TestRun_InitUnmarkedAgentsMdIsANote(t *testing.T) {
	const custom = "# Mine\n\n## Context\n\n- keep\n"
	for _, agent := range []string{"claude", "claude,cursor"} {
		root := writeTempDir(t, map[string]string{"AGENTS.md": custom})
		stdout, stderr := bufs()
		if code := run([]string{"-root", root, "-agent", agent, "init"}, stdout, stderr); code != 0 {
			t.Fatalf("%s: exit %d: %s", agent, code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "note: ") || !strings.Contains(stderr.String(), "<!-- repogov:context:begin -->") {
			t.Errorf("%s: init must say the Context section was not refreshed and how to opt in: %s", agent, stderr.String())
		}
		if data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md")); string(data) != custom {
			t.Errorf("%s: AGENTS.md modified:\n%s", agent, data)
		}
		// -json carries the same note as a notice field.
		stdout, stderr = bufs()
		if code := run([]string{"-root", root, "-agent", agent, "-json", "init"}, stdout, stderr); code != 0 || !strings.Contains(stdout.String(), `"notice": "AGENTS.md Context section not refreshed`) {
			t.Errorf("%s: -json must carry the notice: %d %s", agent, code, stdout.String())
		}
	}
	// layout shows it as an INFO, which never fails the run.
	root := writeTempDir(t, map[string]string{"AGENTS.md": custom + "\n" + "Prose language (en-US).\n"})
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "-agent", "claude", "layout"}, stdout, stderr); code == 2 {
		t.Fatalf("exit 2: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "[INFO] AGENTS.md -- AGENTS.md Context section not refreshed") {
		t.Errorf("layout must report the unmarked AGENTS.md as INFO:\n%s", stdout.String())
	}
}

// layout warns when the configured prose language is not stated in AGENTS.md.
func TestRun_LayoutWarnsWhenAgentsMdOmitsLanguage(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"AGENTS.md":           "# A\n",
		"repogov-config.json": `{"language": "en-US"}`,
	})
	stdout, stderr := bufs()
	if code := run([]string{"-root", root, "-agent", "claude", "layout"}, stdout, stderr); code == 2 {
		t.Fatalf("exit 2: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "[WARN] AGENTS.md -- does not state the prose language (en-US)") {
		t.Errorf("want a language WARN:\n%s", stdout.String())
	}
}
