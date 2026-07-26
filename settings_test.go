package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

// TestCheckClaudeSettings_MissingFilesProduceNoResults verifies the audit is
// silent when no settings files exist -- absence is not a finding.
func TestCheckClaudeSettings_MissingFilesProduceNoResults(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/CLAUDE.md": "# c\n",
	})
	if got := repogov.CheckClaudeSettings(root); len(got) != 0 {
		t.Errorf("expected no results without settings files, got %+v", got)
	}
}

// TestCheckClaudeSettings_SummaryAndClean verifies a clean settings file
// yields exactly one Info summary with line and rule counts.
func TestCheckClaudeSettings_SummaryAndClean(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/settings.json": `{
  "permissions": {
    "allow": ["Bash(go test:*)", "Read(docs/**)"],
    "deny": ["Bash(rm:*)"]
  }
}
`,
	})
	results := repogov.CheckClaudeSettings(root)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for clean settings, got %+v", results)
	}
	r := results[0]
	if r.Status != repogov.Info || r.Path != ".claude/settings.json" {
		t.Errorf("unexpected summary result: %+v", r)
	}
	if !strings.Contains(r.Message, "2 allow, 1 deny, 0 ask") {
		t.Errorf("summary should count rules, got %q", r.Message)
	}
}

// TestCheckClaudeSettings_Findings verifies the three advisory detections:
// exact duplicates, narrower rules shadowed by a broader ":*" rule, and
// allow rules overridden by deny. All are Warn (advisory), never Fail.
func TestCheckClaudeSettings_Findings(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/settings.json": `{
  "permissions": {
    "allow": [
      "Bash(go test:*)",
      "Bash(go test:*)",
      "Bash(go:*)",
      "Bash(gofmt:*)",
      "Bash(git push:*)"
    ],
    "deny": ["Bash(git push:*)"]
  }
}
`,
	})
	results := repogov.CheckClaudeSettings(root)

	var dup, redundant, conflict, gofmtFlagged int
	for _, r := range results {
		if r.Status == repogov.Fail {
			t.Errorf("advisory audit must not Fail on valid JSON: %+v", r)
		}
		switch {
		case strings.Contains(r.Message, "duplicate rule"):
			dup++
		case strings.Contains(r.Message, "appears redundant"):
			redundant++
			if strings.Contains(r.Message, "gofmt") {
				gofmtFlagged++
			}
		case strings.Contains(r.Message, "deny takes precedence"):
			conflict++
		}
	}
	if dup != 1 {
		t.Errorf("duplicate findings = %d, want 1", dup)
	}
	// "Bash(go test:*)" is covered by "Bash(go:*)"; boundary-aware matching
	// must NOT flag "Bash(gofmt:*)".
	if redundant == 0 {
		t.Error("expected a redundant-rule finding for Bash(go test:*) under Bash(go:*)")
	}
	if gofmtFlagged != 0 {
		t.Error("Bash(gofmt:*) must not be treated as covered by Bash(go:*)")
	}
	if conflict != 1 {
		t.Errorf("allow/deny conflict findings = %d, want 1", conflict)
	}
	// Every Warn carries an actionable FIX hint for agent consumption.
	for _, r := range results {
		if r.Status == repogov.Warn && !strings.Contains(r.Message, "FIX:") {
			t.Errorf("advisory finding missing FIX hint: %q", r.Message)
		}
	}
}

// TestCheckClaudeSettings_MalformedJSONFails verifies a syntactically broken
// settings file is a Fail with a FIX hint: Claude Code cannot load it, so it
// is objectively broken rather than advisory.
func TestCheckClaudeSettings_MalformedJSONFails(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/settings.local.json": `{"permissions": {`,
	})
	results := repogov.CheckClaudeSettings(root)
	if len(results) != 1 || results[0].Status != repogov.Fail {
		t.Fatalf("expected exactly one Fail for malformed JSON, got %+v", results)
	}
	if !strings.Contains(results[0].Message, "FIX:") {
		t.Errorf("malformed-JSON failure missing FIX hint: %q", results[0].Message)
	}
}

// TestInitNeverTouchesClaudeSettings pins the invariant that scaffolding
// never creates or modifies .claude/settings.json or settings.local.json --
// they are user and machine configuration owned by Claude Code. Both the
// present case (bytes must survive init, even with InitAlwaysCreate) and the
// absent case (init must not create them) are enforced.
func TestInitNeverTouchesClaudeSettings(t *testing.T) {
	const original = `{"permissions":{"allow":["Bash(go test:*)"]},"custom":true}` + "\n"
	root := writeTempDir(t, map[string]string{
		".claude/settings.json": original,
	})

	cfg := repogov.DefaultConfig()
	cfg.InitAlwaysCreate = true
	if _, err := repogov.InitLayoutWithConfig(root, repogov.DefaultClaudeLayout(), cfg); err != nil {
		t.Fatalf("InitLayoutWithConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("settings.json unreadable after init: %v", err)
	}
	if string(data) != original {
		t.Errorf("init modified settings.json:\n got: %q\nwant: %q", data, original)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "settings.local.json")); !os.IsNotExist(err) {
		t.Error("init must not create settings.local.json")
	}
}

// TestCheckClaudeSettings_ConsolidationSuggestion verifies that five or more
// literal rules sharing a command prefix produce an Info suggestion to
// consolidate into a single ":*" prefix rule, while smaller groups and
// existing wildcards stay silent.
func TestCheckClaudeSettings_ConsolidationSuggestion(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".claude/settings.json": `{
  "permissions": {
    "allow": [
      "Bash(gofmt -w a.go)",
      "Bash(gofmt -w b.go)",
      "Bash(gofmt -w c.go)",
      "Bash(gofmt -w d.go)",
      "Bash(gofmt -w e.go)",
      "Bash(git status)",
      "Bash(git log)",
      "Bash(go vet:*)"
    ]
  }
}
`,
	})
	results := repogov.CheckClaudeSettings(root)

	var suggestions []string
	for _, r := range results {
		if strings.Contains(r.Message, "consolidate into") {
			if r.Status != repogov.Info {
				t.Errorf("consolidation suggestion must be Info (advisory), got %v", r.Status)
			}
			suggestions = append(suggestions, r.Message)
		}
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected exactly 1 consolidation suggestion, got %d: %v", len(suggestions), suggestions)
	}
	if !strings.Contains(suggestions[0], `"Bash(gofmt -w:*)"`) {
		t.Errorf("suggestion should propose Bash(gofmt -w:*), got %q", suggestions[0])
	}
}

// TestCheckClaudeSettings_CredentialFilesNeverParsed pins the guarantee that
// credential files are never opened: .claude/.credentials.json contains
// deliberately malformed bytes, and the audit must report its presence by
// name with remediation guidance -- never a parse error, never a Fail, and
// never any message derived from its content.
func TestCheckClaudeSettings_CredentialFilesNeverParsed(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		// Garbage content: any attempt to parse this as JSON would surface
		// as a malformed-JSON Fail, which must not happen.
		".claude/.credentials.json": "\x00\x01 not json { SENTINEL-SECRET",
		".claude/settings.json":     `{"permissions":{"allow":["Bash(go test:*)"]}}`,
	})
	results := repogov.CheckClaudeSettings(root)

	var credWarn int
	for _, r := range results {
		if strings.Contains(r.Message, "SENTINEL-SECRET") {
			t.Fatalf("credential file content leaked into results: %+v", r)
		}
		if r.Path == ".claude/.credentials.json" {
			if r.Status != repogov.Warn {
				t.Errorf("credential presence must be Warn, got %v", r.Status)
			}
			if !strings.Contains(r.Message, "content not read") || !strings.Contains(r.Message, "FIX:") {
				t.Errorf("credential finding must state content is not read and carry a FIX hint, got %q", r.Message)
			}
			credWarn++
		}
		if r.Status == repogov.Fail {
			t.Errorf("no Fail expected -- a Fail would indicate the credential file was parsed: %+v", r)
		}
	}
	if credWarn != 1 {
		t.Errorf("credential presence findings = %d, want 1", credWarn)
	}
}
