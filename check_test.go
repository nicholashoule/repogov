package repogov_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"two lines", "a\nb", 2},
		{"trailing newline", "a\nb\n", 2},
		{"five lines", "1\n2\n3\n4\n5", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempFile(t, "test.txt", tt.content)
			got, err := repogov.CountLines(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("CountLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountLines_NoSuchFile(t *testing.T) {
	_, err := repogov.CountLines("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCheckFile(t *testing.T) {
	// Create a file with 10 lines.
	content := strings.Repeat("line\n", 10)
	path := writeTempFile(t, "test.md", content)

	tests := []struct {
		name  string
		limit int
		want  repogov.Status
	}{
		{"within limit", 20, repogov.Pass},
		{"at limit", 10, repogov.Warn},
		{"over limit", 5, repogov.Fail},
		{"exactly at warn threshold", 12, repogov.Warn},
		{"exempt", 0, repogov.Skip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := repogov.Config{
				Default:          tt.limit,
				WarningThreshold: 80,
				Files:            map[string]int{},
			}
			if tt.limit == 0 {
				cfg.Default = 500
				cfg.Files[filepath.ToSlash(path)] = 0
			}
			result, err := repogov.CheckFile(path, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tt.want {
				t.Errorf("CheckFile status = %v, want %v (lines=%d, limit=%d)",
					result.Status, tt.want, result.Lines, result.Limit)
			}
		})
	}
}

func TestCheckDir(t *testing.T) {
	files := map[string]string{
		"a.md":        strings.Repeat("line\n", 10),
		"b.md":        strings.Repeat("line\n", 3),
		"c.go":        "package main\n",
		".git/HEAD":   "ref: refs/heads/main\n",
		"vendor/d.go": "package vendor\n",
	}
	root := writeTempDir(t, files)

	cfg := repogov.DefaultConfig()
	cfg.Default = 20

	results, err := repogov.CheckDir(root, []string{".md"}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Should find 2 .md files; .go, .git/, vendor/ should be excluded.
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.Status != repogov.Pass {
			t.Errorf("file %s: status = %v, want PASS", r.Path, r.Status)
		}
	}
}

func TestCheckDir_AllExtensions(t *testing.T) {
	files := map[string]string{
		"a.md": "line\n",
		"b.go": "line\n",
	}
	root := writeTempDir(t, files)
	cfg := repogov.DefaultConfig()

	results, err := repogov.CheckDir(root, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2 (all extensions)", len(results))
	}
}

func TestCheckDirContext_Cancellation(t *testing.T) {
	files := map[string]string{
		"a.md": strings.Repeat("line\n", 100),
		"b.md": strings.Repeat("line\n", 100),
	}
	root := writeTempDir(t, files)
	cfg := repogov.DefaultConfig()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := repogov.CheckDirContext(ctx, root, nil, cfg)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestCheckDir_SkipDirs(t *testing.T) {
	files := map[string]string{
		"src/main.go":       "package main\n",
		"vendor/dep/dep.go": "package dep\n",
	}
	root := writeTempDir(t, files)
	cfg := repogov.DefaultConfig()

	results, err := repogov.CheckDir(root, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if strings.Contains(r.Path, "vendor") {
			t.Errorf("vendor should be skipped, got result for %s", r.Path)
		}
	}
}

// TestCheckDir_FilesOverrideViaCheckDir verifies that per-file entries in
// Config.Files are respected when limits are resolved through CheckDir
// (which walks with absolute paths internally). This is a regression test
// for a bug where CheckDir passed the absolute filesystem path to
// ResolveLimit instead of the repo-relative path, causing Files entries
// to never match.
func TestCheckDir_FilesOverrideViaCheckDir(t *testing.T) {
	files := map[string]string{
		"README.md":                                   strings.Repeat("line\n", 10),
		".github/copilot-instructions.md":             strings.Repeat("line\n", 5),
		".github/instructions/memory.instructions.md": strings.Repeat("line\n", 8),
	}
	root := writeTempDir(t, files)

	cfg := repogov.Config{
		Default:          500,
		WarningThreshold: 85,
		IncludeExts:      []string{".md"},
		SkipDirs:         []string{".git"},
		Files: map[string]int{
			"README.md":                                   1200,
			".github/copilot-instructions.md":             50,
			".github/instructions/memory.instructions.md": 200,
		},
	}

	results, err := repogov.CheckDir(root, []string{".md"}, cfg)
	if err != nil {
		t.Fatal(err)
	}

	limits := make(map[string]int)
	for _, r := range results {
		limits[r.Path] = r.Limit
	}

	want := map[string]int{
		"README.md":                                   1200,
		".github/copilot-instructions.md":             50,
		".github/instructions/memory.instructions.md": 200,
	}
	for path, wantLimit := range want {
		got, ok := limits[path]
		if !ok {
			t.Errorf("missing result for %s", path)
			continue
		}
		if got != wantLimit {
			t.Errorf("CheckDir: %s limit = %d, want %d (files override ignored)", path, got, wantLimit)
		}
	}
}

func TestCheckFile_NonexistentFile(t *testing.T) {
	cfg := repogov.DefaultConfig()
	_, err := repogov.CheckFile(filepath.Join(os.TempDir(), "nonexistent.md"), cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCheckFile_PctAndAction(t *testing.T) {
	content := strings.Repeat("line\n", 10)
	path := writeTempFile(t, "test.txt", content)

	t.Run("fail has action", func(t *testing.T) {
		cfg := repogov.Config{Default: 5, WarningThreshold: 80, Files: map[string]int{}}
		result, err := repogov.CheckFile(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Pct != 200 {
			t.Errorf("Pct = %d, want 200", result.Pct)
		}
		if result.Action == "" {
			t.Error("FAIL result should have an Action")
		}
		if !strings.Contains(result.Action, "over limit") {
			t.Errorf("Action = %q, want 'over limit' substring", result.Action)
		}
	})

	t.Run("warn has action", func(t *testing.T) {
		cfg := repogov.Config{Default: 12, WarningThreshold: 80, Files: map[string]int{}}
		result, err := repogov.CheckFile(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Pct == 0 {
			t.Error("Pct should not be zero for WARN result")
		}
		if result.Action == "" {
			t.Error("WARN result should have an Action")
		}
	})

	t.Run("pass has no action", func(t *testing.T) {
		cfg := repogov.Config{Default: 100, WarningThreshold: 80, Files: map[string]int{}}
		result, err := repogov.CheckFile(path, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if result.Action != "" {
			t.Errorf("PASS result should have empty Action, got %q", result.Action)
		}
		if result.Pct != 10 {
			t.Errorf("Pct = %d, want 10", result.Pct)
		}
	})
}

func TestCheckFile_ZeroWarningThreshold_UsesDefault(t *testing.T) {
	// When WarningThreshold is zero, the built-in default (85%) applies.
	// 10 lines at a limit of 12 = 83%, below the 85% default -> PASS.
	content := strings.Repeat("line\n", 10)
	path := writeTempFile(t, "test.txt", content)

	cfg := repogov.Config{
		Default:          12,
		WarningThreshold: 0, // forces built-in default of 85
		Files:            map[string]int{},
	}
	result, err := repogov.CheckFile(path, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// 10/12 = 83%, which is below 85% → should PASS (not WARN).
	if result.Status != repogov.Pass {
		t.Errorf("with zero WarningThreshold status = %v, want PASS (83%% < default 85%%)", result.Status)
	}
}
