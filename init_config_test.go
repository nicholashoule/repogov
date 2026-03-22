package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

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
	assertExists(t, filepath.Join(root, ".windsurf", "rules", "general.md"))

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
		assertExists(t, filepath.Join(rulesDir2, name))
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
	assertExists(t, filepath.Join(root, ".windsurf", "rules", "general.instructions.md"))

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
	// README.md is exempt — it is infrastructure, not an instruction file.
	rulesDir2 := filepath.Join(root2, ".github", "rules")
	if entries, err := os.ReadDir(rulesDir2); err == nil {
		for _, e := range entries {
			name := e.Name()
			if name == "README.md" {
				continue
			}
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
	assertExists(t, filepath.Join(root, ".cursor", "rules", "general.md"))

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
		assertNotExists(t, filepath.Join(instrDir, name))
		for _, p := range created {
			if strings.HasSuffix(p, "/"+name) {
				t.Errorf("excluded template %s appeared in created list", name)
			}
		}
	}

	// Non-excluded stems SHOULD be created.
	for _, name := range []string{"general.md", "governance.md", "testing.md"} {
		assertExists(t, filepath.Join(instrDir, name))
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
		assertExists(t, filepath.Join(instrDir, name))
	}

	// All other stems must NOT be created.
	for _, name := range []string{
		"codereview.md", "library.md", "testing.md",
		"emoji-prevention.md", "backend.md", "frontend.md", "repo.md",
	} {
		assertNotExists(t, filepath.Join(instrDir, name))
	}

	// IncludeFiles takes precedence: only 2 templates seeded.
	// README.md is infrastructure and not counted as an instruction file.
	instrCount := 0
	for _, p := range created {
		if strings.HasPrefix(p, ".github/rules/") && strings.HasSuffix(p, ".md") && !strings.HasSuffix(p, "README.md") {
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
		assertExists(t, filepath.Join(rulesDir, name))
	}

	// Non-included stems must NOT be created.
	for _, name := range []string{"codereview.md", "backend.md", "frontend.md"} {
		assertNotExists(t, filepath.Join(rulesDir, name))
	}

	// Count seeded rule files.
	// README.md is infrastructure and not counted as a rule file.
	ruleCount := 0
	for _, p := range created {
		if strings.HasPrefix(p, ".claude/rules/") && strings.HasSuffix(p, ".md") && !strings.HasSuffix(p, "README.md") {
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
