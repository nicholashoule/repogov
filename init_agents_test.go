package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestInitLayout_CreatesAgentsMd(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// AGENTS.md should exist at the repo root.
	agentsPath := filepath.Join(root, "AGENTS.md")
	assertExists(t, agentsPath)
	assertFileContains(t, agentsPath, "docs/")
	assertFileContains(t, agentsPath, "README.md")
	assertFileContains(t, agentsPath, ".github/rules/")
	assertFileContains(t, agentsPath, "AGENTS.md")
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
		assertExists(t, filepath.Join(root, "AGENTS.md"))
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

	assertFileContains(t, filepath.Join(root, ".github", "repogov-config.json"), "AGENTS.md")
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
	assertNotExists(t, filepath.Join(root, ".github", "repogov-config.json"))

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
	assertFileNotContains(t, filepath.Join(root, "AGENTS.md"), "## .github Layout")

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
		assertFileNotContains(t, filepath.Join(root, "AGENTS.md"), "## .github Layout")
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
		{
			name:   "zed",
			schema: repogov.DefaultZedLayout(),
			wantLinks: []string{
				"README.md",
				"docs/",
				".rules",
			},
			noLinks: []string{
				"copilot-instructions.md",
				".cursor/", ".claude/",
				"GEMINI.md",
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
		{"zed", repogov.DefaultZedLayout()},
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

// TestInitLayout_CreatesSubdirReadme verifies that init creates README.md in
// each non-NoCreate subdirectory with meaningful content.
func TestInitLayout_CreatesSubdirReadme(t *testing.T) {
	tests := []struct {
		name     string
		schema   repogov.LayoutSchema
		dirs     []string // expected README.md locations (relative to root)
		noDirs   []string // directories that should NOT get a README.md
		contains []string // substrings each README must contain
	}{
		{
			name:   "copilot rules",
			schema: repogov.DefaultCopilotLayout(),
			dirs:   []string{".github/rules/README.md"},
			contains: []string{
				"# Rules",
				"rule files",
			},
		},
		{
			name:   "cursor rules",
			schema: repogov.DefaultCursorLayout(),
			dirs:   []string{".cursor/rules/README.md"},
			contains: []string{
				"# Rules",
			},
		},
		{
			name:   "windsurf rules",
			schema: repogov.DefaultWindsurfLayout(),
			dirs:   []string{".windsurf/rules/README.md"},
			contains: []string{
				"# Rules",
			},
		},
		{
			name:   "claude rules and agents",
			schema: repogov.DefaultClaudeLayout(),
			dirs: []string{
				".claude/rules/README.md",
				".claude/agents/README.md",
			},
			contains: []string{"#"},
		},
		{
			name:   "kiro steering",
			schema: repogov.DefaultKiroLayout(),
			dirs:   []string{".kiro/steering/README.md"},
			contains: []string{
				"# Steering",
			},
		},
		{
			name:   "continue rules",
			schema: repogov.DefaultContinueLayout(),
			dirs:   []string{".continue/rules/README.md"},
			contains: []string{
				"# Rules",
			},
		},
		{
			name:   "roocode rules",
			schema: repogov.DefaultRooCodeLayout(),
			dirs:   []string{".roo/rules/README.md"},
			contains: []string{
				"# Rules",
			},
		},
		{
			name:   "jetbrains rules",
			schema: repogov.DefaultJetBrainsLayout(),
			dirs:   []string{".aiassistant/rules/README.md"},
			contains: []string{
				"# Rules",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			created, err := repogov.InitLayout(root, tt.schema)
			if err != nil {
				t.Fatal(err)
			}

			for _, dir := range tt.dirs {
				path := filepath.Join(root, filepath.FromSlash(dir))
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("expected %s to exist: %v", dir, err)
					continue
				}
				content := string(data)
				if len(content) == 0 {
					t.Errorf("%s is empty", dir)
				}
				for _, sub := range tt.contains {
					if !strings.Contains(content, sub) {
						t.Errorf("%s should contain %q, got:\n%s", dir, sub, content)
					}
				}
				// README.md should be in the created list.
				found := false
				for _, p := range created {
					if p == dir {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s not in created list: %v", dir, created)
				}
			}
		})
	}
}

// TestInitLayout_SubdirReadmeNotOverwritten verifies that existing README.md
// files in subdirectories are never overwritten by init.
func TestInitLayout_SubdirReadmeNotOverwritten(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()

	// Pre-create the rules directory with a custom README.md.
	rulesDir := filepath.Join(root, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customContent := "# My Custom Rules README\n"
	if err := os.WriteFile(filepath.Join(rulesDir, "README.md"), []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the file was NOT overwritten.
	data, _ := os.ReadFile(filepath.Join(rulesDir, "README.md"))
	if string(data) != customContent {
		t.Error("InitLayout overwrote existing README.md in rules/")
	}
}

// TestInitLayout_SubdirReadmeIdempotent verifies that running init twice does
// not create README.md a second time.
func TestInitLayout_SubdirReadmeIdempotent(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCursorLayout()

	created1, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	foundFirst := false
	for _, p := range created1 {
		if p == ".cursor/rules/README.md" {
			foundFirst = true
		}
	}
	if !foundFirst {
		t.Error("first init should create .cursor/rules/README.md")
	}

	// Second init should not re-create README.md.
	created2, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created2 {
		if strings.HasSuffix(p, "README.md") {
			t.Errorf("second init should not re-create README.md, got: %s", p)
		}
	}
}

// TestInitLayout_ClineNoSubdirReadme verifies that Cline's root-dir layout
// (".") does not get a README.md, since that would conflict with a project
// README.
func TestInitLayout_ClineNoSubdirReadme(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultClineLayout()

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	// The "." DirRule should not produce a README.md inside .clinerules/.
	for _, p := range created {
		if strings.HasSuffix(p, "README.md") && strings.HasPrefix(p, ".clinerules") {
			t.Errorf("Cline should not create README.md in .clinerules/ (root-dir layout), got: %s", p)
		}
	}
}

// TestInitLayout_SubdirReadmeNoFrontmatterViolation verifies that README.md in
// a directory with frontmatter requirements does not cause a layout check failure.
func TestInitLayout_SubdirReadmeNoFrontmatterViolation(t *testing.T) {
	root := t.TempDir()
	schema := repogov.DefaultCopilotLayout()

	_, err := repogov.InitLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if strings.Contains(r.Path, "README.md") {
			t.Errorf("README.md should not appear in layout results, got: %s -- %s", r.Path, r.Message)
		}
	}
}
