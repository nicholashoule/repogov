package repogov_test

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status repogov.Status
		want   string
	}{
		{repogov.Pass, "PASS"},
		{repogov.Warn, "WARN"},
		{repogov.Fail, "FAIL"},
		{repogov.Skip, "SKIP"},
		{repogov.Info, "INFO"},
		{repogov.Status(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", int(tt.status), got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := repogov.DefaultConfig()

	if cfg.Default != 300 {
		t.Errorf("Default = %d, want 300", cfg.Default)
	}
	if cfg.WarningThreshold != 85 {
		t.Errorf("WarningThreshold = %d, want 85", cfg.WarningThreshold)
	}
	if len(cfg.SkipDirs) == 0 {
		t.Fatal("SkipDirs is empty, want at least .git")
	}
	if cfg.Files == nil {
		t.Fatal("Files map is nil, should be initialized")
	}

	// Verify default IncludeExts.
	if len(cfg.IncludeExts) == 0 {
		t.Fatal("IncludeExts is empty, want [\".md\", \".mdc\"]")
	}
	if cfg.IncludeExts[0] != ".md" {
		t.Errorf("IncludeExts[0] = %q, want \".md\"", cfg.IncludeExts[0])
	}

	// Verify default rules.
	if len(cfg.Rules) < 1 {
		t.Fatalf("Rules count = %d, want >= 1", len(cfg.Rules))
	}
	if cfg.Rules[0].Glob != ".github/instructions/*.md" || cfg.Rules[0].Limit == nil || *cfg.Rules[0].Limit != 300 {
		t.Errorf("Rules[0] = %+v, want {Glob:.github/instructions/*.md Limit:300}", cfg.Rules[0])
	}

	// Verify copilot-instructions.md per-file limit.
	if cfg.Files[".github/copilot-instructions.md"] != 50 {
		t.Errorf("Files[.github/copilot-instructions.md] = %d, want 50",
			cfg.Files[".github/copilot-instructions.md"])
	}

	// Verify CLAUDE.md per-file limit.
	if cfg.Files[".claude/CLAUDE.md"] != 200 {
		t.Errorf("Files[.claude/CLAUDE.md] = %d, want 200", cfg.Files[".claude/CLAUDE.md"])
	}
}

func TestResolveLimit(t *testing.T) {
	cfg := repogov.Config{
		Default: 300,
		Rules: []repogov.Rule{
			{Glob: "docs/*.md", Limit: repogov.RuleLimit(1000)},
			{Glob: "*.go", Limit: repogov.RuleLimit(600)},
		},
		Files: map[string]int{
			"README.md":    1200,
			"CHANGELOG.md": 0,
		},
	}

	tests := []struct {
		name string
		path string
		want int
	}{
		{"per-file override", "README.md", 1200},
		{"per-file exempt", "CHANGELOG.md", 0},
		{"glob match docs", "docs/design.md", 1000},
		{"glob match go", "main.go", 600},
		{"default fallback", "random.txt", 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repogov.ResolveLimit(tt.path, cfg); got != tt.want {
				t.Errorf("ResolveLimit(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveLimit_NilLimitFallsThrough(t *testing.T) {
	// A rule with nil Limit should fall through to the config default.
	cfg := repogov.Config{
		Default: 500,
		Rules:   []repogov.Rule{{Glob: "*.md"}}, // Limit is nil
	}
	if got := repogov.ResolveLimit("README.md", cfg); got != 500 {
		t.Errorf("nil Limit should fall through to default 500, got %d", got)
	}
}

func TestResolveLimit_ZeroDefault(t *testing.T) {
	cfg := repogov.Config{Default: 0}
	got := repogov.ResolveLimit("anything.txt", cfg)
	if got != 300 {
		t.Errorf("ResolveLimit with zero default = %d, want builtin 300", got)
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	tests := []struct {
		status repogov.Status
		want   string
	}{
		{repogov.Pass, `"PASS"`},
		{repogov.Warn, `"WARN"`},
		{repogov.Fail, `"FAIL"`},
		{repogov.Skip, `"SKIP"`},
		{repogov.Info, `"INFO"`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Errorf("MarshalJSON = %s, want %s", data, tt.want)
			}
		})
	}
}

func TestStatusUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  repogov.Status
	}{
		{`"PASS"`, repogov.Pass},
		{`"WARN"`, repogov.Warn},
		{`"FAIL"`, repogov.Fail},
		{`"SKIP"`, repogov.Skip},
		{`"INFO"`, repogov.Info},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s repogov.Status
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatal(err)
			}
			if s != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, s, tt.want)
			}
		})
	}
}

func TestStatusUnmarshalJSON_Invalid(t *testing.T) {
	var s repogov.Status
	if err := json.Unmarshal([]byte(`"BOGUS"`), &s); err == nil {
		t.Error("expected error for unknown status label")
	}
}

func TestResultJSON_RoundTrip(t *testing.T) {
	r := repogov.Result{
		Path:   "test.go",
		Lines:  42,
		Limit:  500,
		Status: repogov.Pass,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// Verify Status appears as string, not integer.
	if got := string(data); !contains(got, `"Status":"PASS"`) {
		t.Errorf("JSON should contain Status string, got: %s", got)
	}
	var r2 repogov.Result
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatal(err)
	}
	if r2.Status != repogov.Pass {
		t.Errorf("round-trip Status = %v, want PASS", r2.Status)
	}
}

// contains checks if s contains substr (avoids importing strings).
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestZeroDependencies reads go.mod and fails if any "require" directive
// is found. This enforces repogov's core pillar of zero external Go
// dependencies. External tools (like demojify) must be invoked via
// os/exec, never imported.
func TestZeroDependencies(t *testing.T) {
	f, err := os.Open("go.mod")
	if err != nil {
		t.Fatalf("cannot open go.mod: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	inBlock := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "require (" {
			inBlock = true
			t.Errorf("go.mod:%d: found require block — zero-dependency contract violated", lineNum)
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
			}
			continue
		}
		if strings.HasPrefix(line, "require ") {
			t.Errorf("go.mod:%d: found require directive %q — zero-dependency contract violated", lineNum, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading go.mod: %v", err)
	}
}
