// presets.go provides built-in [LayoutSchema] factories for each supported
// AI-agent platform: GitHub Copilot, Cursor, Windsurf, and Claude.
// Each function returns an opinionated default that mirrors the platform's
// published conventions; callers may further customize the returned schema.

package repogov

// DefaultCopilotLayout returns a [LayoutSchema] matching GitHub Copilot
// repository conventions. It expects a .github/ directory with agent
// instruction files, Copilot instructions, and funding configuration.
func DefaultCopilotLayout() LayoutSchema {
	return LayoutSchema{
		Root: ".github",
		Required: []string{
			"copilot-instructions.md",
		},
		Optional: []string{
			"CODEOWNERS",
			"FUNDING.yml",
			"SECURITY.md",
			"dependabot.yml",
			"repogov-config.json",
			"repogov.json",
			"line-limits.json",
		},
		Dirs: map[string]DirRule{
			"instructions": {
				Glob:        "*.instructions.md",
				Min:         0,
				Description: "Scoped instruction files",
			}, "rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Copilot scoped rule files",
			}},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"CODEOWNERS",
				"FUNDING.yml",
				"SECURITY.md",
			},
		},
	}
}

// DefaultCursorLayout returns a [LayoutSchema] matching Cursor AI repository
// conventions. It expects a .cursor/ directory with a rules/ subdirectory
// containing scoped rule files. Cursor supports both .md and .mdc extensions;
// .mdc files support frontmatter (description, globs, alwaysApply) for
// fine-grained scoping.
func DefaultCursorLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".cursor",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "",
				Min:         0,
				Description: "Cursor scoped rule files (.md and .mdc)",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultWindsurfLayout returns a [LayoutSchema] matching Windsurf AI repository
// conventions. It expects a .windsurf/ directory with a rules/ subdirectory
// containing scoped rule files (*.md).
func DefaultWindsurfLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".windsurf",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Windsurf scoped rule files",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultClaudeLayout returns a [LayoutSchema] matching Claude Code (Anthropic)
// repository conventions. It expects a .claude/ directory with subdirectories
// for rules and subagent definitions. CLAUDE.md is the primary instruction file,
// analogous to copilot-instructions.md for GitHub Copilot. Hooks are configured
// inside settings.json, not as a directory.
func DefaultClaudeLayout() LayoutSchema {
	return LayoutSchema{
		Root: ".claude",
		Required: []string{
			"CLAUDE.md",
		},
		Optional: []string{
			"settings.json",
			"settings.local.json",
		},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Scoped instruction files (paths frontmatter)",
			},
			"agents": {
				Glob:        "*.md",
				Min:         0,
				Description: "Claude subagent definitions",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"CLAUDE.md",
			},
		},
	}
}
