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

// TestCountLines_LongLine is a regression test for the bufio.Scanner token
// limit: a single line longer than 64 KiB previously made CountLines fail
// with bufio.ErrTooLong, and CheckDir's skip-unreadable policy then silently
// dropped the file from governance results entirely -- an over-limit file
// could pass CI unreported. The chunked byte-count implementation has no
// line-length limit.
func TestCountLines_LongLine(t *testing.T) {
	long := strings.Repeat("x", 70*1024)
	path := writeTempFile(t, "long.md", long+"\nsecond line\n")
	got, err := repogov.CountLines(path)
	if err != nil {
		t.Fatalf("CountLines with 70KiB line: %v", err)
	}
	if got != 2 {
		t.Errorf("CountLines = %d, want 2", got)
	}
}

// TestCheckDir_LongLineFileNotSkipped verifies the CheckDir half of the same
// regression: a file that is over its line limit AND contains a >64 KiB line
// must appear in the results as FAIL rather than being silently dropped.
func TestCheckDir_LongLineFileNotSkipped(t *testing.T) {
	long := strings.Repeat("x", 70*1024)
	over := long + "\n" + strings.Repeat("line\n", 600) // 601 lines > 500 default
	root := writeTempDir(t, map[string]string{
		"clean.md": "one\ntwo\n",
		"huge.md":  over,
	})

	results, err := repogov.CheckDir(root, []string{".md"}, repogov.DefaultConfig())
	if err != nil {
		t.Fatalf("CheckDir: %v", err)
	}
	var found bool
	for _, r := range results {
		if r.Path == "huge.md" {
			found = true
			if r.Status != repogov.Fail {
				t.Errorf("huge.md status = %v, want FAIL (601 lines > 500)", r.Status)
			}
			if r.Lines != 601 {
				t.Errorf("huge.md lines = %d, want 601", r.Lines)
			}
		}
	}
	if !found {
		t.Error("huge.md missing from results -- long-line file was silently skipped")
	}
}

// TestCheckFile_BinarySkipped verifies binary and compiled files are never
// line-counted or parsed: a NUL byte within the first 512 bytes yields a
// Skip result, and only the sniff prefix is read. Text files are unaffected.
func TestCheckFile_BinarySkipped(t *testing.T) {
	binary := "MZ\x00\x01compiled bytes\x00" + strings.Repeat("\x00garbage\n", 200)
	path := writeTempFile(t, "tool.exe", binary)
	result, err := repogov.CheckFile(path, repogov.DefaultConfig())
	if err != nil {
		t.Fatalf("CheckFile on binary: %v", err)
	}
	if result.Status != repogov.Skip {
		t.Errorf("binary file status = %v, want SKIP (must never be parsed)", result.Status)
	}
	if result.Lines != 0 {
		t.Errorf("binary file lines = %d, want 0 (content must not be counted)", result.Lines)
	}
}

// TestCheckDir_BinarySkippedTextChecked verifies that with no extension
// filter (the -exts all case) a directory walk skips binaries while still
// checking text files normally.
func TestCheckDir_BinarySkippedTextChecked(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		"doc.md":   "one\ntwo\n",
		"blob.bin": "\x00\x01\x02\x03binary\x00",
	})
	results, err := repogov.CheckDir(root, nil, repogov.DefaultConfig())
	if err != nil {
		t.Fatalf("CheckDir: %v", err)
	}
	statuses := make(map[string]repogov.Status, len(results))
	for _, r := range results {
		statuses[r.Path] = r.Status
	}
	if statuses["blob.bin"] != repogov.Skip {
		t.Errorf("blob.bin status = %v, want SKIP", statuses["blob.bin"])
	}
	if statuses["doc.md"] != repogov.Pass {
		t.Errorf("doc.md status = %v, want PASS", statuses["doc.md"])
	}
}

// TestCheckFile_SensitiveExtensionsNeverOpened verifies certificate and key
// files (.crt, .pem, .key) are skipped before the file is even opened: no
// line count, no parse, regardless of content or size. Case-insensitive.
func TestCheckFile_SensitiveExtensionsNeverOpened(t *testing.T) {
	for _, name := range []string{"server.crt", "chain.pem", "private.key", "UPPER.PEM"} {
		t.Run(name, func(t *testing.T) {
			content := "-----BEGIN PRIVATE KEY-----\n" + strings.Repeat("SENTINEL\n", 700) + "-----END PRIVATE KEY-----\n"
			path := writeTempFile(t, name, content)
			result, err := repogov.CheckFile(path, repogov.DefaultConfig())
			if err != nil {
				t.Fatalf("CheckFile: %v", err)
			}
			if result.Status != repogov.Skip {
				t.Errorf("%s status = %v, want SKIP", name, result.Status)
			}
			if result.Lines != 0 {
				t.Errorf("%s lines = %d, want 0 (content must never be read)", name, result.Lines)
			}
		})
	}
}
