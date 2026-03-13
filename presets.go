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
			".aiassistant": {Description: "JetBrains AI Assistant configuration", NoCreate: true},
			".claude":      {Description: "Claude Code configuration", NoCreate: true},
			".clinerules":  {Description: "Cline rule files", NoCreate: true},
			".continue":    {Description: "Continue.dev configuration", NoCreate: true},
			".cursor":      {Description: "Cursor AI configuration", NoCreate: true},
			".github":      {Description: "GitHub configuration", NoCreate: true},
			".gitlab":      {Description: "GitLab configuration", NoCreate: true},
			".kiro":        {Description: "Kiro CLI configuration", NoCreate: true},
			".roo":         {Description: "Roo Code configuration", NoCreate: true},
			".windsurf":    {Description: "Windsurf AI configuration", NoCreate: true},
			// Common project subdirectories.
			"bin":          {Description: "Compiled binaries", NoCreate: true},
			"build":        {Description: "Build output", NoCreate: true},
			"cmd":          {Description: "Main entry points", NoCreate: true},
			"dist":         {Description: "Distribution output", NoCreate: true},
			"docs":         {Description: "Documentation", NoCreate: true},
			"internal":     {Description: "Private packages", NoCreate: true},
			"lib":          {Description: "Library code", NoCreate: true},
			"node_modules": {Description: "Node.js dependencies", NoCreate: true},
			"pkg":          {Description: "Public packages", NoCreate: true},
			"scripts":      {Description: "Helper scripts", NoCreate: true},
			"src":          {Description: "Source code", NoCreate: true},
			"target":       {Description: "Build target (Rust/Maven)", NoCreate: true},
			"test":         {Description: "Test code", NoCreate: true},
			"testdata":     {Description: "Test fixtures", NoCreate: true},
			"tests":        {Description: "Test code", NoCreate: true},
			"vendor":       {Description: "Vendored dependencies", NoCreate: true},
			".vscode":      {Description: "VS Code workspace settings", NoCreate: true},
		},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"AGENTS.md",
				"CHANGELOG.md",
				"CODE_OF_CONDUCT.md",
				"CONTRIBUTING.md",
				"GEMINI.md",
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

// DefaultKiroLayout returns a [LayoutSchema] matching Kiro CLI repository
// conventions. It expects a .kiro/ directory with a steering/ subdirectory
// containing project context files (*.md). Kiro uses steering files to
// provide persistent context to the AI agent across sessions.
func DefaultKiroLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".kiro",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"steering": {
				Glob:        "*.md",
				Min:         0,
				Description: "Kiro steering files (project context)",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultGeminiLayout returns a [LayoutSchema] matching Gemini CLI repository
// conventions. Gemini CLI reads a single GEMINI.md file at the repository root
// as its primary instruction file. No platform-specific subdirectory is used.
func DefaultGeminiLayout() LayoutSchema {
	return LayoutSchema{
		Root: ".",
		Required: []string{
			"GEMINI.md",
		},
		Optional: []string{},
		Dirs:     map[string]DirRule{},
		Naming: NamingRule{
			Case: "lowercase",
			Exceptions: []string{
				"GEMINI.md",
			},
		},
	}
}

// DefaultContinueLayout returns a [LayoutSchema] matching Continue.dev repository
// conventions. It expects a .continue/ directory with a rules/ subdirectory
// containing scoped rule files (*.md).
func DefaultContinueLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".continue",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Continue.dev scoped rule files",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultClineLayout returns a [LayoutSchema] matching Cline repository
// conventions. Cline reads rule files directly from the .clinerules/ directory
// at the repository root; no subdirectory structure is used. The "." DirRule
// matches all *.md files placed directly under .clinerules/.
func DefaultClineLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".clinerules",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			".": {
				Glob:        "*.md",
				Min:         0,
				Description: "Cline rule files",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultRooCodeLayout returns a [LayoutSchema] matching Roo Code repository
// conventions. It expects a .roo/ directory with a rules/ subdirectory
// containing scoped rule files (*.md).
func DefaultRooCodeLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".roo",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "Roo Code scoped rule files",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}

// DefaultJetBrainsLayout returns a [LayoutSchema] matching JetBrains AI
// Assistant repository conventions. It expects a .aiassistant/ directory with
// a rules/ subdirectory containing scoped rule files (*.md).
func DefaultJetBrainsLayout() LayoutSchema {
	return LayoutSchema{
		Root:     ".aiassistant",
		Required: []string{},
		Optional: []string{},
		Dirs: map[string]DirRule{
			"rules": {
				Glob:        "*.md",
				Min:         0,
				Description: "JetBrains AI Assistant scoped rule files",
			},
		},
		Naming: NamingRule{
			Case: "lowercase",
		},
	}
}
