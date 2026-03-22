package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestCheckLayout_GuidanceMessages(t *testing.T) {
	root := t.TempDir()
	// Don't create .github -- should get guidance about running init.

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Required: []string{"workflows/ci.yml"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if !strings.Contains(results[0].Message, "repogov init") {
		t.Errorf("expected guidance about 'repogov init', got: %s", results[0].Message)
	}
}

func TestCheckLayout_MissingFileGuidance(t *testing.T) {
	root := t.TempDir()
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Required: []string{"workflows/ci.yml"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasGuidance := false
	for _, r := range results {
		if r.Status == repogov.Fail && strings.Contains(r.Message, "FIX:") {
			hasGuidance = true
			break
		}
	}
	if !hasGuidance {
		t.Error("expected FIX guidance in failure message for missing required file")
	}
}

func TestCheckLayout_NamingGuidance(t *testing.T) {
	root := t.TempDir()
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ghDir, "MyConfig.yml"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.LayoutSchema{
		Root:   ".github",
		Naming: repogov.NamingRule{Case: "lowercase"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasRenameHint := false
	for _, r := range results {
		if r.Status == repogov.Fail && strings.Contains(r.Message, "rename to") {
			hasRenameHint = true
			break
		}
	}
	if !hasRenameHint {
		t.Error("expected rename guidance in naming violation message")
	}
}

func TestInitLayout_CreatesCopilotInstructions(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Verify copilot-instructions.md was created.
	ciPath := filepath.Join(root, ".github", "copilot-instructions.md")
	data, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatal("copilot-instructions.md was not created")
	}
	content := string(data)

	// On a fresh init (no pre-existing instructions/), should reference rules/ by default.
	if !strings.Contains(content, ".github/rules/") {
		t.Error("copilot-instructions.md should link to .github/rules/ on fresh init")
	}
	// Should cross-reference README and docs.
	if !strings.Contains(content, "README.md") {
		t.Error("copilot-instructions.md should cross-reference README.md")
	}
	if !strings.Contains(content, "docs/") {
		t.Error("copilot-instructions.md should cross-reference docs/")
	}
	// Should include file constraints section with remediation ladder.
	if !strings.Contains(content, "## File Constraints") {
		t.Error("copilot-instructions.md should include File Constraints section")
	}
	if !strings.Contains(content, "Refactor First") {
		t.Error("copilot-instructions.md should document remediation priority")
	}
	// Should include repository commands section.
	if !strings.Contains(content, "## Repository Commands") {
		t.Error("copilot-instructions.md should include Repository Commands section")
	}
	if !strings.Contains(content, "-agent copilot") {
		t.Error("copilot-instructions.md should reference the agent name")
	}
	// Should be in the created list.
	found := false
	for _, p := range created {
		if p == ".github/copilot-instructions.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("copilot-instructions.md not in created list")
	}
}

func TestInitLayout_CopilotInstructionsNotOverwritten(t *testing.T) {
	root := t.TempDir()
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customContent := "# My Custom Instructions\n"
	ciPath := filepath.Join(ghDir, "copilot-instructions.md")
	if err := os.WriteFile(ciPath, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultCopilotLayout()
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(ciPath)
	if string(data) != customContent {
		t.Error("InitLayout overwrote existing copilot-instructions.md")
	}
}

// TestInitLayout_CopilotUsesInstructionsWhenPreExisting verifies that when
// .github/instructions/ already has content, init seeds templates into
// instructions/ and copilot-instructions.md references instructions/ instead
// of the default rules/.
func TestInitLayout_CopilotUsesInstructionsWhenPreExisting(t *testing.T) {
	root := t.TempDir()
	ghDir := filepath.Join(root, ".github")
	instrDir := filepath.Join(ghDir, "instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed a file into instructions/ so it is non-empty.
	if err := os.WriteFile(filepath.Join(instrDir, "existing.md"), []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultCopilotLayout()
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// rules/ should NOT be created when instructions/ already has content.
	assertNotExists(t, filepath.Join(ghDir, "rules"))
	// instructions/ must still exist.
	assertExists(t, instrDir)
	// copilot-instructions.md must reference instructions/, not rules/.
	ciPath := filepath.Join(ghDir, "copilot-instructions.md")
	assertFileContains(t, ciPath, ".github/instructions/")
	assertFileNotContains(t, ciPath, ".github/rules/")
}

func TestInitLayout_CopilotInstructionsLineLimit(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	ciPath := filepath.Join(root, ".github", "copilot-instructions.md")
	lines, err := repogov.CountLines(ciPath)
	if err != nil {
		t.Fatal(err)
	}
	if lines > 50 {
		t.Errorf("copilot-instructions.md has %d lines, want ≤50", lines)
	}
}

func TestInitLayout_SeededConfigOnlyContainsPlatformRules(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".github", "repogov-config.json"))
	if err != nil {
		t.Fatal("repogov-config.json was not created")
	}
	content := string(data)

	// Only .github/ rules should be present.
	for _, foreign := range []string{".cursor/", ".windsurf/", ".claude/"} {
		if strings.Contains(content, foreign) {
			t.Errorf("seeded config should not contain %s rules when initializing github platform", foreign)
		}
	}
	// The .github/rules rule should be present.
	if !strings.Contains(content, ".github/rules/") {
		t.Error("seeded config should contain .github/rules/ rule")
	}
}

// TestInitLayout_CreatesDefaultRuleFiles verifies that a fresh .github init with
// the default config (DescriptiveNames=false) seeds ALL template files into
// rules/ using plain <name>.md naming.
func TestInitLayout_CreatesDefaultRuleFiles(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Default (DescriptiveNames=false): ALL templates seeded as <name>.md in rules/.
	expectedFiles := []string{
		"general.md",
		"codereview.md",
		"governance.md",
		"library.md",
		"testing.md",
		"emoji-prevention.md",
		"backend.md",
		"frontend.md",
		"repo.md",
	}
	rulesDir := filepath.Join(root, ".github", "rules")
	for _, name := range expectedFiles {
		path := filepath.Join(rulesDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to be created in rules/, got error: %v", name, err)
			continue
		} else if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
		rel := ".github/rules/" + name
		found := false
		for _, p := range created {
			if p == rel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not found in created list", rel)
		}
	}
}

// TestInitLayout_CreatesDefaultInstructions verifies that a fresh .github init
// with Descriptive=true seeds all *.instructions.md files into rules/.
func TestInitLayout_CreatesDefaultInstructions(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	cfg := repogov.Config{DescriptiveNames: true}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{
		"general.instructions.md",
		"codereview.instructions.md",
		"governance.instructions.md",
		"library.instructions.md",
		"testing.instructions.md",
		"emoji-prevention.instructions.md",
		"backend.instructions.md",
		"frontend.instructions.md",
	}

	// With Descriptive=true, files land in rules/ as *.instructions.md.
	rulesDir := filepath.Join(root, ".github", "rules")
	for _, name := range expectedFiles {
		path := filepath.Join(rulesDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to be created in rules/, got error: %v", name, err)
			continue
		}
		content := string(data)
		if len(content) == 0 {
			t.Errorf("%s is empty", name)
		}
		// All instruction files should have a YAML front matter header.
		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("%s should start with YAML front matter", name)
		}
		// Should appear in created list.
		rel := ".github/rules/" + name
		found := false
		for _, p := range created {
			if p == rel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not found in created list", rel)
		}
	}
}

func TestInitLayout_DefaultInstructionsIdempotent(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	// First call creates defaults.
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Modify one file in rules/ to verify it's not overwritten.
	// Default is DescriptiveNames=false, so files use plain .md names.
	rulesDir := filepath.Join(root, ".github", "rules")
	customPath := filepath.Join(rulesDir, "general.md")
	custom := "# My Custom General Rules\n"
	if err := os.WriteFile(customPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second call should not overwrite anything.
	created2, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created2 {
		if strings.Contains(p, "rules/") {
			t.Errorf("second call created %s, expected no rule files to be re-seeded", p)
		}
	}

	// Verify the custom file was preserved.
	data, _ := os.ReadFile(customPath)
	if string(data) != custom {
		t.Error("InitLayout overwrote existing instruction file")
	}
}

func TestInitLayout_DefaultInstructionsSkippedWhenDirNonEmpty(t *testing.T) {
	root := t.TempDir()

	// Pre-create instructions directory with a custom file.
	instrDir := filepath.Join(root, ".github", "instructions")
	if err := os.MkdirAll(instrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(instrDir, "custom.instructions.md"),
		[]byte("# Custom\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultCopilotLayout()
	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// No instruction files should be seeded because the directory is non-empty.
	// README.md is infrastructure and is always created when absent.
	for _, p := range created {
		if strings.HasPrefix(p, ".github/instructions/") &&
			!strings.HasSuffix(p, ".gitkeep") &&
			!strings.HasSuffix(p, "README.md") {
			t.Errorf("should not seed %s when instructions/ is non-empty", p)
		}
	}
}

func TestInitLayout_DefaultInstructionsLineLimit(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// On a fresh init, Copilot files land in rules/.
	rulesDir := filepath.Join(root, ".github", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ".gitkeep" {
			continue
		}
		path := filepath.Join(rulesDir, e.Name())
		lines, err := repogov.CountLines(path)
		if err != nil {
			t.Fatalf("counting lines in %s: %v", e.Name(), err)
		}
		if lines > 300 {
			t.Errorf("%s has %d lines, want ≤300", e.Name(), lines)
		}
	}
}

// TestInitLayout_GovernanceLink_DefaultJSON verifies that governance.instructions.md
// links to repogov-config.json when no config file exists yet (the default case).
// This behavior requires Descriptive=true since governance.instructions.md is only
// seeded when the descriptive naming convention is active.
func TestInitLayout_GovernanceLink_DefaultJSON(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	cfg := repogov.Config{DescriptiveNames: true}
	_, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// With Descriptive=true, governance file lands in rules/ as *.instructions.md.
	data, err := os.ReadFile(filepath.Join(root, ".github", "rules", "governance.instructions.md"))
	if err != nil {
		t.Fatal("governance.instructions.md not created:", err)
	}
	content := string(data)
	if !strings.Contains(content, "../repogov-config.json") {
		t.Errorf("governance.instructions.md should link to ../repogov-config.json, got:\n%s", content)
	}
}

// TestInitLayout_GovernanceLink_YAMLConfig verifies that when a YAML config
// already exists and Descriptive=true, governance.instructions.md links to it
// rather than the default JSON filename.
func TestInitLayout_GovernanceLink_YAMLConfig(t *testing.T) {
	root := t.TempDir()

	// Pre-create a YAML config in .github/ to simulate a user who prefers YAML.
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yamlCfg := filepath.Join(ghDir, "repogov-config.yaml")
	if err := os.WriteFile(yamlCfg, []byte("default: 300\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultCopilotLayout()
	cfg := repogov.Config{DescriptiveNames: true}
	_, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// With Descriptive=true, governance file lands in rules/ as *.instructions.md.
	data, err := os.ReadFile(filepath.Join(root, ".github", "rules", "governance.instructions.md"))
	if err != nil {
		t.Fatal("governance.instructions.md not created:", err)
	}
	content := string(data)
	if !strings.Contains(content, "../repogov-config.yaml") {
		t.Errorf("governance.instructions.md should link to ../repogov-config.yaml, got:\n%s", content)
	}
	if strings.Contains(content, "repogov-config.json") {
		t.Errorf("governance.instructions.md should not reference repogov-config.json when YAML config exists")
	}
}
