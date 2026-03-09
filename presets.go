// presets.go provides built-in [LayoutSchema] factories for each supported
// AI-agent platform: GitHub Copilot, Cursor, Windsurf, and Claude.
// Each function returns an opinionated default that mirrors the platform's
// published conventions; callers may further customize the returned schema.

package repogov

// DefaultRootLayout returns a [LayoutSchema] for the repository root directory.
// It recognizes common root-level files shared across GitHub and GitLab
// conventions, and marks well-known subdirectories as managed so their contents
// do not produce unexpected-file warnings. No files are required; all entries
// are optional. Use -agent root to validate or scaffold the root layout.
func DefaultRootLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".",
		Required: []string{},
		Optional: []string{
			".editorconfig",
			".gitattributes",
			".gitignore",
			"AGENTS.md",
			"CHANGELOG.md",
			"CODE_OF_CONDUCT.md",
			"CONTRIBUTING.md",
			"LICENSE",
			"Makefile",
			"README.md",
			"SECURITY.md",
		},
		Dirs: map[string]DirRule{
			// AI-agent platform dirs — managed by their own layout schemas.
			".claude":   {Description: "Claude Code configuration"},
			".cursor":   {Description: "Cursor AI configuration"},
			".github":   {Description: "GitHub configuration"},
			".gitlab":   {Description: "GitLab configuration"},
			".windsurf": {Description: "Windsurf AI configuration"},
			// Common project subdirectories.
			"bin":          {Description: "Compiled binaries"},
			"build":        {Description: "Build output"},
			"cmd":          {Description: "Main entry points"},
			"dist":         {Description: "Distribution output"},
			"docs":         {Description: "Documentation"},
			"internal":     {Description: "Private packages"},
			"lib":          {Description: "Library code"},
			"node_modules": {Description: "Node.js dependencies"},
			"pkg":          {Description: "Public packages"},
			"scripts":      {Description: "Helper scripts"},
			"src":          {Description: "Source code"},
			"target":       {Description: "Build target (Rust/Maven)"},
			"test":         {Description: "Test code"},
			"testdata":     {Description: "Test fixtures"},
			"tests":        {Description: "Test code"},
			"vendor":       {Description: "Vendored dependencies"},
			".vscode":      {Description: "VS Code workspace settings"},
		},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"AGENTS.md",
				"CHANGELOG.md",
				"CODE_OF_CONDUCT.md",
				"CONTRIBUTING.md",
				"LICENSE",
				"Makefile",
				"README.md",
				"SECURITY.md",
			},
		},
	}
}

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
			"CODE_OF_CONDUCT.md",
			"CODEOWNERS",
			"CONTRIBUTING.md",
			"FUNDING.yml",
			"ISSUE_TEMPLATE.md",
			"PULL_REQUEST_TEMPLATE.md",
			"SECURITY.md",
			"SUPPORT.md",
			"dependabot.yml",
			"pull_request_template.md",
			"repogov-config.json",
			"repogov-config.yaml",
			"repogov-config.yml",
		},
		Dirs: map[string]DirRule{
			"ISSUE_TEMPLATE": {
				Glob:        "",
				Min:         0,
				Description: "GitHub issue templates",
				NoCreate:    true,
			},
			"PULL_REQUEST_TEMPLATE": {
				Glob:        "",
				Min:         0,
				Description: "GitHub pull request templates",
				NoCreate:    true,
			},
			"instructions": {
				Glob:        "*.md",
				Min:         0,
				Description: "Scoped instruction files",
			},
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Copilot scoped rule files",
			},
			"workflows": {
				Glob:        "",
				Min:         0,
				Description: "GitHub Actions workflows (recognized; contents not enforced)",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"CODE_OF_CONDUCT.md",
				"CODEOWNERS",
				"CONTRIBUTING.md",
				"FUNDING.yml",
				"ISSUE_TEMPLATE",
				"ISSUE_TEMPLATE.md",
				"PULL_REQUEST_TEMPLATE",
				"PULL_REQUEST_TEMPLATE.md",
				"SECURITY.md",
				"SUPPORT.md",
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

// DefaultGitLabLayout returns a [LayoutSchema] matching GitLab repository
// conventions. It expects a .gitlab/ directory with subdirectories for issue
// and merge request templates, and an optional CODEOWNERS file. Note that
// .gitlab-ci.yml lives at the repository root and is outside the scope of this
// layout schema; enforce its size via the limits config if needed.
func DefaultGitLabLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".gitlab",
		Required: []string{},
		Optional: []string{
			"CODEOWNERS",
		},
		Dirs: map[string]DirRule{
			"issue_templates": {
				Glob:        "",
				Min:         0,
				Description: "GitLab issue templates",
			},
			"merge_request_templates": {
				Glob:        "",
				Min:         0,
				Description: "GitLab merge request templates",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"CODEOWNERS",
			},
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
