package repogov

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// createCopilotInstructions creates .github/copilot-instructions.md with
// references to the instructions/ directory. If the file already exists,
// nothing is created. Returns the list of created paths.
func createCopilotInstructions(root, layoutDir string, schema LayoutSchema, descriptive bool, lang string) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(layoutDir, "copilot-instructions.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		// File exists or stat failed for another reason; either way skip.
		return nil, nil
	}

	content := copilotInstructionsContent(schema, descriptive, lang)
	if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{filepath.ToSlash(filepath.Join(schema.Root, "copilot-instructions.md"))}, nil
}

// copilotInstructionsContent generates the content of copilot-instructions.md
// from the embedded template. The Scoped Instructions section references
// whichever of instructions/ or rules/ is present in schema.Dirs. descriptive
// controls whether the naming-convention hint references *.instructions.md
// (true) or plain *.md (false).
func copilotInstructionsContent(schema LayoutSchema, descriptive bool, lang string) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
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
		LanguageLine    string
	}{
		Root:            schema.Root,
		InstructionsDir: instrDir,
		NamingHint:      namingHint,
		Agent:           schemaRootToAgent(schema.Root),
		LanguageLine:    languageDirective(lang),
	}
	return mustRenderTemplate("agents/copilot-instructions.md.tmpl", data)
}

// createClaudeMd creates .claude/CLAUDE.md with a default template when it
// doesn't already exist. Returns the list of created paths.
func createClaudeMd(layoutDir string, schema LayoutSchema, lang string) ([]string, error) { //nolint:gocritic // hugeParam: intentional value semantics
	filePath := filepath.Join(layoutDir, "CLAUDE.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := claudeMdContent(schemaRootToAgent(schema.Root), schema.Root, lang)
	if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{".claude/CLAUDE.md"}, nil
}

// claudeMdContent returns the default content for .claude/CLAUDE.md rendered
// from the embedded template. agent is the CLI agent name (e.g. "claude").
// root is the schema root directory (e.g. ".claude").
func claudeMdContent(agent, root, lang string) string {
	return mustRenderTemplate("agents/CLAUDE.md.tmpl", struct {
		Agent        string
		Root         string
		LanguageLine string
	}{agent, root, languageDirective(lang)})
}

// createGeminiMd creates GEMINI.md at the repository root with a default
// template when it doesn't already exist. Returns the list of created paths.
func createGeminiMd(root, lang string) ([]string, error) {
	filePath := filepath.Join(root, "GEMINI.md")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := geminiMdContent(lang)
	if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{"GEMINI.md"}, nil
}

// geminiMdContent returns the default content for GEMINI.md rendered from
// the embedded template.
func geminiMdContent(lang string) string {
	return mustRenderTemplate("agents/GEMINI.md.tmpl", struct {
		Agent        string
		LanguageLine string
	}{"gemini", languageDirective(lang)})
}

// createZedRules creates .rules at the repository root with a default template
// when it doesn't already exist. Returns the list of created paths.
func createZedRules(root, lang string) ([]string, error) {
	filePath := filepath.Join(root, ".rules")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return nil, nil
	}
	content := zedRulesContent(lang)
	if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return []string{".rules"}, nil
}

// zedRulesContent returns the default content for .rules rendered from
// the embedded template.
func zedRulesContent(lang string) string {
	return mustRenderTemplate("agents/zed-rules.tmpl", struct {
		Agent        string
		LanguageLine string
	}{"zed", languageDirective(lang)})
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
	// Ignore .gitkeep placeholder and README.md metadata files.
	for _, e := range entries {
		if e.Name() != ".gitkeep" && e.Name() != "README.md" {
			return false
		}
	}
	return true
}

// dirHasGlobFiles reports whether dir contains at least one non-directory
// entry whose name matches the given glob pattern (filepath.Match syntax).
// Infrastructure files (.gitkeep, README.md) are excluded from matching.
func dirHasGlobFiles(dir, glob string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == ".gitkeep" || e.Name() == "README.md" {
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

// detectConfigRelFrom returns the basename and the relative path (from fromDir)
// of the active repogov config file, found with [FindConfig] precedence (repo
// root before .github/, JSON before YAML) so governance links point at the
// file repogov actually reads, never a shadowed one. If no config file exists
// yet, it returns the default JSON name so that governance links are always
// valid after a fresh multi-agent init (which creates repogov-config.json at
// the repository root after seeding).
func detectConfigRelFrom(root, fromDir string) (name, relPath string) {
	if configPath := FindConfig(root); configPath != "" {
		rel, err := filepath.Rel(fromDir, configPath)
		if err != nil {
			rel = configPath
		}
		return filepath.Base(configPath), filepath.ToSlash(rel)
	}
	// Default: init will create repogov-config.json at the repository root.
	defaultPath := filepath.Join(root, "repogov-config.json")
	rel, err := filepath.Rel(fromDir, defaultPath)
	if err != nil {
		rel = defaultPath
	}
	return "repogov-config.json", filepath.ToSlash(rel)
}

// defaultInstructionFilesFor returns the map of template names to content
// functions for the given schema root (e.g. ".github", ".claude"). root is
// passed to templates that need it (e.g. general.md.tmpl). subdir is the
// target subdirectory being seeded ("rules", "instructions", or "steering").
func defaultInstructionFilesFor(root, subdir, lang string) map[string]func() string {
	return map[string]func() string{
		"general.instructions.md":          func() string { return generalInstructionsContent(root, subdir, lang) },
		"memory.instructions.md":           memoryInstructionsContent,
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
	for templateName, contentFn := range defaultInstructionFilesFor(schema.Root, subdir, opts.language) {
		stem := templateStem(templateName)
		if !shouldSeedFile(stem, opts.include, opts.exclude) {
			continue
		}
		actualName := instructionFileName(templateName, opts.descriptive)
		content := contentFn()
		if schema.Root == ".cursor" {
			// Cursor ignores plain .md rules; seed .mdc with Cursor frontmatter.
			actualName, content = cursorRule(templateName, content)
		}
		filePath := filepath.Join(targetDir, actualName)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			// File already exists; never overwrite.
			continue
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return created, err
		}
		if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
			return created, err
		}
		rel := filepath.ToSlash(filepath.Join(schema.Root, subdir, actualName))
		created = append(created, rel)
	}

	// governance file is seeded separately so its config link reflects whichever
	// repogov config file is present at init time.
	if shouldSeedFile("governance", opts.include, opts.exclude) {
		govName := instructionFileName("governance.instructions.md", opts.descriptive)
		var cursorGov string
		if schema.Root == ".cursor" {
			govName = "governance.mdc"
		}
		govFile := filepath.Join(targetDir, govName)
		if _, err := os.Stat(govFile); os.IsNotExist(err) {
			cfgName, cfgRel := detectConfigRelFrom(root, targetDir)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return created, err
			}
			govContent := governanceInstructionsContent(cfgName, cfgRel, schemaRootToAgent(schema.Root))
			if schema.Root == ".cursor" {
				_, cursorGov = cursorRule("governance.instructions.md", govContent)
				govContent = cursorGov
			}
			if err := writeNewFile(govFile, []byte(govContent), 0o644); err != nil {
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
func generalInstructionsContent(root, rulesDir, lang string) string {
	return mustRenderTemplate("rules/general.md.tmpl", struct {
		Root         string
		RulesPath    string
		LanguageLine string
	}{root, filepath.ToSlash(filepath.Join(root, rulesDir)), languageDirective(lang)})
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
	case ".zed":
		return "zed"
	default:
		return root
	}
}

// memoryInstructionsContent returns default content for memory.instructions.md.
func memoryInstructionsContent() string {
	return mustRenderTemplate("rules/memory.md.tmpl", nil)
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
// already exist, or refreshes only its marked Context section when it does
// (see [AgentsMdContextStatus]). The file is cross-agent (GitHub
// Copilot, Cursor, Claude Code, OpenAI Codex) and provides agent-readable
// instructions with links to project documentation and platform-specific
// instruction directories. refresh false skips the update of an existing
// file (multi-schema init refreshes it once, with merged links, at the end).
func createAgentsMd(root string, schema LayoutSchema, lang string, refresh bool) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	filePath := filepath.Join(root, "AGENTS.md")
	if _, err := os.Lstat(filePath); os.IsNotExist(err) {
		// File does not exist: create it with full generated content.
		content := agentsMdContent(schema, lang)
		if err := writeNewFile(filePath, []byte(content), 0o644); err != nil {
			return nil, err
		}
		return []string{"AGENTS.md"}, nil
	}
	if !refresh {
		return nil, nil
	}
	// File exists: update only the marked Context section so it reflects the
	// currently selected agent/platform without overwriting user content.
	if err := spliceContextSection(filePath, agentsMdMergedContextSection([]LayoutSchema{schema}, lang)); err != nil {
		return nil, err
	}
	return nil, nil
}

// maxAgentsMdBytes bounds the AGENTS.md files repogov will rewrite.
const maxAgentsMdBytes = 4 << 20

// The marker comments that delimit the generated Context section of
// AGENTS.md. Only the text between one begin and one end marker is ever
// rewritten; Markdown structure is never parsed to guess the section bounds.
const (
	agentsMdContextBegin = "<!-- repogov:context:begin -->"
	agentsMdContextEnd   = "<!-- repogov:context:end -->"
)

// AgentsMdContextStatus reports whether init can refresh the Context section
// of the AGENTS.md at root: refreshable is true when the file holds exactly
// one "<!-- repogov:context:begin -->" marker followed by exactly one
// "<!-- repogov:context:end -->" marker. Otherwise reason says why (no file,
// markers missing, duplicated or out of order, not a regular file, or
// unreadable) and, for marker problems, how to opt in. Init leaves an
// unrefreshable AGENTS.md byte-for-byte unchanged without returning an
// error, so callers use this to tell the user.
func AgentsMdContextStatus(root string) (refreshable bool, reason string) {
	filePath := filepath.Join(root, "AGENTS.md")
	info, err := os.Lstat(filePath)
	switch {
	case os.IsNotExist(err):
		return false, "AGENTS.md does not exist; 'repogov init' creates it with the Context markers"
	case err != nil:
		return false, fmt.Sprintf("AGENTS.md cannot be read: %v", err)
	case !info.Mode().IsRegular():
		return false, "AGENTS.md is not a regular file (symlink or special file); repogov never follows or rewrites it -- replace it with a regular file"
	case info.Size() > maxAgentsMdBytes:
		return false, fmt.Sprintf("AGENTS.md is %d bytes; repogov refuses to rewrite a file over %d bytes", info.Size(), maxAgentsMdBytes)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Sprintf("AGENTS.md cannot be read: %v", err)
	}
	if _, _, why := contextMarkerBounds(string(data)); why != "" {
		return false, contextNotRefreshedReason(why)
	}
	return true, ""
}

// contextNotRefreshedReason explains a marker problem and how to opt in.
func contextNotRefreshedReason(why string) string {
	return fmt.Sprintf("AGENTS.md Context section not refreshed: the file %s, so it was left unchanged -- to opt in, put %s on the line before \"## Context\" and %s after the section, then re-run 'repogov init'",
		why, agentsMdContextBegin, agentsMdContextEnd)
}

// spliceContextSection replaces the text between the Context markers in
// filePath with a newline and newSection, preserving every byte outside the
// markers. Missing, duplicated or out-of-order markers leave the file
// untouched and are not an error: [AgentsMdContextStatus] reports them. An
// unchanged section is not rewritten. A symlink or special file, an
// oversized or unreadable file and a failed write are errors, and the write
// is atomic.
func spliceContextSection(filePath, newSection string) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file (symlink or special file); refusing to update its Context section -- replace it with a regular file", filePath)
	}
	if info.Size() > maxAgentsMdBytes {
		return fmt.Errorf("%s is %d bytes; refusing to rewrite a file over %d bytes", filePath, info.Size(), maxAgentsMdBytes)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	start, end, why := contextMarkerBounds(content)
	if why != "" {
		return nil // unmarked: left untouched; AgentsMdContextStatus says why
	}

	// Line endings follow the line the begin marker ends; a marker that does
	// not end its line falls back to the file's dominant style.
	nl := "\n"
	if rest := content[start:]; strings.HasPrefix(rest, "\r\n") || (!strings.HasPrefix(rest, "\n") && strings.Contains(content, "\r\n")) {
		nl = "\r\n"
	}
	inner := nl + strings.ReplaceAll(newSection, "\n", nl)
	if content[start:end] == inner {
		return nil
	}
	updated := content[:start] + inner + content[end:]
	return writeFileAtomic(filePath, []byte(updated), info.Mode().Perm())
}

// contextMarkerBounds returns the byte range [start, end) between the
// Context begin and end markers in content. reason is non-empty, naming the
// problem, unless content holds exactly one begin marker followed by exactly
// one end marker.
func contextMarkerBounds(content string) (start, end int, reason string) {
	nb, ne := strings.Count(content, agentsMdContextBegin), strings.Count(content, agentsMdContextEnd)
	switch {
	case nb == 0 && ne == 0:
		return 0, 0, "has no repogov:context markers"
	case nb != 1:
		return 0, 0, fmt.Sprintf("has %d begin markers (want exactly one)", nb)
	case ne != 1:
		return 0, 0, fmt.Sprintf("has %d end markers (want exactly one)", ne)
	}
	start = strings.Index(content, agentsMdContextBegin) + len(agentsMdContextBegin)
	end = strings.Index(content, agentsMdContextEnd)
	if end < start {
		return 0, 0, "has its end marker before its begin marker"
	}
	return start, end, ""
}

// UpdateAgentsMdContextAll refreshes the marked Context section of an
// existing AGENTS.md at the given root with merged links from all provided
// schemas. It is intended for use after initializing multiple agents in one
// command (e.g., -agent all) so that the Context section reflects every
// agent that was scaffolded rather than only the last one processed. The
// section states the default prose language, [DefaultLanguage]; use
// [InitLayoutAllWithConfig] with [Config.Language] to state another.
//
// For GitHub Copilot schemas (.github), the active scoped-rule directory
// (instructions/ or rules/) is detected from the filesystem so the link
// matches what was actually created during init.
//
// Only the text between the "<!-- repogov:context:begin -->" and
// "<!-- repogov:context:end -->" markers is replaced. If AGENTS.md does not
// exist, or lacks a single, ordered marker pair, it is left byte-for-byte
// unchanged and no error is returned; [AgentsMdContextStatus] reports why.
// A symlinked, unreadable or unwritable AGENTS.md is an error.
func UpdateAgentsMdContextAll(root string, schemas []LayoutSchema) error { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	return updateAgentsMdContextAll(root, schemas, "")
}

// updateAgentsMdContextAll is [UpdateAgentsMdContextAll] with the prose
// language tag stated in the regenerated section (en-US when lang is empty).
func updateAgentsMdContextAll(root string, schemas []LayoutSchema, lang string) error { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	// Narrow any Copilot schema to the directory that was actually created.
	narrowed := make([]LayoutSchema, len(schemas))
	copy(narrowed, schemas)
	for i, s := range narrowed {
		if s.Root == ".github" {
			narrowed[i] = copilotNarrowSchema(filepath.Join(root, s.Root), s)
		}
	}

	filePath := filepath.Join(root, "AGENTS.md")
	if _, err := os.Lstat(filePath); os.IsNotExist(err) {
		return nil
	}
	return spliceContextSection(filePath, agentsMdMergedContextSection(narrowed, lang))
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

// agentsMdMergedContextSection generates the "## Context" section that goes
// between the AGENTS.md Context markers: links for all provided schemas,
// deduplicated and in stable order, preceded by the prose-language directive
// (en-US when lang is empty; none for an unknown tag). It starts with the
// "## Context" heading and ends with the last list item followed by "\n",
// byte-for-byte what agents/AGENTS.md.tmpl renders between its markers.
func agentsMdMergedContextSection(schemas []LayoutSchema, lang string) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var b strings.Builder
	b.WriteString("## Context\n\n")
	if d := languageDirective(lang); d != "" {
		b.WriteString(d + "\n\n")
	}
	b.WriteString("- Project overview: [README.md](README.md)\n")
	b.WriteString("- Extended documentation: [docs/](docs/)\n")

	seen := make(map[string]bool)
	writeLine := func(line string) {
		if !seen[line] {
			seen[line] = true
			b.WriteString(line)
		}
	}

	for _, schema := range schemas {
		if _, ok := schema.Dirs["instructions"]; ok {
			writeLine("- Scoped instruction files: [" + schema.Root + "/instructions/](" + schema.Root + "/instructions/) (`applyTo` frontmatter)\n")
		}
		if _, ok := schema.Dirs["rules"]; ok {
			writeLine("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/rules/](" + schema.Root + "/rules/)\n")
		}
		if _, ok := schema.Dirs["agents"]; ok {
			writeLine("- Agent definitions: [" + schema.Root + "/agents/](" + schema.Root + "/agents/)\n")
		}
		if schema.Root == ".github" {
			writeLine("- Copilot repo-wide context: [.github/copilot-instructions.md](.github/copilot-instructions.md)\n")
		}
		if schema.Root == ".claude" {
			writeLine("- Claude repo-wide context: [.claude/CLAUDE.md](.claude/CLAUDE.md)\n")
		}
		if _, ok := schema.Dirs["steering"]; ok {
			writeLine("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/steering/](" + schema.Root + "/steering/)\n")
		}
		if schema.Root == ".clinerules" {
			writeLine("- " + rulesLabel(schema.Root) + ": [" + schema.Root + "/](" + schema.Root + "/)\n")
		}
		if schema.Root == "." {
			for _, req := range schema.Required {
				if req == "GEMINI.md" {
					writeLine("- Gemini repo-wide context: [GEMINI.md](GEMINI.md)\n")
				}
				if req == ".rules" {
					writeLine("- Zed rules: [.rules](.rules)\n")
				}
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
	IsZed           bool
	LanguageLine    string
}

// agentsMdContent generates the content for AGENTS.md from the embedded
// agents/AGENTS.md.tmpl template. It links to the project's docs/ directory
// and to platform-specific instruction directories present in the schema.
// The output must stay within the 200-line limit.
func agentsMdContent(schema LayoutSchema, lang string) string { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	_, hasInstructions := schema.Dirs["instructions"]
	_, hasRules := schema.Dirs["rules"]
	_, hasAgents := schema.Dirs["agents"]
	_, hasSteering := schema.Dirs["steering"]
	isGemini := false
	isZed := false
	if schema.Root == "." {
		for _, req := range schema.Required {
			if req == "GEMINI.md" {
				isGemini = true
			}
			if req == ".rules" {
				isZed = true
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
		IsZed:           isZed,
		LanguageLine:    languageDirective(lang),
	}
	return mustRenderTemplate("agents/AGENTS.md.tmpl", data)
}

// createDefaultConfigAll creates repogov-config.json at root with the full
// default configuration covering all agent platforms. The file is not created
// when any supported repogov config file already exists at root or in
// .github/ (see [FindConfig]): a second config would shadow or be shadowed
// by the first. lang, when a known tag, is written as the language key.
func createDefaultConfigAll(root, lang string) ([]string, error) {
	if FindConfig(root) != "" {
		return nil, nil
	}
	filePath := filepath.Join(root, "repogov-config.json")
	cfg := DefaultConfig()
	cfg.Language = lang
	data := defaultConfigJSON(cfg)
	if err := writeNewFile(filePath, []byte(data), 0o644); err != nil {
		return nil, err
	}
	return []string{"repogov-config.json"}, nil
}

// createDefaultConfig creates repogov-config.json in configDir with config
// values filtered to the schema's root. The file is not created when any
// repogov config file already exists at the repo root or in .github/ (JSON
// or YAML; see [FindConfig]), so init never adds a config that shadows, or
// is shadowed by, the active one. The reported relative path is derived
// from configDir relative to root. descriptive mirrors opts.descriptive:
// when true, descriptive_names is set in the generated config and only
// *.instructions.md file entries are emitted. lang, when a known tag, is
// written as the language key.
func createDefaultConfig(root, configDir string, schema LayoutSchema, descriptive bool, lang string) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	if FindConfig(root) != "" {
		return nil, nil
	}

	filePath := filepath.Join(configDir, "repogov-config.json")
	baseCfg := DefaultConfig()
	baseCfg.DescriptiveNames = descriptive
	baseCfg.Language = lang
	cfg := schemaConfig(baseCfg, schema)
	data := defaultConfigJSON(cfg)
	if err := writeNewFile(filePath, []byte(data), 0o644); err != nil {
		return nil, err
	}
	rel, _ := filepath.Rel(root, filepath.Join(configDir, "repogov-config.json"))
	return []string{filepath.ToSlash(rel)}, nil
}

// schemaConfig returns a copy of cfg with rules and files filtered to only
// those whose glob or path begins with the given root prefix (e.g. ".github/").
// Genuinely cross-agent root-level files (e.g. AGENTS.md) are always preserved.
// Agent-specific root-level files (e.g. GEMINI.md) are only included when the
// file appears in schema.Required, preventing cross-contamination between agents
// that share Root=="." (e.g. Gemini owns GEMINI.md; Zed owns .rules).
// The base scalars (Default, WarningThreshold, SkipDirs) are kept as-is.
func schemaConfig(cfg Config, schema LayoutSchema) Config { //nolint:gocritic // hugeParam: intentional value semantics
	root := schema.Root
	prefix := root + "/"
	out := Config{
		Default:          cfg.Default,
		DescriptiveNames: cfg.DescriptiveNames,
		SkipFrontmatter:  cfg.SkipFrontmatter,
		WarningThreshold: cfg.WarningThreshold,
		IncludeExts:      cfg.IncludeExts,
		SkipDirs:         cfg.SkipDirs,
		Language:         cfg.Language,
	}
	for _, r := range cfg.Rules {
		if strings.HasPrefix(r.Glob, prefix) {
			out.Rules = append(out.Rules, r)
		}
	}
	for _, r := range cfg.MaxBytes {
		switch {
		case strings.HasPrefix(r.Glob, prefix):
			out.MaxBytes = append(out.MaxBytes, r)
		case !strings.Contains(r.Glob, "/") && crossAgentRootFile(r.Glob):
			out.MaxBytes = append(out.MaxBytes, r)
		case root == "." && isInRequired(r.Glob, schema.Required):
			out.MaxBytes = append(out.MaxBytes, r)
		}
	}
	for k, v := range cfg.Files {
		// When descriptive_names is off, skip the .instructions.md variants —
		// those files are never created, so their limits have no effect.
		if !cfg.DescriptiveNames && strings.HasSuffix(k, ".instructions.md") {
			continue
		}
		// When descriptive_names is on, skip plain .md entries that have a
		// corresponding .instructions.md counterpart — only the descriptive
		// variant is seeded, so the plain entry's limit is irrelevant.
		if cfg.DescriptiveNames && strings.HasSuffix(k, ".md") && !strings.HasSuffix(k, ".instructions.md") {
			if _, exists := cfg.Files[strings.TrimSuffix(k, ".md")+".instructions.md"]; exists {
				continue
			}
		}
		include := false
		switch {
		case strings.HasPrefix(k, prefix):
			// File is inside this schema's root directory.
			// When the schema has been narrowed (e.g. Copilot selects
			// instructions/ or rules/ — not both), skip file entries
			// whose subdirectory is absent from schema.Dirs.
			if sub := firstSegment(strings.TrimPrefix(k, prefix)); sub != "" {
				if _, ok := schema.Dirs[sub]; !ok {
					continue
				}
			}
			include = true
		case !strings.Contains(k, "/") && crossAgentRootFile(k):
			// Root-level file that is genuinely cross-agent (e.g. AGENTS.md).
			include = true
		case root == "." && isInRequired(k, schema.Required):
			// Root-level file explicitly owned by this schema (e.g. GEMINI.md for
			// Gemini, .rules for Zed). Scoped to Required to prevent cross-
			// contamination between agents that share Root==".".
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

// isInRequired reports whether name appears in the schema's Required list.
func isInRequired(name string, required []string) bool {
	for _, r := range required {
		if r == name {
			return true
		}
	}
	return false
}

// firstSegment returns the first path segment of rel (the text before the
// first "/"). If rel contains no slash it returns "" — the path is a direct
// child of the schema root, not inside a subdirectory.
func firstSegment(rel string) string {
	if idx := strings.Index(rel, "/"); idx >= 0 {
		return rel[:idx]
	}
	return ""
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
	b.WriteString(agentHints())
	if tag, ok := NormalizeLanguage(effectiveLanguage(cfg.Language)); ok {
		b.WriteString("  \"language\": \"" + tag + "\",\n")
	}
	b.WriteString("  \"default\": " + intStr(cfg.Default) + ",\n")
	if cfg.DescriptiveNames {
		b.WriteString("  \"descriptive_names\": true,\n")
	} else {
		b.WriteString("  \"descriptive_names\": false,\n")
	}
	if cfg.SkipFrontmatter {
		b.WriteString("  \"skip_frontmatter\": true,\n")
	} else {
		b.WriteString("  \"skip_frontmatter\": false,\n")
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

	// max_bytes as compact single-line objects (omitted when empty).
	if len(cfg.MaxBytes) > 0 {
		b.WriteString("  \"max_bytes\": [\n")
		for i, r := range cfg.MaxBytes {
			if r.Limit != nil {
				b.WriteString("    {\"glob\": \"" + r.Glob + "\", \"limit\": " + intStr(*r.Limit) + "}")
			} else {
				b.WriteString("    {\"glob\": \"" + r.Glob + "\"}")
			}
			if i < len(cfg.MaxBytes)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("  ],\n")
	}

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

// subdirReadmeData is the template data passed to readme/README.md.tmpl.
type subdirReadmeData struct {
	DirName     string
	Title       string
	Description string
	FileHint    string
}

// subdirReadmeInfo returns template data for a README.md in the given
// subdirectory. Known directory names (rules, instructions, steering, agents,
// skills) have their content defined inline in the template; unknown names
// fall back to Title/Description derived from the DirRule.
func subdirReadmeInfo(dirName string, rule DirRule) subdirReadmeData { //nolint:gocritic // hugeParam: DirRule is small enough
	title := dirName
	if len(dirName) > 0 {
		r, size := utf8.DecodeRuneInString(dirName)
		title = string(unicode.ToUpper(r)) + dirName[size:]
	}

	desc := rule.Description
	if desc != "" && !strings.HasSuffix(desc, ".") {
		desc += "."
	}

	return subdirReadmeData{
		DirName:     dirName,
		Title:       title,
		Description: desc,
		FileHint:    fileHintFromRule(rule),
	}
}

// fileHintFromRule builds a brief file-convention hint from a DirRule's Glob
// and Frontmatter fields. Returns an empty string when neither is set.
func fileHintFromRule(rule DirRule) string { //nolint:gocritic // hugeParam: DirRule is small enough
	var parts []string
	if rule.Glob != "" {
		parts = append(parts, "Files follow the `"+rule.Glob+"` naming pattern.")
	}
	if len(rule.Frontmatter) > 0 {
		parts = append(parts, "Each file supports YAML frontmatter (`"+strings.Join(rule.Frontmatter, "`, `")+"`).")
	}
	return strings.Join(parts, " ")
}

// subdirReadmeContent renders the README.md content for a subdirectory from
// the embedded readme/README.md.tmpl template.
func subdirReadmeContent(dirName string, rule DirRule) string { //nolint:gocritic // hugeParam: DirRule is small enough
	return mustRenderTemplate("readme/README.md.tmpl", subdirReadmeInfo(dirName, rule))
}

// createSubdirReadmes creates a README.md inside each non-NoCreate
// subdirectory defined in the schema. Existing README.md files are never
// overwritten. The "." directory (used by Cline for root-level rules) is
// skipped because a root-level README.md would conflict with the project's
// own README. Returns the list of created relative paths.
func createSubdirReadmes(layoutDir string, schema LayoutSchema) ([]string, error) { //nolint:gocritic // hugeParam: mirrors public InitLayout signature
	var created []string
	for dirName, rule := range schema.Dirs {
		if rule.NoCreate || dirName == "." {
			continue
		}
		dirPath := filepath.Join(layoutDir, filepath.FromSlash(dirName))
		readmePath := filepath.Join(dirPath, "README.md")
		if _, err := os.Stat(readmePath); !os.IsNotExist(err) {
			continue
		}
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return created, err
		}
		content := subdirReadmeContent(dirName, rule)
		if err := writeNewFile(readmePath, []byte(content), 0o644); err != nil {
			return created, err
		}
		rel := filepath.ToSlash(filepath.Join(schema.Root, dirName, "README.md"))
		created = append(created, rel)
	}
	return created, nil
}
