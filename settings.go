package repogov

// settings.go provides a read-only audit of Claude Code settings files
// (.claude/settings.json and .claude/settings.local.json).
//
// INVARIANT: repogov never creates, modifies, or seeds these files. They are
// user and machine configuration owned by Claude Code; scaffolding lists them
// only as Optional entries in [DefaultClaudeLayout], and init creates only
// Required placeholders. This file opens them strictly for reading.
//
// The audit reports on the permissions lists, which tend to grow organically
// as an agent accumulates allow rules: exact duplicates, rules made redundant
// by a broader ":*" prefix rule, and allow entries that a deny rule
// overrides. Every non-PASS result carries an actionable "-- FIX:" hint so
// the output can gate and guide an AI agent directly.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// claudeSettingsFiles lists the settings filenames audited under .claude/,
// in report order. This is a strict allowlist: the audit opens these two
// files and nothing else. In particular, credential files such as
// .claude/.credentials.json are NEVER opened or parsed -- see
// credentialFileResults, which reports their presence by name only.
var claudeSettingsFiles = []string{"settings.json", "settings.local.json"}

// claudeCredentialFiles lists credential filenames under .claude/ whose
// presence is reported by stat only. Their content is never read.
var claudeCredentialFiles = []string{".credentials.json"}

// claudeSettings models the subset of a Claude Code settings file that the
// audit inspects. Unknown fields are ignored.
type claudeSettings struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
		Ask   []string `json:"ask"`
	} `json:"permissions"`
}

// CheckClaudeSettings audits the Claude Code settings files under
// root/.claude and returns advisory [LayoutResult] entries. Files that do
// not exist produce no results. The audit is strictly read-only -- repogov
// never creates or modifies settings files.
//
// Per existing file it reports:
//   - Info: a summary line with the line count and permission-rule counts.
//   - Fail: the file is not valid JSON (Claude Code cannot load it).
//   - Warn: exact duplicate rules within a permissions list.
//   - Warn: rules made redundant by a broader ":*" prefix rule in the
//     same list (e.g. "Bash(go test:*)" alongside "Bash(go:*)").
//   - Warn: allow entries that also appear in deny (deny takes precedence,
//     so the allow entry is ineffective).
//
// Messages follow the same actionable "-- FIX:" convention as layout
// checks, so results feed the same text, JSON, and exit-code gates.
//
// Credential files (e.g. .claude/.credentials.json) are never opened or
// parsed. Their presence is detected by filename alone and reported with
// remediation guidance; the content is not read under any circumstance.
func CheckClaudeSettings(root string) []LayoutResult {
	var results []LayoutResult
	for _, name := range claudeSettingsFiles {
		absPath := filepath.Join(root, ".claude", name)
		displayPath := ".claude/" + name
		if _, err := os.Stat(absPath); err != nil {
			continue
		}
		results = append(results, auditSettingsFile(absPath, displayPath)...)
	}
	results = append(results, credentialFileResults(root)...)
	return results
}

// credentialFileResults reports credential files present under .claude/ by
// filename only -- the content is never opened or parsed. A credential file
// inside a repository tree is a leak risk the moment it is committed, so
// the finding carries explicit remediation guidance.
func credentialFileResults(root string) []LayoutResult {
	var results []LayoutResult
	for _, name := range claudeCredentialFiles {
		if _, err := os.Stat(filepath.Join(root, ".claude", name)); err != nil {
			continue
		}
		results = append(results, LayoutResult{
			Path:   ".claude/" + name,
			Status: Warn,
			Message: "credential file present (content not read by repogov) -- " +
				"FIX: ensure it is listed in .gitignore and never committed; rotate the credentials if it ever was",
		})
	}
	return results
}

// auditSettingsFile audits a single settings file and returns its results.
func auditSettingsFile(absPath, displayPath string) []LayoutResult {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return []LayoutResult{{
			Path:    displayPath,
			Status:  Warn,
			Message: fmt.Sprintf("unable to read settings file: %v", err),
		}}
	}

	var cfg claudeSettings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return []LayoutResult{{
			Path:    displayPath,
			Status:  Fail,
			Message: fmt.Sprintf("not valid JSON (%v) -- FIX: repair the syntax; Claude Code cannot load a malformed settings file", err),
		}}
	}

	lines, err := CountLines(absPath)
	if err != nil {
		lines = 0
	}

	results := []LayoutResult{{
		Path:   displayPath,
		Status: Info,
		Message: fmt.Sprintf("%d lines; permissions: %d allow, %d deny, %d ask",
			lines, len(cfg.Permissions.Allow), len(cfg.Permissions.Deny), len(cfg.Permissions.Ask)),
	}}

	lists := []struct {
		label string
		rules []string
	}{
		{"allow", cfg.Permissions.Allow},
		{"deny", cfg.Permissions.Deny},
		{"ask", cfg.Permissions.Ask},
	}
	for _, l := range lists {
		results = append(results, duplicateRuleResults(displayPath, l.label, l.rules)...)
		results = append(results, redundantRuleResults(displayPath, l.label, l.rules)...)
		results = append(results, consolidationResults(displayPath, l.label, l.rules)...)
	}

	results = append(results, allowDenyConflictResults(displayPath, cfg.Permissions.Allow, cfg.Permissions.Deny)...)
	return results
}

// duplicateRuleResults reports exact duplicate rules within one permissions
// list. Duplicates accumulate as agents append rules without scanning the
// existing list; each duplicate is pure noise.
func duplicateRuleResults(displayPath, label string, rules []string) []LayoutResult {
	seen := make(map[string]int, len(rules))
	for _, r := range rules {
		seen[r]++
	}
	var results []LayoutResult
	reported := make(map[string]bool)
	for _, r := range rules {
		if seen[r] > 1 && !reported[r] {
			reported[r] = true
			results = append(results, LayoutResult{
				Path:   displayPath,
				Status: Warn,
				Message: fmt.Sprintf("permissions.%s: duplicate rule %q (%d occurrences) -- FIX: keep one and delete the rest",
					label, r, seen[r]),
			})
		}
	}
	return results
}

// splitRule splits a permission rule into its tool name and inner pattern:
// "Bash(go test:*)" -> ("Bash", "go test:*"). A rule without parentheses is
// a bare tool name with an empty inner pattern.
func splitRule(rule string) (tool, inner string) {
	if i := strings.IndexByte(rule, '('); i >= 0 && strings.HasSuffix(rule, ")") {
		return rule[:i], rule[i+1 : len(rule)-1]
	}
	return rule, ""
}

// redundantRuleResults reports rules that a broader ":*" prefix rule in the
// same list already covers, e.g. "Bash(go test:*)" alongside "Bash(go:*)".
// The prefix comparison is boundary-aware (the narrower inner pattern must
// equal the prefix or continue with a space or colon), so "Bash(go:*)" is
// never claimed to cover "Bash(gofmt:*)".
func redundantRuleResults(displayPath, label string, rules []string) []LayoutResult {
	var results []LayoutResult
	reported := make(map[string]bool)
	for _, broad := range rules {
		tool, inner := splitRule(broad)
		if !strings.HasSuffix(inner, ":*") {
			continue
		}
		prefix := strings.TrimSuffix(inner, ":*")
		for _, narrow := range rules {
			if narrow == broad || reported[narrow] {
				continue
			}
			nTool, nInner := splitRule(narrow)
			if nTool != tool {
				continue
			}
			if nInner == prefix ||
				strings.HasPrefix(nInner, prefix+" ") ||
				strings.HasPrefix(nInner, prefix+":") {
				reported[narrow] = true
				results = append(results, LayoutResult{
					Path:   displayPath,
					Status: Warn,
					Message: fmt.Sprintf("permissions.%s: rule %q appears redundant -- broader rule %q already covers it -- FIX: remove the narrower rule",
						label, narrow, broad),
				})
			}
		}
	}
	return results
}

// consolidationMinGroup is the number of same-prefix rules in one list at
// which a consolidation suggestion is emitted. Below this, individual rules
// are cheap; above it, the list is growing one near-identical entry at a
// time and a single prefix rule would stop the churn.
const consolidationMinGroup = 5

// commandPrefix returns the grouping prefix for an inner rule pattern: the
// first two whitespace-separated tokens when present (e.g. "gofmt -w"),
// otherwise the single token. Patterns that already end in ":*" return ""
// -- they are wildcards, not candidates for consolidation.
func commandPrefix(inner string) string {
	if inner == "" || strings.HasSuffix(inner, ":*") {
		return ""
	}
	fields := strings.Fields(inner)
	switch {
	case len(fields) >= 2:
		return fields[0] + " " + fields[1]
	case len(fields) == 1:
		return fields[0]
	default:
		return ""
	}
}

// consolidationResults suggests replacing many near-identical literal rules
// with one prefix wildcard. Permission lists grow organically -- an agent
// records "Bash(gofmt -w a.go)", then "Bash(gofmt -w b.go)", and so on --
// and past [consolidationMinGroup] same-prefix entries a single
// "Tool(prefix:*)" rule is easier to review and stops the churn. The
// suggestion is Info, not Warn: widening a permission is a human decision.
func consolidationResults(displayPath, label string, rules []string) []LayoutResult {
	type group struct {
		tool   string
		prefix string
		count  int
	}
	groups := make(map[string]*group)
	var order []string
	for _, r := range rules {
		tool, inner := splitRule(r)
		prefix := commandPrefix(inner)
		if prefix == "" {
			continue
		}
		key := tool + "\x00" + prefix
		if g, ok := groups[key]; ok {
			g.count++
		} else {
			groups[key] = &group{tool: tool, prefix: prefix, count: 1}
			order = append(order, key)
		}
	}
	var results []LayoutResult
	for _, key := range order {
		g := groups[key]
		if g.count < consolidationMinGroup {
			continue
		}
		results = append(results, LayoutResult{
			Path:   displayPath,
			Status: Info,
			Message: fmt.Sprintf("permissions.%s: %d rules share the prefix %q -- FIX: if broad access is acceptable, consolidate into %q and delete the individual entries",
				label, g.count, g.prefix, g.tool+"("+g.prefix+":*)"),
		})
	}
	return results
}

// allowDenyConflictResults reports allow rules that also appear verbatim in
// deny. Deny takes precedence in Claude Code, so the allow entry is
// ineffective and usually indicates a stale grant.
func allowDenyConflictResults(displayPath string, allow, deny []string) []LayoutResult {
	denySet := make(map[string]bool, len(deny))
	for _, r := range deny {
		denySet[r] = true
	}
	var results []LayoutResult
	reported := make(map[string]bool)
	for _, r := range allow {
		if denySet[r] && !reported[r] {
			reported[r] = true
			results = append(results, LayoutResult{
				Path:   displayPath,
				Status: Warn,
				Message: fmt.Sprintf("permissions.allow: rule %q is also in permissions.deny; deny takes precedence -- FIX: remove it from one list",
					r),
			})
		}
	}
	return results
}
