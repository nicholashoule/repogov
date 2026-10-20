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
// string-to-int map (files). Keys may be quoted ("default": 300). Keys
// starting with "_" are metadata (e.g. _agent_hints) and are skipped
// together with their indented children.
//
// Anything outside that subset is an error rather than being silently
// ignored, so a typo cannot quietly fall back to defaults: unknown keys,
// tab indentation, and flow syntax (inline [a, b] or {k: v}) other than an
// empty [] (or {} for files) all fail with the offending line number.
func unmarshalYAML(data []byte, cfg *Config) error {
	type section int
	const (
		sNone section = iota
		sMeta         // inside a _-prefixed metadata key: children skipped
		sSkipDirs
		sIncludeExts
		sRules
		sFiles
		sInitInclude
		sInitExclude
		sMaxBytes
	)

	var (
		sec     section
		curRule *Rule
		// ruleDst is the rule list the section being parsed appends to
		// (rules or max_bytes; both are lists of glob/limit objects).
		ruleDst *[]Rule
	)

	flushRule := func() {
		if curRule != nil && ruleDst != nil {
			*ruleDst = append(*ruleDst, *curRule)
		}
		curRule = nil
	}

	isList := func(s section) bool {
		switch s {
		case sSkipDirs, sIncludeExts, sRules, sMaxBytes, sInitInclude, sInitExclude:
			return true
		}
		return false
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
		if strings.ContainsRune(line[:len(line)-len(strings.TrimLeft(line, " \t"))], '\t') {
			return fmt.Errorf("line %d: tab indentation is not supported; indent with spaces", i+1)
		}

		// An indentless block sequence ("skip_dirs:\n- vendor") is valid
		// YAML: treat a top-level "- " item as part of the open list section.
		indentlessItem := indent == 0 && isList(sec) && (strings.HasPrefix(trimmed, "- ") || trimmed == "-")

		if indent == 0 && !indentlessItem {
			// Top-level key.
			flushRule()
			key, val, ok := yamlSplitKV(trimmed)
			if !ok {
				return fmt.Errorf("line %d: expected \"key: value\" at top level, got %q", i+1, trimmed)
			}
			key = yamlUnquote(key) // "default": 300 is the same key as default: 300
			if strings.HasPrefix(key, "_") {
				sec = sMeta
				continue
			}
			// Flow collections are not supported, except an explicit empty
			// list ([]) for list keys or empty map ({}) for files.
			if strings.HasPrefix(val, "[") || strings.HasPrefix(val, "{") {
				switch {
				case val == "[]" && yamlIsListKey(key):
				case val == "{}" && key == "files":
				default:
					return fmt.Errorf("line %d: inline (flow) value %q for %q is not supported; use a block list or map, or [] for an empty list", i+1, val, key)
				}
			} else if val != "" && (yamlIsListKey(key) || key == "files") {
				return fmt.Errorf("line %d: %q takes a block list or map on the following lines, not the value %q", i+1, key, val)
			}
			sec = sNone
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
			case "language":
				cfg.Language = yamlUnquote(val)
			case "skip_dirs":
				sec = sSkipDirs
				cfg.SkipDirs = nil
				if val == "[]" {
					cfg.SkipDirs = []string{} // explicit empty: skip nothing
				}
			case "include_exts":
				sec = sIncludeExts
				cfg.IncludeExts = []string{} // explicit empty means scan all
			case "rules":
				sec = sRules
				cfg.Rules = nil
				if val == "[]" {
					cfg.Rules = []Rule{} // explicit empty: no glob rules
				}
				ruleDst = &cfg.Rules
			case "max_bytes":
				sec = sMaxBytes
				cfg.MaxBytes = []Rule{}
				ruleDst = &cfg.MaxBytes
			case "files":
				sec = sFiles
				cfg.Files = make(map[string]int)
			case "init_always_create":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("line %d: invalid init_always_create: %w", i+1, err)
				}
				cfg.InitAlwaysCreate = b
			case "descriptive_names":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("line %d: invalid descriptive_names: %w", i+1, err)
				}
				cfg.DescriptiveNames = b
			case "skip_frontmatter":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("line %d: invalid skip_frontmatter: %w", i+1, err)
				}
				cfg.SkipFrontmatter = b
			case "init_include_files":
				sec = sInitInclude
				cfg.InitIncludeFiles = []string{}
			case "init_exclude_files":
				sec = sInitExclude
				cfg.InitExcludeFiles = []string{}
			default:
				return fmt.Errorf("line %d: unknown key %q; check the spelling against the documented config keys (prefix metadata keys with _)", i+1, key)
			}
			continue
		}

		// Indented line (or indentless list item) -- part of current section.
		item, isItem := yamlListItem(trimmed)
		switch sec {
		case sMeta:
			// Children of a _-prefixed metadata key are not interpreted.

		case sSkipDirs, sIncludeExts, sInitInclude, sInitExclude:
			if !isItem {
				return fmt.Errorf("line %d: expected a \"- value\" list item, got %q", i+1, trimmed)
			}
			if strings.HasPrefix(item, "[") || strings.HasPrefix(item, "{") {
				return fmt.Errorf("line %d: inline (flow) list item %q is not supported; use one \"- value\" per line", i+1, item)
			}
			val := yamlUnquote(item)
			switch sec {
			case sSkipDirs:
				cfg.SkipDirs = append(cfg.SkipDirs, val)
			case sIncludeExts:
				cfg.IncludeExts = append(cfg.IncludeExts, val)
			case sInitInclude:
				cfg.InitIncludeFiles = append(cfg.InitIncludeFiles, val)
			case sInitExclude:
				cfg.InitExcludeFiles = append(cfg.InitExcludeFiles, val)
			}

		case sRules, sMaxBytes:
			rest := trimmed
			if isItem {
				flushRule()
				curRule = &Rule{}
				rest = item
			} else if curRule == nil {
				return fmt.Errorf("line %d: expected a \"- glob: ...\" list item, got %q", i+1, trimmed)
			}
			key, val, ok := yamlSplitKV(rest)
			if !ok {
				return fmt.Errorf("line %d: expected \"glob: ...\" or \"limit: ...\", got %q", i+1, rest)
			}
			if err := yamlSetRuleField(curRule, yamlUnquote(key), val); err != nil {
				return fmt.Errorf("line %d: %w", i+1, err)
			}

		case sFiles:
			if cfg.Files == nil {
				cfg.Files = make(map[string]int)
			}
			key, val, ok := yamlSplitKV(trimmed)
			if !ok || isItem {
				return fmt.Errorf("line %d: expected \"path: limit\" under files, got %q", i+1, trimmed)
			}
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("line %d: invalid file limit for %q: %w", i+1, key, err)
			}
			cfg.Files[yamlUnquote(key)] = n

		default:
			return fmt.Errorf("line %d: unexpected indented line %q; only list and map keys take indented values", i+1, trimmed)
		}
	}

	flushRule()
	return nil
}

// yamlIsListKey reports whether key is a top-level config key whose value
// is a block list.
func yamlIsListKey(key string) bool {
	switch key {
	case "skip_dirs", "include_exts", "rules", "max_bytes", "init_include_files", "init_exclude_files":
		return true
	}
	return false
}

// yamlListItem returns the text after a block-sequence "- " marker.
func yamlListItem(trimmed string) (string, bool) {
	if strings.HasPrefix(trimmed, "- ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	return "", false
}

// marshalYAML produces a minimal YAML representation of cfg.
// Map keys are sorted for deterministic output.
func marshalYAML(cfg Config) ([]byte, error) { //nolint:gocritic // hugeParam: intentional value semantics
	var b bytes.Buffer
	// The prose language always comes first, written out even when it is
	// the default (an empty value means en-US).
	fmt.Fprintf(&b, "language: %s\n", yamlQuote(effectiveLanguage(cfg.Language)))
	fmt.Fprintf(&b, "default: %d\n", cfg.Default)
	fmt.Fprintf(&b, "warning_threshold: %d%%\n", int(cfg.WarningThreshold))

	if cfg.SkipDirs != nil && len(cfg.SkipDirs) == 0 {
		fmt.Fprintln(&b, "skip_dirs: []")
	} else if len(cfg.SkipDirs) > 0 {
		fmt.Fprintln(&b, "skip_dirs:")
		for _, d := range cfg.SkipDirs {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(d))
		}
	}

	if cfg.IncludeExts != nil {
		fmt.Fprintln(&b, "include_exts:")
		for _, e := range cfg.IncludeExts {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(e))
		}
	}

	if cfg.Rules != nil && len(cfg.Rules) == 0 {
		fmt.Fprintln(&b, "rules: []") // explicit empty: no glob rules
	} else if len(cfg.Rules) > 0 {
		fmt.Fprintln(&b, "rules:")
		for _, r := range cfg.Rules {
			fmt.Fprintf(&b, "  - glob: %s\n", yamlQuote(r.Glob))
			if r.Limit != nil {
				fmt.Fprintf(&b, "    limit: %d\n", *r.Limit)
			}
		}
	}

	if len(cfg.MaxBytes) > 0 {
		fmt.Fprintln(&b, "max_bytes:")
		for _, r := range cfg.MaxBytes {
			fmt.Fprintf(&b, "  - glob: %s\n", yamlQuote(r.Glob))
			if r.Limit != nil {
				fmt.Fprintf(&b, "    limit: %d\n", *r.Limit)
			}
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

	if cfg.InitAlwaysCreate {
		fmt.Fprintln(&b, "init_always_create: true")
	}
	if cfg.DescriptiveNames {
		fmt.Fprintln(&b, "descriptive_names: true")
	}
	if cfg.SkipFrontmatter {
		fmt.Fprintln(&b, "skip_frontmatter: true")
	}
	if len(cfg.InitIncludeFiles) > 0 {
		fmt.Fprintln(&b, "init_include_files:")
		for _, f := range cfg.InitIncludeFiles {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(f))
		}
	}
	if len(cfg.InitExcludeFiles) > 0 {
		fmt.Fprintln(&b, "init_exclude_files:")
		for _, f := range cfg.InitExcludeFiles {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(f))
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

// yamlQuote returns s quoted if it contains YAML-special characters. The
// empty string is quoted ("") so that a list item or value holding it
// round-trips instead of reading back as a bare "-" or a missing value.
func yamlQuote(s string) string {
	if s == "" || strings.ContainsAny(s, ":{}[]#&*!|>'\"%@`,?") ||
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
		r.Limit = &n
	default:
		return fmt.Errorf("unknown rule key %q; rules take only glob and limit", key)
	}
	return nil
}
