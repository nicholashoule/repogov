package repogov

// DefaultGitHubLayout returns a [LayoutSchema] matching standard GitHub
// repository conventions. It expects a .github/ directory with agent
// instruction files, Copilot instructions, and funding configuration.
func DefaultGitHubLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".github",
		Required: []string{},
		Optional: []string{
			"copilot-instructions.md",
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
			},
		},
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
// for rules and subagent definitions. Hooks are configured inside
// settings.json, not as a directory. CLAUDE.md may live at the repo root or
// at .claude/CLAUDE.md; both locations are valid.
func DefaultClaudeLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".claude",
		Required: []string{},
		Optional: []string{
			"CLAUDE.md",
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

// DefaultGitLabLayout returns a [LayoutSchema] matching standard GitLab
// repository conventions. It expects a .gitlab/ directory with merge
// request templates, issue templates, and CI includes.
func DefaultGitLabLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".gitlab",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"merge_request_templates": {
				Glob:        "*.md",
				Min:         0,
				Description: "GitLab merge request templates",
			},
			"issue_templates": {
				Glob:        "*.md",
				Min:         0,
				Description: "GitLab issue templates",
			},
			"ci": {
				Glob:        "*.yml",
				Min:         0,
				Description: "GitLab CI includes",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}
