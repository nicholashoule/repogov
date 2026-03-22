package repogov_test

import (
	"os"
	"path/filepath"
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

	// Verify .github directory and expected subdirectories.
	ghDir := filepath.Join(root, ".github")
	assertExists(t, ghDir)
	assertExists(t, filepath.Join(ghDir, "rules"))
	assertNotExists(t, filepath.Join(ghDir, "instructions"))
	assertExists(t, filepath.Join(ghDir, "copilot-instructions.md"))
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

	assertExists(t, filepath.Join(root, ".cursor"))
	assertExists(t, filepath.Join(root, ".cursor", "rules"))
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

	assertExists(t, filepath.Join(root, ".windsurf"))
	assertExists(t, filepath.Join(root, ".windsurf", "rules"))
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
	assertExists(t, claudeDir)
	for _, dir := range []string{"rules", "agents"} {
		assertExists(t, filepath.Join(claudeDir, dir))
	}

	// CLAUDE.md must be created with {{.Agent}} rendered — not emitted literally.
	claudeMd := filepath.Join(claudeDir, "CLAUDE.md")
	assertExists(t, claudeMd)
	assertFileNotContains(t, claudeMd, "{{.Agent}}")
	assertFileContains(t, claudeMd, "-agent claude")
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
	assertExists(t, filepath.Join(root, ".kiro"))
	assertExists(t, filepath.Join(root, ".kiro", "steering"))
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
	assertExists(t, geminiMd)
	assertFileNotContains(t, geminiMd, "{{.Agent}}")
	assertFileContains(t, geminiMd, "-agent gemini")
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
	assertExists(t, filepath.Join(root, ".continue"))
	assertExists(t, filepath.Join(root, ".continue", "rules"))
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
	assertExists(t, clinerDir)

	// Rule files go directly into .clinerules/ (the "." DirRule).
	generalPath := filepath.Join(clinerDir, "general.md")
	assertExists(t, generalPath)
	assertFileContains(t, generalPath, ".clinerules/emoji-prevention.md")
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
	assertExists(t, filepath.Join(root, ".roo"))
	assertExists(t, filepath.Join(root, ".roo", "rules"))
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
	assertExists(t, filepath.Join(root, ".aiassistant"))
	assertExists(t, filepath.Join(root, ".aiassistant", "rules"))
}
