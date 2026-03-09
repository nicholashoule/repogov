// Command repogov is a CLI tool for auditing repository file lengths and
// directory layout conventions.
//
// Usage:
//
//	repogov [flags] <subcommand>
//
// Subcommands:
//
//	limits    Check file line counts against configured limits.
//	layout    Validate directory structure against a platform preset.
//	init      Scaffold the platform directory structure.
//	validate  Validate the configuration file and report issues.
//	all       Run both limits and layout checks.
//	version   Print version and exit.
//
// Flags:
//
//	-config <path>        Path to config file (default: auto-discovered)
//	-root <dir>           Repository root directory (default: .)
//	-exts .md,.mdc        Extension filter override; default from config include_exts; use "all" to scan every type
//	-agent <name[,name…]>  Agent/layout preset(s): copilot, cursor, windsurf, claude, gitlab, root, or all (required for init)
//	-descriptive          Use *.instructions.md naming convention for seeded files (overrides config descriptive_names)
//	-quiet                Suppress output; exit code only
//	-json                 Output results as JSON
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicholashoule/repogov"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. It parses args, dispatches to the
// appropriate subcommand, and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("repogov", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		configPath  string
		root        string
		exts        string
		agent       string
		quiet       bool
		jsonOut     bool
		descriptive bool
	)

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: repogov [flags] <subcommand>

Subcommands:
  limits    Check file line counts against configured limits (default)
  layout    Validate directory structure against a platform preset
  init      Scaffold the platform directory structure
  validate  Validate the configuration file and report issues
  all       Run both limits and layout checks
  version   Print version and exit

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), `
Examples:
  # Run all checks (limits + layout) for Copilot
  repogov -root . -agent copilot

  # Check line limits only
  repogov -root . limits

  # Check layout only for a specific agent
  repogov -root . -agent cursor layout

  # Scaffold Copilot + Windsurf support
  repogov -root . -agent copilot,windsurf init

  # Check root-level layout (README, LICENSE, CONTRIBUTING, etc.)
  repogov -root . -agent root layout

  # Use a custom config file
  repogov -root . -config path/to/config.json limits

  # Scan all file types (not just .md/.mdc)
  repogov -root . -exts all limits

  # Validate the config file
  repogov -root . validate

  # Output results as JSON
  repogov -root . -json limits
`)
	}

	fs.StringVar(&configPath, "config", "", "path to config file (JSON or YAML; auto-discovered if omitted)")
	fs.StringVar(&root, "root", ".", "repository root directory")
	fs.StringVar(&exts, "exts", "", "comma-separated extension filter override (default: from config include_exts; use \"all\" to scan every file type)")
	fs.StringVar(&agent, "agent", "", "agent/layout preset(s): copilot, cursor, windsurf, claude, gitlab, root, all, or comma-separated list (required for init)")
	fs.BoolVar(&quiet, "quiet", false, "suppress output; exit code only")
	fs.BoolVar(&jsonOut, "json", false, "output results as JSON")
	fs.BoolVar(&descriptive, "descriptive", false, "use *.instructions.md naming convention for seeded files (overrides config descriptive_names)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Allow extra agent names as positional args before the subcommand so that
	//   repogov -agent windsurf copilot init
	// is equivalent to
	//   repogov -agent windsurf,copilot init
	remaining := fs.Args()
	for len(remaining) > 0 && isKnownAgentName(remaining[0]) {
		if agent == "" {
			agent = remaining[0]
		} else {
			agent += "," + remaining[0]
		}
		remaining = remaining[1:]
	}

	sub := ""
	if len(remaining) > 0 {
		sub = remaining[0]
	}
	if sub == "" {
		sub = "all"
	}

	// Auto-detect the git repository root when root is the default ".".
	// This prevents the tool from scaffolding inside a subdirectory (e.g.
	// .github/ or .gitlab/) when the user forgot to pass an explicit -root.
	root = resolveRoot(root)

	switch sub {
	case "version":
		fmt.Fprintln(stdout, "repogov", version)
		return 0
	case "init":
		return runInit(root, configPath, agent, quiet, jsonOut, descriptive, stdout, stderr)
	case "validate":
		return runValidate(root, configPath, quiet, jsonOut, stdout, stderr)
	case "limits":
		return runLimits(root, configPath, exts, false, quiet, jsonOut, stdout, stderr)
	case "layout":
		// Default to checking all agents when none is specified.
		if agent == "" {
			agent = "all"
		}
		return runLayout(root, agent, quiet, jsonOut, stdout, stderr)
	case "all":
		// Default to checking all agents when none is specified.
		effective := agent
		if effective == "" {
			effective = "all"
		}
		// Print config preamble once before any checks.
		if !quiet && !jsonOut {
			cfgPath := resolveConfigPath(root, configPath)
			printConfigSource(root, cfgPath, configPath, stdout)
		}
		// Logical order: layout first (validates required files exist),
		// then limits (checks content of those files).
		code := 0
		if c := runLayout(root, effective, quiet, jsonOut, stdout, stderr); c != 0 {
			code = c
		}
		if c := runLimits(root, configPath, exts, true, quiet, jsonOut, stdout, stderr); c != 0 {
			code = c
		}
		return code
	default:
		fmt.Fprintf(stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintf(stderr, "usage: repogov [flags] <limits|layout|init|validate|all|version>\n")
		return 2
	}
}

// resolveConfigPath returns the absolute config file path to use.
// If configPath is explicit and relative, it is joined to root.
// If empty, standard locations are searched and a default is returned.
func resolveConfigPath(root, configPath string) string {
	if configPath != "" {
		if !isAbsolute(configPath) {
			return filepath.Join(root, configPath)
		}
		return configPath
	}
	if found := repogov.FindConfig(root); found != "" {
		return found
	}
	return filepath.Join(root, ".github", "repogov-config.json")
}

func runLimits(root, configPath, exts string, suppressConfig, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	cfgPath := resolveConfigPath(root, configPath)

	cfg, err := repogov.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return 2
	}

	var extList []string
	// CLI -exts flag takes highest precedence.
	if exts != "" {
		// The sentinel value "all" skips extension filtering entirely.
		if strings.ToLower(strings.TrimSpace(exts)) != "all" {
			for _, e := range strings.Split(exts, ",") {
				e = strings.TrimSpace(e)
				if e != "" {
					if !strings.HasPrefix(e, ".") {
						e = "." + e
					}
					extList = append(extList, e)
				}
			}
		}
		// "all" -> extList remains nil (no filter)
	} else {
		// No CLI flag: use include_exts from config.
		// An empty cfg.IncludeExts means scan all files.
		for _, e := range cfg.IncludeExts {
			e = strings.TrimSpace(e)
			if e != "" {
				if !strings.HasPrefix(e, ".") {
					e = "." + e
				}
				extList = append(extList, e)
			}
		}
	}

	results, err := repogov.CheckDir(root, extList, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error scanning directory: %v\n", err)
		return 2
	}

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(results) //nolint:errcheck
		if !repogov.Passed(results) {
			return 1
		}
		return 0
	}

	if !quiet {
		if !suppressConfig {
			printConfigSource(root, cfgPath, configPath, stdout)
		}
		fmt.Fprintln(stdout, "Checking limits...")
		fmt.Fprintln(stdout)
		fmt.Fprint(stdout, repogov.Summary(results))
	}

	if !repogov.Passed(results) {
		return 1
	}
	return 0
}

// printConfigSource prints the config check section in the same
// [STATUS] results + labeled footer format used by Layout and Limits.
func printConfigSource(root, activePath, explicit string, stdout io.Writer) {
	rel := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return filepath.ToSlash(r)
		}
		return filepath.ToSlash(p)
	}

	fmt.Fprintln(stdout, "Checking config...")
	fmt.Fprintln(stdout)

	// If the user supplied an explicit path, just report it as active.
	if explicit != "" {
		fmt.Fprintf(stdout, "  [PASS] %s (active, explicit)\n", rel(activePath))
		fmt.Fprintf(stdout, "\nConfig: 1 checks | 1 pass | 0 warn | 0 fail | 0 info\n\n")
		return
	}

	all := repogov.FindAllConfigs(root)
	if len(all) == 0 {
		fmt.Fprintf(stdout, "  [WARN] no config file found -- using defaults\n")
		fmt.Fprintf(stdout, "\nConfig: 1 checks | 0 pass | 1 warn | 0 fail | 0 info\n\n")
		return
	}

	pass, info := 0, 0
	for i, p := range all {
		if i == 0 {
			fmt.Fprintf(stdout, "  [PASS] %s (active)\n", rel(p))
			pass++
		} else {
			fmt.Fprintf(stdout, "  [INFO] %s (overridden by %s)\n", rel(p), rel(all[0]))
			info++
		}
	}
	fmt.Fprintf(stdout, "\nConfig: %d checks | %d pass | 0 warn | 0 fail | %d info\n\n",
		len(all), pass, info)
}

// filterConfigInfos removes [Info] layout results for config files.
// Config files are separately reported by printConfigSource, so
// listing them again as "optional file present" is redundant.
func filterConfigInfos(root string, results []repogov.LayoutResult) []repogov.LayoutResult {
	allConfigs := repogov.FindAllConfigs(root)
	if len(allConfigs) == 0 {
		return results
	}
	configPaths := make(map[string]bool, len(allConfigs))
	for _, cp := range allConfigs {
		if rel, err := filepath.Rel(root, cp); err == nil {
			configPaths[filepath.ToSlash(rel)] = true
		}
	}
	filtered := make([]repogov.LayoutResult, 0, len(results))
	for _, r := range results {
		if r.Status == repogov.Info && configPaths[r.Path] {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// platformEntry pairs a platform name with its layout schema.
type platformEntry struct {
	name   string
	schema repogov.LayoutSchema
}

// isKnownAgentName reports whether s is a recognized single-platform agent
// name. "all" is intentionally excluded because it is also a subcommand
// keyword; it is handled separately in run.
func isKnownAgentName(s string) bool {
	switch strings.ToLower(s) {
	case "copilot", "cursor", "windsurf", "claude", "gitlab", "root":
		return true
	}
	return false
}

// allPlatformSchemas returns all supported platforms in a stable order.
func allPlatformSchemas() []platformEntry {
	return []platformEntry{
		{"copilot", repogov.DefaultCopilotLayout()},
		{"cursor", repogov.DefaultCursorLayout()},
		{"windsurf", repogov.DefaultWindsurfLayout()},
		{"claude", repogov.DefaultClaudeLayout()},
		{"gitlab", repogov.DefaultGitLabLayout()},
	}
}

// resolvePlatform returns the schema for a named platform, or an error
// message for unknown names. Returns nil schema and "" message for "all".
func resolvePlatform(platform string) (repogov.LayoutSchema, string) {
	switch strings.ToLower(platform) {
	case "copilot":
		return repogov.DefaultCopilotLayout(), ""
	case "cursor":
		return repogov.DefaultCursorLayout(), ""
	case "windsurf":
		return repogov.DefaultWindsurfLayout(), ""
	case "claude":
		return repogov.DefaultClaudeLayout(), ""
	case "gitlab":
		return repogov.DefaultGitLabLayout(), ""
	case "root":
		return repogov.DefaultRootLayout(), ""
	case "all":
		return repogov.LayoutSchema{}, ""
	}
	return repogov.LayoutSchema{}, "unknown agent: " + platform + " (use copilot, cursor, windsurf, claude, gitlab, root, or all)"
}

func runLayout(root, platform string, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	if strings.EqualFold(platform, "all") {
		platforms := allPlatformSchemas()
		if jsonOut {
			out := make(map[string]interface{})
			code := 0
			for i := range platforms {
				p := &platforms[i]
				// Skip platforms whose root directory is absent (same as text path).
				platformRoot := filepath.Join(root, filepath.FromSlash(p.schema.Root))
				if _, statErr := os.Stat(platformRoot); os.IsNotExist(statErr) {
					continue
				}
				results, err := repogov.CheckLayout(root, p.schema)
				if err != nil {
					fmt.Fprintf(stderr, "error checking %s layout: %v\n", p.name, err)
					return 2
				}
				out[p.name] = results
				if !repogov.LayoutPassed(results) {
					code = 1
				}
			}
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out) //nolint:errcheck
			return code
		}
		code := 0
		for i := range platforms {
			p := &platforms[i]
			// When running all platforms, skip any whose root directory is absent.
			// Repos that only implement a subset of platforms should not fail
			// for platforms they haven't adopted.
			platformRoot := filepath.Join(root, filepath.FromSlash(p.schema.Root))
			if _, statErr := os.Stat(platformRoot); os.IsNotExist(statErr) {
				continue
			}
			results, err := repogov.CheckLayout(root, p.schema)
			if err != nil {
				fmt.Fprintf(stderr, "error checking %s layout: %v\n", p.name, err)
				return 2
			}
			if !quiet {
				fmt.Fprintf(stdout, "Checking layout (%s)...\n\n", p.name)
				fmt.Fprint(stdout, repogov.LayoutSummary(filterConfigInfos(root, results)))
			}
			if !repogov.LayoutPassed(results) {
				code = 1
			}
		}
		return code
	}

	schema, errMsg := resolvePlatform(platform)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 2
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		fmt.Fprintf(stderr, "error checking layout: %v\n", err)
		return 2
	}

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(results) //nolint:errcheck
		if !repogov.LayoutPassed(results) {
			return 1
		}
		return 0
	}

	if !quiet {
		fmt.Fprintf(stdout, "Checking layout (%s)...\n\n", platform)
		fmt.Fprint(stdout, repogov.LayoutSummary(filterConfigInfos(root, results)))
	}

	if !repogov.LayoutPassed(results) {
		return 1
	}
	return 0
}

func runInit(root, configPath, platform string, quiet, jsonOut, descriptive bool, stdout, stderr io.Writer) int {
	if platform == "" {
		fmt.Fprintln(stderr, "error: -agent is required for init")
		fmt.Fprintln(stderr, "usage: repogov -agent <copilot|cursor|windsurf|claude|all[,...]> init")
		return 2
	}

	// Load config so that init_always_create (and any future init options) are
	// honored. Config is optional; missing files silently use defaults.
	cfgPath := configPath
	if cfgPath == "" {
		if found := repogov.FindConfig(root); found != "" {
			cfgPath = found
		}
	} else if !isAbsolute(cfgPath) {
		cfgPath = filepath.Join(root, cfgPath)
	}
	cfg, err := repogov.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return 2
	}
	if descriptive {
		cfg.DescriptiveNames = true
	}

	// Normalize: split comma-separated list and trim spaces.
	parts := strings.Split(platform, ",")
	var agents []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			agents = append(agents, p)
		}
	}

	// Expand "all" to every platform.
	forAll := false
	for _, a := range agents {
		if a == "all" {
			forAll = true
			break
		}
	}

	if forAll || len(agents) > 1 {
		var schemas []repogov.LayoutSchema
		var platformNames []string
		if forAll {
			allSchemas := allPlatformSchemas()
			for i := range allSchemas {
				p := &allSchemas[i]
				schemas = append(schemas, p.schema)
				platformNames = append(platformNames, p.name)
			}
		} else {
			for _, a := range agents {
				schema, errMsg := resolvePlatform(a)
				if errMsg != "" {
					fmt.Fprintln(stderr, errMsg)
					return 2
				}
				schemas = append(schemas, schema)
				platformNames = append(platformNames, a)
			}
		}
		created, err := repogov.InitLayoutAllWithConfig(root, schemas, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "error initializing layouts: %v\n", err)
			return 2
		}
		displayName := strings.Join(platformNames, "+")
		if forAll {
			displayName = "all"
		}
		if !quiet && !jsonOut && len(created) > 0 {
			fmt.Fprintf(stdout, "Scaffolded %s layouts (%d items created):\n", displayName, len(created))
			for _, item := range created {
				fmt.Fprintf(stdout, "  + %s\n", item)
			}
		}
		if jsonOut {
			type initResult struct {
				Platform string   `json:"platform"`
				Created  []string `json:"created"`
			}
			if created == nil {
				created = []string{}
			}
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			enc.Encode([]initResult{{Platform: displayName, Created: created}}) //nolint:errcheck
		}
		return 0
	}

	// Single-agent path.
	schema, errMsg := resolvePlatform(agents[0])
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 2
	}

	created, err := repogov.InitLayoutWithConfig(root, schema, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error initializing layout: %v\n", err)
		return 2
	}

	if jsonOut {
		out := struct {
			Platform string   `json:"platform"`
			Created  []string `json:"created"`
		}{
			Platform: agents[0],
			Created:  created,
		}
		if out.Created == nil {
			out.Created = []string{}
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out) //nolint:errcheck
		return 0
	}

	if len(created) == 0 {
		if !quiet {
			fmt.Fprintln(stdout, "Layout already exists -- nothing to create.")
		}
		return 0
	}

	if !quiet {
		fmt.Fprintf(stdout, "Scaffolded %s layout (%d items created):\n", agents[0], len(created))
		for _, p := range created {
			fmt.Fprintf(stdout, "  + %s\n", p)
		}
	}
	return 0
}

func runValidate(root, configPath string, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	// Resolve config path the same way as runLimits.
	cfgPath := configPath
	if cfgPath == "" {
		if found := repogov.FindConfig(root); found != "" {
			cfgPath = found
		} else {
			if !quiet {
				fmt.Fprintln(stderr, "No config file found.")
				fmt.Fprintln(stderr, "Create repogov-config.json or use -config to specify a path.")
			}
			return 2
		}
	} else if !isAbsolute(cfgPath) {
		cfgPath = filepath.Join(root, cfgPath)
	}

	// Display path: slash-separated, relative to root when possible.
	displayPath := cfgPath
	if rel, err := filepath.Rel(root, cfgPath); err == nil {
		displayPath = filepath.ToSlash(rel)
	}

	cfg, err := repogov.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config %s: %v\n", displayPath, err)
		return 2
	}

	violations := repogov.ValidateConfig(cfg)

	if jsonOut {
		out := struct {
			Path       string              `json:"path"`
			Valid      bool                `json:"valid"`
			Violations []repogov.Violation `json:"violations"`
		}{
			Path:       displayPath,
			Valid:      len(violations) == 0,
			Violations: violations,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out) //nolint:errcheck
		if !out.Valid {
			return 1
		}
		return 0
	}

	if quiet {
		if len(violations) > 0 {
			return 1
		}
		return 0
	}

	if len(violations) == 0 {
		fmt.Fprintf(stdout, "Config %s is valid.\n", displayPath)
		return 0
	}

	errors, warnings := 0, 0
	fmt.Fprintf(stdout, "Config %s has issues:\n\n", displayPath)
	for _, v := range violations {
		icon := "WARNING"
		if v.Severity == "error" {
			icon = "[FAIL]"
			errors++
		} else {
			warnings++
		}
		fmt.Fprintf(stdout, "  %s [%s] %s: %s\n", icon, v.Severity, v.Field, v.Message)
	}
	fmt.Fprintf(stdout, "\n%d error(s), %d warning(s)\n", errors, warnings)

	if errors > 0 {
		return 1
	}
	return 0
}

// findGitRoot walks up from dir searching for a directory that contains a
// ".git" entry (directory or file, to support Git worktrees). It returns the
// path to that directory, or "" when none is found before the filesystem root.
func findGitRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root with no .git found.
			return ""
		}
		dir = parent
	}
}

// resolveRoot returns the effective repository root to use for all operations.
// When root is the default "." it resolves the working directory and walks up
// to find the nearest git repository root, preventing confusing results when
// the tool is invoked from a subdirectory such as ".github" or ".gitlab".
// Falls back to the literal value when no git root is found.
func resolveRoot(root string) string {
	if root != "." {
		return root
	}
	wd, err := os.Getwd()
	if err != nil {
		return root
	}
	if git := findGitRoot(wd); git != "" {
		return git
	}
	return root
}

// isAbsolute returns true if the path looks absolute on any platform.
func isAbsolute(path string) bool {
	if len(path) == 0 {
		return false
	}
	if path[0] == '/' || path[0] == '\\' {
		return true
	}
	// Windows drive letter: C:\...
	if len(path) >= 3 && path[1] == ':' && (path[2] == '/' || path[2] == '\\') {
		return true
	}
	return false
}
