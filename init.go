package repogov

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// initOptions bundles all options that influence scaffolding behavior so that
// internal helpers can be updated without cascading signature changes.
type initOptions struct {
	// skipConfig suppresses writing repogov-config.json (used by all-agent init).
	skipConfig bool
	// alwaysCreate seeds template files into existing non-empty directories.
	alwaysCreate bool
	// descriptive selects <name>.instructions.md naming (true) vs <name>.md (false).
	descriptive bool
	// include is an allowlist of template stems; when non-empty only listed stems are created.
	include []string
	// exclude is a blocklist of template stems; listed stems are skipped (ignored when include is set).
	exclude []string
}

// templateStem strips any ".instructions.md", ".mdc", or ".md" suffix from
// name and returns the bare stem (e.g. "general.instructions.md" -> "general").
func templateStem(name string) string {
	name = strings.TrimSuffix(name, ".instructions.md")
	name = strings.TrimSuffix(name, ".mdc")
	name = strings.TrimSuffix(name, ".md")
	return name
}

// shouldSeedFile reports whether the template with the given stem should be
// created during init. When include is non-empty, only stems present in that
// list are created (takes precedence over exclude). When exclude is non-empty
// and include is empty, stems present in exclude are skipped.
func shouldSeedFile(stem string, include, exclude []string) bool {
	if len(include) > 0 {
		for _, n := range include {
			if templateStem(n) == stem {
				return true
			}
		}
		return false
	}
	for _, n := range exclude {
		if templateStem(n) == stem {
			return false
		}
	}
	return true
}

// InitLayout creates the directory structure defined by a [LayoutSchema]
// under the given repository root. It creates:
//   - The root layout directory (e.g., .github/)
//   - All subdirectories defined in schema.Dirs
//   - Placeholder files for each required file that does not already exist
//   - A copilot-instructions.md linking to the instructions/ directory
//     (GitHub schemas only, when no existing file is present)
//
// Existing files and directories are never overwritten or modified.
// Default template files are only seeded when their target directory is
// empty or newly created. Use [InitLayoutWithConfig] with
// [Config.InitAlwaysCreate] set to true to seed missing files into existing
// non-empty directories.
// InitLayout returns the list of paths that were created.
func InitLayout(root string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: LayoutSchema is part of the public API; changing to pointer would be a breaking change
	return initLayoutSingle(root, schema, initOptions{})
}

// InitLayoutWithConfig is like [InitLayout] but honors configuration options
// that influence scaffolding behavior, in particular [Config.InitAlwaysCreate].
// When cfg.InitAlwaysCreate is true, default template files are seeded into
// existing non-empty directories whenever an individual file is absent.
func InitLayoutWithConfig(root string, schema LayoutSchema, cfg Config) ([]string, error) { //nolint:gocritic // hugeParam: LayoutSchema is part of the public API; changing to pointer would be a breaking change
	return initLayoutSingle(root, schema, initOptions{
		alwaysCreate: cfg.InitAlwaysCreate,
		descriptive:  cfg.DescriptiveNames,
		include:      cfg.InitIncludeFiles,
		exclude:      cfg.InitExcludeFiles,
	})
}

// InitLayoutAll initializes all schemas in a single pass and writes a single
// repogov-config.json at the repository root covering all agent platforms.
// It is the preferred entry point when scaffolding multiple platforms at once.
// Use [InitLayout] for a single-platform init.
//
// InitLayoutAll also updates AGENTS.md with a merged ## Context section that
// references every scaffolded platform directory.
func InitLayoutAll(root string, schemas []LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	return initLayoutAllWithOptions(root, schemas, initOptions{})
}

// InitLayoutAllWithConfig is like [InitLayoutAll] but honors configuration
// options that influence scaffolding behavior, in particular
// [Config.InitAlwaysCreate].
func InitLayoutAllWithConfig(root string, schemas []LayoutSchema, cfg Config) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	return initLayoutAllWithOptions(root, schemas, initOptions{
		alwaysCreate: cfg.InitAlwaysCreate,
		descriptive:  cfg.DescriptiveNames,
		include:      cfg.InitIncludeFiles,
		exclude:      cfg.InitExcludeFiles,
	})
}

// initLayoutAllWithOptions is the shared implementation for [InitLayoutAll]
// and [InitLayoutAllWithConfig].
func initLayoutAllWithOptions(root string, schemas []LayoutSchema, opts initOptions) ([]string, error) {
	var allCreated []string
	for i := range schemas {
		created, err := initLayoutSingle(root, schemas[i], initOptions{
			skipConfig:   true,
			alwaysCreate: opts.alwaysCreate,
			descriptive:  opts.descriptive,
			include:      opts.include,
			exclude:      opts.exclude,
		})
		if err != nil {
			return allCreated, err
		}
		allCreated = append(allCreated, created...)
	}
	paths, err := createDefaultConfigAll(root)
	if err != nil {
		return allCreated, err
	}
	allCreated = append(allCreated, paths...)
	if err := UpdateAgentsMdContextAll(root, schemas); err != nil {
		return allCreated, err
	}
	return allCreated, nil
}

// [InitLayoutAll] handle config placement centrally. When opts.alwaysCreate is
// true, default template files are seeded into existing non-empty directories
// as well as empty or new ones; existing files are never overwritten.
// opts.descriptive controls whether template files use the <name>.instructions.md
// convention (true) or the plain <name>.md convention (false).
func initLayoutSingle(root string, schema LayoutSchema, opts initOptions) ([]string, error) { //nolint:gocritic // hugeParam: intentional value semantics
	// Validate include/exclude stems before touching the filesystem.
	// This prevents unsafe characters from JSON/YAML config values reaching
	// file creation paths (allowed: A-Z, a-z, 0-9, _, -, .).
	for i, s := range opts.include {
		if bare := templateStem(s); !isSafeFileSegment(bare) {
			return nil, fmt.Errorf("init_include_files[%d]: unsafe stem %q (allowed: A-Z, a-z, 0-9, _, -, .)", i, s)
		}
	}
	for i, s := range opts.exclude {
		if bare := templateStem(s); !isSafeFileSegment(bare) {
			return nil, fmt.Errorf("init_exclude_files[%d]: unsafe stem %q (allowed: A-Z, a-z, 0-9, _, -, .)", i, s)
		}
	}

	var created []string

	// For GitHub Copilot schemas, detect whether to seed into instructions/ or
	// rules/ based on what already exists, then narrow the schema so only the
	// chosen directory is created and referenced by generated files.
	if schema.Root == ".github" {
		schema = copilotNarrowSchema(filepath.Join(root, schema.Root), schema)
	}

	layoutDir := filepath.Join(root, filepath.FromSlash(schema.Root))

	// Create the root layout directory.
	if dirIsNew(layoutDir) {
		if err := os.MkdirAll(layoutDir, 0o755); err != nil {
			return nil, err
		}
		created = append(created, schema.Root)
	}

	// Create subdirectories defined in Dirs.
	for dirName, rule := range schema.Dirs {
		dirPath := filepath.Join(layoutDir, filepath.FromSlash(dirName))
		isNew := dirIsNew(dirPath)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
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
				if err := os.WriteFile(gitkeep, []byte(""), 0o644); err != nil {
					return created, err
				}
				rel := filepath.ToSlash(filepath.Join(schema.Root, dirName, ".gitkeep"))
				created = append(created, rel)
			}
		}
	}

	// Create placeholder files for required entries that don't exist.
	for _, req := range schema.Required {
		// copilot-instructions.md is handled by createCopilotInstructions
		// below with purpose-built content; skip it here.
		if schema.Root == ".github" && req == "copilot-instructions.md" {
			continue
		}
		// CLAUDE.md is handled by createClaudeMd below with purpose-built
		// content; skip it here.
		if schema.Root == ".claude" && req == "CLAUDE.md" {
			continue
		}
		filePath := filepath.Join(layoutDir, filepath.FromSlash(req))

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return created, err
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			content := placeholderContent(req)
			if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
				return created, err
			}
			rel := filepath.ToSlash(filepath.Join(schema.Root, req))
			created = append(created, rel)
		}
	}

	// Create copilot-instructions.md for GitHub schemas when it doesn't
	// already exist. References whichever scoped-rule directory is in schema.Dirs.
	if schema.Root == ".github" {
		paths, err := createCopilotInstructions(root, layoutDir, schema, opts.descriptive)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Create CLAUDE.md for Claude schemas when it doesn't already exist.
	if schema.Root == ".claude" {
		paths, err := createClaudeMd(layoutDir, schema)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Create repogov-config.json with sensible defaults when it doesn't
	// already exist. Prefer .github/ if it already exists (FindConfig checks
	// there first); otherwise write it into the agent's own layout directory.
	// Skipped when called from InitLayoutAll, which writes a single root-level
	// config covering all platforms instead.
	if !opts.skipConfig {
		configDir := layoutDir
		if schema.Root != ".github" {
			if info, err := os.Stat(filepath.Join(root, ".github")); err == nil && info.IsDir() {
				configDir = filepath.Join(root, ".github")
			}
		}
		paths, err := createDefaultConfig(root, configDir, schema)
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	// Seed instruction/rule template files.
	//
	// Copilot (.github): seeds the full template set into whichever directory
	// was selected by copilotNarrowSchema (instructions/ or rules/ — not both).
	//
	// All other agents (Claude, Cursor, Windsurf, …): seed the full template
	// set into rules/ — the same rich set Copilot gets.
	// Existing .md and .mdc files are never overwritten.
	if schema.Root == ".github" {
		for _, seedDir := range []string{"instructions", "rules"} {
			if _, ok := schema.Dirs[seedDir]; ok {
				paths, err := createDefaultInstructions(root, layoutDir, schema, opts, seedDir)
				if err != nil {
					return created, err
				}
				created = append(created, paths...)
			}
		}
	} else {
		if _, ok := schema.Dirs["rules"]; ok {
			paths, err := createDefaultInstructions(root, layoutDir, schema, opts, "rules")
			if err != nil {
				return created, err
			}
			created = append(created, paths...)
		}
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
// references to the instructions/ directory. If the file already exists,
// nothing is created. Returns the list of created paths.
func createCopilotInstructions(root, layoutDir string, schema LayoutSchema, descriptive bool) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(layoutDir, "copilot-instructions.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		// File exists or stat failed for another reason; either way skip.
		return nil, nil
	}

	content := copilotInstructionsContent(schema, descriptive)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{filepath.ToSlash(filepath.Join(schema.Root, "copilot-instructions.md"))}, nil
}

// copilotInstructionsContent generates the content of copilot-instructions.md.
// The Scoped Instructions section references whichever of instructions/ or
// rules/ is present in schema.Dirs (never both). The output is kept under
// 50 lines to comply with the default copilot-instructions.md line limit.
// descriptive controls whether the naming-convention hint references
// *.instructions.md (true) or plain *.md (false).
func copilotInstructionsContent(schema LayoutSchema, descriptive bool) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("# Copilot Instructions\n\n")
	b.WriteString("This file provides repository-level context to GitHub Copilot and compatible AI coding agents.\n\n")

	// namingHint reflects the actual file-naming convention in use.
	namingHint := "Use the `*.md` naming convention and set `applyTo` in the YAML frontmatter (e.g. `applyTo: \"**/*.go\"`) to scope each file.\n"
	if descriptive {
		namingHint = "Use the `*.instructions.md` naming convention and set `applyTo` in the YAML header (e.g. `applyTo: \"**/*.go\"`) to scope each file.\n"
	}

	// Link to the scoped-rule directory (exactly one of instructions/ or rules/).
	if _, ok := schema.Dirs["instructions"]; ok {
		b.WriteString("## Scoped Instructions\n\n")
		b.WriteString("See modular instruction files in `" + schema.Root + "/instructions/` for scoped rules.\n")
		b.WriteString(namingHint)
		b.WriteString("\n")
	} else if _, ok := schema.Dirs["rules"]; ok {
		b.WriteString("## Scoped Instructions\n\n")
		b.WriteString("See modular instruction files in `" + schema.Root + "/rules/` for scoped rules.\n")
		b.WriteString(namingHint)
		b.WriteString("\n")
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
	b.WriteString("Run checks: `go run ./cmd/repogov -agent copilot`\n")
	b.WriteString("Re-scaffold missing files: `go run ./cmd/repogov -agent copilot init`\n\n")

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

// createClaudeMd creates .claude/CLAUDE.md with a default template when it
// doesn't already exist. Returns the list of created paths.
func createClaudeMd(layoutDir string, _ LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: intentional value semantics
	filePath := filepath.Join(layoutDir, "CLAUDE.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := claudeMdContent()
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{".claude/CLAUDE.md"}, nil
}

// claudeMdContent returns the default content for .claude/CLAUDE.md.
func claudeMdContent() string {
	var b strings.Builder
	b.WriteString("# CLAUDE.md\n\n")
	b.WriteString("Project-level instructions for Claude Code in this repository.\n\n")
	b.WriteString("## Context\n\n")
	b.WriteString("- Project overview: [README.md](../README.md)\n")
	b.WriteString("- Extended documentation: [docs/](../docs/)\n")
	b.WriteString("- Scoped rule files: [.claude/rules/](rules/)\n")
	b.WriteString("- Subagent definitions: [.claude/agents/](agents/)\n\n")
	b.WriteString("## Repository Conventions\n\n")
	b.WriteString("- Follow existing code style and patterns\n")
	b.WriteString("- Keep files within configured line limits\n")
	b.WriteString("- Write tests for new functionality\n")
	b.WriteString("\n## Repository Commands\n\n")
	b.WriteString("Run checks: `go run ./cmd/repogov -agent claude`\n")
	b.WriteString("Re-scaffold missing files: `go run ./cmd/repogov -agent claude init`\n")
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

// detectCopilotTargetDir returns the subdirectory name that should be used for
// Copilot scoped rule files during init: "instructions" when .github/instructions/
// already has content; otherwise "rules" (the default, consistent with all other
// agents). This lets init honor an existing instructions/ layout without
// forcing it on repos that have not opted in.
func detectCopilotTargetDir(ghDir string) string {
	if !isDirEmpty(filepath.Join(ghDir, "instructions")) {
		return "instructions"
	}
	return "rules"
}

// copilotNarrowSchema returns schema with only the chosen scoped-rule directory
// (instructions/ or rules/) retained in Dirs; all other directories are
// preserved. It is used during init so that exactly one sibling directory is
// created rather than both.
func copilotNarrowSchema(ghDir string, schema LayoutSchema) LayoutSchema { //nolint:gocritic // hugeParam: intentional value semantics
	targetDir := detectCopilotTargetDir(ghDir)
	narrowed := make(map[string]DirRule, len(schema.Dirs))
	for k, v := range schema.Dirs {
		if (k == "instructions" || k == "rules") && k != targetDir {
			continue
		}
		narrowed[k] = v
	}
	schema.Dirs = narrowed
	return schema
}

// detectConfigFile returns the basename and the relative path (from
// detectConfigRelFrom returns the basename and the relative path (from fromDir)
// of the first repogov config file found at root or inside root/.github/.
// If no config file exists yet, it returns the default JSON name so that
// governance links are always valid after a fresh init (which always creates
// repogov-config.json in .github/).
func detectConfigRelFrom(root, fromDir string) (name, relPath string) {
	candidates := []string{
		filepath.Join(root, ".github"),
		root,
	}
	for _, dir := range candidates {
		for _, n := range configNames {
			configPath := filepath.Join(dir, n)
			if _, err := os.Stat(configPath); err == nil {
				rel, err := filepath.Rel(fromDir, configPath)
				if err != nil {
					rel = configPath
				}
				return n, filepath.ToSlash(rel)
			}
		}
	}
	// Default: init will create repogov-config.json in .github/.
	defaultPath := filepath.Join(root, ".github", "repogov-config.json")
	rel, err := filepath.Rel(fromDir, defaultPath)
	if err != nil {
		rel = defaultPath
	}
	return "repogov-config.json", filepath.ToSlash(rel)
}

// defaultInstructionFiles defines the instruction files seeded by
// [InitLayout] when the instructions/ directory is empty. Each entry
// maps a filename to a content-generating function. Content must stay
// within the 300-line limit for .github/instructions/*.md.
// governance.instructions.md is handled separately in createDefaultInstructions
// because its content depends on which config file exists at init time.
var defaultInstructionFiles = map[string]func() string{
	"general.instructions.md":          generalInstructionsContent,
	"codereview.instructions.md":       codereviewInstructionsContent,
	"library.instructions.md":          libraryInstructionsContent,
	"testing.instructions.md":          testingInstructionsContent,
	"emoji-prevention.instructions.md": emojiPreventionInstructionsContent,
	"backend.instructions.md":          backendInstructionsContent,
	"frontend.instructions.md":         frontendInstructionsContent,
	"repo.instructions.md":             repoInstructionsContent,
}

// instructionFileName returns the on-disk filename for a default template.
// When descriptive is true the *.instructions.md convention is kept unchanged;
// when false the ".instructions" segment is stripped so that, for example,
// "general.instructions.md" becomes "general.md".
func instructionFileName(name string, descriptive bool) string {
	if descriptive {
		return name
	}
	return strings.TrimSuffix(name, ".instructions.md") + ".md"
}

// createDefaultInstructions seeds a target subdirectory (instructions/ or rules/)
// with default template files when the directory is empty (or contains only a
// .gitkeep). When opts.alwaysCreate is true, individual missing files are created
// even when the directory already has content. Existing .md and .mdc files are
// never overwritten regardless of this setting.
// subdir is the target directory name ("instructions" or "rules").
// opts.descriptive controls filename style: true -> *.instructions.md,
// false -> *.md (e.g., general.md, codereview.md).
// opts.include/exclude filter which templates are seeded (see shouldSeedFile).
// root is the repository root; used to detect the active repogov config file
// so that the governance file can link to it accurately.
func createDefaultInstructions(root, layoutDir string, schema LayoutSchema, opts initOptions, subdir string) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	targetDir := filepath.Join(layoutDir, subdir)
	if !opts.alwaysCreate && !isDirEmpty(targetDir) {
		return nil, nil
	}

	var created []string
	for templateName, contentFn := range defaultInstructionFiles {
		stem := templateStem(templateName)
		if !shouldSeedFile(stem, opts.include, opts.exclude) {
			continue
		}
		actualName := instructionFileName(templateName, opts.descriptive)
		filePath := filepath.Join(targetDir, actualName)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			// File already exists; never overwrite.
			continue
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(filePath, []byte(contentFn()), 0o644); err != nil {
			return created, err
		}
		rel := filepath.ToSlash(filepath.Join(schema.Root, subdir, actualName))
		created = append(created, rel)
	}

	// governance file is seeded separately so its config link reflects whichever
	// repogov config file is present at init time.
	if shouldSeedFile("governance", opts.include, opts.exclude) {
		govName := instructionFileName("governance.instructions.md", opts.descriptive)
		govFile := filepath.Join(targetDir, govName)
		if _, err := os.Stat(govFile); os.IsNotExist(err) {
			cfgName, cfgRel := detectConfigRelFrom(root, targetDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return created, err
			}
			if err := os.WriteFile(govFile, []byte(governanceInstructionsContent(cfgName, cfgRel, schemaRootToAgent(schema.Root))), 0o644); err != nil {
				return created, err
			}
			rel := filepath.ToSlash(filepath.Join(schema.Root, subdir, govName))
			created = append(created, rel)
		}
	}

	return created, nil
}

// repoInstructionsContent returns default content for repo.instructions.md.
// It documents the .github folder layout, pull request / merge request
// conventions, and commit standards (including the conventional-commits type
// table) for AI agents.
func repoInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Repository Instructions\n\n")

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
	b.WriteString("- `.github/dependabot.yml`\n\n")

	b.WriteString("## .gitlab Layout\n\n")
	b.WriteString("Standard files and directories for this repository's `.gitlab/` folder:\n\n")
	b.WriteString("- `.gitlab/issue_templates/`\n")
	b.WriteString("- `.gitlab/merge_request_templates/`\n")
	b.WriteString("- `.gitlab/CODEOWNERS`\n")
	b.WriteString("- `.gitlab-ci.yml`\n\n")

	b.WriteString("## Shared Root Files\n\n")
	b.WriteString("Files in the repository root recognized by both GitHub and GitLab:\n\n")
	b.WriteString("- `README.md`\n")
	b.WriteString("- `LICENSE`\n")
	b.WriteString("- `CHANGELOG.md`\n")
	b.WriteString("- `CONTRIBUTING.md`\n")
	b.WriteString("- `CODE_OF_CONDUCT.md`\n")
	b.WriteString("- `SECURITY.md`\n")
	b.WriteString("- `AGENTS.md`\n")
	b.WriteString("- `.gitignore`\n")
	b.WriteString("- `.gitattributes`\n\n")

	b.WriteString("## Pull Requests / Merge Requests\n\n")
	b.WriteString("- Keep pull requests focused on a single concern\n")
	b.WriteString("- Use a descriptive title that summarizes the change (imperative mood)\n")
	b.WriteString("- Reference related issues in the PR description\n")
	b.WriteString("- Ensure all CI checks pass before requesting review\n")
	b.WriteString("- Resolve or respond to every review comment before merging\n")
	b.WriteString("- Update documentation in the same PR that changes behavior\n\n")

	b.WriteString("## Commit Standards\n\n")
	b.WriteString("Format: `<type>(<scope>): <subject>` -- subject in imperative mood, under 72 characters.\n\n")
	b.WriteString("- Separate subject from body with a blank line when detail is needed\n")
	b.WriteString("- Reference issue or PR numbers in the body when relevant\n")
	b.WriteString("- Do not include generated, vendor, or binary files in commits\n")
	b.WriteString("- Do not commit secrets, credentials, or environment-specific values\n\n")
	b.WriteString("| Type | Use |\n")
	b.WriteString("|------|-----|\n")
	b.WriteString("| `feat:` | New exported symbol or option |\n")
	b.WriteString("| `fix:` | Bug fix |\n")
	b.WriteString("| `docs:` | Documentation only |\n")
	b.WriteString("| `style:` | Formatting (no logic change) |\n")
	b.WriteString("| `refactor:` | Code restructuring |\n")
	b.WriteString("| `test:` | Adding/updating tests |\n")
	b.WriteString("| `chore:` | Maintenance, dependencies |\n")
	b.WriteString("| `perf:` | Performance improvement |\n")
	b.WriteString("| `ci:` | CI/CD changes |\n")
	return b.String()
}

// generalInstructionsContent returns default content for general.instructions.md.
func generalInstructionsContent() string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# General Instructions\n\n")
	b.WriteString("## Writing Style\n\n")
	b.WriteString("- Use clear, concise language\n")
	b.WriteString("- Prefer active voice over passive voice\n")
	b.WriteString("- Write in complete sentences\n")
	b.WriteString("- Keep paragraphs focused on a single idea\n")
	b.WriteString("- Avoid jargon unless the audience expects it\n")
	b.WriteString(`- Use US English spelling (e.g., "behavior" not "behaviour", "summarize" not "summarise")` + "\n\n") //nolint:misspell // intentional British-English example
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
// configName is the basename of the config file (e.g. "repogov-config.yaml") and
// configRelPath is its relative path from .github/instructions/ (e.g. "../repogov-config.yaml").
// agent is the CLI agent name (e.g. "copilot", "claude", "cursor", "windsurf").
func governanceInstructionsContent(configName, configRelPath, agent string) string {
	var b strings.Builder
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Governance Instructions\n\n")
	b.WriteString("## Line Limits\n\n")
	b.WriteString("All files must stay within their configured line limit.\n")
	b.WriteString("See [" + configName + "](" + configRelPath + ") for limits and rules.\n\n")
	b.WriteString("- **Resolution order**: per-file override, first matching glob, default\n")
	b.WriteString("- A limit of `0` exempts the file (status = SKIP)\n")
	b.WriteString("- WARN at the configured `warning_threshold` percentage\n\n")
	b.WriteString("## Enforcing Limits\n\n")
	b.WriteString("### Minimal CLI Example\n\n")
	b.WriteString("```sh\n")
	b.WriteString("go run ./cmd/repogov -agent " + agent + "\n")
	b.WriteString("go run ./cmd/repogov -agent " + agent + " init\n")
	b.WriteString("```\n\n")
	b.WriteString("Pre-commit hook (`.git/hooks/pre-commit`):\n\n")
	b.WriteString("```sh\n#!/bin/sh\n")
	b.WriteString("go run github.com/nicholashoule/repogov/cmd/repogov@latest -root . limits\n```\n")
	return b.String()
}

// schemaRootToAgent maps a schema root directory name to its CLI -agent flag value.
func schemaRootToAgent(root string) string {
	switch root {
	case ".github":
		return "copilot"
	case ".claude":
		return "claude"
	case ".cursor":
		return "cursor"
	case ".windsurf":
		return "windsurf"
	default:
		return root
	}
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
	b.WriteString("---\napplyTo: \"**\"\n---\n\n")
	b.WriteString("# Emoji Prevention Instructions\n\n")
	b.WriteString("## Rule\n\n")
	b.WriteString("Do **not** introduce emoji or Unicode pictographic characters into\n")
	b.WriteString("source code, documentation, comments, commit messages, or CI output.\n")
	b.WriteString("Use plain-text equivalents instead.\n\n")
	b.WriteString("## Why\n\n")
	b.WriteString("- Emoji render inconsistently across terminals, editors, and fonts\n")
	b.WriteString("- They can complicate grep, diff, and other line-oriented tools\n")
	b.WriteString("- Screen readers may announce them unexpectedly or skip them entirely\n")
	b.WriteString("- They inflate token counts in LLM-assisted workflows\n")
	b.WriteString("- They add no semantic value that plain text cannot convey\n\n")
	b.WriteString("## Scope\n\n")
	b.WriteString("These rules apply to all files tracked by git:\n\n")
	b.WriteString("- Source files like (`.go`, `.py`, `.js`, etc.)\n")
	b.WriteString("- Markdown documentation (`.md`, `.markdown`, etc.)\n")
	b.WriteString("- YAML and JSON configuration (`.yml`, `.yaml`, `.json`)\n")
	b.WriteString("- Shell scripts and Makefiles (`.sh`, `Makefile`, etc.)\n")
	b.WriteString("- Commit messages and PR descriptions\n\n")
	b.WriteString("## Exceptions\n\n")
	b.WriteString("- Test fixtures that explicitly exercise emoji handling\n")
	b.WriteString("- Docs that describe or reference emoji (e.g., unicode-coverage.md)\n")
	b.WriteString("- Third-party files vendored as-is\n\n")
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
	b.WriteString("| lock emoji | `Security` |\n")
	b.WriteString("| fire emoji | `HOT:` or `[BREAKING]` |\n")
	b.WriteString("| bug emoji | `BUG:` or `[BUG]` |\n")
	b.WriteString("| wrench/gear emoji | `Config:` or `Setup:` |\n")
	b.WriteString("| package emoji | `Package:` or `Build:` |\n")
	b.WriteString("| magnifying glass emoji | `Search:` or `[AUDIT]` |\n")
	b.WriteString("| clipboard emoji | `TODO:` or `[TASK]` |\n")
	b.WriteString("| calendar emoji | `Date:` or `Schedule:` |\n")
	b.WriteString("| clock emoji | `Time:` or `Timeout:` |\n")
	b.WriteString("| folder emoji | `Dir:` or `Path:` |\n")
	b.WriteString("| link emoji | `URL:` or `Ref:` |\n")
	b.WriteString("| pencil/pen emoji | `Edit:` or `Draft:` |\n")
	b.WriteString("| trash emoji | `Removed:` or `[DEPRECATED]` |\n")
	b.WriteString("| recycle emoji | `Refactor:` or `Reuse:` |\n")
	b.WriteString("| shield emoji | `Security:` or `[PROTECTED]` |\n")
	b.WriteString("| key emoji | `Auth:` or `Credentials:` |\n")
	b.WriteString("| electrical plug emoji | `Plugin:` or `Integration:` |\n")
	b.WriteString("| books emoji | `Docs:` or `Reference:` |\n")
	b.WriteString("| test tube emoji | `Test:` or `[EXPERIMENTAL]` |\n")
	b.WriteString("| seedling/tree emoji | `[NEW]` or `[GROWING]` |\n")
	b.WriteString("| arrow emoji (right) | `->` or `=>` |\n")
	b.WriteString("| thumbs up emoji | `[APPROVED]` or `[ACK]` |\n")
	b.WriteString("| thumbs down emoji | `[REJECTED]` or `[NAK]` |\n")
	b.WriteString("| construction emoji | `[WIP]` or `Draft:` |\n\n")
	b.WriteString("## Enforcement\n\n")
	b.WriteString("Use the `demojify-sanitize` tool to detect emoji:\n\n")
	b.WriteString("```sh\ngo install github.com/nicholashoule/demojify-sanitize/cmd/demojify@latest\n```\n\n")
	b.WriteString("Run as a command (audit only -- exit 1 if emoji found):\n\n")
	b.WriteString("```sh\ngo run github.com/nicholashoule/demojify-sanitize/cmd/demojify -root . -exts .go,.md\n```\n\n")
	b.WriteString("Strip emoji in place (`-fix`):\n\n")
	b.WriteString("```sh\ngo run github.com/nicholashoule/demojify-sanitize/cmd/demojify -root . -exts .go,.md -fix\n```\n\n")
	b.WriteString("Pre-commit hook (`.git/hooks/pre-commit`):\n\n")
	b.WriteString("```sh\n#!/bin/sh\n")
	b.WriteString("go run github.com/nicholashoule/demojify-sanitize/cmd/demojify -root . -exts .go,.md -quiet || { echo \"ERROR: emoji found -- run: demojify -root . -exts .go,.md -fix\"; exit 1; }\n```\n")
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
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
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
func updateAgentsMdContext(filePath string, schema LayoutSchema) error { //nolint:gocritic // hugeParam: intentional value semantics
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

	return os.WriteFile(filePath, []byte(updated), 0o644)
}

// agentsMdContextSection generates the "## Context" section for AGENTS.md
// with links appropriate to the given schema. The returned string starts with
// the "## Context" heading and ends with the last list item followed by "\n";
// the blank-line separator before the next section is NOT included so that
// updateAgentsMdContext can splice it in without introducing extra blank lines.

// UpdateAgentsMdContextAll updates the "## Context" section of an existing
// AGENTS.md at the given root with merged links from all provided schemas.
// It is intended for use after initializing multiple agents in one command
// (e.g., -agent all) so that the Context section reflects every agent
// that was scaffolded rather than only the last one processed.
//
// For GitHub Copilot schemas (.github), the active scoped-rule directory
// (instructions/ or rules/) is detected from the filesystem so the link
// matches what was actually created during init.
//
// If AGENTS.md does not exist, or has no ## Context section, it is left
// unchanged and no error is returned.
func UpdateAgentsMdContextAll(root string, schemas []LayoutSchema) error { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	// Narrow any Copilot schema to the directory that was actually created.
	narrowed := make([]LayoutSchema, len(schemas))
	copy(narrowed, schemas)
	for i, s := range narrowed {
		if s.Root == ".github" {
			narrowed[i] = copilotNarrowSchema(filepath.Join(root, s.Root), s)
		}
	}

	filePath := filepath.Join(root, "AGENTS.md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	const sectionMarker = "## Context\n"
	startIdx := strings.Index(content, sectionMarker)
	if startIdx == -1 {
		return nil
	}

	newSection := agentsMdMergedContextSection(narrowed)
	afterSection := startIdx + len(sectionMarker)
	nextHeadingOffset := strings.Index(content[afterSection:], "\n## ")

	var updated string
	if nextHeadingOffset == -1 {
		updated = content[:startIdx] + newSection
	} else {
		splitIdx := afterSection + nextHeadingOffset
		updated = content[:startIdx] + newSection + content[splitIdx:]
	}

	return os.WriteFile(filePath, []byte(updated), 0o644)
}

// rulesLabel returns a human-readable label for the rules directory link based
// on the agent platform root directory.
func rulesLabel(root string) string {
	switch root {
	case ".github":
		return "Copilot rule files"
	case ".cursor":
		return "Cursor rule files"
	case ".windsurf":
		return "Windsurf rule files"
	case ".claude":
		return "Claude rule files"
	default:
		return "Agent rule files"
	}
}

// agentsMdMergedContextSection generates a "## Context" section that includes
// links for all provided schemas, deduplicated and in stable order. It is used
// when multiple platforms are initialized together so that the Context section
// references every scaffolded platform directory.
func agentsMdMergedContextSection(schemas []LayoutSchema) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("## Context\n\n")
	b.WriteString("- Project overview: [README.md](README.md)\n")
	b.WriteString("- Extended documentation: [docs/](docs/)\n")

	seen := make(map[string]bool)
	for _, schema := range schemas {
		if _, ok := schema.Dirs["instructions"]; ok {
			line := "- Scoped instruction files: [" + schema.Root + "/instructions/](" + schema.Root + "/instructions/) (`applyTo` frontmatter)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if _, ok := schema.Dirs["rules"]; ok {
			line := "- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/rules/](" + schema.Root + "/rules/)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if _, ok := schema.Dirs["agents"]; ok {
			line := "- Agent definitions: [" + schema.Root + "/agents/](" + schema.Root + "/agents/)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if schema.Root == ".github" {
			line := "- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if schema.Root == ".claude" {
			line := "- Claude repo-wide context: [.claude/CLAUDE.md](.claude/CLAUDE.md)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
	}
	return b.String()
}

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
		b.WriteString("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/rules/](" + schema.Root + "/rules/)\n")
	}
	if _, ok := schema.Dirs["agents"]; ok {
		b.WriteString("- Agent definitions: [" + schema.Root + "/agents/](" + schema.Root + "/agents/)\n")
	}
	if schema.Root == ".github" {
		b.WriteString("- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)\n")
	}
	if schema.Root == ".claude" {
		b.WriteString("- Claude repo-wide context: [.claude/CLAUDE.md](.claude/CLAUDE.md)\n")
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

	return b.String()
}

// createDefaultConfigAll creates repogov-config.json at root with the full
// default configuration covering all agent platforms. The file is not created
// when any supported repogov config file already exists at root.
func createDefaultConfigAll(root string) ([]string, error) {
	for _, n := range configNames {
		if _, err := os.Stat(filepath.Join(root, n)); err == nil {
			return nil, nil
		}
	}
	filePath := filepath.Join(root, "repogov-config.json")
	cfg := DefaultConfig()
	data := defaultConfigJSON(cfg)
	if err := os.WriteFile(filePath, []byte(data), 0o644); err != nil {
		return nil, err
	}
	return []string{"repogov-config.json"}, nil
}

// createDefaultConfig creates repogov-config.json in configDir with config
// values filtered to the schema's root. The file is not created when any
// repogov config file already exists in configDir (JSON or YAML). The
// reported relative path is derived from configDir relative to root.
func createDefaultConfig(root, configDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	// Skip if any supported config filename already exists in configDir.
	for _, n := range configNames {
		if _, err := os.Stat(filepath.Join(configDir, n)); err == nil {
			return nil, nil
		}
	}

	filePath := filepath.Join(configDir, "repogov-config.json")
	cfg := schemaConfig(DefaultConfig(), schema.Root)
	data := defaultConfigJSON(cfg)
	if err := os.WriteFile(filePath, []byte(data), 0o644); err != nil {
		return nil, err
	}
	rel, _ := filepath.Rel(root, filepath.Join(configDir, "repogov-config.json"))
	return []string{filepath.ToSlash(rel)}, nil
}

// schemaConfig returns a copy of cfg with rules and files filtered to only
// those whose glob or path begins with the given root prefix (e.g. ".github/").
// Root-level file entries (no "/" in key) are always preserved because they
// are cross-platform (e.g. AGENTS.md).
// The base scalars (Default, WarningThreshold, SkipDirs) are kept as-is.
func schemaConfig(cfg Config, root string) Config { //nolint:gocritic // hugeParam: intentional value semantics
	prefix := root + "/"
	out := Config{
		Default:          cfg.Default,
		WarningThreshold: cfg.WarningThreshold,
		IncludeExts:      cfg.IncludeExts,
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
func defaultConfigJSON(cfg Config) string { //nolint:gocritic // hugeParam: intentional value semantics
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  \"default\": " + intStr(cfg.Default) + ",\n")
	if cfg.DescriptiveNames {
		b.WriteString("  \"descriptive_names\": true,\n")
	} else {
		b.WriteString("  \"descriptive_names\": false,\n")
	}
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

	// include_exts as compact array.
	b.WriteString("  \"include_exts\": [")
	for i, e := range cfg.IncludeExts {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("\"" + e + "\"")
	}
	b.WriteString("],\n")

	// rules as compact single-line objects.
	b.WriteString("  \"rules\": [\n")
	for i, r := range cfg.Rules {
		if r.Limit != nil {
			b.WriteString("    {\"glob\": \"" + r.Glob + "\", \"limit\": " + intStr(*r.Limit) + "}")
		} else {
			b.WriteString("    {\"glob\": \"" + r.Glob + "\"}")
		}
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
