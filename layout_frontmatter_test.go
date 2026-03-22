package repogov_test

import (
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestCheckLayout_FrontmatterValid(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/instructions/general.instructions.md": "---\napplyTo: \"**\"\n---\n# General\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Status == repogov.Fail {
			t.Errorf("unexpected FAIL: path=%s msg=%s", r.Path, r.Message)
		}
	}

	// Should have a PASS result for the frontmatter key.
	hasPass := false
	for _, r := range results {
		if r.Status == repogov.Pass && strings.Contains(r.Message, "applyTo") {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("expected PASS result for valid frontmatter key 'applyTo'")
	}
}

func TestCheckLayout_FrontmatterMissingDelimiter(t *testing.T) {
	// File does NOT start with "---".
	root := writeTempDir(t, map[string]string{
		".github/instructions/bad.instructions.md": "# No frontmatter\napplyTo: ignored\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFail := false
	for _, r := range results {
		if r.Status == repogov.Fail && strings.Contains(r.Message, "frontmatter delimiter") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for missing frontmatter delimiter")
	}
}

func TestCheckLayout_FrontmatterMissingKey(t *testing.T) {
	// File has frontmatter but is missing the required key.
	root := writeTempDir(t, map[string]string{
		".github/instructions/partial.instructions.md": "---\ntitle: \"Something\"\n---\n# Content\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFail := false
	for _, r := range results {
		if r.Status == repogov.Fail && strings.Contains(r.Message, "applyTo") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for missing required frontmatter key 'applyTo'")
	}
}

func TestCheckLayout_FrontmatterUnclosed(t *testing.T) {
	// File starts with "---" but never closes the frontmatter block.
	root := writeTempDir(t, map[string]string{
		".github/instructions/unclosed.instructions.md": "---\napplyTo: \"**\"\n# No closing delimiter\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFail := false
	for _, r := range results {
		if r.Status == repogov.Fail && strings.Contains(r.Message, "not closed") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for unclosed frontmatter block")
	}
}

func TestCheckLayout_FrontmatterMultipleKeys(t *testing.T) {
	// Require two frontmatter keys; file has only one.
	root := writeTempDir(t, map[string]string{
		".github/instructions/multi.instructions.md": "---\napplyTo: \"**/*.go\"\n---\n# Content\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo", "description"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	passCount, failCount := 0, 0
	for _, r := range results {
		if strings.Contains(r.Message, "frontmatter key") {
			if r.Status == repogov.Pass {
				passCount++
			} else if r.Status == repogov.Fail {
				failCount++
			}
		}
	}
	if passCount != 1 {
		t.Errorf("expected 1 PASS for 'applyTo', got %d", passCount)
	}
	if failCount != 1 {
		t.Errorf("expected 1 FAIL for 'description', got %d", failCount)
	}
}

func TestCheckLayout_FrontmatterNonMatchingGlob(t *testing.T) {
	// File doesn't match the Glob, so frontmatter should NOT be checked.
	root := writeTempDir(t, map[string]string{
		".github/instructions/readme.txt": "no frontmatter here\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"instructions": {
				Glob:        "*.md",
				Frontmatter: []string{"applyTo"},
			},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if strings.Contains(r.Message, "frontmatter") {
			t.Errorf("should not check frontmatter for non-matching glob: %s", r.Message)
		}
	}
}

func TestCheckLayout_CopilotFrontmatter(t *testing.T) {
	// Integration test: DefaultCopilotLayout should enforce applyTo in instructions/.
	root := writeTempDir(t, map[string]string{
		".github/copilot-instructions.md":                          "# Copilot\n",
		".github/instructions/general.instructions.md":             "---\napplyTo: \"**\"\n---\n# General\n",
		".github/instructions/missing-frontmatter.instructions.md": "# Missing frontmatter\n",
	})

	schema := repogov.DefaultCopilotLayout()
	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFrontmatterFail := false
	for _, r := range results {
		if strings.Contains(r.Path, "missing-frontmatter") && r.Status == repogov.Fail && strings.Contains(r.Message, "frontmatter") {
			hasFrontmatterFail = true
		}
	}
	if !hasFrontmatterFail {
		t.Error("expected FAIL for instruction file missing YAML frontmatter")
	}
}

func TestStripFrontmatter(t *testing.T) {
	schema := repogov.DefaultCopilotLayout()

	// Verify the original schema has Frontmatter.
	hasOriginal := false
	for _, rule := range schema.Dirs {
		if len(rule.Frontmatter) > 0 {
			hasOriginal = true
			break
		}
	}
	if !hasOriginal {
		t.Fatal("DefaultCopilotLayout should have Frontmatter requirements")
	}

	stripped := repogov.StripFrontmatter(schema)

	// Stripped schema should have no Frontmatter.
	for dir, rule := range stripped.Dirs {
		if len(rule.Frontmatter) > 0 {
			t.Errorf("StripFrontmatter: dir %q still has Frontmatter %v", dir, rule.Frontmatter)
		}
	}

	// Original schema should be unchanged.
	for _, rule := range schema.Dirs {
		if len(rule.Frontmatter) > 0 {
			return // original preserved
		}
	}
	t.Error("StripFrontmatter mutated the original schema")
}

func TestStripFrontmatter_SkipsFrontmatterCheck(t *testing.T) {
	// With StripFrontmatter, a file missing frontmatter should NOT fail.
	root := writeTempDir(t, map[string]string{
		".github/copilot-instructions.md":                     "# Copilot\n",
		".github/instructions/no-frontmatter.instructions.md": "# No frontmatter\n",
	})

	schema := repogov.StripFrontmatter(repogov.DefaultCopilotLayout())
	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if strings.Contains(r.Message, "frontmatter") {
			t.Errorf("should not check frontmatter after StripFrontmatter: %s", r.Message)
		}
	}
}
