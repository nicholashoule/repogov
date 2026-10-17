package repogov

import (
	"fmt"
	"strings"
)

// Passed returns true when every [Result] has status [Pass], [Warn],
// or [Skip]. A single [Fail] makes the overall check fail.
func Passed(results []Result) bool {
	for _, r := range results {
		if r.Status == Fail {
			return false
		}
	}
	return true
}

// Summary returns a human-readable multi-line summary of the results.
// Each line shows the status, path, line count vs. limit, percentage,
// and an actionable hint for WARN/FAIL outcomes. The format is designed
// to be consumed by AI agents, LLMs, and MCP tools.
func Summary(results []Result) string {
	if len(results) == 0 {
		return "No files checked."
	}
	var b strings.Builder
	pass, warn, fail, skip := 0, 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case Pass:
			pass++
		case Warn:
			warn++
		case Fail:
			fail++
		case Skip:
			skip++
		}
		switch {
		case r.Limit > 0:
			// Recompute the percentage from Lines/Limit rather than trusting
			// r.Pct: callers may construct Result literals by hand (see
			// ExampleSummary) without populating Pct, and the summary must
			// still render correctly for them.
			pct := 100 * r.Lines / r.Limit
			if r.Action != "" {
				fmt.Fprintf(&b, "  [%s] %s (%d / %d, %d%%) -- %s\n",
					r.Status, cleanText(r.Path), r.Lines, r.Limit, pct, cleanText(r.Action))
			} else {
				fmt.Fprintf(&b, "  [%s] %s (%d / %d, %d%%)\n",
					r.Status, cleanText(r.Path), r.Lines, r.Limit, pct)
			}
		case r.Action != "":
			// Line-exempt files can still fail a byte limit; show why.
			fmt.Fprintf(&b, "  [%s] %s (%d / %d) -- %s\n",
				r.Status, cleanText(r.Path), r.Lines, r.Limit, cleanText(r.Action))
		default:
			fmt.Fprintf(&b, "  [%s] %s (%d / %d)\n",
				r.Status, cleanText(r.Path), r.Lines, r.Limit)
		}
	}
	fmt.Fprintf(&b, "\nLimits: %d files | %d pass | %d warn | %d fail | %d skip\n\n",
		len(results), pass, warn, fail, skip)
	return b.String()
}

// LayoutPassed returns true when no [LayoutResult] has status [Fail].
// Warnings (e.g. unexpected files) and informational results do not fail
// the layout check.
func LayoutPassed(results []LayoutResult) bool {
	for _, r := range results {
		if r.Status == Fail {
			return false
		}
	}
	return true
}

// LayoutSummary returns a human-readable multi-line summary of layout
// check results. The format includes actionable messages for FAIL and
// WARN outcomes, designed for AI agents, LLMs, and MCP tools.
func LayoutSummary(results []LayoutResult) string {
	if len(results) == 0 {
		return "No layout checks performed."
	}
	var b strings.Builder
	pass, warn, fail, info := 0, 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case Pass:
			pass++
		case Warn:
			warn++
		case Fail:
			fail++
		case Info:
			info++
		}
		fmt.Fprintf(&b, "  [%s] %s -- %s\n", r.Status, cleanText(r.Path), cleanText(r.Message))
	}
	fmt.Fprintf(&b, "\nLayout: %d checks | %d pass | %d warn | %d fail | %d info\n\n",
		len(results), pass, warn, fail, info)
	return b.String()
}

// cleanText makes untrusted text (file names and values read from files)
// safe to print to a terminal or CI log. Control characters (including ESC
// and newline, which could forge extra log lines or GitHub Actions workflow
// commands such as "::error::") and Unicode bidirectional overrides
// ("Trojan Source") are replaced with '?'. JSON output is unaffected: the
// encoder already escapes these.
func cleanText(s string) string {
	clean := true
	for _, r := range s {
		if isUnsafeRune(r) {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isUnsafeRune(r) {
			return '?'
		}
		return r
	}, s)
}

func isUnsafeRune(r rune) bool {
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) ||
		(r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069)
}
