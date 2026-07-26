// Package repogov enforces repository governance conventions: line-count
// limits on text files and directory-structure validation for GitHub and
// AI-agent repositories.
//
// # Line-Count Limits
//
// [Config] defines per-file, glob-based, and default line-count limits.
// [LoadConfig] reads a JSON or YAML configuration file; [DefaultConfig]
// provides sensible defaults when no file is present. [FindConfig] searches
// standard locations for a config file, and [FindAllConfigs] lists every
// config present in precedence order; [SaveConfig] writes a config to disk.
// [ValidateConfig] checks a loaded config for structural and semantic issues.
//
//   - [CheckFile] checks a single file against its resolved limit.
//   - [CheckDir] walks a directory tree and checks every file matching
//     the given extensions.
//   - [CheckDirContext] is like [CheckDir] but accepts a [context.Context]
//     for cancellation support.
//   - [ResolveLimit] returns the effective limit for a path given the config.
//   - [CountLines] counts lines using buffered I/O without loading the
//     entire file into memory.
//
// # Layout Governance
//
// [LayoutSchema] declares the expected directory structure for a repository's
// platform directory (e.g., .github/ or .cursor/).
//
//   - AI-agent presets: [DefaultCopilotLayout], [DefaultCursorLayout],
//     [DefaultWindsurfLayout], [DefaultClaudeLayout], [DefaultKiroLayout],
//     [DefaultGeminiLayout], [DefaultContinueLayout], [DefaultClineLayout],
//     [DefaultRooCodeLayout], [DefaultJetBrainsLayout], and
//     [DefaultZedLayout] return the built-in schemas matching each agent's
//     conventions.
//   - Repository-platform presets: [DefaultGitHubLayout],
//     [DefaultGitLabLayout], [DefaultBitbucketLayout], and
//     [DefaultRootLayout].
//   - [DetectPlatforms] inspects a repository for platform markers
//     (.github/, .gitlab/, bitbucket-pipelines.*) to drive auto-detection.
//   - [CheckLayout] validates a directory against a schema and returns
//     [LayoutResult] entries for required, optional, and unexpected files.
//   - [CheckLayoutContext] is like [CheckLayout] with cancellation support.
//   - [StripFrontmatter] returns a schema copy with frontmatter requirements
//     cleared, honoring [Config.SkipFrontmatter].
//   - [CheckClaudeSettings] audits .claude/settings.json and
//     settings.local.json read-only: duplicate, redundant, and conflicting
//     permission rules, plus consolidation suggestions. repogov never
//     creates or modifies settings files, and credential files are never
//     opened or parsed.
//
// # Scaffolding
//
//   - [InitLayout] creates the directory structure defined by a [LayoutSchema].
//   - [InitLayoutWithConfig] is like [InitLayout] but honors [Config] options
//     such as [Config.InitAlwaysCreate] and [Config.DescriptiveNames].
//   - [InitLayoutAll] initializes multiple schemas in a single pass.
//   - [InitLayoutAllWithConfig] is like [InitLayoutAll] with config support.
//   - [UpdateAgentsMdContextAll] refreshes the "## Context" section of an
//     existing AGENTS.md with merged links for all scaffolded platforms.
//
// # Output Helpers
//
//   - [Passed] and [LayoutPassed] report whether all checks succeeded.
//   - [Summary] and [LayoutSummary] produce human-readable reports.
//
// Typical usage:
//
//	cfg, _ := repogov.LoadConfig(".github/repogov-config.json")
//	results, _ := repogov.CheckDir(".", []string{".md"}, cfg)
//	if !repogov.Passed(results) {
//	    fmt.Fprint(os.Stderr, repogov.Summary(results))
//	    os.Exit(1)
//	}
package repogov
