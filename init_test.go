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
	schema := repogov.DefaultCopilotLayout()

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

	// On a fresh init Copilot seeds into rules/ only (consistent with all other
	// agents). instructions/ is only used when it already existed before init.
	if _, err := os.Stat(filepath.Join(ghDir, "rules")); os.IsNotExist(err) {
		t.Error("subdirectory rules was not created")
	}
	if _, err := os.Stat(filepath.Join(ghDir, "instructions")); !os.IsNotExist(err) {
		t.Error("subdirectory instructions should NOT be created on a fresh Copilot init")
	}
	// rules/ MUST be created on a fresh init for Copilot.
	if _, err := os.Stat(filepath.Join(ghDir, "rules")); os.IsNotExist(err) {
		t.Error(".github/rules should be created on a fresh Copilot init")
	}

	// Verify copilot-instructions.md was seeded.
	if _, err := os.Stat(filepath.Join(ghDir, "copilot-instructions.md")); os.IsNotExist(err) {
		t.Error("copilot-instructions.md was not created")
	}
}

func TestInitLayout_Idempotent(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

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
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customContent := `{"default":100}`
	cfgPath := filepath.Join(ghDir, "repogov-config.json")
	if err := os.WriteFile(cfgPath, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	schema := repogov.DefaultCopilotLayout()
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

	// CLAUDE.md must be created with {{.Agent}} rendered — not emitted literally.
	claudeMd := filepath.Join(claudeDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMd); os.IsNotExist(err) {
		t.Error(".claude/CLAUDE.md was not created")
	}
	if data, err := os.ReadFile(claudeMd); err == nil {
		if strings.Contains(string(data), "{{.Agent}}") {
			t.Error(".claude/CLAUDE.md contains literal {{.Agent}} — template was not rendered")
		}
		if !strings.Contains(string(data), "-agent claude") {
			t.Error(".claude/CLAUDE.md missing '-agent claude'; {{.Agent}} was not substituted")
		}
	}
}

func TestInitLayout_KiroSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultKiroLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected Kiro directories to be created")
	}
	kiroDir := filepath.Join(root, ".kiro")
	if _, err := os.Stat(kiroDir); os.IsNotExist(err) {
		t.Error(".kiro directory was not created")
	}
	steeringDir := filepath.Join(kiroDir, "steering")
	if _, err := os.Stat(steeringDir); os.IsNotExist(err) {
		t.Error(".kiro/steering directory was not created")
	}
}

func TestInitLayout_GeminiSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultGeminiLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected Gemini files to be created")
	}

	// GEMINI.md must be created at root with {{.Agent}} rendered.
	geminiMd := filepath.Join(root, "GEMINI.md")
	if _, err := os.Stat(geminiMd); os.IsNotExist(err) {
		t.Error("GEMINI.md was not created at repository root")
	}
	if data, err := os.ReadFile(geminiMd); err == nil {
		if strings.Contains(string(data), "{{.Agent}}") {
			t.Error("GEMINI.md contains literal {{.Agent}} — template was not rendered")
		}
		if !strings.Contains(string(data), "-agent gemini") {
			t.Error("GEMINI.md missing '-agent gemini'; {{.Agent}} was not substituted")
		}
	}
}

func TestInitLayout_ContinueSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultContinueLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected Continue directories to be created")
	}
	continueDir := filepath.Join(root, ".continue")
	if _, err := os.Stat(continueDir); os.IsNotExist(err) {
		t.Error(".continue directory was not created")
	}
	rulesDir := filepath.Join(continueDir, "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Error(".continue/rules directory was not created")
	}
}

func TestInitLayout_ClineSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultClineLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected Cline directories to be created")
	}
	clinerDir := filepath.Join(root, ".clinerules")
	if _, err := os.Stat(clinerDir); os.IsNotExist(err) {
		t.Error(".clinerules directory was not created")
	}

	// Rule files go directly into .clinerules/ (the "." DirRule).
	generalPath := filepath.Join(clinerDir, "general.md")
	if _, err := os.Stat(generalPath); os.IsNotExist(err) {
		t.Error(".clinerules/general.md was not seeded")
	}
	// Verify the emoji-prevention link points to .clinerules/ (not .clinerules/rules/).
	data, _ := os.ReadFile(generalPath)
	if content := string(data); !strings.Contains(content, ".clinerules/emoji-prevention.md") {
		t.Errorf("general.md emoji link should be .clinerules/emoji-prevention.md, got:\n%s", content)
	}
}

func TestInitLayout_RooCodeSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultRooCodeLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected Roo Code directories to be created")
	}
	rooDir := filepath.Join(root, ".roo")
	if _, err := os.Stat(rooDir); os.IsNotExist(err) {
		t.Error(".roo directory was not created")
	}
	rulesDir := filepath.Join(rooDir, "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Error(".roo/rules directory was not created")
	}
}

func TestInitLayout_JetBrainsSchema(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultJetBrainsLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("expected JetBrains directories to be created")
	}
	jbDir := filepath.Join(root, ".aiassistant")
	if _, err := os.Stat(jbDir); os.IsNotExist(err) {
		t.Error(".aiassistant directory was not created")
	}
	rulesDir := filepath.Join(jbDir, "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Error(".aiassistant/rules directory was not created")
	}
}

// TestInitLayoutWithConfig_AlwaysCreate verifies that when
// Config.InitAlwaysCreate is true, missing template files are seeded into
// a pre-existing non-empty directory.
func TestInitLayoutWithConfig_AlwaysCreate(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()

	// Pre-create the .cursor/rules directory with an existing file so that
	// isDirEmpty returns false.
	rulesDir := filepath.Join(root, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "existing.mdc"), []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without InitAlwaysCreate, the default rule file should NOT be seeded
	// because the directory is not empty.
	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created {
		if p == ".cursor/rules/general.mdc" {
			t.Error("InitLayout should not seed general.mdc into non-empty rules dir")
		}
	}

	// With InitAlwaysCreate=true, the missing default file should be seeded.
	cfg := repogov.Config{InitAlwaysCreate: true}
	created2, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range created2 {
		if p == ".cursor/rules/general.md" {
			found = true
		}
	}
	if !found {
		t.Error("InitLayoutWithConfig with InitAlwaysCreate should seed general.md into non-empty rules dir")
	}

	// The pre-existing file must not be overwritten.
	data, _ := os.ReadFile(filepath.Join(rulesDir, "existing.mdc"))
	if string(data) != "# existing\n" {
		t.Error("existing.mdc was overwritten")
	}
}

// TestInitLayoutWithConfig_AlwaysCreateFalse verifies the default behavior:
// template files are NOT seeded into an existing non-empty directory.
func TestInitLayoutWithConfig_AlwaysCreateFalse(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()

	// Pre-create the rules directory with an existing file.
	rulesDir := filepath.Join(root, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "custom.mdc"), []byte("# custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := repogov.Config{InitAlwaysCreate: false}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created {
		if p == ".cursor/rules/general.mdc" {
			t.Errorf("InitLayoutWithConfig with InitAlwaysCreate=false should not seed into non-empty dir; created: %v", created)
		}
	}
}

// TestInitLayoutAllWithConfig_AlwaysCreate verifies that InitLayoutAllWithConfig
// respects InitAlwaysCreate across all schemas.
func TestInitLayoutAllWithConfig_AlwaysCreate(t *testing.T) {
	root := t.TempDir()
	schemas := []repogov.LayoutSchema{repogov.DefaultCursorLayout()}

	// Pre-create the rules directory with a file so it is not empty.
	rulesDir := filepath.Join(root, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "existing.mdc"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := repogov.Config{InitAlwaysCreate: true}
	created, err := repogov.InitLayoutAllWithConfig(root, schemas, cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range created {
		if p == ".cursor/rules/general.md" {
			found = true
		}
	}
	if !found {
		t.Error("InitLayoutAllWithConfig with InitAlwaysCreate should seed general.md")
	}
}

// TestInitLayoutWithConfig_Descriptive_False verifies that with DescriptiveNames=false
// (the default), non-GitHub agents seed the full template set as plain <name>.md files
// into rules/, and GitHub Copilot seeds them into instructions/.
func TestInitLayoutWithConfig_Descriptive_False(t *testing.T) {
	// Non-GitHub agent: .windsurf rules should include general.md (no .instructions suffix).
	root := t.TempDir()
	schema := repogov.DefaultWindsurfLayout()
	cfg := repogov.Config{DescriptiveNames: false}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range created {
		if p == ".windsurf/rules/general.md" {
			found = true
		}
		if strings.Contains(p, "instructions.md") {
			t.Errorf("DescriptiveNames=false must not create *.instructions.md files, got: %s", p)
		}
	}
	if !found {
		t.Errorf("DescriptiveNames=false should create .windsurf/rules/general.md; created: %v", created)
	}
	if _, err := os.Stat(filepath.Join(root, ".windsurf", "rules", "general.md")); os.IsNotExist(err) {
		t.Error(".windsurf/rules/general.md was not created on disk")
	}

	// GitHub fresh init with DescriptiveNames=false: ALL templates seeded as <name>.md into rules/.
	allTemplates := []string{
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
	root2 := t.TempDir()
	schema2 := repogov.DefaultCopilotLayout()
	cfg2 := repogov.Config{DescriptiveNames: false}
	created2, err := repogov.InitLayoutWithConfig(root2, schema2, cfg2)
	if err != nil {
		t.Fatal(err)
	}
	rulesDir2 := filepath.Join(root2, ".github", "rules")
	for _, name := range allTemplates {
		if _, err := os.Stat(filepath.Join(rulesDir2, name)); os.IsNotExist(err) {
			t.Errorf("DescriptiveNames=false: %s was not created on disk", name)
		}
	}
	for _, p := range created2 {
		if strings.HasPrefix(p, ".github/rules/") && strings.HasSuffix(p, ".instructions.md") {
			t.Errorf("DescriptiveNames=false must not create *.instructions.md in rules/, got: %s", p)
		}
	}
}

// TestInitLayoutWithConfig_Descriptive_True verifies that with Descriptive=true,
// instruction files use the <name>.instructions.md naming convention. For .github
// this is provided by createDefaultInstructions into instructions/; for other agents
// by createDefaultInstructions into rules/ with the descriptive flag.
func TestInitLayoutWithConfig_Descriptive_True(t *testing.T) {
	// Non-GitHub agent: .windsurf rules should use general.instructions.md.
	root := t.TempDir()
	schema := repogov.DefaultWindsurfLayout()
	cfg := repogov.Config{DescriptiveNames: true}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range created {
		if p == ".windsurf/rules/general.instructions.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("Descriptive=true should create .windsurf/rules/general.instructions.md; created: %v", created)
	}
	if _, err := os.Stat(filepath.Join(root, ".windsurf", "rules", "general.instructions.md")); os.IsNotExist(err) {
		t.Error(".windsurf/rules/general.instructions.md was not created on disk")
	}

	// GitHub fresh init with Descriptive=true: rules/ should contain *.instructions.md files.
	root2 := t.TempDir()
	schema2 := repogov.DefaultCopilotLayout()
	cfg2 := repogov.Config{DescriptiveNames: true}
	created2, err := repogov.InitLayoutWithConfig(root2, schema2, cfg2)
	if err != nil {
		t.Fatal(err)
	}
	foundInstructions := false
	for _, p := range created2 {
		if strings.HasPrefix(p, ".github/rules/") && strings.HasSuffix(p, ".instructions.md") {
			foundInstructions = true
			break
		}
	}
	if !foundInstructions {
		t.Errorf("Descriptive=true should create .github/rules/*.instructions.md files; created: %v", created2)
	}
	// Ensure no plain .md files were created in .github/rules/ (only *.instructions.md).
	rulesDir2 := filepath.Join(root2, ".github", "rules")
	if entries, err := os.ReadDir(rulesDir2); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".instructions.md") {
				t.Errorf("Descriptive=true must not create plain .md files in rules/, got: %s", name)
			}
		}
	}
}

// TestInitLayoutWithConfig_Descriptive_Cursor verifies that .cursor/rules/ receives
// the full template set. With descriptive=false files use plain <name>.md; with
// descriptive=true files use <name>.instructions.md.
func TestInitLayoutWithConfig_Descriptive_Cursor(t *testing.T) {
	// descriptive=false: files use plain .md
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()
	cfg := repogov.Config{DescriptiveNames: false}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatalf("descriptive=false: %v", err)
	}
	found := false
	for _, p := range created {
		if p == ".cursor/rules/general.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("descriptive=false: expected .cursor/rules/general.md; created: %v", created)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "rules", "general.md")); os.IsNotExist(err) {
		t.Error("descriptive=false: .cursor/rules/general.md was not created on disk")
	}

	// descriptive=true: files use .instructions.md
	root2 := t.TempDir()
	cfg2 := repogov.Config{DescriptiveNames: true}
	created2, err := repogov.InitLayoutWithConfig(root2, schema, cfg2)
	if err != nil {
		t.Fatalf("descriptive=true: %v", err)
	}
	found2 := false
	for _, p := range created2 {
		if p == ".cursor/rules/general.instructions.md" {
			found2 = true
		}
	}
	if !found2 {
		t.Errorf("descriptive=true: expected .cursor/rules/general.instructions.md; created: %v", created2)
	}
}

// TestInitLayoutWithConfig_ExcludeFiles verifies that InitExcludeFiles prevents
// listed templates from being seeded during init.
func TestInitLayoutWithConfig_ExcludeFiles(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()
	cfg := repogov.Config{
		InitExcludeFiles: []string{"backend", "frontend", "emoji-prevention"},
	}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	instrDir := filepath.Join(root, ".github", "rules")

	// Excluded stems must NOT be created.
	for _, name := range []string{"backend.md", "frontend.md", "emoji-prevention.md"} {
		if _, err := os.Stat(filepath.Join(instrDir, name)); err == nil {
			t.Errorf("excluded template %s should not have been created", name)
		}
		for _, p := range created {
			if strings.HasSuffix(p, "/"+name) {
				t.Errorf("excluded template %s appeared in created list", name)
			}
		}
	}

	// Non-excluded stems SHOULD be created.
	for _, name := range []string{"general.md", "governance.md", "testing.md"} {
		if _, err := os.Stat(filepath.Join(instrDir, name)); os.IsNotExist(err) {
			t.Errorf("non-excluded template %s was not created", name)
		}
	}
}

// TestInitLayoutWithConfig_IncludeFiles verifies that InitIncludeFiles restricts
// seeding to only the listed templates.
func TestInitLayoutWithConfig_IncludeFiles(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()
	cfg := repogov.Config{
		InitIncludeFiles: []string{"general", "governance"},
	}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	instrDir := filepath.Join(root, ".github", "rules")

	// Included stems MUST be created.
	for _, name := range []string{"general.md", "governance.md"} {
		if _, err := os.Stat(filepath.Join(instrDir, name)); os.IsNotExist(err) {
			t.Errorf("included template %s was not created", name)
		}
	}

	// All other stems must NOT be created.
	for _, name := range []string{
		"codereview.md", "library.md", "testing.md",
		"emoji-prevention.md", "backend.md", "frontend.md", "repo.md",
	} {
		if _, err := os.Stat(filepath.Join(instrDir, name)); err == nil {
			t.Errorf("non-included template %s should not have been created", name)
		}
	}

	// IncludeFiles takes precedence: only 2 templates seeded.
	instrCount := 0
	for _, p := range created {
		if strings.HasPrefix(p, ".github/rules/") && strings.HasSuffix(p, ".md") {
			instrCount++
		}
	}
	if instrCount != 2 {
		t.Errorf("expected exactly 2 instruction files seeded, got %d; created: %v", instrCount, created)
	}
}

// TestInitLayoutWithConfig_IncludeFiles_NonCopilot verifies that include/exclude
// also works for non-Copilot agents (rules/ dir).
func TestInitLayoutWithConfig_IncludeFiles_NonCopilot(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultClaudeLayout()
	cfg := repogov.Config{
		InitIncludeFiles: []string{"general", "testing"},
	}
	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		t.Fatal(err)
	}

	rulesDir := filepath.Join(root, ".claude", "rules")

	// Included stems MUST be created.
	for _, name := range []string{"general.md", "testing.md"} {
		if _, err := os.Stat(filepath.Join(rulesDir, name)); os.IsNotExist(err) {
			t.Errorf("included template %s was not created", name)
		}
	}

	// Non-included stems must NOT be created.
	for _, name := range []string{"codereview.md", "backend.md", "frontend.md"} {
		if _, err := os.Stat(filepath.Join(rulesDir, name)); err == nil {
			t.Errorf("non-included template %s should not have been created", name)
		}
	}

	// Count seeded rule files.
	ruleCount := 0
	for _, p := range created {
		if strings.HasPrefix(p, ".claude/rules/") && strings.HasSuffix(p, ".md") {
			ruleCount++
		}
	}
	if ruleCount != 2 {
		t.Errorf("expected exactly 2 rule files seeded for claude, got %d; created: %v", ruleCount, created)
	}
}

// TestInitLayoutWithConfig_UnsafeIncludeStem verifies that an unsafe stem in
// InitIncludeFiles causes InitLayoutWithConfig to return an error before any
// filesystem changes are made.
func TestInitLayoutWithConfig_UnsafeIncludeStem(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()
	cfg := repogov.Config{
		InitIncludeFiles: []string{"../hack"},
	}
	_, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err == nil {
		t.Fatal("expected an error for unsafe stem '../hack', got nil")
	}
}

// TestInitLayoutWithConfig_UnsafeExcludeStem verifies that an unsafe stem in
// InitExcludeFiles causes InitLayoutWithConfig to return an error.
func TestInitLayoutWithConfig_UnsafeExcludeStem(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()
	cfg := repogov.Config{
		InitExcludeFiles: []string{"my template!"},
	}
	_, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err == nil {
		t.Fatal("expected an error for unsafe stem 'my template!', got nil")
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
	if _, err := os.Stat(filepath.Join(ghDir, "rules")); !os.IsNotExist(err) {
		t.Error(".github/rules should NOT be created when instructions/ is pre-existing")
	}
	// instructions/ must still exist.
	if _, err := os.Stat(instrDir); os.IsNotExist(err) {
		t.Error(".github/instructions should still exist")
	}
	// copilot-instructions.md must reference instructions/, not rules/.
	ciPath := filepath.Join(ghDir, "copilot-instructions.md")
	data, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatal("copilot-instructions.md was not created:", err)
	}
	ciContent := string(data)
	if !strings.Contains(ciContent, ".github/instructions/") {
		t.Error("copilot-instructions.md should reference .github/instructions/ when instructions/ is pre-existing")
	}
	if strings.Contains(ciContent, ".github/rules/") {
		t.Error("copilot-instructions.md should NOT reference .github/rules/ when instructions/ is pre-existing")
	}
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
	for _, p := range created {
		if strings.HasPrefix(p, ".github/instructions/") &&
			!strings.HasSuffix(p, ".gitkeep") {
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

func TestInitLayout_CreatesAgentsMd(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

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
	// Should link to .github/rules/ for GitHub schema.
	if !strings.Contains(content, ".github/rules/") {
		t.Error("AGENTS.md should link to .github/rules/ for GitHub schema")
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
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout())
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
	if _, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}
	dataAfterGitHub, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dataAfterGitHub), ".github/rules/") {
		t.Fatal("expected .github/rules/ after GitHub init")
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

	// Extract the ## Context section only (from heading to the next ## heading).
	ctxStart := strings.Index(content, "## Context\n")
	if ctxStart == -1 {
		t.Fatal("## Context section not found")
	}
	ctxEnd := strings.Index(content[ctxStart+len("## Context\n"):], "\n## ")
	var contextSection string
	if ctxEnd == -1 {
		contextSection = content[ctxStart:]
	} else {
		contextSection = content[ctxStart : ctxStart+len("## Context\n")+ctxEnd]
	}

	// Context section now reflects Cursor.
	if !strings.Contains(contextSection, ".cursor/rules/") {
		t.Error("expected .cursor/rules/ link in Context after Cursor init")
	}

	// GitHub-specific context links must be gone from the Context section.
	if strings.Contains(contextSection, ".github/rules/") {
		t.Error("stale .github/rules/ link should be removed from Context after Cursor init")
	}
	if strings.Contains(contextSection, "copilot-instructions.md") {
		t.Error("stale copilot-instructions.md link should be removed from Context after Cursor init")
	}

	// Non-context sections must be preserved.
	if !strings.Contains(content, "## Nested Instructions") {
		t.Error("## Nested Instructions section should be preserved")
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
	_, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout())
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
		repogov.DefaultCopilotLayout(),
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
	_, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout())
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

// TestInitLayoutAll_ConfigAtRoot verifies that InitLayoutAll places
// repogov-config.json at the repository root (not inside a per-agent dir)
// and that the config covers all agent platforms.
func TestInitLayoutAll_ConfigAtRoot(t *testing.T) {
	root := t.TempDir()
	schemas := []repogov.LayoutSchema{
		repogov.DefaultCopilotLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	_, err := repogov.InitLayoutAll(root, schemas)
	if err != nil {
		t.Fatal(err)
	}

	// Config must be at root.
	rootCfg := filepath.Join(root, "repogov-config.json")
	data, err := os.ReadFile(rootCfg)
	if err != nil {
		t.Fatalf("repogov-config.json not found at root: %v", err)
	}
	content := string(data)

	// Root config must cover all platforms.
	for _, prefix := range []string{".github/", ".cursor/", ".windsurf/", ".claude/"} {
		if !strings.Contains(content, prefix) {
			t.Errorf("root config missing rules for %s", prefix)
		}
	}

	// Config must NOT be duplicated in .github/.
	githubCfg := filepath.Join(root, ".github", "repogov-config.json")
	if _, err := os.Stat(githubCfg); !os.IsNotExist(err) {
		t.Error(".github/repogov-config.json should not exist when InitLayoutAll is used")
	}

	// Second call must be idempotent.
	created, err := repogov.InitLayoutAll(root, schemas)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("second InitLayoutAll should create nothing, got %v", created)
	}
}

// TestAgentsMdContent_GitHubLayoutSection verifies that repo.instructions.md
// is seeded for the copilot platform and contains the .github Layout section
// with all standard .github/ files and directories. It also confirms the
// section is no longer inlined in AGENTS.md itself.
func TestAgentsMdContent_GitHubLayoutSection(t *testing.T) {
	root := t.TempDir()
	cfg := repogov.Config{DescriptiveNames: true}
	if _, err := repogov.InitLayoutWithConfig(root, repogov.DefaultCopilotLayout(), cfg); err != nil {
		t.Fatal(err)
	}

	// AGENTS.md must NOT inline the .github Layout section.
	agData, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agData), "## .github Layout") {
		t.Error("AGENTS.md should not inline '## .github Layout'; it belongs in repo.instructions.md")
	}

	// repo.instructions.md must be seeded into rules/ and contain the layout section.
	repoInstr := filepath.Join(root, ".github", "rules", "repo.instructions.md")
	data, err := os.ReadFile(repoInstr)
	if err != nil {
		t.Fatal("repo.instructions.md was not seeded:", err)
	}
	content := string(data)

	if !strings.Contains(content, "## .github Layout") {
		t.Error("repo.instructions.md should contain '## .github Layout' section")
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
			t.Errorf("repo.instructions.md missing layout item: %s", item)
		}
	}

	if !strings.Contains(content, "## Pull Requests / Merge Requests") {
		t.Error("repo.instructions.md should contain '## Pull Requests / Merge Requests' section")
	}
	if !strings.Contains(content, "## Commit Standards") {
		t.Error("repo.instructions.md should contain '## Commit Standards' section")
	}
}

// TestAgentsMdContent_NoInlineLayoutSection verifies that AGENTS.md generated
// for any platform does not inline the .github Layout section; that content
// now lives exclusively in repo.instructions.md.
func TestAgentsMdContent_NoInlineLayoutSection(t *testing.T) {
	platforms := []repogov.LayoutSchema{
		repogov.DefaultCopilotLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	for _, schema := range platforms {
		root := t.TempDir()
		if _, err := repogov.InitLayout(root, schema); err != nil {
			t.Fatalf("%s: %v", schema.Root, err)
		}
		data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatalf("%s: %v", schema.Root, err)
		}
		if strings.Contains(string(data), "## .github Layout") {
			t.Errorf("%s: AGENTS.md should NOT inline '.github Layout' section", schema.Root)
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
			name:   "copilot",
			schema: repogov.DefaultCopilotLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".github/rules/",
				".github/copilot-instructions.md",
			},
			noLinks: []string{
				".cursor/", ".windsurf/", ".claude/",
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
				".windsurf/", ".claude/",
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
				".cursor/", ".claude/",
			},
		},
		{
			name:   "claude",
			schema: repogov.DefaultClaudeLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".claude/rules/",
				".claude/agents/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".github/instructions/",
				".cursor/", ".windsurf/",
			},
		},
		{
			name:   "kiro",
			schema: repogov.DefaultKiroLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".kiro/steering/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
			},
		},
		{
			name:   "gemini",
			schema: repogov.DefaultGeminiLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				"GEMINI.md",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
			},
		},
		{
			name:   "continue",
			schema: repogov.DefaultContinueLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".continue/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
			},
		},
		{
			name:   "cline",
			schema: repogov.DefaultClineLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".clinerules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
			},
		},
		{
			name:   "roocode",
			schema: repogov.DefaultRooCodeLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".roo/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
			},
		},
		{
			name:   "jetbrains",
			schema: repogov.DefaultJetBrainsLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".aiassistant/rules/",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
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
		{"copilot", repogov.DefaultCopilotLayout()},
		{"cursor", repogov.DefaultCursorLayout()},
		{"windsurf", repogov.DefaultWindsurfLayout()},
		{"claude", repogov.DefaultClaudeLayout()},
		{"kiro", repogov.DefaultKiroLayout()},
		{"gemini", repogov.DefaultGeminiLayout()},
		{"continue", repogov.DefaultContinueLayout()},
		{"cline", repogov.DefaultClineLayout()},
		{"roocode", repogov.DefaultRooCodeLayout()},
		{"jetbrains", repogov.DefaultJetBrainsLayout()},
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

			// AGENTS.md must never inline the .github Layout section.
			if strings.Contains(content, "## .github Layout") {
				t.Errorf("%s AGENTS.md should not inline .github Layout section", p.name)
			}
		})
	}
}

// TestUpdateAgentsMdContextAll_MergedLinksForAllPlatforms verifies that after
// calling UpdateAgentsMdContextAll with all supported platform schemas, the
// ## Context section of AGENTS.md contains links for every platform directory
// and does not contain stale links from unrelated platforms.
func TestUpdateAgentsMdContextAll_MergedLinksForAllPlatforms(t *testing.T) {
	root := t.TempDir()

	// Seed AGENTS.md via a single-platform init so it has a ## Context section.
	if _, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}

	schemas := []repogov.LayoutSchema{
		repogov.DefaultCopilotLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	if err := repogov.UpdateAgentsMdContextAll(root, schemas); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	wantLinks := []string{
		"README.md",
		"docs/",
		".github/rules/",
		".github/copilot-instructions.md",
		".cursor/rules/",
		".windsurf/rules/",
		".claude/rules/",
		".claude/agents/",
	}
	for _, link := range wantLinks {
		if !strings.Contains(content, link) {
			t.Errorf("merged AGENTS.md missing expected link %q", link)
		}
	}

	// Sections outside Context must be preserved.
	if !strings.Contains(content, "## Nested Instructions") {
		t.Error("## Nested Instructions section should be preserved")
	}
}

// TestUpdateAgentsMdContextAll_NoContextSection verifies that a file without
// a ## Context heading is left unchanged.
func TestUpdateAgentsMdContextAll_NoContextSection(t *testing.T) {
	root := t.TempDir()
	original := "# AGENTS.md\n\nNo context section here.\n"
	agentsPath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := repogov.UpdateAgentsMdContextAll(root, []repogov.LayoutSchema{repogov.DefaultCopilotLayout()}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(agentsPath)
	if string(data) != original {
		t.Error("UpdateAgentsMdContextAll modified a file with no ## Context section")
	}
}

// TestUpdateAgentsMdContextAll_FileNotExist verifies that a missing AGENTS.md
// is silently ignored (no error, no file created).
func TestUpdateAgentsMdContextAll_FileNotExist(t *testing.T) {
	root := t.TempDir()
	if err := repogov.UpdateAgentsMdContextAll(root, []repogov.LayoutSchema{repogov.DefaultCopilotLayout()}); err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("UpdateAgentsMdContextAll should not create AGENTS.md when it does not exist")
	}
}

// TestUpdateAgentsMdContextAll_DeduplicatesLinks verifies that passing the same
// schema twice does not produce duplicate lines in the Context section.
func TestUpdateAgentsMdContextAll_DeduplicatesLinks(t *testing.T) {
	root := t.TempDir()
	if _, err := repogov.InitLayout(root, repogov.DefaultCopilotLayout()); err != nil {
		t.Fatal(err)
	}
	// Pass github schema twice -- should not produce duplicate lines.
	schemas := []repogov.LayoutSchema{
		repogov.DefaultCopilotLayout(),
		repogov.DefaultCopilotLayout(),
	}
	if err := repogov.UpdateAgentsMdContextAll(root, schemas); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	content := string(data)

	// Count occurrences of the full rules entry line -- must be exactly 1.
	instrLine := "- Copilot rule files: [.github/rules/](.github/rules/)"
	count := strings.Count(content, instrLine)
	if count != 1 {
		t.Errorf("expected exactly 1 Copilot rules entry, got %d", count)
	}
}

// TestInitAllPlatforms_AllToTemp_MergedContext verifies that initializing all
// platforms in sequence produces an AGENTS.md Context section that references
// every scaffolded platform directory -- not only the last one.
func TestInitAllPlatforms_AllToTemp_MergedContext(t *testing.T) {
	root := t.TempDir()

	schemas := []repogov.LayoutSchema{
		repogov.DefaultCopilotLayout(),
		repogov.DefaultCursorLayout(),
		repogov.DefaultWindsurfLayout(),
		repogov.DefaultClaudeLayout(),
	}
	for _, schema := range schemas {
		if _, err := repogov.InitLayout(root, schema); err != nil {
			t.Fatalf("%s: InitLayout error: %v", schema.Root, err)
		}
	}
	if err := repogov.UpdateAgentsMdContextAll(root, schemas); err != nil {
		t.Fatalf("UpdateAgentsMdContextAll error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Every platform with a rules/ dir must appear in Context.
	wantLinks := []string{
		".github/rules/",
		".github/copilot-instructions.md",
		".cursor/rules/",
		".windsurf/rules/",
		".claude/rules/",
		".claude/agents/",
	}
	for _, link := range wantLinks {
		if !strings.Contains(content, link) {
			t.Errorf("all-platforms AGENTS.md missing context link %q", link)
		}
	}

	// Context must not duplicate any link entry. Check by counting the full
	// entry lines rather than the raw path strings (which appear twice per
	// markdown link: once in text, once in URL).
	wantEntries := []string{
		"- Copilot rule files: [.github/rules/](.github/rules/)",
		"- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)",
		"- Cursor rule files: [.cursor/rules/](.cursor/rules/)",
		"- Windsurf rule files: [.windsurf/rules/](.windsurf/rules/)",
		"- Claude rule files: [.claude/rules/](.claude/rules/)",
		"- Agent definitions: [.claude/agents/](.claude/agents/)",
	}
	for _, entry := range wantEntries {
		if count := strings.Count(content, entry); count != 1 {
			t.Errorf("expected entry %q exactly once, found %d times", entry, count)
		}
	}

	// Non-context sections preserved.
	if !strings.Contains(content, "## Nested Instructions") {
		t.Error("## Nested Instructions section should be preserved after all-platforms init")
	}
}
