package repogov

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// isYAML reports whether the file path has a .yaml or .yml extension.
func isYAML(path string) bool {
	p := strings.ToLower(path)
	return strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml")
}

// unmarshalYAML parses a minimal YAML configuration into cfg.
//
// Supported schema: top-level int scalars (default, warning_threshold), a string
// list (skip_dirs), a list of objects (rules with glob/limit), and a
// string-to-int map (files). Unknown keys (including _-prefixed metadata)
// are silently ignored. Flow syntax (inline [] and {}) is not supported;
// use block style.
func unmarshalYAML(data []byte, cfg *Config) error {
	type section int
	const (
		sNone section = iota
		sSkipDirs
		sRules
		sFiles
	)

	var (
		sec     section
		curRule *Rule
	)

	flushRule := func() {
		if curRule != nil {
			cfg.Rules = append(cfg.Rules, *curRule)
			curRule = nil
		}
	}

	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	for i, raw := range lines {
		line := yamlStripComment(raw)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "---" || trimmed == "..." {
			continue
		}

		indent := yamlCountIndent(line)

		if indent == 0 {
			// Top-level key.
			flushRule()
			key, val, ok := yamlSplitKV(trimmed)
			if !ok {
				continue
			}
			switch key {
			case "default":
				n, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("line %d: invalid default: %w", i+1, err)
				}
				cfg.Default = n
			case "warning_threshold":
				val = strings.TrimSuffix(strings.TrimSpace(val), "%")
				n, err := strconv.Atoi(strings.TrimSpace(val))
				if err != nil {
					return fmt.Errorf("line %d: invalid warning_threshold: %w", i+1, err)
				}
				cfg.WarningThreshold = PercentInt(n)
			case "skip_dirs":
				sec = sSkipDirs
				cfg.SkipDirs = nil
			case "rules":
				sec = sRules
				cfg.Rules = nil
			case "files":
				sec = sFiles
				cfg.Files = make(map[string]int)
			default:
				sec = sNone
			}
			continue
		}

		// Indented line — part of current section.
		switch sec {
		case sSkipDirs:
			if strings.HasPrefix(trimmed, "- ") {
				val := yamlUnquote(strings.TrimSpace(trimmed[2:]))
				cfg.SkipDirs = append(cfg.SkipDirs, val)
			}

		case sRules:
			if strings.HasPrefix(trimmed, "- ") {
				flushRule()
				curRule = &Rule{}
				rest := strings.TrimSpace(trimmed[2:])
				key, val, ok := yamlSplitKV(rest)
				if ok {
					if err := yamlSetRuleField(curRule, key, val); err != nil {
						return fmt.Errorf("line %d: %w", i+1, err)
					}
				}
			} else if curRule != nil {
				key, val, ok := yamlSplitKV(trimmed)
				if ok {
					if err := yamlSetRuleField(curRule, key, val); err != nil {
						return fmt.Errorf("line %d: %w", i+1, err)
					}
				}
			}

		case sFiles:
			if cfg.Files == nil {
				cfg.Files = make(map[string]int)
			}
			key, val, ok := yamlSplitKV(trimmed)
			if ok {
				n, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("line %d: invalid file limit for %q: %w", i+1, key, err)
				}
				cfg.Files[yamlUnquote(key)] = n
			}
		}
	}

	flushRule()
	return nil
}

// marshalYAML produces a minimal YAML representation of cfg.
// Map keys are sorted for deterministic output.
func marshalYAML(cfg Config) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "default: %d\n", cfg.Default)
	fmt.Fprintf(&b, "warning_threshold: %d%%\n", int(cfg.WarningThreshold))

	if len(cfg.SkipDirs) > 0 {
		fmt.Fprintln(&b, "skip_dirs:")
		for _, d := range cfg.SkipDirs {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(d))
		}
	}

	if len(cfg.Rules) > 0 {
		fmt.Fprintln(&b, "rules:")
		for _, r := range cfg.Rules {
			fmt.Fprintf(&b, "  - glob: %s\n", yamlQuote(r.Glob))
			fmt.Fprintf(&b, "    limit: %d\n", r.Limit)
		}
	}

	if len(cfg.Files) > 0 {
		fmt.Fprintln(&b, "files:")
		keys := make([]string, 0, len(cfg.Files))
		for k := range cfg.Files {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s: %d\n", yamlQuote(k), cfg.Files[k])
		}
	}

	return b.Bytes(), nil
}

// yamlSplitKV splits "key: value" into (key, value, true).
// Handles "key:" with no value for section headers.
func yamlSplitKV(s string) (string, string, bool) {
	idx := strings.Index(s, ": ")
	if idx >= 0 {
		return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+2:]), true
	}
	if strings.HasSuffix(s, ":") {
		return s[:len(s)-1], "", true
	}
	return "", "", false
}

// yamlStripComment removes a trailing # comment from a YAML line,
// preserving # characters inside quoted strings.
func yamlStripComment(line string) string {
	inSingle, inDouble := false, false
	for i, ch := range line {
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
				return line[:i]
			}
		}
	}
	return line
}

// yamlCountIndent returns the number of leading spaces.
func yamlCountIndent(s string) int {
	for i, ch := range s {
		if ch != ' ' {
			return i
		}
	}
	return len(s)
}

// yamlUnquote removes surrounding single or double quotes from s.
func yamlUnquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// yamlQuote returns s quoted if it contains YAML-special characters.
func yamlQuote(s string) string {
	if strings.ContainsAny(s, ":{}[]#&*!|>'\"%@`,?") ||
		strings.ContainsAny(s, " \t") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}

// yamlSetRuleField sets a field on a Rule by key name.
func yamlSetRuleField(r *Rule, key, val string) error {
	switch key {
	case "glob":
		r.Glob = yamlUnquote(val)
	case "limit":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid limit %q: %w", val, err)
		}
		r.Limit = n
	}
	return nil
}
