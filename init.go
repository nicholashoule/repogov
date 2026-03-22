package repogov

import (
	"fmt"
	"os"
	"path/filepath"
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
		// VS Code only auto-applies files from instructions/ when they carry
		// the *.instructions.md suffix; force descriptive naming when that
		// directory was selected so seeded files and generated config match.
		if _, ok := schema.Dirs["instructions"]; ok {
			opts.descriptive = true
		}
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
		// .rules is handled by createZedRules below with purpose-built
		// content; skip it here.
		if schema.Root == "." && req == ".rules" {
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

	// Create .rules for Zed schemas when it doesn't already exist.
	if schema.Root == "." {
		for _, req := range schema.Required {
			if req == ".rules" {
				paths, err := createZedRules(root)
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
	// there first); otherwise write it at the repo root so FindConfig can
	// discover it without an explicit -config flag. Skipped when called from
	// InitLayoutAll, which writes a single root-level config covering all
	// platforms instead.
	if !opts.skipConfig {
		configDir := layoutDir
		if schema.Root != ".github" {
			if info, err := os.Stat(filepath.Join(root, ".github")); err == nil && info.IsDir() {
				configDir = filepath.Join(root, ".github")
			} else {
				// No .github/ present: write config at repo root so FindConfig
				// can discover it without an explicit -config flag.
				configDir = root
			}
		}
		paths, err := createDefaultConfig(root, configDir, schema, opts.descriptive)
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

	// Create README.md in each non-NoCreate subdirectory to describe the
	// intent of the directory and its files. Existing README.md files are
	// never overwritten.
	readmePaths, readmeErr := createSubdirReadmes(layoutDir, schema)
	if readmeErr != nil {
		return created, readmeErr
	}
	created = append(created, readmePaths...)

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
