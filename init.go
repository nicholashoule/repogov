package repogov

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

// mustReadTemplate reads and returns the content of an embedded template file
// by its name within the templates/ directory. Line endings are normalized to
// LF (\n) so that content is consistent across platforms regardless of how
// the file was checked out. It panics if the file is missing, since that
// indicates a broken build rather than a runtime error.
func mustReadTemplate(name string) string {
	data, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		panic(fmt.Sprintf("repogov: missing embedded template %q: %v", name, err))
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// mustRenderTemplate reads the named embedded template and renders it with
// text/template using the provided data value (pass nil for static templates).
// Unlike mustReadTemplate, this ensures that any future {{.Placeholder}}
// additions to a template are never silently emitted as literal text.
func mustRenderTemplate(name string, data any) string {
	tmpl := template.Must(template.New(name).Parse(mustReadTemplate(name)))
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		panic(fmt.Sprintf("repogov: template render error %q: %v", name, err))
	}
	return b.String()
}

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
		if rule.NoCreate {
			continue
		}
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
		// GEMINI.md is handled by createGeminiMd below with purpose-built
		// content; skip it here.
		if schema.Root == "." && req == "GEMINI.md" {
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

	// Create GEMINI.md for Gemini schemas when it doesn't already exist.
	if schema.Root == "." {
		for _, req := range schema.Required {
			if req == "GEMINI.md" {
				paths, err := createGeminiMd(root)
				if err != nil {
					return created, err
				}
				created = append(created, paths...)
				break
			}
		}
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
		for _, seedDir := range []string{".", "rules", "steering"} {
			if _, ok := schema.Dirs[seedDir]; ok {
				paths, err := createDefaultInstructions(root, layoutDir, schema, opts, seedDir)
				if err != nil {
					return created, err
				}
				created = append(created, paths...)
			}
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

// copilotInstructionsContent generates the content of copilot-instructions.md
// from the embedded template. The Scoped Instructions section references
// whichever of instructions/ or rules/ is present in schema.Dirs. descriptive
// controls whether the naming-convention hint references *.instructions.md
// (true) or plain *.md (false).
func copilotInstructionsContent(schema LayoutSchema, descriptive bool) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	instrDir := ""
	if _, ok := schema.Dirs["instructions"]; ok {
		instrDir = "instructions"
	} else if _, ok := schema.Dirs["rules"]; ok {
		instrDir = "rules"
	}

	namingHint := "Use the `*.md` naming convention and set `applyTo` in the YAML frontmatter (e.g. `applyTo: \"**/*.go\"`) to scope each file."
	if descriptive {
		namingHint = "Use the `*.instructions.md` naming convention and set `applyTo` in the YAML header (e.g. `applyTo: \"**/*.go\"`) to scope each file."
	}

	data := struct {
		Root            string
		InstructionsDir string
		NamingHint      string
		Agent           string
	}{
		Root:            schema.Root,
		InstructionsDir: instrDir,
		NamingHint:      namingHint,
		Agent:           schemaRootToAgent(schema.Root),
	}
	return mustRenderTemplate("agents/copilot-instructions.md.tmpl", data)
}

// createClaudeMd creates .claude/CLAUDE.md with a default template when it
// doesn't already exist. Returns the list of created paths.
func createClaudeMd(layoutDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: intentional value semantics
	filePath := filepath.Join(layoutDir, "CLAUDE.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := claudeMdContent(schemaRootToAgent(schema.Root), schema.Root)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{".claude/CLAUDE.md"}, nil
}

// claudeMdContent returns the default content for .claude/CLAUDE.md rendered
// from the embedded template. agent is the CLI agent name (e.g. "claude").
// root is the schema root directory (e.g. ".claude").
func claudeMdContent(agent, root string) string {
	return mustRenderTemplate("agents/CLAUDE.md.tmpl", struct {
		Agent string
		Root  string
	}{agent, root})
}

// createGeminiMd creates GEMINI.md at the repository root with a default
// template when it doesn't already exist. Returns the list of created paths.
func createGeminiMd(root string) ([]string, error) {
	filePath := filepath.Join(root, "GEMINI.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := geminiMdContent()
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{"GEMINI.md"}, nil
}

// geminiMdContent returns the default content for GEMINI.md rendered from
// the embedded template.
func geminiMdContent() string {
	return mustRenderTemplate("agents/GEMINI.md.tmpl", struct {
		Agent string
	}{"gemini"})
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

// dirHasGlobFiles reports whether dir contains at least one non-directory
// entry whose name matches the given glob pattern (filepath.Match syntax).
func dirHasGlobFiles(dir, glob string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ok, _ := filepath.Match(glob, e.Name()); ok {
			return true
		}
	}
	return false
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
// defaultInstructionFilesFor returns the map of template names to content
// functions for the given schema root (e.g. ".github", ".claude"). root is
// passed to templates that need it (e.g. general.md.tmpl). subdir is the
// target subdirectory being seeded ("rules", "instructions", or "steering").
func defaultInstructionFilesFor(root, subdir string) map[string]func() string {
	return map[string]func() string{
		"general.instructions.md":          func() string { return generalInstructionsContent(root, subdir) },
		"codereview.instructions.md":       codereviewInstructionsContent,
		"library.instructions.md":          libraryInstructionsContent,
		"testing.instructions.md":          testingInstructionsContent,
		"emoji-prevention.instructions.md": emojiPreventionInstructionsContent,
		"backend.instructions.md":          backendInstructionsContent,
		"frontend.instructions.md":         frontendInstructionsContent,
		"security.instructions.md":         securityInstructionsContent,
		"repo.instructions.md":             repoInstructionsContent,
	}
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
	// For the root-dir case (subdir=="."), infrastructure files like
	// repogov-config.json live alongside templates, so we must only block
	// seeding when *.md template files already exist. For named subdirs the
	// existing "non-empty dir" check is correct.
	var targetPopulated bool
	if subdir == "." {
		targetPopulated = dirHasGlobFiles(targetDir, "*.md")
	} else {
		targetPopulated = !isDirEmpty(targetDir)
	}
	if !opts.alwaysCreate && targetPopulated {
		return nil, nil
	}

	var created []string
	for templateName, contentFn := range defaultInstructionFilesFor(schema.Root, subdir) {
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
func repoInstructionsContent() string {
	return mustRenderTemplate("rules/repo.md.tmpl", nil)
}

// generalInstructionsContent returns default content for general.instructions.md.
// root is the schema root directory (e.g. ".github") and rulesDir is the
// subdirectory being seeded (e.g. "rules", "instructions", "steering", or "." for
// layouts that place files directly under schema.Root).
func generalInstructionsContent(root, rulesDir string) string {
	return mustRenderTemplate("rules/general.md.tmpl", struct {
		Root      string
		RulesPath string
	}{root, filepath.ToSlash(filepath.Join(root, rulesDir))})
}

// codereviewInstructionsContent returns default content for codereview.instructions.md.
func codereviewInstructionsContent() string {
	return mustRenderTemplate("rules/codereview.md.tmpl", nil)
}

// governanceInstructionsContent returns default content for governance.instructions.md
// rendered from the embedded template. configName is the basename of the config
// file (e.g. "repogov-config.yaml") and configRelPath is its relative path from
// the target directory. agent is the CLI agent name (e.g. "copilot", "claude").
func governanceInstructionsContent(configName, configRelPath, agent string) string {
	data := struct {
		ConfigName    string
		ConfigRelPath string
		Agent         string
	}{configName, configRelPath, agent}
	return mustRenderTemplate("rules/governance.md.tmpl", data)
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
	case ".kiro":
		return "kiro"
	case ".":
		return "gemini"
	case ".continue":
		return "continue"
	case ".clinerules":
		return "cline"
	case ".roo":
		return "roocode"
	case ".aiassistant":
		return "jetbrains"
	default:
		return root
	}
}

// libraryInstructionsContent returns default content for library.instructions.md.
func libraryInstructionsContent() string {
	return mustRenderTemplate("rules/library.md.tmpl", nil)
}

// testingInstructionsContent returns default content for testing.instructions.md.
func testingInstructionsContent() string {
	return mustRenderTemplate("rules/testing.md.tmpl", nil)
}

// emojiPreventionInstructionsContent returns default content for emoji-prevention.instructions.md.
func emojiPreventionInstructionsContent() string {
	return mustRenderTemplate("rules/emoji-prevention.md.tmpl", nil)
}

// securityInstructionsContent returns default content for security.instructions.md.
func securityInstructionsContent() string {
	return mustRenderTemplate("rules/security.md.tmpl", nil)
}

// backendInstructionsContent returns default content for backend.instructions.md.
func backendInstructionsContent() string {
	return mustRenderTemplate("rules/backend.md.tmpl", nil)
}

// frontendInstructionsContent returns default content for frontend.instructions.md.
func frontendInstructionsContent() string {
	return mustRenderTemplate("rules/frontend.md.tmpl", nil)
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
	case ".kiro":
		return "Kiro steering files"
	case ".continue":
		return "Continue rule files"
	case ".clinerules":
		return "Cline rule files"
	case ".roo":
		return "Roo Code rule files"
	case ".aiassistant":
		return "JetBrains rule files"
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
		if _, ok := schema.Dirs["steering"]; ok {
			line := "- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/steering/](" + schema.Root + "/steering/)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if schema.Root == ".clinerules" {
			line := "- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/](" + schema.Root + "/)\n"
			if !seen[line] {
				seen[line] = true
				b.WriteString(line)
			}
		}
		if schema.Root == "." {
			for _, req := range schema.Required {
				if req == "GEMINI.md" {
					line := "- Gemini repo-wide context: [GEMINI.md](GEMINI.md)\n"
					if !seen[line] {
						seen[line] = true
						b.WriteString(line)
					}
					break
				}
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
	if _, ok := schema.Dirs["steering"]; ok {
		b.WriteString("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/steering/](" + schema.Root + "/steering/)\n")
	}
	if schema.Root == ".clinerules" {
		b.WriteString("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/](" + schema.Root + "/)\n")
	}
	if schema.Root == "." {
		for _, req := range schema.Required {
			if req == "GEMINI.md" {
				b.WriteString("- Gemini repo-wide context: [GEMINI.md](GEMINI.md)\n")
				break
			}
		}
	}
	return b.String()
}

// agentsMdTemplateData is the template data passed to agents/AGENTS.md.tmpl.
type agentsMdTemplateData struct {
	Root            string
	HasInstructions bool
	HasRules        bool
	RulesLabel      string
	HasAgents       bool
	IsCopilot       bool
	IsClaude        bool
	HasSteering     bool
	HasRootRules    bool
	IsGemini        bool
}

// agentsMdContent generates the content for AGENTS.md from the embedded
// agents/AGENTS.md.tmpl template. It links to the project's docs/ directory
// and to platform-specific instruction directories present in the schema.
// The output must stay within the 200-line limit.
func agentsMdContent(schema LayoutSchema) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	_, hasInstructions := schema.Dirs["instructions"]
	_, hasRules := schema.Dirs["rules"]
	_, hasAgents := schema.Dirs["agents"]
	_, hasSteering := schema.Dirs["steering"]
	isGemini := false
	if schema.Root == "." {
		for _, req := range schema.Required {
			if req == "GEMINI.md" {
				isGemini = true
				break
			}
		}
	}
	data := agentsMdTemplateData{
		Root:            schema.Root,
		HasInstructions: hasInstructions,
		HasRules:        hasRules,
		RulesLabel:      rulesLabel(schema.Root),
		HasAgents:       hasAgents,
		IsCopilot:       schema.Root == ".github",
		IsClaude:        schema.Root == ".claude",
		HasSteering:     hasSteering,
		HasRootRules:    schema.Root == ".clinerules",
		IsGemini:        isGemini,
	}
	return mustRenderTemplate("agents/AGENTS.md.tmpl", data)
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
// Genuinely cross-agent root-level files (e.g. AGENTS.md) are always preserved.
// Agent-specific root-level files (e.g. GEMINI.md) are only included when the
// schema root matches their agent (root == ".").
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
		include := false
		switch {
		case strings.HasPrefix(k, prefix):
			// File is inside this schema's root directory.
			include = true
		case !strings.Contains(k, "/") && crossAgentRootFile(k):
			// Root-level file that is genuinely cross-agent (e.g. AGENTS.md).
			include = true
		case root == "." && !strings.Contains(k, "/"):
			// Gemini schema (Root==".") governs all root-level files it owns.
			include = true
		}
		if include {
			if out.Files == nil {
				out.Files = make(map[string]int)
			}
			out.Files[k] = v
		}
	}
	return out
}

// crossAgentRootFile reports whether a root-level filename (no path separator)
// is a cross-agent standard that should appear in every agent's generated config.
// AGENTS.md is read by Copilot, Cursor, Claude, Kiro, Cline, Roo Code, and others.
// Agent-specific files like GEMINI.md are excluded here and only appear when their
// owning schema is active.
func crossAgentRootFile(name string) bool {
	return name == "AGENTS.md"
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

	// files map — sorted for deterministic output.
	b.WriteString("  \"files\": {\n")
	fileKeys := make([]string, 0, len(cfg.Files))
	for k := range cfg.Files {
		fileKeys = append(fileKeys, k)
	}
	sort.Strings(fileKeys)
	for i, k := range fileKeys {
		b.WriteString("    \"" + k + "\": " + intStr(cfg.Files[k]))
		if i < len(fileKeys)-1 {
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
