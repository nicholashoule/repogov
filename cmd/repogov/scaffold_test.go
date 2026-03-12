package main

// scaffold_test.go exercises every agent and subcommand combination and
// writes the generated file trees to ./tmp so the output can be inspected
// after a run.  TestMain wipes and recreates ./tmp before each run.
//
// Test layout under ./tmp:
//   tmp/copilot/   -- repogov -root . -agent copilot init
//   tmp/cursor/    -- repogov -root . -agent cursor  init
//   tmp/windsurf/  -- repogov -root . -agent windsurf init
//   tmp/claude/    -- repogov -root . -agent claude   init
//   tmp/all/       -- repogov -root . -agent all      init
//
// Error-path tests use t.TempDir() since they test failure conditions
// rather than inspecting what was created.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tmpRoot is the persistent output directory, located next to this file so
// that generated scaffolding is easy to inspect after a run.
const tmpRoot = "tmp"

func TestMain(m *testing.M) {
	// Wipe and recreate ./tmp before every run so the on-disk state always
	// reflects the most recent test execution.
	if err := os.RemoveAll(tmpRoot); err != nil {
		panic("scaffold_test: cannot remove " + tmpRoot + ": " + err.Error())
	}
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		panic("scaffold_test: cannot create " + tmpRoot + ": " + err.Error())
	}
	os.Exit(m.Run())
}

// scaffoldDir returns the absolute path to the per-agent tmp subdirectory
// and creates it if it does not yet exist.
func scaffoldDir(t *testing.T, agent string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(tmpRoot, agent))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// assertFileExists fails the test when path does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

// assertDirExists fails the test when path is not a directory.
func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("expected directory to exist: %s", path)
		return
	}
	if err != nil || !info.IsDir() {
		t.Errorf("expected a directory at: %s", path)
	}
}

// assertFileContains fails when path does not contain substr.
func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("%s: expected to contain %q", path, substr)
	}
}

// assertFileNotContains fails when path contains substr.
func assertFileNotContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		// If the file doesn't exist the assertion trivially passes.
		return
	}
	if strings.Contains(string(data), substr) {
		t.Errorf("%s: should not contain %q", path, substr)
	}
}

// assertValidJSON fails when the file at path is not valid JSON.
func assertValidJSON(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("%s: invalid JSON: %v", path, err)
	}
}

// assertConfigHasDefault fails when the JSON config at path does not have a
// positive "default" field.
func assertConfigHasDefault(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	var v struct {
		Default int `json:"default"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("%s: invalid JSON: %v", path, err)
		return
	}
	if v.Default <= 0 {
		t.Errorf("%s: default must be > 0, got %d", path, v.Default)
	}
}

// -------------------------------------------------------------------
// Copilot
// -------------------------------------------------------------------

func TestScaffold_Copilot_Init(t *testing.T) {
	root := scaffoldDir(t, "copilot")
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit copilot: exit %d\nstderr: %s", code, stderr.String())
	}

	// Output must report items created.
	if !strings.Contains(stdout.String(), "Scaffolded") {
		t.Errorf("expected 'Scaffolded' in stdout, got: %s", stdout.String())
	}

	// Required directories: on a fresh Copilot init only rules/ is created
	// (instructions/ is only used when it already exists before init).
	assertDirExists(t, filepath.Join(root, ".github"))
	assertDirExists(t, filepath.Join(root, ".github", "rules"))

	// copilot-instructions.md content.
	ciPath := filepath.Join(root, ".github", "copilot-instructions.md")
	assertFileExists(t, ciPath)
	assertFileContains(t, ciPath, "# Copilot Instructions")
	assertFileContains(t, ciPath, "## Context")
	assertFileContains(t, ciPath, "File Constraints")
	assertFileContains(t, ciPath, "Repository Commands")
	assertFileContains(t, ciPath, "-agent copilot")

	// Default naming convention (DescriptiveNames=false): all templates seeded
	// as plain <name>.md files into rules/.
	for _, name := range []string{
		"general.md", "codereview.md", "governance.md",
		"library.md", "testing.md", "emoji-prevention.md",
		"backend.md", "frontend.md", "security.md", "repo.md",
	} {
		p := filepath.Join(root, ".github", "rules", name)
		assertFileExists(t, p)
		assertFileContains(t, p, "---") // YAML frontmatter delimiter
	}

	// Config must be in .github/ and be valid JSON with a positive default.
	cfgPath := filepath.Join(root, ".github", "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	// AGENTS.md must exist at the repo root with correct context links.
	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, "copilot-instructions.md")
	assertFileContains(t, agPath, ".github/rules/")
	assertFileNotContains(t, agPath, ".github/instructions/")

	// AGENTS.md must NOT contain links for other platforms.
	assertFileNotContains(t, agPath, ".cursor/")
	assertFileNotContains(t, agPath, ".windsurf/")
	assertFileNotContains(t, agPath, ".claude/")

	// Layout must pass after init.
	stdout.Reset()
	if code := runLayout(root, "copilot", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout copilot after init: exit %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "Layout:") {
		t.Errorf("expected Layout summary in stdout, got: %s", stdout.String())
	}

	// limits must pass on the generated files.
	stdout.Reset()
	if code := runLimits(root, "", "", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLimits copilot after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// TestScaffold_Copilot_Init_Descriptive verifies that a fresh .github init with
// descriptive=true in the config seeds all *.instructions.md files into rules/.
func TestScaffold_Copilot_Init_Descriptive(t *testing.T) {
	root := scaffoldDir(t, "copilot-descriptive")
	stdout, stderr := bufs()

	// Pre-create .github/ with a full working config that enables descriptive naming.
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgData := "{\n" +
		"  \"default\": 300,\n" +
		"  \"descriptive_names\": true,\n" +
		"  \"warning_threshold\": \"85%\",\n" +
		"  \"skip_dirs\": [\".git\", \"vendor\"],\n" +
		"  \"include_exts\": [\".md\", \".mdc\"],\n" +
		"  \"rules\": [\n" +
		"    {\"glob\": \".github/rules/*.md\", \"limit\": 300}\n" +
		"  ],\n" +
		"  \"files\": {\n" +
		"    \".github/copilot-instructions.md\": 50,\n" +
		"    \"AGENTS.md\": 200\n" +
		"  }\n" +
		"}\n"
	cfgPath := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runInit(root, cfgPath, "copilot", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit copilot descriptive: exit %d\nstderr: %s", code, stderr.String())
	}

	// Every default instruction file must exist in rules/ with *.instructions.md names.
	instructionFiles := []string{
		"general.instructions.md",
		"codereview.instructions.md",
		"governance.instructions.md",
		"library.instructions.md",
		"testing.instructions.md",
		"emoji-prevention.instructions.md",
		"backend.instructions.md",
		"frontend.instructions.md",
		"repo.instructions.md",
	}
	for _, name := range instructionFiles {
		p := filepath.Join(root, ".github", "rules", name)
		assertFileExists(t, p)
		assertFileContains(t, p, "---") // YAML frontmatter delimiter
	}

	// governance.instructions.md must contain key sections.
	govPath := filepath.Join(root, ".github", "rules", "governance.instructions.md")
	assertFileContains(t, govPath, "## Line Limits")
	assertFileContains(t, govPath, "## Enforcing Limits")
	assertFileContains(t, govPath, "### Minimal CLI Example")

	// emoji-prevention.instructions.md must include the text-alternatives table.
	emojiPath := filepath.Join(root, ".github", "rules", "emoji-prevention.instructions.md")
	assertFileContains(t, emojiPath, "## Enforcement")
	assertFileContains(t, emojiPath, "demojify")
}

// -------------------------------------------------------------------
// Cursor
// -------------------------------------------------------------------

func TestScaffold_Cursor_Init(t *testing.T) {
	root := scaffoldDir(t, "cursor")
	stdout, stderr := bufs()

	if code := runInit(root, "", "cursor", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit cursor: exit %d\nstderr: %s", code, stderr.String())
	}

	// Required directories.
	assertDirExists(t, filepath.Join(root, ".cursor"))
	assertDirExists(t, filepath.Join(root, ".cursor", "rules"))

	// general.md must exist with standard instruction frontmatter (full template set).
	mdPath := filepath.Join(root, ".cursor", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	// Config must exist within .cursor/ (no .github/ present for this init).
	cfgPath := filepath.Join(root, ".cursor", "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	// AGENTS.md context.
	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".cursor/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")
	assertFileNotContains(t, agPath, ".github/instructions/")
	assertFileNotContains(t, agPath, ".windsurf/")
	assertFileNotContains(t, agPath, ".claude/")

	// Layout passes.
	if code := runLayout(root, "cursor", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout cursor after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Windsurf
// -------------------------------------------------------------------

func TestScaffold_Windsurf_Init(t *testing.T) {
	root := scaffoldDir(t, "windsurf")
	stdout, stderr := bufs()

	if code := runInit(root, "", "windsurf", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit windsurf: exit %d\nstderr: %s", code, stderr.String())
	}

	// Required directories.
	assertDirExists(t, filepath.Join(root, ".windsurf"))
	assertDirExists(t, filepath.Join(root, ".windsurf", "rules"))

	// general.md must exist with standard instruction frontmatter and content.
	mdPath := filepath.Join(root, ".windsurf", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	// Config must exist within .windsurf/.
	cfgPath := filepath.Join(root, ".windsurf", "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	// AGENTS.md context.
	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".windsurf/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")
	assertFileNotContains(t, agPath, ".cursor/")
	assertFileNotContains(t, agPath, ".claude/")

	// Layout passes.
	if code := runLayout(root, "windsurf", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout windsurf after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Claude
// -------------------------------------------------------------------

func TestScaffold_Claude_Init(t *testing.T) {
	root := scaffoldDir(t, "claude")
	stdout, stderr := bufs()

	if code := runInit(root, "", "claude", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit claude: exit %d\nstderr: %s", code, stderr.String())
	}

	// Required directories.
	assertDirExists(t, filepath.Join(root, ".claude"))
	assertDirExists(t, filepath.Join(root, ".claude", "rules"))
	assertDirExists(t, filepath.Join(root, ".claude", "agents"))

	// CLAUDE.md must exist with context section and {{.Agent}} rendered.
	claudePath := filepath.Join(root, ".claude", "CLAUDE.md")
	assertFileExists(t, claudePath)
	assertFileContains(t, claudePath, "# CLAUDE.md")
	assertFileContains(t, claudePath, "## Context")
	assertFileContains(t, claudePath, ".claude/rules/")
	assertFileContains(t, claudePath, ".claude/agents/")
	assertFileContains(t, claudePath, "-agent claude")
	assertFileNotContains(t, claudePath, "{{.Agent}}")

	// general.md must exist with standard instruction frontmatter and content.
	mdPath := filepath.Join(root, ".claude", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	// Config must exist within .claude/.
	cfgPath := filepath.Join(root, ".claude", "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	// AGENTS.md must have both claude dirs linked but not other platforms.
	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".claude/rules/")
	assertFileContains(t, agPath, ".claude/agents/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")
	assertFileNotContains(t, agPath, ".cursor/")
	assertFileNotContains(t, agPath, ".windsurf/")

	// Layout passes.
	if code := runLayout(root, "claude", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout claude after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Kiro
// -------------------------------------------------------------------

func TestScaffold_Kiro_Init(t *testing.T) {
	root := scaffoldDir(t, "kiro")
	stdout, stderr := bufs()

	if code := runInit(root, "", "kiro", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit kiro: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".kiro"))
	assertDirExists(t, filepath.Join(root, ".kiro", "steering"))

	// general.md must be seeded into steering/.
	mdPath := filepath.Join(root, ".kiro", "steering", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".kiro/steering/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")
	assertFileNotContains(t, agPath, ".cursor/")

	if code := runLayout(root, "kiro", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout kiro after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Gemini
// -------------------------------------------------------------------

func TestScaffold_Gemini_Init(t *testing.T) {
	root := scaffoldDir(t, "gemini")
	stdout, stderr := bufs()

	if code := runInit(root, "", "gemini", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit gemini: exit %d\nstderr: %s", code, stderr.String())
	}

	// GEMINI.md must be created at root with {{.Agent}} rendered.
	geminiPath := filepath.Join(root, "GEMINI.md")
	assertFileExists(t, geminiPath)
	assertFileContains(t, geminiPath, "# GEMINI.md")
	assertFileContains(t, geminiPath, "## Context")
	assertFileContains(t, geminiPath, "-agent gemini")
	assertFileNotContains(t, geminiPath, "{{.Agent}}")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, "GEMINI.md")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "gemini", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout gemini after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Continue
// -------------------------------------------------------------------

func TestScaffold_Continue_Init(t *testing.T) {
	root := scaffoldDir(t, "continue")
	stdout, stderr := bufs()

	if code := runInit(root, "", "continue", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit continue: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".continue"))
	assertDirExists(t, filepath.Join(root, ".continue", "rules"))

	mdPath := filepath.Join(root, ".continue", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".continue/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "continue", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout continue after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Cline
// -------------------------------------------------------------------

func TestScaffold_Cline_Init(t *testing.T) {
	root := scaffoldDir(t, "cline")
	stdout, stderr := bufs()

	if code := runInit(root, "", "cline", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit cline: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".clinerules"))

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".clinerules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "cline", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout cline after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Roo Code
// -------------------------------------------------------------------

func TestScaffold_RooCode_Init(t *testing.T) {
	root := scaffoldDir(t, "roocode")
	stdout, stderr := bufs()

	if code := runInit(root, "", "roocode", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit roocode: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".roo"))
	assertDirExists(t, filepath.Join(root, ".roo", "rules"))

	mdPath := filepath.Join(root, ".roo", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".roo/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "roocode", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout roocode after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// JetBrains
// -------------------------------------------------------------------

func TestScaffold_JetBrains_Init(t *testing.T) {
	root := scaffoldDir(t, "jetbrains")
	stdout, stderr := bufs()

	if code := runInit(root, "", "jetbrains", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit jetbrains: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".aiassistant"))
	assertDirExists(t, filepath.Join(root, ".aiassistant", "rules"))

	mdPath := filepath.Join(root, ".aiassistant", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".aiassistant/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "jetbrains", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout jetbrains after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// All platforms
// -------------------------------------------------------------------

func TestScaffold_All_Init(t *testing.T) {
	root := scaffoldDir(t, "all")
	stdout, stderr := bufs()

	if code := runInit(root, "", "all", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit all: exit %d\nstderr: %s", code, stderr.String())
	}

	// All agent directories must be present.
	for _, dir := range []string{
		".github", ".github/rules",
		".cursor", ".cursor/rules",
		".windsurf", ".windsurf/rules",
		".claude", ".claude/rules", ".claude/agents",
		".kiro", ".kiro/steering",
		".continue", ".continue/rules",
		".clinerules",
		".roo", ".roo/rules",
		".aiassistant", ".aiassistant/rules",
	} {
		assertDirExists(t, filepath.Join(root, filepath.FromSlash(dir)))
	}

	// Root-level repogov-config.json is written by InitLayoutAll.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	// Spot-check one artifact per platform.
	assertFileExists(t, filepath.Join(root, ".github", "copilot-instructions.md"))
	assertFileExists(t, filepath.Join(root, ".cursor", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".windsurf", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".claude", "CLAUDE.md"))
	assertFileExists(t, filepath.Join(root, ".claude", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".kiro", "steering", "general.md"))
	assertFileExists(t, filepath.Join(root, "GEMINI.md"))
	assertFileExists(t, filepath.Join(root, ".continue", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".roo", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".aiassistant", "rules", "general.md"))

	// AGENTS.md must contain context links for every platform.
	agPath := filepath.Join(root, "AGENTS.md")
	wantLinks := []string{
		"README.md",
		"docs/",
		".github/rules/",
		".github/copilot-instructions.md",
		".cursor/rules/",
		".windsurf/rules/",
		".claude/rules/",
		".claude/agents/",
		".claude/CLAUDE.md",
		".kiro/steering/",
		"GEMINI.md",
		".continue/rules/",
		".clinerules/",
		".roo/rules/",
		".aiassistant/rules/",
	}
	for _, link := range wantLinks {
		assertFileContains(t, agPath, link)
	}

	// No duplicate context entry lines.
	wantEntries := []string{
		"- Copilot rule files: [.github/rules/](.github/rules/)",
		"- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)",
		"- Cursor rule files: [.cursor/rules/](.cursor/rules/)",
		"- Windsurf rule files: [.windsurf/rules/](.windsurf/rules/)",
		"- Claude rule files: [.claude/rules/](.claude/rules/)",
		"- Agent definitions: [.claude/agents/](.claude/agents/)",
		"- Claude repo-wide context: [.claude/CLAUDE.md](.claude/CLAUDE.md)",
		"- Kiro steering files: [.kiro/steering/](.kiro/steering/)",
		"- Gemini repo-wide context: [GEMINI.md](GEMINI.md)",
		"- Continue rule files: [.continue/rules/](.continue/rules/)",
		"- Cline rule files: [.clinerules/](.clinerules/)",
		"- Roo Code rule files: [.roo/rules/](.roo/rules/)",
		"- JetBrains rule files: [.aiassistant/rules/](.aiassistant/rules/)",
	}
	data, _ := os.ReadFile(agPath)
	content := string(data)
	for _, entry := range wantEntries {
		if count := strings.Count(content, entry); count != 1 {
			t.Errorf("AGENTS.md: entry %q appears %d times, want exactly 1", entry, count)
		}
	}

	// Standard sections must still be present.
	assertFileContains(t, agPath, "## Nested Instructions")

	// Layout must pass for every platform.
	for _, platform := range []string{"copilot", "cursor", "windsurf", "claude", "kiro", "gemini", "continue", "cline", "roocode", "jetbrains"} {
		stdout.Reset()
		if code := runLayout(root, platform, false, false, stdout, stderr); code != 0 {
			t.Errorf("runLayout %s after all-init: exit %d\nstdout: %s", platform, code, stdout.String())
		}
	}

	// limits must pass on the generated files.
	stdout.Reset()
	if code := runLimits(root, "", "", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLimits after all-init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Idempotency: second init must create nothing new
// -------------------------------------------------------------------

func TestScaffold_Idempotent(t *testing.T) {
	tests := []struct {
		agent string
	}{
		{"copilot"},
		{"cursor"},
		{"windsurf"},
		{"claude"},
		{"kiro"},
		{"gemini"},
		{"continue"},
		{"cline"},
		{"roocode"},
		{"jetbrains"},
		{"all"},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()

			// First init.
			if code := runInit(root, "", tc.agent, true, false, false, stdout, stderr); code != 0 {
				t.Fatalf("first runInit %s: exit %d\nstderr: %s", tc.agent, code, stderr.String())
			}

			// Second init in verbose mode; "nothing to create" or empty created list.
			stdout.Reset()
			if code := runInit(root, "", tc.agent, false, false, false, stdout, stderr); code != 0 {
				t.Fatalf("second runInit %s: exit %d\nstderr: %s", tc.agent, code, stderr.String())
			}
			out := stdout.String()
			if strings.Contains(out, "Scaffolded") {
				t.Errorf("second init for %s should create nothing; got: %s", tc.agent, out)
			}
		})
	}
}

// -------------------------------------------------------------------
// No-overwrite: hand-edited files are preserved
// -------------------------------------------------------------------

func TestScaffold_DoesNotOverwrite(t *testing.T) {
	tests := []struct {
		agent   string
		relPath string // file to pre-create
		marker  string // unique string that must survive
	}{
		{"copilot", ".github/copilot-instructions.md", "DO-NOT-OVERWRITE-COPILOT"},
		{"cursor", ".cursor/rules/general.mdc", "DO-NOT-OVERWRITE-CURSOR"},
		{"windsurf", ".windsurf/rules/general.md", "DO-NOT-OVERWRITE-WINDSURF"},
		{"claude", ".claude/CLAUDE.md", "DO-NOT-OVERWRITE-CLAUDE"},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()

			// First init to create the directory structure.
			runInit(root, "", tc.agent, true, false, false, stdout, stderr)

			// Overwrite the file with a custom marker.
			abs := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.WriteFile(abs, []byte("# Custom\n\n"+tc.marker+"\n"), 0o644); err != nil {
				t.Fatalf("cannot write marker file: %v", err)
			}

			// Second init must not overwrite the file.
			stdout.Reset()
			runInit(root, "", tc.agent, true, false, false, stdout, stderr)
			assertFileContains(t, abs, tc.marker)
		})
	}
}

// -------------------------------------------------------------------
// Error paths
// -------------------------------------------------------------------

func TestScaffold_Init_NoAgent(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "", true, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-agent") {
		t.Errorf("expected -agent hint in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Init_UnknownAgent(t *testing.T) {
	stdout, stderr := bufs()
	if code := runInit(t.TempDir(), "", "notion", true, false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Errorf("expected 'unknown agent' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Init_OutputReportsCreatedPaths(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit: exit %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	// Every reported path must actually exist.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+ ") {
			rel := strings.TrimPrefix(line, "+ ")
			abs := filepath.Join(root, filepath.FromSlash(rel))
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				t.Errorf("reported created path %q does not exist", rel)
			}
		}
	}
}

func TestScaffold_Init_JSON_ReportsCreatedPaths(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit JSON: exit %d, stderr: %s", code, stderr.String())
	}
	var result struct {
		Platform string   `json:"platform"`
		Created  []string `json:"created"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout.String())
	}
	if result.Platform != "copilot" {
		t.Errorf("expected platform=copilot, got %q", result.Platform)
	}
	if len(result.Created) == 0 {
		t.Error("expected non-empty created list in JSON output")
	}
	// Every JSON-reported path must actually exist on disk.
	for _, rel := range result.Created {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			t.Errorf("JSON reported created %q but file does not exist", rel)
		}
	}
}

// -------------------------------------------------------------------
// limits subcommand
// -------------------------------------------------------------------

func TestScaffold_Limits_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()

	// Init first so a valid config and .md files are present.
	if code := runInit(root, "", "copilot", true, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit: exit %d", code)
	}
	stdout.Reset()
	// Scan .md and .mdc files; all generated files must be within limits.
	if code := runLimits(root, "", ".md,.mdc", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLimits after init: exit %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "[PASS]") {
		t.Errorf("expected [PASS] in limits output, got: %s", stdout.String())
	}
}

func TestScaffold_Limits_ExceedLimit(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.md":                      nlines(500),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (limit exceeded), got %d", code)
	}
}

func TestScaffold_Limits_WarnBeforeExceed(t *testing.T) {
	// File at 90% of the warning-threshold limit should produce a WARN, not FAIL.
	root := writeTempDir(t, map[string]string{
		"near.md":                     nlines(90),
		".github/repogov-config.json": `{"default": 100, "warning_threshold": "80%"}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0 (warning only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARN") {
		t.Errorf("expected WARN in output, got: %s", stdout.String())
	}
}

func TestScaffold_Limits_NoConfig_UsesDefaults(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"tiny.md": nlines(5),
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestScaffold_Limits_AllSentinelIncludesGo(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"big.go":                      nlines(500),
		".github/repogov-config.json": `{"default": 50}`,
	})
	stdout, stderr := bufs()
	// "all" sentinel removes extension filter so .go exceeds limit.
	if code := runLimits(root, "", "all", false, true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (all sentinel includes big.go), got %d", code)
	}
}

func TestScaffold_Limits_BadConfigJSON(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": "not-a-number"}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (bad config), got %d", code)
	}
}

func TestScaffold_Limits_JSONOutput(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"a.md":                        nlines(5),
		".github/repogov-config.json": `{"default": 300}`,
	})
	stdout, stderr := bufs()
	if code := runLimits(root, "", ".md", false, false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []interface{}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

// -------------------------------------------------------------------
// layout subcommand
// -------------------------------------------------------------------

func TestScaffold_Layout_PassAfterInit(t *testing.T) {
	tests := []struct {
		agent    string
		platform string
	}{
		{"copilot", "copilot"},
		{"cursor", "cursor"},
		{"windsurf", "windsurf"},
		{"claude", "claude"},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()
			if code := runInit(root, "", tc.agent, true, false, false, stdout, stderr); code != 0 {
				t.Fatalf("runInit %s: exit %d", tc.agent, code)
			}
			if code := runLayout(root, tc.platform, true, false, stdout, stderr); code != 0 {
				t.Fatalf("runLayout %s after init: exit %d", tc.platform, code)
			}
		})
	}
}

func TestScaffold_Layout_MissingCopilotInstructions(t *testing.T) {
	// Init copilot, then delete the required file.
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)

	required := filepath.Join(root, ".github", "copilot-instructions.md")
	if err := os.Remove(required); err != nil {
		t.Fatalf("cannot remove required file: %v", err)
	}

	if code := runLayout(root, "copilot", true, false, stdout, stderr); code != 1 {
		t.Fatalf("expected layout to fail without copilot-instructions.md, got exit %d", code)
	}
}

func TestScaffold_Layout_MissingRootDir(t *testing.T) {
	// Fresh empty directory: every platform dir is absent.
	tests := []string{"copilot", "cursor", "windsurf", "claude"}
	for _, platform := range tests {
		t.Run(platform, func(t *testing.T) {
			stdout, stderr := bufs()
			if code := runLayout(t.TempDir(), platform, true, false, stdout, stderr); code != 1 {
				t.Fatalf("expected exit 1 (missing layout dir), got %d", code)
			}
		})
	}
}

func TestScaffold_Layout_UnknownPlatform(t *testing.T) {
	stdout, stderr := bufs()
	if code := runLayout(t.TempDir(), "jira", true, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown agent") {
		t.Errorf("expected 'unknown agent' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Layout_All_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "all", true, false, false, stdout, stderr)

	if code := runLayout(root, "all", true, false, stdout, stderr); code != 0 {
		t.Fatalf("expected layout all to pass after init, got exit %d\nstderr: %s", code, stderr.String())
	}
}

func TestScaffold_Layout_JSONOutput(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)

	stdout.Reset()
	if code := runLayout(root, "copilot", false, true, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	var results []interface{}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
}

// -------------------------------------------------------------------
// validate subcommand
// -------------------------------------------------------------------

func TestScaffold_Validate_ValidConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": "80%"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0, got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("expected 'is valid' in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_InvalidNegativeDefault(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -1}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (negative default), got %d", code)
	}
	if !strings.Contains(stdout.String(), "[FAIL]") {
		t.Errorf("expected [FAIL] in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_InvalidThresholdOver100(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": 300, "warning_threshold": "200%"}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 (threshold > 100), got %d", code)
	}
}

func TestScaffold_Validate_MissingConfig(t *testing.T) {
	stdout, stderr := bufs()
	if code := runValidate(t.TempDir(), "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (no config), got %d", code)
	}
	if !strings.Contains(stderr.String(), "No config file found") {
		t.Errorf("expected 'No config file found' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Validate_MalformedJSON(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{broken json`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2 (malformed JSON), got %d", code)
	}
}

func TestScaffold_Validate_BackslashPathWarning(t *testing.T) {
	// A backslash in a file key is a warning-level violation; exit must be 0.
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": "{\"default\": 300, \"files\": {\"a\\\\b.md\": 100}}",
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected 0 (warning only), got %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "WARNING") {
		t.Errorf("expected WARNING in output, got: %s", stdout.String())
	}
}

func TestScaffold_Validate_JSONOutput_Valid(t *testing.T) {
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
		t.Error("expected valid=true in JSON output")
	}
}

func TestScaffold_Validate_JSONOutput_Invalid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/repogov-config.json": `{"default": -5}`,
	})
	stdout, stderr := bufs()
	if code := runValidate(root, "", false, true, stdout, stderr); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	var result struct {
		Valid      bool        `json:"valid"`
		Violations interface{} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false in JSON output")
	}
}

func TestScaffold_Validate_PassAfterInit(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := bufs()
	runInit(root, "", "copilot", true, false, false, stdout, stderr)

	stdout.Reset()
	if code := runValidate(root, "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected validate to pass on init-generated config, exit %d\nstdout: %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("expected 'is valid' in output, got: %s", stdout.String())
	}
}

// -------------------------------------------------------------------
// run() dispatcher integration
// -------------------------------------------------------------------

func TestScaffold_Run_EachSubcommand(t *testing.T) {
	root := t.TempDir()
	// Pre-init so all subcommands have a config and layout dirs.
	bufs1, bufs2 := bufs()
	runInit(root, "", "all", true, false, false, bufs1, bufs2)

	tests := []struct {
		args     []string
		wantCode int
		desc     string
	}{
		{
			args:     []string{"-root", root, "-agent", "copilot", "-quiet", "init"},
			wantCode: 0,
			desc:     "init copilot (already exists)",
		},
		{
			args:     []string{"-root", root, "-agent", "all", "-quiet", "layout"},
			wantCode: 0,
			desc:     "layout all after init",
		},
		{
			args:     []string{"-root", root, "-quiet", "limits"},
			wantCode: 0,
			desc:     "limits with default config",
		},
		{
			args:     []string{"-root", root, "-quiet", "validate"},
			wantCode: 0,
			desc:     "validate with init-generated config",
		},
		{
			args:     []string{"-root", root, "-agent", "all", "-quiet"},
			wantCode: 0,
			desc:     "default (all) subcommand",
		},
		{
			args:     []string{"version"},
			wantCode: 0,
			desc:     "version prints",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			stdout, stderr := bufs()
			if code := run(tc.args, stdout, stderr); code != tc.wantCode {
				t.Errorf("%s: expected exit %d, got %d\nstdout: %s\nstderr: %s",
					tc.desc, tc.wantCode, code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestScaffold_Run_UnknownSubcommand(t *testing.T) {
	stdout, stderr := bufs()
	if code := run([]string{"bogus-command"}, stdout, stderr); code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in stderr, got: %s", stderr.String())
	}
}

func TestScaffold_Run_NoArgsShowsHelp(t *testing.T) {
	stdout, stderr := bufs()
	// No -root provided: run uses cwd; config likely absent so limits might
	// return 0 or 1, but the process must not panic.
	run([]string{"-h"}, stdout, stderr)
	// Help exits 0 or 2 depending on flag implementation -- just confirm
	// it outputs usage text.
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "repogov") {
		t.Errorf("expected repogov in help output, got: %s", combined)
	}
}

// -------------------------------------------------------------------
// AGENTS.md context section per-agent accuracy
// -------------------------------------------------------------------

func TestScaffold_AgentsMd_ContextLinks(t *testing.T) {
	tests := []struct {
		agent       string
		mustHave    []string
		mustNotHave []string
	}{
		{
			agent:       "copilot",
			mustHave:    []string{"README.md", "docs/", ".github/rules/", "copilot-instructions.md"},
			mustNotHave: []string{".cursor/", ".windsurf/", ".claude/"},
		},
		{
			agent:       "cursor",
			mustHave:    []string{"README.md", "docs/", ".cursor/rules/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".windsurf/", ".claude/"},
		},
		{
			agent:       "windsurf",
			mustHave:    []string{"README.md", "docs/", ".windsurf/rules/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".cursor/", ".claude/"},
		},
		{
			agent:       "claude",
			mustHave:    []string{"README.md", "docs/", ".claude/rules/", ".claude/agents/"},
			mustNotHave: []string{"copilot-instructions.md", ".github/instructions/", ".cursor/", ".windsurf/"},
		},
		{
			agent:    "kiro",
			mustHave: []string{"README.md", "docs/", ".kiro/steering/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "gemini",
			mustHave: []string{"README.md", "docs/", "GEMINI.md"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "continue",
			mustHave: []string{"README.md", "docs/", ".continue/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "cline",
			mustHave: []string{"README.md", "docs/", ".clinerules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "roocode",
			mustHave: []string{"README.md", "docs/", ".roo/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
		{
			agent:    "jetbrains",
			mustHave: []string{"README.md", "docs/", ".aiassistant/rules/"},
			mustNotHave: []string{
				"copilot-instructions.md", ".cursor/", ".claude/",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()
			runInit(root, "", tc.agent, true, false, false, stdout, stderr)

			agPath := filepath.Join(root, "AGENTS.md")
			for _, link := range tc.mustHave {
				assertFileContains(t, agPath, link)
			}
			for _, link := range tc.mustNotHave {
				assertFileNotContains(t, agPath, link)
			}
		})
	}
}
