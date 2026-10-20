package repogov

import (
	"strings"
	"testing"
)

func TestCursorRule(t *testing.T) {
	always := "---\napplyTo: \"**\"\n---\n\n# General Instructions\n\nBody.\n"
	name, body := cursorRule("general.instructions.md", always)
	if name != "general.mdc" {
		t.Errorf("name = %q", name)
	}
	if !strings.HasPrefix(body, "---\ndescription: \"General Instructions\"\nalwaysApply: true\n---\n\n# General Instructions") {
		t.Errorf("always-apply frontmatter wrong:\n%s", body)
	}
	if strings.Contains(body, "applyTo") {
		t.Error("Copilot applyTo must not leak into a Cursor rule")
	}

	scoped := "---\napplyTo: \"**/*.md\"\n---\n\n# Documentation Review Instructions\n"
	name, body = cursorRule("codereview.instructions.md", scoped)
	if name != "codereview.mdc" || !strings.Contains(body, "globs: \"**/*.md\"\nalwaysApply: false\n") {
		t.Errorf("scoped rule wrong: %s\n%s", name, body)
	}

	// No frontmatter in the template: defaults to always-apply, heading as description.
	_, body = cursorRule("x.instructions.md", "# Plain\n\ntext\n")
	if !strings.Contains(body, "alwaysApply: true") || !strings.Contains(body, `description: "Plain"`) {
		t.Errorf("fallback wrong:\n%s", body)
	}
}
