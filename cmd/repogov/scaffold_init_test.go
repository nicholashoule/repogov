package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -------------------------------------------------------------------
// Copilot
// -------------------------------------------------------------------

func TestScaffold_Copilot_Init(t *testing.T) {
	root := scaffoldDir(t, "copilot")
	stdout, stderr := bufs()

	if code := runInit(root, "", "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
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
		"general.md", "memory.md", "codereview.md", "governance.md",
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
	if code := runLayout(root, "", "copilot", "", false, false, stdout, stderr); code != 0 {
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
		"    \".github/rules/memory.instructions.md\": 200,\n" +
		"    \"AGENTS.md\": 200\n" +
		"  }\n" +
		"}\n"
	cfgPath := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(cfgData), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runInit(root, cfgPath, "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit copilot descriptive: exit %d\nstderr: %s", code, stderr.String())
	}

	// Every default instruction file must exist in rules/ with *.instructions.md names.
	instructionFiles := []string{
		"general.instructions.md",
		"memory.instructions.md",
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

// TestScaffold_Copilot_Init_Instructions verifies that when .github/instructions/
// already has content, init detects it, sets descriptive_names=true, and generates
// a config that references only instructions/ paths — not rules/ duplicates.
// Template files are not re-seeded into a non-empty directory (by design);
// the test focuses on config correctness and AGENTS.md links.
func TestScaffold_Copilot_Init_Instructions(t *testing.T) {
	root := scaffoldDir(t, "copilot-instructions")
	stdout, stderr := bufs()

	// Pre-create .github/instructions/ with a file so detectCopilotTargetDir
	// picks instructions/ over rules/.
	instrDir := filepath.Join(root, ".github", "instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(instrDir, "existing.instructions.md")
	if err := os.WriteFile(seed, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := runInit(root, "", "copilot", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit copilot instructions: exit %d\nstderr: %s", code, stderr.String())
	}

	// instructions/ must exist; rules/ must NOT be created.
	assertDirExists(t, instrDir)
	if _, err := os.Stat(filepath.Join(root, ".github", "rules")); err == nil {
		t.Error("expected .github/rules/ to NOT exist when instructions/ was detected")
	}

	// The pre-existing seed file must still be present.
	assertFileExists(t, seed)

	// Generated config must set descriptive_names=true.
	cfgPath := filepath.Join(root, ".github", "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertFileContains(t, cfgPath, `"descriptive_names": true`)

	// Config must reference instructions/ paths, not rules/ paths.
	assertFileContains(t, cfgPath, ".github/instructions/memory.instructions.md")
	assertFileNotContains(t, cfgPath, ".github/rules/memory")

	// Rules glob is always .github/rules/*.md (harmless when dir absent).
	assertFileContains(t, cfgPath, ".github/rules/*.md")

	// AGENTS.md must reference instructions/, not rules/.
	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".github/instructions/")
	assertFileNotContains(t, agPath, ".github/rules/")

	// copilot-instructions.md must reference instructions/ dir.
	ciPath := filepath.Join(root, ".github", "copilot-instructions.md")
	assertFileExists(t, ciPath)
	assertFileContains(t, ciPath, "instructions/")
}

// -------------------------------------------------------------------
// Cursor
// -------------------------------------------------------------------

func TestScaffold_Cursor_Init(t *testing.T) {
	root := scaffoldDir(t, "cursor")
	stdout, stderr := bufs()

	if code := runInit(root, "", "cursor", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit cursor: exit %d\nstderr: %s", code, stderr.String())
	}

	// Required directories.
	assertDirExists(t, filepath.Join(root, ".cursor"))
	assertDirExists(t, filepath.Join(root, ".cursor", "rules"))

	// general.md and memory.md must exist with standard instruction frontmatter.
	mdPath := filepath.Join(root, ".cursor", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".cursor", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root (no .github/ present; FindConfig discovers root before agent dirs).
	cfgPath := filepath.Join(root, "repogov-config.json")
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
	if code := runLayout(root, "", "cursor", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout cursor after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Windsurf
// -------------------------------------------------------------------

func TestScaffold_Windsurf_Init(t *testing.T) {
	root := scaffoldDir(t, "windsurf")
	stdout, stderr := bufs()

	if code := runInit(root, "", "windsurf", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit windsurf: exit %d\nstderr: %s", code, stderr.String())
	}

	// Required directories.
	assertDirExists(t, filepath.Join(root, ".windsurf"))
	assertDirExists(t, filepath.Join(root, ".windsurf", "rules"))

	// general.md and memory.md must exist with standard instruction frontmatter.
	mdPath := filepath.Join(root, ".windsurf", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".windsurf", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root (FindConfig discovers root before agent dirs).
	cfgPath := filepath.Join(root, "repogov-config.json")
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
	if code := runLayout(root, "", "windsurf", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout windsurf after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Claude
// -------------------------------------------------------------------

func TestScaffold_Claude_Init(t *testing.T) {
	root := scaffoldDir(t, "claude")
	stdout, stderr := bufs()

	if code := runInit(root, "", "claude", "", false, false, false, false, stdout, stderr); code != 0 {
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

	// general.md and memory.md must exist with standard instruction frontmatter.
	mdPath := filepath.Join(root, ".claude", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "---")
	assertFileContains(t, mdPath, "applyTo:")
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".claude", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root (FindConfig discovers root before agent dirs).
	cfgPath := filepath.Join(root, "repogov-config.json")
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
	if code := runLayout(root, "", "claude", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout claude after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Kiro
// -------------------------------------------------------------------

func TestScaffold_Kiro_Init(t *testing.T) {
	root := scaffoldDir(t, "kiro")
	stdout, stderr := bufs()

	if code := runInit(root, "", "kiro", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit kiro: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".kiro"))
	assertDirExists(t, filepath.Join(root, ".kiro", "steering"))

	// general.md and memory.md must be seeded into steering/.
	mdPath := filepath.Join(root, ".kiro", "steering", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".kiro", "steering", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".kiro/steering/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")
	assertFileNotContains(t, agPath, ".cursor/")

	if code := runLayout(root, "", "kiro", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout kiro after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Gemini
// -------------------------------------------------------------------

func TestScaffold_Gemini_Init(t *testing.T) {
	root := scaffoldDir(t, "gemini")
	stdout, stderr := bufs()

	if code := runInit(root, "", "gemini", "", false, false, false, false, stdout, stderr); code != 0 {
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

	if code := runLayout(root, "", "gemini", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout gemini after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Continue
// -------------------------------------------------------------------

func TestScaffold_Continue_Init(t *testing.T) {
	root := scaffoldDir(t, "continue")
	stdout, stderr := bufs()

	if code := runInit(root, "", "continue", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit continue: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".continue"))
	assertDirExists(t, filepath.Join(root, ".continue", "rules"))

	mdPath := filepath.Join(root, ".continue", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".continue", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".continue/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "", "continue", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout continue after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Cline
// -------------------------------------------------------------------

func TestScaffold_Cline_Init(t *testing.T) {
	root := scaffoldDir(t, "cline")
	stdout, stderr := bufs()

	if code := runInit(root, "", "cline", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit cline: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".clinerules"))

	// Rule files are seeded directly into .clinerules/.
	for _, name := range []string{
		"general.md", "memory.md", "codereview.md", "governance.md",
		"library.md", "testing.md", "emoji-prevention.md",
		"backend.md", "frontend.md", "security.md", "repo.md",
	} {
		p := filepath.Join(root, ".clinerules", name)
		assertFileExists(t, p)
		assertFileContains(t, p, "---") // YAML frontmatter delimiter
	}

	// The emoji-prevention link in general.md must point to .clinerules/ directly.
	assertFileContains(t, filepath.Join(root, ".clinerules", "general.md"), ".clinerules/emoji-prevention.md")

	// Config must exist at repo root.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".clinerules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "", "cline", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout cline after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Roo Code
// -------------------------------------------------------------------

func TestScaffold_RooCode_Init(t *testing.T) {
	root := scaffoldDir(t, "roocode")
	stdout, stderr := bufs()

	if code := runInit(root, "", "roocode", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit roocode: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".roo"))
	assertDirExists(t, filepath.Join(root, ".roo", "rules"))

	mdPath := filepath.Join(root, ".roo", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".roo", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".roo/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "", "roocode", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout roocode after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// JetBrains
// -------------------------------------------------------------------

func TestScaffold_JetBrains_Init(t *testing.T) {
	root := scaffoldDir(t, "jetbrains")
	stdout, stderr := bufs()

	if code := runInit(root, "", "jetbrains", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit jetbrains: exit %d\nstderr: %s", code, stderr.String())
	}

	assertDirExists(t, filepath.Join(root, ".aiassistant"))
	assertDirExists(t, filepath.Join(root, ".aiassistant", "rules"))

	mdPath := filepath.Join(root, ".aiassistant", "rules", "general.md")
	assertFileExists(t, mdPath)
	assertFileContains(t, mdPath, "# General Instructions")

	memPath := filepath.Join(root, ".aiassistant", "rules", "memory.md")
	assertFileExists(t, memPath)
	assertFileContains(t, memPath, "# Project Memory")

	// Config must exist at repo root.
	cfgPath := filepath.Join(root, "repogov-config.json")
	assertFileExists(t, cfgPath)
	assertValidJSON(t, cfgPath)
	assertConfigHasDefault(t, cfgPath)

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".aiassistant/rules/")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "", "jetbrains", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout jetbrains after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// Zed
// -------------------------------------------------------------------

func TestScaffold_Zed_Init(t *testing.T) {
	root := scaffoldDir(t, "zed")
	stdout, stderr := bufs()

	if code := runInit(root, "", "zed", "", false, false, false, false, stdout, stderr); code != 0 {
		t.Fatalf("runInit zed: exit %d\nstderr: %s", code, stderr.String())
	}

	// .rules must be created at root with {{.Agent}} rendered.
	rulesPath := filepath.Join(root, ".rules")
	assertFileExists(t, rulesPath)
	assertFileContains(t, rulesPath, "# Project Rules")
	assertFileContains(t, rulesPath, "-agent zed")
	assertFileNotContains(t, rulesPath, "{{.Agent}}")

	agPath := filepath.Join(root, "AGENTS.md")
	assertFileExists(t, agPath)
	assertFileContains(t, agPath, ".rules")
	assertFileNotContains(t, agPath, "copilot-instructions.md")

	if code := runLayout(root, "", "zed", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("runLayout zed after init: exit %d\nstdout: %s", code, stdout.String())
	}
}

// -------------------------------------------------------------------
// All platforms
// -------------------------------------------------------------------

func TestScaffold_All_Init(t *testing.T) {
	root := scaffoldDir(t, "all")
	stdout, stderr := bufs()

	if code := runInit(root, "", "all", "", false, false, false, false, stdout, stderr); code != 0 {
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
	assertFileExists(t, filepath.Join(root, ".cursor", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".windsurf", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".windsurf", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".claude", "CLAUDE.md"))
	assertFileExists(t, filepath.Join(root, ".claude", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".claude", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".kiro", "steering", "general.md"))
	assertFileExists(t, filepath.Join(root, ".kiro", "steering", "memory.md"))
	assertFileExists(t, filepath.Join(root, "GEMINI.md"))
	assertFileExists(t, filepath.Join(root, ".continue", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".continue", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".roo", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".roo", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".aiassistant", "rules", "general.md"))
	assertFileExists(t, filepath.Join(root, ".aiassistant", "rules", "memory.md"))
	assertFileExists(t, filepath.Join(root, ".rules"))

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
		".rules",
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
		"- Zed rules: [.rules](.rules)",
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
	for _, platform := range []string{"copilot", "cursor", "windsurf", "claude", "kiro", "gemini", "continue", "cline", "roocode", "jetbrains", "zed"} {
		stdout.Reset()
		if code := runLayout(root, "", platform, "", false, false, stdout, stderr); code != 0 {
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
		{"zed"},
		{"all"},
	}
	for _, tc := range tests {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			stdout, stderr := bufs()

			// First init.
			if code := runInit(root, "", tc.agent, "", true, false, false, false, stdout, stderr); code != 0 {
				t.Fatalf("first runInit %s: exit %d\nstderr: %s", tc.agent, code, stderr.String())
			}

			// Second init in verbose mode; "nothing to create" or empty created list.
			stdout.Reset()
			if code := runInit(root, "", tc.agent, "", false, false, false, false, stdout, stderr); code != 0 {
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
			runInit(root, "", tc.agent, "", true, false, false, false, stdout, stderr)

			// Overwrite the file with a custom marker.
			abs := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.WriteFile(abs, []byte("# Custom\n\n"+tc.marker+"\n"), 0o644); err != nil {
				t.Fatalf("cannot write marker file: %v", err)
			}

			// Second init must not overwrite the file.
			stdout.Reset()
			runInit(root, "", tc.agent, "", true, false, false, false, stdout, stderr)
			assertFileContains(t, abs, tc.marker)
		})
	}
}
