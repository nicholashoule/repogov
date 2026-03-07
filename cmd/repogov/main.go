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
//	-exts .go,.md         Comma-separated extension filter for limits check
//	-platform <name>      Layout preset: github, gitlab, cursor, windsurf, claude, or all (required for init)
//	-quiet                Suppress output; exit code only
//	-json                 Output results as JSON
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
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
		configPath string
		root       string
		exts       string
		platform   string
		quiet      bool
		jsonOut    bool
	)

	fs.StringVar(&configPath, "config", "", "path to config file (JSON or YAML; auto-discovered if omitted)")
	fs.StringVar(&root, "root", ".", "repository root directory")
	fs.StringVar(&exts, "exts", "", "comma-separated extension filter (e.g., .go,.md)")
	fs.StringVar(&platform, "platform", "", "layout preset: github, gitlab, cursor, windsurf, claude, or all (required for init)")
	fs.BoolVar(&quiet, "quiet", false, "suppress output; exit code only")
	fs.BoolVar(&jsonOut, "json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	sub := fs.Arg(0)
	if sub == "" {
		sub = "all"
	}

	switch sub {
	case "version":
		fmt.Fprintln(stdout, "repogov", version)
		return 0
	case "init":
		return runInit(root, platform, quiet, jsonOut, stdout, stderr)
	case "validate":
		return runValidate(root, configPath, quiet, jsonOut, stdout, stderr)
	case "limits":
		return runLimits(root, configPath, exts, quiet, jsonOut, stdout, stderr)
	case "layout":
		// Default to checking all platforms when none is specified.
		if platform == "" {
			platform = "all"
		}
		return runLayout(root, platform, quiet, jsonOut, stdout, stderr)
	case "all":
		// Default to checking all platforms when none is specified.
		effective := platform
		if effective == "" {
			effective = "all"
		}
		code := 0
		if c := runLimits(root, configPath, exts, quiet, jsonOut, stdout, stderr); c != 0 {
			code = c
		}
		if c := runLayout(root, effective, quiet, jsonOut, stdout, stderr); c != 0 {
			code = c
		}
		return code
	default:
		fmt.Fprintf(stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintf(stderr, "usage: repogov [flags] <limits|layout|init|validate|all|version>\n")
		return 2
	}
}

func runLimits(root, configPath, exts string, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	// Resolve config path: if the user specified a custom path use it;
	// otherwise search standard locations (repo root, then .github/).
	cfgPath := configPath
	if cfgPath == "" {
		// Auto-discover: search repo root then .github/ for any
		// supported config filename (JSON or YAML).
		if found := repogov.FindConfig(root); found != "" {
			cfgPath = found
		} else {
			cfgPath = root + "/.github/repogov-config.json"
		}
	} else if !isAbsolute(cfgPath) {
		cfgPath = root + "/" + cfgPath
	}

	cfg, err := repogov.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return 2
	}

	var extList []string
	if exts != "" {
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
		fmt.Fprint(stdout, repogov.Summary(results))
	}

	if !repogov.Passed(results) {
		return 1
	}
	return 0
}

// platformEntry pairs a platform name with its layout schema.
type platformEntry struct {
	name   string
	schema repogov.LayoutSchema
}

// allPlatformSchemas returns all supported platforms in a stable order.
func allPlatformSchemas() []platformEntry {
	return []platformEntry{
		{"github", repogov.DefaultGitHubLayout()},
		{"gitlab", repogov.DefaultGitLabLayout()},
		{"cursor", repogov.DefaultCursorLayout()},
		{"windsurf", repogov.DefaultWindsurfLayout()},
		{"claude", repogov.DefaultClaudeLayout()},
	}
}

// resolvePlatform returns the schema for a named platform, or an error
// message for unknown names. Returns nil schema and "" message for "all".
func resolvePlatform(platform string) (repogov.LayoutSchema, string) {
	switch strings.ToLower(platform) {
	case "github":
		return repogov.DefaultGitHubLayout(), ""
	case "gitlab":
		return repogov.DefaultGitLabLayout(), ""
	case "cursor":
		return repogov.DefaultCursorLayout(), ""
	case "windsurf":
		return repogov.DefaultWindsurfLayout(), ""
	case "claude":
		return repogov.DefaultClaudeLayout(), ""
	case "all":
		return repogov.LayoutSchema{}, ""
	}
	return repogov.LayoutSchema{}, "unknown platform: " + platform + " (use github, gitlab, cursor, windsurf, claude, or all)"
}

func runLayout(root, platform string, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	if strings.ToLower(platform) == "all" {
		if jsonOut {
			out := make(map[string]interface{})
			code := 0
			for _, p := range allPlatformSchemas() {
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
		for _, p := range allPlatformSchemas() {
			results, err := repogov.CheckLayout(root, p.schema)
			if err != nil {
				fmt.Fprintf(stderr, "error checking %s layout: %v\n", p.name, err)
				return 2
			}
			if !quiet {
				fmt.Fprint(stdout, repogov.LayoutSummary(results))
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
		fmt.Fprint(stdout, repogov.LayoutSummary(results))
	}

	if !repogov.LayoutPassed(results) {
		return 1
	}
	return 0
}

func runInit(root, platform string, quiet, jsonOut bool, stdout, stderr io.Writer) int {
	if platform == "" {
		fmt.Fprintln(stderr, "error: -platform is required for init")
		fmt.Fprintln(stderr, "usage: repogov -platform <github|gitlab|cursor|windsurf|claude|all> init")
		return 2
	}
	if strings.ToLower(platform) == "all" {
		type initResult struct {
			Platform string   `json:"platform"`
			Created  []string `json:"created"`
		}
		var allResults []initResult
		for _, p := range allPlatformSchemas() {
			created, err := repogov.InitLayout(root, p.schema)
			if err != nil {
				fmt.Fprintf(stderr, "error initializing %s layout: %v\n", p.name, err)
				return 2
			}
			if created == nil {
				created = []string{}
			}
			allResults = append(allResults, initResult{Platform: p.name, Created: created})
			if !quiet && !jsonOut && len(created) > 0 {
				fmt.Fprintf(stdout, "Scaffolded %s layout (%d items created):\n", p.name, len(created))
				for _, item := range created {
					fmt.Fprintf(stdout, "  + %s\n", item)
				}
			}
		}
		if jsonOut {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			enc.Encode(allResults) //nolint:errcheck
		}
		return 0
	}

	schema, errMsg := resolvePlatform(platform)
	if errMsg != "" {
		fmt.Fprintln(stderr, errMsg)
		return 2
	}

	created, err := repogov.InitLayout(root, schema)
	if err != nil {
		fmt.Fprintf(stderr, "error initializing layout: %v\n", err)
		return 2
	}

	if jsonOut {
		out := struct {
			Platform string   `json:"platform"`
			Created  []string `json:"created"`
		}{
			Platform: platform,
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
		fmt.Fprintf(stdout, "Scaffolded %s layout (%d items created):\n", platform, len(created))
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
		cfgPath = root + "/" + cfgPath
	}

	cfg, err := repogov.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config %s: %v\n", cfgPath, err)
		return 2
	}

	violations := repogov.ValidateConfig(cfg)

	if jsonOut {
		out := struct {
			Path       string              `json:"path"`
			Valid      bool                `json:"valid"`
			Violations []repogov.Violation `json:"violations"`
		}{
			Path:       cfgPath,
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
		fmt.Fprintf(stdout, "Config %s is valid.\n", cfgPath)
		return 0
	}

	errors, warnings := 0, 0
	fmt.Fprintf(stdout, "Config %s has issues:\n\n", cfgPath)
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
