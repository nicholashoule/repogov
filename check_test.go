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
