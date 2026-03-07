package repogov

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// InitLayout creates the directory structure defined by a [LayoutSchema]
// under the given repository root. It creates:
//   - The root layout directory (e.g., .github/)
//   - All subdirectories defined in schema.Dirs
//   - Placeholder files for each required file that does not already exist
//   - A copilot-instructions.md linking to the instructions/ directory
//     (GitHub schemas only, when no existing file is present)
//
// Existing files and directories are never overwritten or modified.
// InitLayout returns the list of paths that were created.
func InitLayout(root string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: LayoutSchema is part of the public API; changing to pointer would be a breaking change
	var created []string

	layoutDir := filepath.Join(root, filepath.FromSlash(schema.Root))

	// Create the root layout directory.
	if dirIsNew(layoutDir) {
		if err := os.MkdirAll(layoutDir, 0755); err != nil {
			return nil, err
		}
		created = append(created, schema.Root)
	}

	// Create subdirectories defined in Dirs.
	for dirName, rule := range schema.Dirs {

		dirPath := filepath.Join(layoutDir, filepath.FromSlash(dirName))
		isNew := dirIsNew(dirPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return created, err
		}
		if isNew {
			rel := filepath.ToSlash(filepath.Join(schema.Root, dirName))
			created = append(created, rel)
		}

		// If the directory has a minimum, create a .gitkeep so git
		// tracks the otherwise-empty directory.
		if rule.Min > 0 {
			gitkeep := filepath.Join(dirPath, ".gitkeep")
			if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
				if err := os.WriteFile(gitkeep, []byte(""), 0644); err != nil {
					return created, err
				}
				rel := filepath.ToSlash(filepath.Join(schema.Root, dirName, ".gitkeep"))
				created = append(created, rel)
			}
		}
	}

	// Create placeholder files for required entries that don't exist.
	for _, req := range schema.Required {
		filePath := filepath.Join(layoutDir, filepath.FromSlash(req))

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return created, err
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			content := placeholderContent(req)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return created, err
			}
			rel := filepath.ToSlash(filepath.Join(schema.Root, req))
			created = append(created, rel)
		}
	}

	// Create copilot-instructions.md for GitHub schemas when it doesn't
	// already exist. The file references the instructions/ directory for
	// scoped instruction files.
	if schema.Root == ".github" {
		paths, err := createCopilotInstructions(root, layoutDir, schema)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Create repogov-config.json with sensible defaults when it doesn't
	// already exist.
	if schema.Root == ".github" {
		paths, err := createDefaultConfig(layoutDir, schema)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Seed the instructions directory with default scoped instruction
	// files when it exists but is empty (only GitHub schemas).
	if _, ok := schema.Dirs["instructions"]; ok {
		paths, err := createDefaultInstructions(layoutDir, schema)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Create AGENTS.md at the repo root when it doesn't already exist.
	// AGENTS.md is a cross-agent open standard (https://agents.md) and is
	// recognized by GitHub Copilot, Cursor, Claude Code, and OpenAI Codex.
	agentsPaths, agentsErr := createAgentsMd(root, schema)
	if agentsErr != nil {
		return created, agentsErr
	}
	created = append(created, agentsPaths...)

	return created, nil
}

// createCopilotInstructions creates .github/copilot-instructions.md with
// references to the instructions/ directory. If the file
// already exists, nothing is created. Returns the list of created paths.
func createCopilotInstructions(root, layoutDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(layoutDir, "copilot-instructions.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		// File exists or stat failed for another reason; either way skip.
		return nil, nil
	}

	content := copilotInstructionsContent(schema)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, err
	}
	return []string{filepath.ToSlash(filepath.Join(schema.Root, "copilot-instructions.md"))}, nil
}

// copilotInstructionsContent generates the content of copilot-instructions.md.
// It detects whether the schema includes an instructions/ directory and links
// to it accordingly. The output is kept under
// 50 lines to comply with the default copilot-instructions.md line limit.
func copilotInstructionsContent(schema LayoutSchema) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("# Copilot Instructions\n\n")
	b.WriteString("This file provides repository-level context to GitHub Copilot and compatible AI coding agents.\n\n")

	// Link to instructions directory if present in the schema.
	if _, ok := schema.Dirs["instructions"]; ok {
		b.WriteString("## Scoped Instructions\n\n")
		b.WriteString("See modular instruction files in [instructions/](instructions/) for scoped rules.\n")
		b.WriteString("Place scoped instruction files in [")
		b.WriteString(schema.Root + "/instructions/](")
		b.WriteString(schema.Root + "/instructions/) using `*.instructions.md`.\n\n")
	}

	b.WriteString("Long-form project context lives in [README.md](../README.md) and [docs/](../docs/).\n\n")

	// File constraints section.
	b.WriteString("## File Constraints\n\n")
	b.WriteString("**`" + schema.Root + "` File Limit**: All files in `" + schema.Root + "/` must not exceed the configured line limit (see `repogov-config.json`). Use `docs/` for detailed explanations.\n\n")
	b.WriteString("**Handling the line limit (priority order):**\n\n")
	b.WriteString("1. **Refactor First** - Remove redundancy, condense verbose explanations\n")
	b.WriteString("2. **Move to `docs/`** - Relocate detailed content to `docs/` directory\n")
	b.WriteString("3. **Link, Don't Repeat** - Reference external docs instead of duplicating\n")
	b.WriteString("4. **Split Only When Necessary** - Only when content is a distinct concern; use cohesive files, descriptive names, and cross-references\n\n")

	// File naming conventions.
	b.WriteString("## File Naming Conventions\n\n")
	b.WriteString("**Prefer lowercase filenames** in `docs/` and `" + schema.Root + "/` directories:\n\n")
	b.WriteString("- Use `kebab-case` or `snake_case` for multi-word filenames\n")
	b.WriteString("- Exception: `*_AUDIT.md` files in `docs/` may use uppercase\n")
	b.WriteString("- Exception: GitHub-mandated filenames remain uppercase (e.g., `CODEOWNERS`)\n\n")

	// Repository conventions.
	b.WriteString("## Repository Conventions\n\n")
	b.WriteString("- Follow existing code style and patterns\n")
	b.WriteString("- Keep files within configured line limits (see repogov-config.json)\n")
	b.WriteString("- Write tests for new functionality\n")

	return b.String()
}

// dirIsNew returns true if the directory does not yet exist.
func dirIsNew(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// isDirEmpty returns true if the directory at path exists and contains no
// entries (ignoring . and ..), or if the directory does not exist.
func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true // treat unreadable/missing as empty
	}
	// Ignore .gitkeep placeholder files.
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			return false
		}
	}
	return true
}

// defaultInstructionFiles defines the instruction files seeded by
// [InitLayout] when the instructions/ directory is empty. Each entry
// maps a filename to a content-generating function. Content must stay
// within the 300-line limit for .github/instructions/*.md.
var defaultInstructionFiles = map[string]func() string{
	"general.instructions.md":          generalInstructionsContent,
	"codereview.instructions.md":       codereviewInstructionsContent,
	"governance.instructions.md":       governanceInstructionsContent,
	"library.instructions.md":          libraryInstructionsContent,
	"testing.instructions.md":          testingInstructionsContent,
	"emoji-prevention.instructions.md": emojiPreventionInstructionsContent,
	"backend.instructions.md":          backendInstructionsContent,
	"frontend.instructions.md":         frontendInstructionsContent,
}

// createDefaultInstructions seeds the instructions/ directory with default
// scoped instruction files when the directory is empty (or contains only a
// .gitkeep). Existing files are never overwritten. Returns created paths.
func createDefaultInstructions(layoutDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	instrDir := filepath.Join(layoutDir, "instructions")
	if !isDirEmpty(instrDir) {
		return nil, nil
	}

	var created []string
	for name, contentFn := range defaultInstructionFiles {
		filePath := filepath.Join(instrDir, name)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			continue
		}
		if err := os.MkdirAll(instrDir, 0755); err != nil {
			return created, err
		}
		if err := os.WriteFile(filePath, []byte(contentFn()), 0644); err != nil {
			return created, err
		}
		rel := filepath.ToSlash(filepath.Join(schema.Root, "instructions", name))
		created = append(created, rel)
	}
	return created, nil
}

// generalInstructionsContent returns default content for general.instructions.md.
func generalInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**/*.md\"\n---\n\n")
	b.WriteString("# General Instructions\n\n")
	b.WriteString("## Writing Style\n\n")
	b.WriteString("- Use clear, concise language\n")
	b.WriteString("- Prefer active voice over passive voice\n")
	b.WriteString("- Write in complete sentences\n")
	b.WriteString("- Keep paragraphs focused on a single idea\n")
	b.WriteString("- Avoid jargon unless the audience expects it\n\n")
	b.WriteString("## Document Structure\n\n")
	b.WriteString("- Start every document with a level-1 heading (`# Title`)\n")
	b.WriteString("- Use heading levels sequentially -- do not skip levels\n")
	b.WriteString("- Keep headings descriptive and concise\n")
	b.WriteString("- Add a blank line before and after headings\n")
	b.WriteString("- Use lists for three or more related items\n\n")
	b.WriteString("## Formatting\n\n")
	b.WriteString("- Wrap inline code, commands, file paths, and symbols in backticks\n")
	b.WriteString("- Use fenced code blocks with a language identifier for multi-line code\n")
	b.WriteString("- Use bold for emphasis on key terms, not for entire sentences\n")
	b.WriteString("- Use tables when presenting structured comparisons or reference data\n")
	b.WriteString("- Keep lines under 120 characters where possible for diff readability\n\n")
	b.WriteString("## File Organization\n\n")
	b.WriteString("- Use consistent file naming conventions (see copilot-instructions.md)\n")
	b.WriteString("- Keep files focused and within configured line limits\n")
	b.WriteString("- Place detailed content in `docs/` and link from top-level files\n")
	b.WriteString("- Do not duplicate content -- link to the canonical source instead\n\n")
	b.WriteString("## Links and References\n\n")
	b.WriteString("- Use relative links for in-repo references\n")
	b.WriteString("- Verify links are valid before committing\n")
	b.WriteString("- Use descriptive link text -- avoid \"click here\" or bare URLs\n")
	b.WriteString("- Reference specific sections with anchor links when useful\n\n")
	b.WriteString("## Maintenance\n\n")
	b.WriteString("- Keep documentation in sync with the feature it describes\n")
	b.WriteString("- Remove or update stale content promptly\n")
	b.WriteString("- Date-stamp time-sensitive information\n")
	b.WriteString("- Review docs as part of the pull request process\n")
	return b.String()
}

// codereviewInstructionsContent returns default content for codereview.instructions.md.
func codereviewInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**/*.md\"\n---\n\n")
	b.WriteString("# Code Review Instructions\n\n")
	b.WriteString("## Review Priorities for Documentation\n\n")
	b.WriteString("When reviewing documentation changes, prioritize in this order:\n\n")
	b.WriteString("1. **Accuracy** -- Does the documentation match the current behavior?\n")
	b.WriteString("2. **Completeness** -- Are all necessary topics covered?\n")
	b.WriteString("3. **Clarity** -- Can the target audience understand this?\n")
	b.WriteString("4. **Consistency** -- Does it follow project conventions?\n")
	b.WriteString("5. **Brevity** -- Is it concise without losing important detail?\n\n")
	b.WriteString("## What to Look For\n\n")
	b.WriteString("### Content Quality\n\n")
	b.WriteString("- Factual accuracy -- verify claims against the implementation\n")
	b.WriteString("- No stale references to removed features or old APIs\n")
	b.WriteString("- Examples are runnable and produce the stated output\n")
	b.WriteString("- Edge cases and limitations are documented\n\n")
	b.WriteString("### Structure and Navigation\n\n")
	b.WriteString("- Heading hierarchy is logical and sequential\n")
	b.WriteString("- Related topics are grouped together\n")
	b.WriteString("- Cross-references use working relative links\n")
	b.WriteString("- Table of contents is present for long documents\n\n")
	b.WriteString("### Formatting and Style\n\n")
	b.WriteString("- Follows general.instructions.md conventions\n")
	b.WriteString("- Code blocks use correct language identifiers\n")
	b.WriteString("- Tables are well-formed and readable in plain text\n")
	b.WriteString("- No emoji (see emoji-prevention.instructions.md)\n\n")
	b.WriteString("### File Governance\n\n")
	b.WriteString("- File stays within its configured line limit\n")
	b.WriteString("- Content belongs in this file, not in docs/ or another document\n")
	b.WriteString("- No content duplication -- link instead of repeating\n")
	b.WriteString("- File naming follows project conventions\n\n")
	b.WriteString("## Review Etiquette\n\n")
	b.WriteString("- Be specific -- reference lines and suggest concrete alternatives\n")
	b.WriteString("- Distinguish must-fix items from suggestions and nits\n")
	b.WriteString("- Acknowledge good improvements, not just problems\n")
	b.WriteString("- Ask questions rather than making assumptions about intent\n")
	return b.String()
}

// governanceInstructionsContent returns default content for governance.instructions.md.
func governanceInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Governance Instructions\n\n")
	b.WriteString("## Line Limits\n\n")
	b.WriteString("All files must stay within their configured line limit.\n")
	b.WriteString("See [repogov-config.json](../repogov-config.json) for limits and rules.\n\n")
	b.WriteString("- **Resolution order**: per-file override, first matching glob, default\n")
	b.WriteString("- A limit of `0` exempts the file (status = SKIP)\n")
	b.WriteString("- WARN at the configured `warning_threshold` percentage\n\n")
	b.WriteString("## Enforcing Limits\n\n")
	b.WriteString("Run `repogov -root . limits` or use the library before committing.\n")
	b.WriteString("Refactor or split files that approach their limit -- do not raise limits\n")
	b.WriteString("without justification.\n\n")
	b.WriteString("## Layout\n\n")
	b.WriteString("The `.github/` directory must satisfy the GitHub layout preset.\n")
	b.WriteString("Run `repogov -root . layout` to validate structure.\n\n")
	b.WriteString("## AI Agent Support\n\n")
	b.WriteString("When adding or modifying agent support, consult [docs/ai-agents-audit.md](../../docs/ai-agents-audit.md)\n")
	b.WriteString("for the current support matrix, per-agent config patterns, and maintenance steps.\n\n")
	b.WriteString("## Zero Dependencies\n\n")
	b.WriteString("This repository has no external Go module dependencies.\n")
	b.WriteString("Do not add `require` directives to the root `go.mod`.\n\n")
	b.WriteString("## Pre-commit Hook Example\n\n")
	b.WriteString("Use repogov as a dependency-free pre-commit hook:\n\n")
	b.WriteString("```go\npackage main\n\nimport (\n")
	b.WriteString("    \"fmt\"\n    \"os\"\n    \"github.com/nicholashoule/repogov\"\n)\n\n")
	b.WriteString("func main() {\n")
	b.WriteString("    cfg := repogov.DefaultConfig()\n")
	b.WriteString("    results, _ := repogov.CheckDir(\".\", []string{\".md\"}, cfg)\n")
	b.WriteString("    fmt.Fprint(os.Stderr, repogov.Summary(results))\n")
	b.WriteString("    layout, _ := repogov.CheckLayout(\".\", repogov.DefaultGitHubLayout())\n")
	b.WriteString("    fmt.Fprint(os.Stderr, repogov.LayoutSummary(layout))\n")
	b.WriteString("    if !repogov.Passed(results) || !repogov.LayoutPassed(layout) {\n")
	b.WriteString("        os.Exit(1)\n    }\n}\n```\n\n")
	b.WriteString("## Minimal CLI Example\n\n")
	b.WriteString("```bash\ngo install github.com/nicholashoule/repogov/cmd/repogov@latest\n")
	b.WriteString("repogov -root . -exts .md all\n```\n")
	return b.String()
}

// libraryInstructionsContent returns default content for library.instructions.md.
func libraryInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**/*.md\"\n---\n\n")
	b.WriteString("# Library Documentation Instructions\n\n")
	b.WriteString("## API Reference Documentation\n\n")
	b.WriteString("- Document every public symbol in a reference table or section\n")
	b.WriteString("- Include the function signature, a one-line summary, and the source file\n")
	b.WriteString("- Group symbols by concern (types, functions, configuration)\n")
	b.WriteString("- Keep the public API table up to date when symbols are added or removed\n")
	b.WriteString("- Link to doc comments or pkg.go.dev for full details\n\n")
	b.WriteString("## README Structure\n\n")
	b.WriteString("- Start with a one-paragraph summary of what the library does\n")
	b.WriteString("- Include install and quick-start sections early\n")
	b.WriteString("- Provide at least one runnable example\n")
	b.WriteString("- List configuration options with defaults and valid ranges\n")
	b.WriteString("- Add a public API table linking symbols to source files\n")
	b.WriteString("- End with status codes, development instructions, and license\n\n")
	b.WriteString("## Design Documents\n\n")
	b.WriteString("- Place architecture and design docs in `docs/`\n")
	b.WriteString("- Start with purpose, then architecture, then details\n")
	b.WriteString("- Document constraints and trade-offs, not just decisions\n")
	b.WriteString("- Include diagrams or tables when they clarify structure\n")
	b.WriteString("- Keep design docs under their configured line limit\n\n")
	b.WriteString("## Changelog\n\n")
	b.WriteString("- Maintain a CHANGELOG.md following Keep a Changelog format\n")
	b.WriteString("- Group entries under Added, Changed, Deprecated, Removed, Fixed, Security\n")
	b.WriteString("- Reference issue or PR numbers for traceability\n")
	b.WriteString("- Write entries from the user's perspective, not the developer's\n\n")
	b.WriteString("## Cross-References\n\n")
	b.WriteString("- Link between README, docs/, and instruction files -- do not duplicate\n")
	b.WriteString("- Use relative links for all in-repo references\n")
	b.WriteString("- Reference specific sections with anchor links\n")
	b.WriteString("- Keep the link graph shallow: two hops maximum to find any topic\n\n")
	b.WriteString("## Versioning Documentation\n\n")
	b.WriteString("- Document breaking changes prominently in CHANGELOG and README\n")
	b.WriteString("- Mark deprecated features with a timeline for removal\n")
	b.WriteString("- Update install instructions when the minimum version changes\n")
	b.WriteString("- Tag documentation updates alongside code releases\n")
	return b.String()
}

// testingInstructionsContent returns default content for testing.instructions.md.
func testingInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**/*.md\"\n---\n\n")
	b.WriteString("# Testing Documentation Instructions\n\n")
	b.WriteString("## Documenting Test Coverage\n\n")
	b.WriteString("- List what is tested and what is not in a testing section or file\n")
	b.WriteString("- Group tests by feature or module for easier navigation\n")
	b.WriteString("- Note any known gaps with a plan or issue reference\n")
	b.WriteString("- Keep test documentation close to the feature it covers\n\n")
	b.WriteString("## Test Examples in Documentation\n\n")
	b.WriteString("- Include runnable examples when documenting public APIs\n")
	b.WriteString("- Show both input and expected output\n")
	b.WriteString("- Use fenced code blocks with the correct language identifier\n")
	b.WriteString("- Verify examples still work when updating documentation\n\n")
	b.WriteString("## CI and Automation Docs\n\n")
	b.WriteString("- Document how to run tests locally (e.g., `make test`)\n")
	b.WriteString("- List available test-related Makefile or script targets\n")
	b.WriteString("- Explain CI workflow steps in workflow YAML or a docs/ file\n")
	b.WriteString("- Document required environment variables or setup steps\n\n")
	b.WriteString("## Test Result Formatting\n\n")
	b.WriteString("- Use plain-text status markers: `[PASS]`, `[FAIL]`, `[SKIP]`\n")
	b.WriteString("- Do not use emoji in test output (see emoji-prevention.instructions.md)\n")
	b.WriteString("- Present results in tables when summarizing multiple checks\n")
	b.WriteString("- Include counts: total, passed, failed, skipped\n\n")
	b.WriteString("## Development Section\n\n")
	b.WriteString("- Include a Development section in README.md with test commands\n")
	b.WriteString("- List commands in a code block for easy copy-paste\n")
	b.WriteString("- Cover: unit tests, race detection, coverage, formatting, linting\n")
	b.WriteString("- Keep the section concise -- link to docs/ for details\n")
	return b.String()
}

// emojiPreventionInstructionsContent returns default content for emoji-prevention.instructions.md.
func emojiPreventionInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**/*.md\"\n---\n\n")
	b.WriteString("# Emoji Prevention Instructions\n\n")
	b.WriteString("## Rule\n\n")
	b.WriteString("Do **not** introduce emoji or Unicode pictographic characters into\n")
	b.WriteString("source code, documentation, comments, commit messages, or CI output.\n")
	b.WriteString("Use plain-text equivalents instead.\n\n")
	b.WriteString("## Why\n\n")
	b.WriteString("- Emoji render inconsistently across terminals, editors, and fonts\n")
	b.WriteString("- They break grep, diff, and other line-oriented tools\n")
	b.WriteString("- Screen readers may announce them unexpectedly or skip them entirely\n")
	b.WriteString("- They inflate token counts in LLM-assisted workflows\n")
	b.WriteString("- They add no semantic value that plain text cannot convey\n\n")
	b.WriteString("## Common Substitutions\n\n")
	b.WriteString("| Instead of | Write |\n")
	b.WriteString("|------------|-------|\n")
	b.WriteString("| U+2705 / U+2714 | `[PASS]`, `OK`, `yes` |\n")
	b.WriteString("| U+274C / U+2716 | `[FAIL]`, `ERROR`, `no` |\n")
	b.WriteString("| U+26A0 | `[WARN]`, `WARNING` |\n")
	b.WriteString("| U+2139 | `[INFO]`, `NOTE` |\n")
	b.WriteString("| U+1F680 | `deploy`, `launch`, `ship` |\n")
	b.WriteString("| U+1F41B | `bug`, `defect`, `issue` |\n")
	b.WriteString("| U+1F527 | `fix`, `patch`, `repair` |\n")
	b.WriteString("| U+2728 | `new`, `feature`, `add` |\n")
	b.WriteString("| U+1F4DD | `docs`, `note`, `document` |\n")
	b.WriteString("| U+1F525 | `hot`, `critical`, `urgent` |\n")
	b.WriteString("| U+27A1 / U+2192 | `->`, `-->`, `==>` |\n")
	b.WriteString("| U+1F504 | `refactor`, `rework`, `restructure` |\n\n")
	b.WriteString("## Text alternatives\n\n")
	b.WriteString("| Instead of | Write |\n")
	b.WriteString("|------------|-------|\n")
	b.WriteString("| checkmark emoji | `[PASS]` or `[OK]` |\n")
	b.WriteString("| cross/X emoji | `[FAIL]` or `[ERROR]` |\n")
	b.WriteString("| warning emoji | `WARNING:` |\n")
	b.WriteString("| info/note emoji | `[INFO]` or `NOTE:` |\n")
	b.WriteString("| lightbulb emoji | `TIP:` |\n")
	b.WriteString("| rocket emoji | `Deployment` or `Released` |\n")
	b.WriteString("| chart emoji | `Report` or `Metrics` |\n")
	b.WriteString("| star emoji | `[FEATURED]` |\n")
	b.WriteString("| lock emoji | `Security` |\n\n")
	b.WriteString("## Enforcement\n\n")
	b.WriteString("Use the `demojify-sanitize` tool to detect emoji:\n\n")
	b.WriteString("```sh\ngo install github.com/nicholashoule/demojify-sanitize/cmd/demojify@latest\n```\n\n")
	b.WriteString("- **Pre-commit hook**: install `demojify` as a pre-commit hook to prevent new emoji from being committed.\n")
	b.WriteString("- **Manual audit**: `go run github.com/nicholashoule/demojify-sanitize/cmd/demojify -root .`\n")
	b.WriteString("- **Auto-fix with removal**: `demojify -root . -fix`\n\n")
	b.WriteString("## Scope\n\n")
	b.WriteString("This rule applies to all files tracked by git:\n\n")
	b.WriteString("- Go source files (`.go`)\n")
	b.WriteString("- Markdown documentation (`.md`)\n")
	b.WriteString("- YAML and JSON configuration (`.yml`, `.yaml`, `.json`)\n")
	b.WriteString("- Shell scripts and Makefiles\n")
	b.WriteString("- Commit messages and PR descriptions\n\n")
	b.WriteString("## Exceptions\n\n")
	b.WriteString("- Test fixtures that explicitly exercise emoji handling\n")
	b.WriteString("- Docs that describe or reference emoji (e.g., unicode-coverage.md)\n")
	b.WriteString("- Third-party files vendored as-is\n")
	return b.String()
}

// backendInstructionsContent returns default content for backend.instructions.md.
func backendInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Backend Instructions\n\n")
	b.WriteString("## API Design\n\n")
	b.WriteString("- Follow RESTful conventions: use nouns for resources, HTTP verbs for actions\n")
	b.WriteString("- Version APIs from the start (e.g., `/api/v1/`)\n")
	b.WriteString("- Return consistent JSON response shapes: `{ data, error, meta }`\n")
	b.WriteString("- Use standard HTTP status codes accurately (200, 201, 400, 401, 403, 404, 500)\n")
	b.WriteString("- Document all endpoints in an OpenAPI/Swagger spec or equivalent\n\n")
	b.WriteString("## Error Handling\n\n")
	b.WriteString("- Never expose internal stack traces or system paths in API responses\n")
	b.WriteString("- Return structured error objects with a code, message, and optional detail field\n")
	b.WriteString("- Log errors server-side with context (request ID, user ID where applicable)\n")
	b.WriteString("- Distinguish between client errors (4xx) and server errors (5xx)\n\n")
	b.WriteString("## Authentication and Authorization\n\n")
	b.WriteString("- Validate and sanitize all input at the API boundary\n")
	b.WriteString("- Use bearer tokens or session cookies consistently -- do not mix approaches\n")
	b.WriteString("- Check authorization before loading data, not after\n")
	b.WriteString("- Document required permissions for each endpoint\n\n")
	b.WriteString("## Data and Storage\n\n")
	b.WriteString("- Use parameterized queries or ORM methods -- never string-interpolate SQL\n")
	b.WriteString("- Keep business logic out of SQL; prefer service-layer transforms\n")
	b.WriteString("- Handle connection errors gracefully with appropriate retries or fallback\n")
	b.WriteString("- Document schema migrations and keep them reversible\n\n")
	b.WriteString("## Testing\n\n")
	b.WriteString("- Write unit tests for service and business logic in isolation\n")
	b.WriteString("- Write integration tests for database-touching code against a test database\n")
	b.WriteString("- Test error paths and boundary conditions, not just happy paths\n")
	b.WriteString("- Use table-driven or data-driven tests for multiple input/output combinations\n\n")
	b.WriteString("## Documentation\n\n")
	b.WriteString("- Document request and response shapes with examples\n")
	b.WriteString("- Note rate limits, authentication requirements, and pagination behavior\n")
	b.WriteString("- Keep API docs co-located with implementation or in `docs/api/`\n")
	b.WriteString("- Update docs in the same PR that changes the API\n")
	return b.String()
}

// frontendInstructionsContent returns default content for frontend.instructions.md.
func frontendInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Frontend Instructions\n\n")
	b.WriteString("## Component Structure\n\n")
	b.WriteString("- Keep components focused on a single responsibility\n")
	b.WriteString("- Separate presentational components from data-fetching logic\n")
	b.WriteString("- Co-locate styles, tests, and stories with the component file\n")
	b.WriteString("- Use consistent file naming: `ComponentName.tsx`, `ComponentName.test.tsx`\n\n")
	b.WriteString("## State Management\n\n")
	b.WriteString("- Prefer local state over global state where possible\n")
	b.WriteString("- Document the shape of shared state with types or interfaces\n")
	b.WriteString("- Avoid prop-drilling beyond two levels -- use context or a state store\n")
	b.WriteString("- Keep side effects isolated and clearly labeled\n\n")
	b.WriteString("## API Integration\n\n")
	b.WriteString("- Centralize API calls in a dedicated service or hook layer\n")
	b.WriteString("- Handle loading, error, and empty states explicitly in UI components\n")
	b.WriteString("- Do not hard-code base URLs -- use environment variables\n")
	b.WriteString("- Type API responses at the boundary to catch shape mismatches early\n\n")
	b.WriteString("## Accessibility\n\n")
	b.WriteString("- All interactive elements must be keyboard-navigable\n")
	b.WriteString("- Use semantic HTML elements before reaching for ARIA attributes\n")
	b.WriteString("- Provide alt text for all images and icons\n")
	b.WriteString("- Test with a screen reader before shipping new interactive components\n\n")
	b.WriteString("## Testing\n\n")
	b.WriteString("- Write unit tests for utility functions and hooks\n")
	b.WriteString("- Write component tests for rendering and interaction logic\n")
	b.WriteString("- Prefer testing user-visible behavior over implementation details\n")
	b.WriteString("- Use end-to-end tests sparingly for critical user journeys\n\n")
	b.WriteString("## Documentation\n\n")
	b.WriteString("- Document component props with types and descriptions\n")
	b.WriteString("- Note any setup requirements (environment variables, feature flags)\n")
	b.WriteString("- Keep a Storybook or equivalent for shared UI components\n")
	b.WriteString("- Link to design specs or Figma files for visual reference\n")
	return b.String()
}

// createAgentsMd creates AGENTS.md at the repository root when it does not
// already exist, or updates only its ## Context section when it does. The file
// is cross-agent (GitHub Copilot, Cursor, Claude Code, OpenAI Codex) and
// provides agent-readable instructions with links to project documentation and
// platform-specific instruction directories.
func createAgentsMd(root string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(root, "AGENTS.md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File does not exist: create it with full generated content.
		content := agentsMdContent(schema)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return nil, err
		}
		return []string{"AGENTS.md"}, nil
	}
	// File exists: update only the ## Context section so it reflects the
	// currently selected agent/platform without overwriting user content.
	if err := updateAgentsMdContext(filePath, schema); err != nil {
		return nil, err
	}
	return nil, nil
}

// updateAgentsMdContext rewrites the "## Context" section of an existing
// AGENTS.md file using links derived from schema. All other sections are
// preserved exactly. If no "## Context" heading is found the file is left
// unchanged.
func updateAgentsMdContext(filePath string, schema LayoutSchema) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	const sectionMarker = "## Context\n"
	startIdx := strings.Index(content, sectionMarker)
	if startIdx == -1 {
		// No ## Context section present; leave the file unchanged.
		return nil
	}

	// Build the replacement context section (heading + items, no trailing
	// blank line -- the separator is provided by the "\n## " that follows).
	newSection := agentsMdContextSection(schema)

	// Find where the next heading starts after the context section.
	afterSection := startIdx + len(sectionMarker)
	nextHeadingOffset := strings.Index(content[afterSection:], "\n## ")

	var updated string
	if nextHeadingOffset == -1 {
		// Context is the last section in the file.
		updated = content[:startIdx] + newSection
	} else {
		// Preserve everything from the blank-line separator onward.
		splitIdx := afterSection + nextHeadingOffset
		updated = content[:startIdx] + newSection + content[splitIdx:]
	}

	return os.WriteFile(filePath, []byte(updated), 0644)
}

// agentsMdContextSection generates the "## Context" section for AGENTS.md
// with links appropriate to the given schema. The returned string starts with
// the "## Context" heading and ends with the last list item followed by "\n";
// the blank-line separator before the next section is NOT included so that
// updateAgentsMdContext can splice it in without introducing extra blank lines.
func agentsMdContextSection(schema LayoutSchema) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("## Context\n\n")
	b.WriteString("- Project overview: [README.md](README.md)\n")
	b.WriteString("- Extended documentation: [docs/](docs/)\n")

	// Link to schema-specific instruction directories.
	if _, ok := schema.Dirs["instructions"]; ok {
		b.WriteString("- Scoped instruction files: [" + schema.Root + "/instructions/](" + schema.Root + "/instructions/) (`applyTo` frontmatter)\n")
	}
	if _, ok := schema.Dirs["rules"]; ok {
		b.WriteString("- Agent rule files: [" + schema.Root + "/rules/](" + schema.Root + "/rules/)\n")
	}
	if schema.Root == ".github" {
		b.WriteString("- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)\n")
	}
	return b.String()
}

// agentsMdContent generates the content for AGENTS.md. It links to the
// project's docs/ directory and to platform-specific instruction directories
// present in the schema. The output must stay within the 200-line limit.
func agentsMdContent(schema LayoutSchema) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("# AGENTS.md\n\n")
	b.WriteString("Agent instructions for AI coding assistants working in this repository.\n")
	b.WriteString("See [agents.md](https://agents.md) for the open format specification.\n\n")

	b.WriteString(agentsMdContextSection(schema))
	b.WriteString("\n")

	b.WriteString("## Nested Instructions\n\n")
	b.WriteString("Place an `AGENTS.md` in any subdirectory to provide directory-scoped instructions.\n")
	b.WriteString("Agents load the nearest `AGENTS.md` walking up to the repo root; more specific\n")
	b.WriteString("files take precedence over less specific ones.\n\n")

	if schema.Root == ".github" {
		b.WriteString("## .github Layout\n\n")
		b.WriteString("Standard files and directories for this repository's `.github/` folder:\n\n")
		b.WriteString("- `.github/ISSUE_TEMPLATE/`\n")
		b.WriteString("- `.github/PULL_REQUEST_TEMPLATE/`\n")
		b.WriteString("- `.github/workflows/`\n")
		b.WriteString("- `.github/ISSUE_TEMPLATE.md`\n")
		b.WriteString("- `.github/pull_request_template.md`\n")
		b.WriteString("- `.github/CONTRIBUTING.md`\n")
		b.WriteString("- `.github/CODE_OF_CONDUCT.md`\n")
		b.WriteString("- `.github/SECURITY.md`\n")
		b.WriteString("- `.github/SUPPORT.md`\n")
		b.WriteString("- `.github/FUNDING.yml`\n")
		b.WriteString("- `.github/CODEOWNERS`\n")
		b.WriteString("- `.github/dependabot.yml`\n")
		b.WriteString("\n")
	}

	b.WriteString("## Dev Environment\n\n")
	b.WriteString("TODO: describe setup steps (dependencies, tools, environment variables).\n\n")

	b.WriteString("## Testing\n\n")
	b.WriteString("Validate our boundaries by snapshotting the current `" + schema.Root + "/` contents, running all init commands, and inspecting the generated files in a temp path (e.g., `./temp`) for every agent flag and config. Confirm that `AGENTS.md` reflects the correct context for the init\u2011selected agent.\n\n")

	b.WriteString("## PR Instructions\n\n")
	b.WriteString("TODO: describe pull request conventions (title format, required checks, review expectations).\n")

	return b.String()
}

// createDefaultConfig creates .github/repogov-config.json with config values
// filtered to the schema's root (e.g. only .github/ rules for a GitHub init).
// The file is not created when it already exists.
func createDefaultConfig(layoutDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(layoutDir, "repogov-config.json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}

	cfg := schemaConfig(DefaultConfig(), schema.Root)
	data := defaultConfigJSON(cfg)
	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		return nil, err
	}
	rel := filepath.ToSlash(filepath.Join(schema.Root, "repogov-config.json"))
	return []string{rel}, nil
}

// schemaConfig returns a copy of cfg with rules and files filtered to only
// those whose glob or path begins with the given root prefix (e.g. ".github/").
// Root-level file entries (no "/" in key) are always preserved because they
// are cross-platform (e.g. AGENTS.md).
// The base scalars (Default, WarningThreshold, SkipDirs) are kept as-is.
func schemaConfig(cfg Config, root string) Config {
	prefix := root + "/"
	out := Config{
		Default:          cfg.Default,
		WarningThreshold: cfg.WarningThreshold,
		SkipDirs:         cfg.SkipDirs,
	}
	for _, r := range cfg.Rules {
		if strings.HasPrefix(r.Glob, prefix) {
			out.Rules = append(out.Rules, r)
		}
	}
	for k, v := range cfg.Files {
		// Include files inside this schema's root OR bare root-level filenames.
		if strings.HasPrefix(k, prefix) || !strings.Contains(k, "/") {
			if out.Files == nil {
				out.Files = make(map[string]int)
			}
			out.Files[k] = v
		}
	}
	return out
}

// defaultConfigJSON returns a compact JSON representation of a Config
// that matches the preferred on-disk style: skip_dirs on one line,
// rules as compact single-line objects.
func defaultConfigJSON(cfg Config) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  \"default\": " + intStr(cfg.Default) + ",\n")
	b.WriteString("  \"warning_threshold\": \"" + intStr(int(cfg.WarningThreshold)) + "%\",\n")

	// skip_dirs as compact array.
	b.WriteString("  \"skip_dirs\": [")
	for i, d := range cfg.SkipDirs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("\"" + d + "\"")
	}
	b.WriteString("],\n")

	// rules as compact single-line objects.
	b.WriteString("  \"rules\": [\n")
	for i, r := range cfg.Rules {
		b.WriteString("    {\"glob\": \"" + r.Glob + "\",\"limit\": " + intStr(r.Limit) + "}")
		if i < len(cfg.Rules)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ],\n")

	// files map.
	b.WriteString("  \"files\": {\n")
	i := 0
	for k, v := range cfg.Files {
		b.WriteString("    \"" + k + "\": " + intStr(v))
		i++
		if i < len(cfg.Files) {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

// intStr converts an int to its string representation.
func intStr(n int) string {
	return strconv.Itoa(n)
}

// placeholderContent returns minimal starter content for a required file
// based on its filename extension and name.
func placeholderContent(relPath string) string {
	name := filepath.Base(relPath)
	ext := filepath.Ext(name)

	switch ext {
	case ".yml", ".yaml":
		return "# " + name + "\n# TODO: configure this file\n"
	case ".md":
		return "# " + name[:len(name)-len(ext)] + "\n\nTODO: add content\n"
	default:
		return "# " + name + "\n"
	}
}
