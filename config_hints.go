package repogov

import (
	"encoding/json"
	"fmt"
	"strings"
)

// defaultAgentHints is the "_agent_hints" block written at the top of every
// generated repogov-config.json. Keys starting with "_" are ignored by
// [LoadConfig], so the block is pure guidance for AI agents that open the
// file. Each entry is one line an agent can act on: a situation and what to
// do about it, never an essay. Documentation lives in docs/, not here.
//
// The block is the same for every config: the "language" key sits in the
// same file, and its one-line hint is all an agent needs to honor it.
// The repository's own .github/repogov-config.json carries the same block
// (pinned by TestDefaultConfigJSON_AgentHints) so the dogfooded file and the
// generator cannot drift apart.
const defaultAgentHints = `  "_agent_hints": {
    "about": "Keys starting with _ guide AI agents; repogov ignores them.",
    "on_fail": "Over its limit: split the file or move detail to docs/. Raise a limit only if the user asks.",
    "on_warn": "Near its limit: trim before adding.",
    "on_pitfall": "The agent silently ignores that file: apply the FIX hint.",
    "on_settings": "An agent settings finding: report it to the user; never widen permissions.",
    "exempt": "Limit 0 under files, only for generated or append-only files; tell the user.",
    "never": [
      "Edit files outside of this repo.",
      "Read or paste secrets: .env, keys, certificates.",
      "Bypass security controls (signing, hooks, permissions) to finish a task."
    ],
    "verify": "Run 'repogov -root . all' and the tests; apply each FIX hint.",
    "keys": {
      "language": "Prose language (BCP 47, e.g. en-US): write docs, comments and commits in it.",
      "default": "Fallback line limit (500).",
      "warning_threshold": "Percent of a limit where WARN starts (85 or \"85%\").",
      "skip_dirs": "Directories never scanned.",
      "include_exts": "Extensions scanned; [] scans all.",
      "rules": "Line limits by glob; first match wins.",
      "max_bytes": "Byte limits by glob; some agents truncate large files.",
      "files": "Per-file line limits; 0 exempts."
    }
  },
`

// agentHints returns the "_agent_hints" member every generated config
// carries.
func agentHints() string { return defaultAgentHints }

// agentHintsYAML renders [agentHints] as a YAML "_agent_hints:" block (no
// trailing newline). It is derived from the JSON constant so the two
// formats cannot drift. Values are double-quoted scalars, which every YAML
// parser (and repogov's minimal one, which skips _-prefixed keys) reads as
// plain strings.
func agentHintsYAML() string {
	// agentHints is a `"_agent_hints": {...},` member; wrap to decode.
	body := strings.TrimSuffix(strings.TrimSpace(agentHints()), ",")
	dec := json.NewDecoder(strings.NewReader("{" + body + "}"))
	var b strings.Builder
	if _, err := dec.Token(); err != nil { // outer '{'
		return ""
	}
	if err := yamlWriteMember(dec, &b, 0); err != nil {
		return ""
	}
	return strings.TrimRight(b.String(), "\n")
}

// yamlWriteMember writes `key: value` for the next object member, where the
// value is a string, an array of strings, or an object of the same.
func yamlWriteMember(dec *json.Decoder, b *strings.Builder, depth int) error {
	keyTok, err := dec.Token()
	if err != nil {
		return err
	}
	pad := strings.Repeat("  ", depth)
	key, _ := keyTok.(string)
	return yamlWriteValue(dec, b, pad, key, depth)
}

func yamlWriteValue(dec *json.Decoder, b *strings.Builder, pad, key string, depth int) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch v := tok.(type) {
	// key is a member name of the defaultAgentHints constant ([a-z_]), never
	// config text, so it is written bare; every value is double-quoted.
	case string:
		fmt.Fprintf(b, "%s%s: %s\n", pad, key, yamlDoubleQuote(v))
	case json.Delim:
		fmt.Fprintf(b, "%s%s:\n", pad, key)
		switch v {
		case '[':
			for dec.More() {
				t, err := dec.Token()
				if err != nil {
					return err
				}
				s, _ := t.(string)
				fmt.Fprintf(b, "%s  - %s\n", pad, yamlDoubleQuote(s))
			}
		case '{':
			for dec.More() {
				if err := yamlWriteMember(dec, b, depth+1); err != nil {
					return err
				}
			}
		}
		if _, err := dec.Token(); err != nil { // closing ']' or '}'
			return err
		}
	}
	if depth == 0 {
		// Top-level wrapper member finished; consume the outer '}' lazily.
		return nil
	}
	return nil
}

// yamlDoubleQuote returns s as a YAML double-quoted scalar, escaping
// backslashes, double quotes and control characters so the value cannot end
// the scalar early or spill onto a new line.
func yamlDoubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02X`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
