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
	// Skip indicates the file is exempt (limit = 0) or does not match the extension filter.
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
	// the check result changes from PASS to WARN. Default: 80.
	WarningThreshold PercentInt `json:"warning_threshold"`

	// SkipDirs lists directory names to skip during directory walks.
	SkipDirs []string `json:"skip_dirs"`

	// Rules are glob-based limit rules evaluated in order. The first
	// matching rule wins.
	Rules []Rule `json:"rules"`

	// Files maps repo-relative paths (forward slashes) to per-file
	// limits. A limit of 0 exempts the file from checking.
	Files map[string]int `json:"files"`
}

// Rule maps a glob pattern to a line limit.
type Rule struct {
	// Glob is a filepath.Match pattern using forward slashes.
	Glob string `json:"glob"`

	// Limit is the maximum number of lines. Zero exempts matched files.
	Limit int `json:"limit"`
}

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

	// Status is the check outcome: Pass, Warn, Fail, or Skip.
	Status Status

	// Action is a remediation hint for WARN and FAIL outcomes.
	// Empty for PASS and SKIP. Designed to be consumed by AI agents,
	// LLMs, and MCP tools that need actionable context.
	Action string
}

// defaultLimit is the built-in fallback when Config.Default is zero.
const defaultLimit = 300

// defaultWarningThreshold is the built-in warning threshold percentage.
const defaultWarningThreshold PercentInt = 85

// DefaultConfig returns a Config with sensible defaults:
// Default=300, WarningThreshold=85, standard SkipDirs,
// .github/instructions/*.md at 300, and .github/copilot-instructions.md
// at 50 lines.
func DefaultConfig() Config {
	return Config{
		Default:          defaultLimit,
		WarningThreshold: defaultWarningThreshold,
		SkipDirs:         []string{".git", "vendor"},
		Rules: []Rule{
			{Glob: ".github/instructions/*.md", Limit: 300},
			{Glob: ".cursor/rules/*.md", Limit: 300},
			{Glob: ".cursor/rules/*.mdc", Limit: 300},
			{Glob: ".windsurf/rules/*.md", Limit: 300},
			{Glob: ".claude/rules/*.md", Limit: 300},
			{Glob: ".claude/agents/*.md", Limit: 300},
		},
		Files: map[string]int{
			".github/copilot-instructions.md": 50,
			"AGENTS.md":                       200,
		},
	}
}

// ResolveLimit returns the effective line limit for a repo-relative path
// (forward slashes) given the config. Returns 0 when the file is exempt.
//
// Resolution priority:
//  1. Per-file override in Config.Files (exact match)
//  2. First matching glob in Config.Rules
//  3. Config.Default (falls back to 300 if zero)
func ResolveLimit(path string, cfg Config) int {
	// 1. Per-file override.
	if v, ok := cfg.Files[path]; ok {
		return v
	}

	// 2. First matching glob rule.
	for _, r := range cfg.Rules {
		if ok, _ := filepath.Match(
			filepath.FromSlash(r.Glob),
			filepath.FromSlash(path),
		); ok {
			return r.Limit
		}
	}

	// 3. Config-level default, then built-in default.
	if cfg.Default > 0 {
		return cfg.Default
	}
	return defaultLimit
}

// effectiveWarningThreshold returns the warning percentage threshold,
// falling back to the built-in default when cfg.WarningThreshold is zero.
func effectiveWarningThreshold(cfg Config) int {
	if cfg.WarningThreshold > 0 {
		return int(cfg.WarningThreshold)
	}
	return int(defaultWarningThreshold)
}
