package repogov_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestCheckDir_SymlinkNotOpened(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(target, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	results, err := repogov.CheckDir(root, []string{".md"}, repogov.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != repogov.Skip || results[0].Lines != 0 {
		t.Fatalf("symlink must be skipped without counting lines, got %+v", results)
	}
}

func TestInit_DoesNotWriteThroughSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "victim.md")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dangling-style plant: AGENTS.md is a symlink to an outside file.
	if err := os.Symlink(outside, filepath.Join(root, "AGENTS.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, _ = repogov.InitLayout(root, repogov.DefaultCopilotLayout())
	got, _ := os.ReadFile(outside)
	if string(got) != "keep" {
		t.Fatalf("init wrote through a symlink: %q", got)
	}
}
