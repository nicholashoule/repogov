package repogov

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Status classifies a check outcome.
type Status int

const (
	// Pass indicates the file is within its limit or all layout rules are satisfied.
	Pass Status = iota
	// Warn indicates the file is at or above the warning threshold percentage.
	Warn
	// Fail indicates the file exceeds its limit or a required layout rule is violated.
	Fail
	// Skip indicates the file is exempt (limit = 0), does not match the
	// extension filter, is binary (NUL byte within the first 512 bytes and
	// no UTF-16/UTF-32 byte order mark), is a symlink or special file, or
	// holds certificate/key material (.crt, .pem, .key, .p12, .pfx, .jks).
	// Binary and sensitive files are never line-counted or parsed;
	// sensitive files and symlinks are never opened at all. A binary file
	// is still held to its byte limit ([Config.MaxBytes]).
	Skip
	// Info indicates an optional file is present (layout checks only).
	Info
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case Pass:
		return "PASS"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	case Skip:
		return "SKIP"
	case Info:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON encodes a Status as its string label ("PASS", "WARN", etc.)
// so JSON consumers see human-readable values instead of integers.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes a Status from its string label.
func (s *Status) UnmarshalJSON(data []byte) error {
	var label string
	if err := json.Unmarshal(data, &label); err != nil {
		return err
	}
	switch label {
	case "PASS":
		*s = Pass
	case "WARN":
		*s = Warn
	case "FAIL":
		*s = Fail
	case "SKIP":
		*s = Skip
	case "INFO":
		*s = Info
	default:
		return fmt.Errorf("unknown status: %q", label)
	}
	return nil
}

// PercentInt is an integer percentage that tolerates a trailing "%" sign
// in JSON and YAML input. It serializes to JSON as a string with a
// trailing "%" (e.g. "80%") for readability.
type PercentInt int

// MarshalJSON encodes a PercentInt as a percent string ("80%").
func (p PercentInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%d%%", int(p)))
}

// UnmarshalJSON decodes a PercentInt from a bare number (80) or a
// percent string ("80%").
func (p *PercentInt) UnmarshalJSON(data []byte) error {
	// Try as number first.
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*p = PercentInt(n)
		return nil
	}
	// Try as string (e.g. "80%" or "80").
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("warning_threshold: expected number or string, got %s", string(data))
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("invalid warning_threshold %q: %w", string(data), err)
	}
	*p = PercentInt(v)
	return nil
}

// Config holds the complete line-limit configuration.
type Config struct {
	// Default is the global fallback limit applied when no per-file
	// override or glob rule matches. Zero uses the built-in default (500).
	Default int `json:"default"`

	// WarningThreshold is the percentage of a file's line limit at which
	// the check result changes from PASS to WARN. Default: 85.
	WarningThreshold PercentInt `json:"warning_threshold"`

	// IncludeExts lists the file extensions that the limits check scans.
	// An empty slice means all files are scanned (no extension filter).
	// Default: [".md", ".mdc"].
	IncludeExts []string `json:"include_exts"`

	// SkipDirs lists directory names to skip during directory walks.
	SkipDirs []string `json:"skip_dirs"`

	// Rules are glob-based limit rules evaluated in order. The first
	// matching rule wins.
	Rules []Rule `json:"rules"`

	// Files maps repo-relative paths (forward slashes) to per-file
	// limits. A limit of 0 exempts the file from checking.
	Files map[string]int `json:"files"`

	// MaxBytes are glob-based byte-size limits, evaluated in order (first
	// match wins) and checked in addition to the line limits. Some agents
	// truncate or drop instruction files by size rather than line count
	// (e.g. Antigravity truncates rule files over 24,000 bytes), so a file
	// inside its line limit can still be cut off. Rule.Limit is in bytes;
	// 0 or nil means no byte limit. Exact paths work as globs.
	MaxBytes []Rule `json:"max_bytes,omitempty"`

	// InitAlwaysCreate controls whether [InitLayout] and [InitLayoutAll]
	// seed default template files even when the target directory already
	// exists and contains files. When false (the default), default files
	// are only seeded into empty or newly-created directories. When true,
	// any individual template file that is missing from an existing directory
	// is created; existing files are never overwritten regardless of this
	// setting.
	InitAlwaysCreate bool `json:"init_always_create,omitempty"`

	// DescriptiveNames controls the filename convention used when scaffolding
	// instruction and rule files. When false (the default), all template files
	// use plain <name>.md (e.g., general.md, codereview.md). When true, files
	// use the <name>.instructions.md convention (e.g., general.instructions.md).
	DescriptiveNames bool `json:"descriptive_names"`

	// InitIncludeFiles is an allowlist of template stem names to seed during
	// init (e.g., ["general", "governance", "testing"]). When non-empty, only
	// templates whose stem matches an entry are created. Takes precedence over
	// InitExcludeFiles. Stem matching strips any ".instructions.md", ".md", or
	// ".mdc" suffix before comparison.
	InitIncludeFiles []string `json:"init_include_files,omitempty"`

	// InitExcludeFiles is a blocklist of template stem names to skip during
	// init (e.g., ["backend", "frontend", "emoji-prevention"]). Entries whose
	// stem matches are not created. Ignored when InitIncludeFiles is non-empty.
	InitExcludeFiles []string `json:"init_exclude_files,omitempty"`

	// SkipFrontmatter disables YAML frontmatter validation during layout
	// checks. When false (the default), files in directories whose
	// [DirRule.Frontmatter] is non-empty are checked for the required
	// keys. When true, frontmatter requirements are stripped from all
	// schemas before checking.
	SkipFrontmatter bool `json:"skip_frontmatter,omitempty"`

	// Language is the BCP 47 tag of the natural language the repository's
	// prose is written in -- Markdown and text files and code comments --
	// e.g. "en-US" or "pt-BR". It is not a programming language. Scaffolded
	// agent instruction files (AGENTS.md, CLAUDE.md, copilot-instructions.md,
	// GEMINI.md, .rules, the general rule) state it so AI agents write in
	// that language. Must be one of [KnownLanguages] (aliases and bare
	// subtags are accepted; see [NormalizeLanguage]). The default is
	// [DefaultLanguage] ("en-US"): [DefaultConfig] sets it and [LoadConfig]
	// fills a missing or empty value with it.
	Language string `json:"language,omitempty"`
}

// Rule maps a glob pattern to a line limit.
type Rule struct {
	// Glob is a filepath.Match pattern using forward slashes.
	Glob string `json:"glob"`

	// Limit is the maximum number of lines. When nil, the config-level
	// default is used. When set to 0, matched files are exempt from checking.
	Limit *int `json:"limit,omitempty"`
}

// RuleLimit returns a pointer to n for use in Rule literals.
// A nil Limit falls through to the config default; use RuleLimit(0) to
// explicitly exempt all files matched by the rule.
func RuleLimit(n int) *int { return &n }

// Result holds the check outcome for a single file.
type Result struct {
	// Path is the repo-relative path using forward slashes.
	Path string

	// Lines is the actual line count of the file.
	Lines int

	// Limit is the resolved line limit (0 = exempt).
	Limit int

	// Pct is the percentage of the limit used (Lines * 100 / Limit).
	// Zero when the file is exempt (Limit = 0).
	Pct int

	// Bytes is the file size in bytes. Set only when a byte limit applies.
	Bytes int `json:"Bytes,omitempty"`

	// ByteLimit is the resolved byte limit (0 = none).
	ByteLimit int `json:"ByteLimit,omitempty"`

	// Status is the check outcome: Pass, Warn, Fail, or Skip.
	Status Status

	// Action is a remediation hint for WARN and FAIL outcomes.
	// Empty for PASS and SKIP. Designed to be consumed by AI agents,
	// LLMs, and MCP tools that need actionable context.
	Action string
}

// defaultLimit is the built-in fallback when Config.Default is zero.
const defaultLimit = 500

// defaultWarningThreshold is the built-in warning threshold percentage.
const defaultWarningThreshold PercentInt = 85

// DefaultConfig returns a Config with sensible defaults:
// Default=500, WarningThreshold=85, Language="en-US" ([DefaultLanguage]),
// standard SkipDirs, and .md/.mdc includes.
// It applies 300-line limits to AI rule files for:
//   - GitHub Copilot: .github/rules/*.md
//   - Cursor: .cursor/rules/*.md, .cursor/rules/*.mdc
//   - Windsurf: .windsurf/rules/*.md
//   - Claude: .claude/rules/*.md, .claude/agents/*.md
//   - Kiro: .kiro/steering/*.md
//   - Continue: .continue/rules/*.md
//   - Cline: .clinerules/*.md
//   - Roo Code: .roo/rules/*.md
//   - JetBrains AI Assistant: .aiassistant/rules/*.md
//
// and sets specific file limits:
//   - .github/copilot-instructions.md at 50 lines
//   - .claude/CLAUDE.md, AGENTS.md, GEMINI.md at 200 lines each
func DefaultConfig() Config {
	return Config{
		Default:          defaultLimit,
		WarningThreshold: defaultWarningThreshold,
		Language:         DefaultLanguage,
		IncludeExts:      []string{".md", ".mdc"},
		SkipDirs:         []string{".git", "vendor", "workflows"},
		Rules: []Rule{
			{Glob: ".github/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".cursor/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".cursor/rules/*.mdc", Limit: RuleLimit(300)},
			{Glob: ".windsurf/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".claude/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".claude/agents/*.md", Limit: RuleLimit(300)},
			{Glob: ".kiro/steering/*.md", Limit: RuleLimit(300)},
			{Glob: ".continue/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".clinerules/*.md", Limit: RuleLimit(300)},
			{Glob: ".roo/rules/*.md", Limit: RuleLimit(300)},
			{Glob: ".aiassistant/rules/*.md", Limit: RuleLimit(300)},
		},
		// Byte limits: Antigravity truncates each rule file (AGENTS.md,
		// GEMINI.md, .agents/rules/*.md) over 24,000 bytes after include
		// expansion; Codex caps combined project instructions at 32 KiB, so
		// the stricter documented figure is used.
		MaxBytes: []Rule{
			{Glob: "AGENTS.md", Limit: RuleLimit(24000)},
			{Glob: "GEMINI.md", Limit: RuleLimit(24000)},
			{Glob: ".agents/rules/*.md", Limit: RuleLimit(24000)},
		},
		Files: map[string]int{
			".github/copilot-instructions.md": 50,
			".claude/CLAUDE.md":               200,
			"AGENTS.md":                       200,
			"GEMINI.md":                       200,
			".rules":                          200,
			// memory.md is read holistically as session context (not a scoped rule),
			// so it shares the 200-line limit used by AGENTS.md, CLAUDE.md, etc.
			// Claude Code empirically loads the first 200 lines of its MEMORY.md
			// at every session start, which establishes 200 as the practical ceiling.
			// Both naming conventions are covered: plain .md (default) and
			// .instructions.md (used when descriptive_names is enabled).
			// .github/instructions/ is also covered for repos where Copilot
			// init targeted that directory instead of rules/.
			".github/rules/memory.md":                     200,
			".github/rules/memory.instructions.md":        200,
			".github/instructions/memory.instructions.md": 200,
			".cursor/rules/memory.md":                     200,
			".cursor/rules/memory.mdc":                    200,
			".cursor/rules/memory.instructions.md":        200,
			".windsurf/rules/memory.md":                   200,
			".windsurf/rules/memory.instructions.md":      200,
			".claude/rules/memory.md":                     200,
			".claude/rules/memory.instructions.md":        200,
			".kiro/steering/memory.md":                    200,
			".kiro/steering/memory.instructions.md":       200,
			".continue/rules/memory.md":                   200,
			".continue/rules/memory.instructions.md":      200,
			".clinerules/memory.md":                       200,
			".clinerules/memory.instructions.md":          200,
			".roo/rules/memory.md":                        200,
			".roo/rules/memory.instructions.md":           200,
			".aiassistant/rules/memory.md":                200,
			".aiassistant/rules/memory.instructions.md":   200,
		},
	}
}

// ResolveLimit returns the effective line limit for a repo-relative path
// (forward slashes) given the config. Returns 0 when the file is exempt.
//
// Resolution priority:
//  1. Per-file override in Config.Files (exact match)
//  2. First matching glob in Config.Rules
//  3. Config.Default (falls back to the built-in default of 500 if zero)
//
// Glob matching uses [filepath.Match]. As a special case, a glob that ends
// with "/" is treated as a recursive directory prefix: it matches any file
// whose path starts with that prefix (e.g. "docs/" matches "docs/a.md" and
// "docs/sub/b.md"). This allows directory-scoped rules without requiring
// shell-style "**" patterns, which [filepath.Match] does not support.
func ResolveLimit(path string, cfg Config) int { //nolint:gocritic // hugeParam: stable public API
	// 1. Per-file override.
	if v, ok := cfg.Files[path]; ok {
		return v
	}

	// 2. First matching glob rule.
	for _, r := range cfg.Rules {
		if ruleGlobMatches(r.Glob, path) {
			if r.Limit == nil {
				break // nil limit -> fall through to config default
			}
			return *r.Limit
		}
	}

	// 3. Config-level default, then built-in default.
	if cfg.Default > 0 {
		return cfg.Default
	}
	return defaultLimit
}

// ResolveByteLimit returns the effective byte-size limit for a repo-relative
// path (forward slashes): the Limit of the first matching [Config.MaxBytes]
// rule, or 0 when no byte limit applies. Matching is identical to
// [ResolveLimit]'s glob handling.
func ResolveByteLimit(path string, cfg Config) int { //nolint:gocritic // hugeParam: stable public API
	for _, r := range cfg.MaxBytes {
		if ruleGlobMatches(r.Glob, path) {
			if r.Limit == nil {
				return 0
			}
			return *r.Limit
		}
	}
	return 0
}

// ruleGlobMatches reports whether a [Rule.Glob] pattern matches a
// repo-relative path (forward slashes). It is the single matching
// implementation shared by [ResolveLimit] and [ValidateConfig] so that
// validation warnings agree with resolution behavior on every platform:
// a trailing-slash glob matches as a recursive directory prefix, and all
// other globs are normalized through [filepath.FromSlash] before
// [filepath.Match] so separator semantics are consistent across
// operating systems.
func ruleGlobMatches(glob, path string) bool {
	if strings.HasSuffix(glob, "/") {
		// Trailing-slash glob: match any file under this directory prefix.
		return strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(glob))
	}
	ok, _ := filepath.Match(filepath.FromSlash(glob), filepath.FromSlash(path))
	return ok
}

// effectiveWarningThreshold returns the warning percentage threshold,
// falling back to the built-in default when cfg.WarningThreshold is zero.
func effectiveWarningThreshold(cfg Config) int { //nolint:gocritic // hugeParam: intentional value semantics
	if cfg.WarningThreshold > 0 {
		return int(cfg.WarningThreshold)
	}
	return int(defaultWarningThreshold)
}

// isSafeFileSegment reports whether s is a safe, cross-platform filename
// segment. Allowed characters: A–Z, a–z, 0–9, underscore, hyphen, and dot.
// Empty strings, ".", and ".." are rejected. The constraint prevents
// path-separator injection, path-traversal via "..", and characters reserved
// on Windows (e.g. *, ?, :, <, >, |, "\") from reaching the filesystem via
// JSON/YAML config values.
func isSafeFileSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') &&
			(r < '0' || r > '9') && r != '_' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
