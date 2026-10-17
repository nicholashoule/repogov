package repogov

import (
	"encoding/json"
	"strings"
)

// defaultAgentHints is the "_agent_hints" block written at the top of every
// generated repogov-config.json. Keys starting with "_" are ignored by
// [LoadConfig], so the block is pure guidance for AI agents that open the
// file: what each finding means, what they must not do, and how to verify.
// The repository's own .github/repogov-config.json carries the same block
// (pinned by TestDefaultConfigJSON_AgentHints) so the dogfooded file and the
// generator cannot drift apart.
const defaultAgentHints = `  "_agent_hints": {
    "about": "Keys starting with _ are ignored by repogov and exist only to guide AI agents. Read before editing files or this config.",
    "on_limit_fail": "A FAIL means a file is over its line limit. Split or trim the file (move detail into docs/ and link to it). Do NOT raise a limit in this config just to make the check pass unless the user asks.",
    "on_limit_warn": "A WARN means the file is at or above the warning threshold. Prefer trimming before adding more content.",
    "adding_exemptions": "To exempt a file set its limit to 0 under files. Only do this for generated or append-only files (e.g. CHANGELOG.md), and tell the user.",
    "never_do": [
      "Never create, edit, or seed .claude/settings.json or .claude/settings.local.json; they belong to the user.",
      "Never open or paste the contents of .claude/.credentials.json, *.pem, *.key, *.crt, *.p12, *.pfx, *.jks or .env files.",
      "Never overwrite existing instruction or rule files; repogov init only creates missing files."
    ],
    "verify": "Run 'repogov -root . all' (or with -agent <name>) and the project's tests before finishing. Every non-PASS finding includes a '-- FIX:' hint; apply it.",
    "settings_audit": "For the claude agent, layout also audits .claude/settings*.json. Fix duplicate, redundant, or allow/deny-conflicting permission rules by editing the list the finding names, and report to the user rather than widening permissions.",
    "silent_failures": "Findings under 'agent pitfalls' mean an agent will silently ignore that file (no error from the agent). Apply the FIX hint; do not leave the file as is. A byte-limit FAIL means the agent may truncate the file even though the line count passes.",
    "keys": {
      "default": "Fallback line limit (0 falls back to 500).",
      "warning_threshold": "Percent of limit at which WARN begins; accepts 85 or \"85%\".",
      "skip_dirs": "Directory names never entered.",
      "include_exts": "Extensions scanned; [] scans everything.",
      "rules": "Glob rules, first match wins; per-file entries in files take precedence.",
      "max_bytes": "Byte-size limits by glob (first match wins), checked in addition to line limits; some agents truncate instruction files by size, e.g. 24000 bytes.",
    "max_bytes": "Byte-size limits by glob (first match wins), checked in addition to line limits; some agents truncate instruction files by size, e.g. 24000 bytes.",
    "files": "Exact repo-relative path (forward slashes) to limit; 0 exempts."
    }
  },
`

// agentHintsYAML renders [defaultAgentHints] as a YAML "_agent_hints:" block
// (no trailing newline). It is derived from the JSON constant so the two
// formats cannot drift. Values are double-quoted scalars, which every YAML
// parser (and repogov's minimal one, which skips unknown keys) reads as plain
// strings.
func agentHintsYAML() string {
	// defaultAgentHints is a `"_agent_hints": {...},` member; wrap to decode.
	body := strings.TrimSuffix(strings.TrimSpace(defaultAgentHints), ",")
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
	case string:
		b.WriteString(pad + key + ": " + yamlDoubleQuote(v) + "\n")
	case json.Delim:
		b.WriteString(pad + key + ":\n")
		switch v {
		case '[':
			for dec.More() {
				t, err := dec.Token()
				if err != nil {
					return err
				}
				s, _ := t.(string)
				b.WriteString(pad + "  - " + yamlDoubleQuote(s) + "\n")
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

// yamlDoubleQuote returns s as a YAML double-quoted scalar.
func yamlDoubleQuote(s string) string {
	r := strings.NewReplacer(`\`, `\`, `"`, `\"`)
	return `"` + r.Replace(s) + `"`
}
