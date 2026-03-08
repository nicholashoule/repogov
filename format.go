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
		if r.Limit > 0 {
			pct := 100 * r.Lines / r.Limit
			if r.Action != "" {
				fmt.Fprintf(&b, "  [%s] %s (%d / %d, %d%%) -- %s\n",
					r.Status, r.Path, r.Lines, r.Limit, pct, r.Action)
			} else {
				fmt.Fprintf(&b, "  [%s] %s (%d / %d, %d%%)\n",
					r.Status, r.Path, r.Lines, r.Limit, pct)
			}
		} else {
			fmt.Fprintf(&b, "  [%s] %s (%d / %d)\n",
				r.Status, r.Path, r.Lines, r.Limit)
		}
	}
	fmt.Fprintf(&b, "\nLimits: %d files | %d pass | %d warn | %d fail | %d skip\n\n",
		len(results), pass, warn, fail, skip)
	return b.String()
}

// LayoutPassed returns true when every [LayoutResult] has status [Pass],
// [Info], or [Skip].
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
		fmt.Fprintf(&b, "  [%s] %s -- %s\n", r.Status, r.Path, r.Message)
	}
	fmt.Fprintf(&b, "\nLayout: %d checks | %d pass | %d warn | %d fail | %d info\n\n",
		len(results), pass, warn, fail, info)
	return b.String()
}
