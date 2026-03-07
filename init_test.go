package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestInitLayout_CreatesStructure(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(created) == 0 {
		t.Fatal("expected items to be created")
	}

	// Verify .github directory exists.
	ghDir := filepath.Join(root, ".github")
	if _, err := os.Stat(ghDir); os.IsNotExist(err) {
		t.Error(".github directory was not created")
	}

	// Verify subdirectories exist.
	for _, dir := range []string{"instructions"} {
		dirPath := filepath.Join(ghDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("subdirectory %s was not created", dir)
		}
	}

	// Verify copilot-instructions.md was seeded.
	if _, err := os.Stat(filepath.Join(ghDir, "copilot-instructions.md")); os.IsNotExist(err) {
		t.Error("copilot-instructions.md was not created")
	}
}

func TestInitLayout_Idempotent(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

	// First call creates structure.
	created1, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created1) == 0 {
		t.Fatal("first call should create items")
	}

	// Second call should not create anything new.
	created2, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created2) != 0 {
		t.Errorf("second call created %d items, want 0 (idempotent)", len(created2))
	}
}

func TestInitLayout_DoesNotOverwrite(t *testing.T) {
	root := t.TempDir()

	// Pre-create repogov-config.json with custom content before init.
	ghDir := filepath.Join(root, ".github")
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatal(err)
	}
	customContent := `{"default":100}`
	cfgPath := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(customContent), 0644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultGitHubLayout()
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the file was NOT overwritten.
	data, _ := os.ReadFile(cfgPath)
	if string(data) != customContent {
		t.Error("InitLayout overwrote existing repogov-config.json")
	}
}

func TestInitLayout_GitLabSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitLabLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(created) == 0 {
		t.Fatal("expected GitLab directories to be created")
	}

	// Verify .gitlab directory exists.
	glDir := filepath.Join(root, ".gitlab")
	if _, err := os.Stat(glDir); os.IsNotExist(err) {
		t.Error(".gitlab directory was not created")
	}
}

func TestInitLayout_CursorSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(created) == 0 {
		t.Fatal("expected Cursor directories to be created")
	}

	cursorDir := filepath.Join(root, ".cursor")
	if _, err := os.Stat(cursorDir); os.IsNotExist(err) {
		t.Error(".cursor directory was not created")
	}

	rulesDir := filepath.Join(cursorDir, "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Error(".cursor/rules directory was not created")
	}
}

func TestInitLayout_WindsurfSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultWindsurfLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(created) == 0 {
		t.Fatal("expected Windsurf directories to be created")
	}

	wsDir := filepath.Join(root, ".windsurf")
	if _, err := os.Stat(wsDir); os.IsNotExist(err) {
		t.Error(".windsurf directory was not created")
	}

	rulesDir := filepath.Join(wsDir, "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Error(".windsurf/rules directory was not created")
	}
}

func TestInitLayout_ClaudeSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultClaudeLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	if len(created) == 0 {
		t.Fatal("expected Claude directories to be created")
	}

	claudeDir := filepath.Join(root, ".claude")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		t.Error(".claude directory was not created")
	}

	for _, dir := range []string{"rules", "agents"} {
		dirPath := filepath.Join(claudeDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf(".claude/%s directory was not created", dir)
		}
	}
}

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
	if err := os.MkdirAll(ghDir, 0755); err != nil {
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
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ghDir, "MyConfig.yml"), []byte("test\n"), 0644); err != nil {
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
	schema := repogov.DefaultGitHubLayout()

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

	// Should reference the instructions directory.
	if !strings.Contains(content, ".github/instructions/") {
		t.Error("copilot-instructions.md should link to .github/instructions/")
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
	// Should include file naming conventions.
	if !strings.Contains(content, "## File Naming Conventions") {
		t.Error("copilot-instructions.md should include File Naming Conventions section")
	}
	if !strings.Contains(content, "kebab-case") {
		t.Error("copilot-instructions.md should reference kebab-case convention")
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
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatal(err)
	}
	customContent := "# My Custom Instructions\n"
	ciPath := filepath.Join(ghDir, "copilot-instructions.md")
	if err := os.WriteFile(ciPath, []byte(customContent), 0644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultGitHubLayout()
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(ciPath)
	if string(data) != customContent {
		t.Error("InitLayout overwrote existing copilot-instructions.md")
	}
}

func TestInitLayout_CopilotInstructionsLineLimit(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()
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

func TestInitLayout_CopilotNotCreatedForGitLab(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitLabLayout()

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	ciPath := filepath.Join(root, ".gitlab", "copilot-instructions.md")
	if _, err := os.Stat(ciPath); !os.IsNotExist(err) {
		t.Error("copilot-instructions.md should not be created for GitLab schemas")
	}
}

func TestInitLayout_SeededConfigOnlyContainsPlatformRules(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

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
	// The .github/instructions rule should be present.
	if !strings.Contains(content, ".github/instructions/") {
		t.Error("seeded config should contain .github/instructions/ rule")
	}
}

func TestInitLayout_CreatesDefaultInstructions(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{
		"general.instructions.md",
		"codereview.instructions.md",
		"library.instructions.md",
		"testing.instructions.md",
		"emoji-prevention.instructions.md",
		"backend.instructions.md",
		"frontend.instructions.md",
	}

	instrDir := filepath.Join(root, ".github", "instructions")
	for _, name := range expectedFiles {
		path := filepath.Join(instrDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to be created, got error: %v", name, err)
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
		rel := ".github/instructions/" + name
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
	schema := repogov.DefaultGitHubLayout()

	// First call creates defaults.
	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Modify one file to verify it's not overwritten.
	instrDir := filepath.Join(root, ".github", "instructions")
	customPath := filepath.Join(instrDir, "general.instructions.md")
	custom := "# My Custom General Instructions\n"
	if err := os.WriteFile(customPath, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}

	// Second call should not overwrite anything.
	created2, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created2 {
		if strings.Contains(p, "instructions/") {
			t.Errorf("second call created %s, expected no instruction files", p)
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
	if err := os.MkdirAll(instrDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(instrDir, "custom.instructions.md"),
		[]byte("# Custom\n"), 0644,
	); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultGitHubLayout()
	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// No instruction files should be seeded because the directory is non-empty.
	for _, p := range created {
		if strings.HasPrefix(p, ".github/instructions/") &&
			!strings.HasSuffix(p, ".gitkeep") {
			t.Errorf("should not seed %s when instructions/ is non-empty", p)
		}
	}
}

func TestInitLayout_DefaultInstructionsLineLimit(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	instrDir := filepath.Join(root, ".github", "instructions")
	entries, err := os.ReadDir(instrDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ".gitkeep" {
			continue
		}
		path := filepath.Join(instrDir, e.Name())
		lines, err := repogov.CountLines(path)
		if err != nil {
			t.Fatalf("counting lines in %s: %v", e.Name(), err)
		}
		if lines > 300 {
			t.Errorf("%s has %d lines, want ≤300", e.Name(), lines)
		}
	}
}

func TestInitLayout_CreatesAgentsMd(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGitHubLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// AGENTS.md should exist at the repo root.
	agentsPath := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal("AGENTS.md was not created")
	}
	content := string(data)

	// Should link to docs/ and README.md.
	if !strings.Contains(content, "docs/") {
		t.Error("AGENTS.md should link to docs/")
	}
	if !strings.Contains(content, "README.md") {
		t.Error("AGENTS.md should link to README.md")
	}
	// Should link to .github/instructions/ for GitHub schema.
	if !strings.Contains(content, ".github/instructions/") {
		t.Error("AGENTS.md should link to .github/instructions/ for GitHub schema")
	}
	// Should document nested AGENTS.md scoping.
	if !strings.Contains(content, "AGENTS.md") {
		t.Error("AGENTS.md should mention nested AGENTS.md files")
	}
	// Should be in the created list.
	found := false
	for _, p := range created {
		if p == "AGENTS.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AGENTS.md not in created list")
	}
}

func TestInitLayout_AgentsMdNotOverwritten(t *testing.T) {
	root := t.TempDir()
	custom := "# My Custom AGENTS.md\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := repogov.InitLayout(root, repogov.DefaultGitHubLayout())
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if string(data) != custom {
		t.Error("InitLayout overwrote existing AGENTS.md")
	}
}

// TestInitLayout_AgentsMdContextUpdated verifies that running InitLayout with
// a different platform schema on a repo that already has AGENTS.md updates only
// the ## Context section while preserving all other content.
func TestInitLayout_AgentsMdContextUpdated(t *testing.T) {
	root := t.TempDir()

	// First init: GitHub schema -- creates AGENTS.md with GitHub context.
	if _, err := repogov.InitLayout(root, repogov.DefaultGitHubLayout()); err != nil {
		t.Fatal(err)
	}
	dataAfterGitHub, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dataAfterGitHub), ".github/instructions/") {
		t.Fatal("expected .github/instructions/ after GitHub init")
	}
	if !strings.Contains(string(dataAfterGitHub), ".github/copilot-instructions.md") {
		t.Fatal("expected copilot-instructions.md link after GitHub init")
	}

	// Second init: Cursor schema -- should update ## Context to Cursor links.
	if _, err := repogov.InitLayout(root, repogov.DefaultCursorLayout()); err != nil {
		t.Fatal(err)
	}
	dataAfterCursor, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(dataAfterCursor)

	// Context section now reflects Cursor.
	if !strings.Contains(content, ".cursor/rules/") {
		t.Error("expected .cursor/rules/ link after Cursor init")
	}

	// GitHub-specific context links must be gone.
	if strings.Contains(content, ".github/instructions/") {
		t.Error("stale .github/instructions/ link should be removed after Cursor init")
	}
	if strings.Contains(content, "copilot-instructions.md") {
		t.Error("stale copilot-instructions.md link should be removed after Cursor init")
	}

	// Non-context sections must be preserved.
	if !strings.Contains(content, "## Nested Instructions") {
		t.Error("## Nested Instructions section should be preserved")
	}
	if !strings.Contains(content, "## Dev Environment") {
		t.Error("## Dev Environment section should be preserved")
	}
	if !strings.Contains(content, "## Testing") {
		t.Error("## Testing section should be preserved")
	}
	if !strings.Contains(content, "## PR Instructions") {
		t.Error("## PR Instructions section should be preserved")
	}

	// README and docs links are always present.
	if !strings.Contains(content, "README.md") {
		t.Error("README.md link should always be present in Context")
	}
	if !strings.Contains(content, "docs/") {
		t.Error("docs/ link should always be present in Context")
	}
}

func TestInitLayout_AgentsMdLineLimit(t *testing.T) {
	root := t.TempDir()
	_, err := repogov.InitLayout(root, repogov.DefaultGitHubLayout())
	if err != nil {
		t.Fatal(err)
	}
	lines, err := repogov.CountLines(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if lines > 200 {
		t.Errorf("AGENTS.md has %d lines, want ≤200", lines)
	}
}

func TestInitLayout_AgentsMdCreatedForAllPlatforms(t *testing.T) {
	schemas := []repogov.LayoutSchema{
		repogov.DefaultGitHubLayout(),
		repogov.DefaultGitLabLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	for _, schema := range schemas {
		root := t.TempDir()
		created, err := repogov.InitLayout(root, schema)
		if err != nil {
			t.Fatalf("%s: %v", schema.Root, err)
		}
		if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
			t.Errorf("%s: AGENTS.md not created", schema.Root)
		}
		found := false
		for _, p := range created {
			if p == "AGENTS.md" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: AGENTS.md not in created list", schema.Root)
		}
	}
}

func TestInitLayout_SeededConfigIncludesAgentsMd(t *testing.T) {
	root := t.TempDir()
	_, err := repogov.InitLayout(root, repogov.DefaultGitHubLayout())
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".github", "repogov-config.json"))
	if err != nil {
		t.Fatal("repogov-config.json not found")
	}
	if !strings.Contains(string(data), "AGENTS.md") {
		t.Error("seeded repogov-config.json should include AGENTS.md line limit")
	}
}

// TestAgentsMdContent_GitHubLayoutSection verifies that AGENTS.md generated
// for the github platform includes the full .github Layout section listing
// all standard .github/ files and directories.
func TestAgentsMdContent_GitHubLayoutSection(t *testing.T) {
	root := t.TempDir()
	if _, err := repogov.InitLayout(root, repogov.DefaultGitHubLayout()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "## .github Layout") {
		t.Error("github AGENTS.md should contain '## .github Layout' section")
	}

	wantItems := []string{
		".github/ISSUE_TEMPLATE/",
		".github/PULL_REQUEST_TEMPLATE/",
		".github/workflows/",
		".github/ISSUE_TEMPLATE.md",
		".github/pull_request_template.md",
		".github/CONTRIBUTING.md",
		".github/CODE_OF_CONDUCT.md",
		".github/SECURITY.md",
		".github/SUPPORT.md",
		".github/FUNDING.yml",
		".github/CODEOWNERS",
		".github/dependabot.yml",
	}
	for _, item := range wantItems {
		if !strings.Contains(content, item) {
			t.Errorf("github AGENTS.md missing layout item: %s", item)
		}
	}
}

// TestAgentsMdContent_NonGitHubPlatformsNoLayoutSection verifies that
// AGENTS.md generated for non-github platforms does NOT contain the
// .github Layout section.
func TestAgentsMdContent_NonGitHubPlatformsNoLayoutSection(t *testing.T) {
	nonGitHub := []repogov.LayoutSchema{
		repogov.DefaultGitLabLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	for _, schema := range nonGitHub {
		root := t.TempDir()
		if _, err := repogov.InitLayout(root, schema); err != nil {
			t.Fatalf("%s: %v", schema.Root, err)
		}
		data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatalf("%s: %v", schema.Root, err)
		}
		if strings.Contains(string(data), "## .github Layout") {
			t.Errorf("%s: AGENTS.md should NOT contain '.github Layout' section", schema.Root)
		}
	}
}

// TestAgentsMdContent_PerPlatformContextLinks verifies the Context section of
// AGENTS.md contains platform-appropriate links for each supported platform.
func TestAgentsMdContent_PerPlatformContextLinks(t *testing.T) {
	tests := []struct {
		name      string
		schema    repogov.LayoutSchema
		wantLinks []string
		noLinks   []string
	}{
		{
			name:   "github",
			schema: repogov.DefaultGitHubLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".github/instructions/",
				".github/copilot-instructions.md",
			},
			noLinks: []string{
				".cursor/", ".windsurf/", ".claude/", ".gitlab/",
			},
		},
		{
			name:   "cursor",
			schema: repogov.DefaultCursorLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".cursor/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".github/instructions/",
				".windsurf/", ".claude/", ".gitlab/",
			},
		},
		{
			name:   "windsurf",
			schema: repogov.DefaultWindsurfLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".windsurf/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".github/instructions/",
				".cursor/", ".claude/", ".gitlab/",
			},
		},
		{
			name:   "claude",
			schema: repogov.DefaultClaudeLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".claude/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".github/instructions/",
				".cursor/", ".windsurf/", ".gitlab/",
			},
		},
		{
			name:   "gitlab",
			schema: repogov.DefaultGitLabLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".github/", ".cursor/", ".windsurf/", ".claude/",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := repogov.InitLayout(root, tc.schema); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)

			for _, want := range tc.wantLinks {
				if !strings.Contains(content, want) {
					t.Errorf("platform %s: AGENTS.md missing expected link %q", tc.name, want)
				}
			}
			for _, noWant := range tc.noLinks {
				if strings.Contains(content, noWant) {
					t.Errorf("platform %s: AGENTS.md should not contain %q", tc.name, noWant)
				}
			}
		})
	}
}

// TestInitAllPlatforms_TempDir runs init for every platform into an isolated
// temp directory, then validates that AGENTS.md is present, within the line
// limit, and carries the correct platform-specific context.
func TestInitAllPlatforms_TempDir(t *testing.T) {
	platforms := []struct {
		name   string
		schema repogov.LayoutSchema
	}{
		{"github", repogov.DefaultGitHubLayout()},
		{"gitlab", repogov.DefaultGitLabLayout()},
		{"cursor", repogov.DefaultCursorLayout()},
		{"windsurf", repogov.DefaultWindsurfLayout()},
		{"claude", repogov.DefaultClaudeLayout()},
	}

	for _, p := range platforms {
		t.Run(p.name, func(t *testing.T) {
			root := t.TempDir()
			created, err := repogov.InitLayout(root, p.schema)
			if err != nil {
				t.Fatalf("InitLayout error: %v", err)
			}

			// AGENTS.md must be created.
			agentsPath := filepath.Join(root, "AGENTS.md")
			data, err := os.ReadFile(agentsPath)
			if err != nil {
				t.Fatal("AGENTS.md not created")
			}
			content := string(data)

			// Must be within the 200-line limit.
			lines, err := repogov.CountLines(agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if lines > 200 {
				t.Errorf("AGENTS.md has %d lines, want ≤200", lines)
			}

			// Must appear in the created list.
			found := false
			for _, c := range created {
				if c == "AGENTS.md" {
					found = true
					break
				}
			}
			if !found {
				t.Error("AGENTS.md not in created list")
			}

			// Platform dir must exist.
			if _, err := os.Stat(filepath.Join(root, p.schema.Root)); err != nil {
				t.Errorf("platform dir %s not created", p.schema.Root)
			}

			// README and docs links always present.
			if !strings.Contains(content, "README.md") {
				t.Error("AGENTS.md missing README.md link")
			}
			if !strings.Contains(content, "docs/") {
				t.Error("AGENTS.md missing docs/ link")
			}

			// .github Layout section only for github.
			hasLayout := strings.Contains(content, "## .github Layout")
			if p.name == "github" && !hasLayout {
				t.Error("github AGENTS.md missing .github Layout section")
			}
			if p.name != "github" && hasLayout {
				t.Errorf("%s AGENTS.md should not contain .github Layout section", p.name)
			}
		})
	}
}
