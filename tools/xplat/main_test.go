package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// call runs the command, capturing stdout and stderr separately.
func call(args ...string) (rc int, stdout, stderr string) {
	var o, e bytes.Buffer
	rc = run(args, &o, &e)
	return rc, o.String(), e.String()
}

func TestMkdirIdempotentAndNested(t *testing.T) {
	d := filepath.Join(t.TempDir(), "a", "b", "build")
	if rc, _, e := call("mkdir", d); rc != 0 {
		t.Fatalf("mkdir rc=%d stderr=%q", rc, e)
	}
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		t.Fatalf("dir not created: %v", err)
	}
	// Idempotent: a second call on the existing dir is a silent success.
	if rc, _, e := call("mkdir", d); rc != 0 || e != "" {
		t.Fatalf("mkdir not idempotent: rc=%d stderr=%q", rc, e)
	}
}

func TestRmrfMultipleAndAbsentIsSilent(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "build")
	file := filepath.Join(tmp, "coverage.out")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Mix of an existing dir, an existing file, and an absent path -> all ok.
	if rc, _, e := call("rmrf", dir, file, filepath.Join(tmp, "missing")); rc != 0 {
		t.Fatalf("rmrf rc=%d stderr=%q", rc, e)
	}
	for _, p := range []string{dir, file} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists (err=%v)", p, err)
		}
	}
}

func TestCpCreatesParentAndIsExecutable(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "pre-commit")
	if err := os.WriteFile(src, []byte("#!/bin/sh\necho hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(tmp, ".git", "hooks", "pre-commit") // parent absent
	if rc, _, e := call("cp", src, dst); rc != 0 {
		t.Fatalf("cp rc=%d stderr=%q", rc, e)
	}
	got, err := os.ReadFile(dst)
	if err != nil || !strings.Contains(string(got), "echo hi") {
		t.Fatalf("dst content wrong: %q err=%v", got, err)
	}
	// The executable bit only exists on POSIX; Windows reports -rw-rw-rw-
	// regardless (git there does not need it). The cp intent ("runnable
	// hook on POSIX") is only assertable off Windows.
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(dst); fi != nil && fi.Mode().Perm()&0o100 == 0 {
			t.Errorf("dst not executable on %s: mode=%v", runtime.GOOS, fi.Mode())
		}
	}
}

func TestRequirePresentAndMissing(t *testing.T) {
	tmp := t.TempDir()
	present := filepath.Join(tmp, ".env")
	if err := os.WriteFile(present, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if rc, _, e := call("require", present, "should not print"); rc != 0 || e != "" {
		t.Fatalf("require(present) rc=%d stderr=%q, want 0/empty", rc, e)
	}
	rc, _, e := call("require", filepath.Join(tmp, "nope"), "ERROR: .env missing")
	if rc != 1 || !strings.Contains(e, "ERROR: .env missing") {
		t.Fatalf("require(missing) rc=%d stderr=%q, want 1 + message", rc, e)
	}
}

func TestHelpRendersTargetsAndGroups(t *testing.T) {
	mf := filepath.Join(t.TempDir(), "Makefile")
	const body = "" +
		"##@ Build\n" +
		"build: ## Build the binary\n" +
		"\t@go build ./...\n" +
		".PHONY: build\n" + // must be skipped (leading dot, no '## ')
		"clean: deps ## Remove artifacts\n" +
		"# not-a-target: ## comment line, not a rule\n"
	if err := os.WriteFile(mf, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rc, out, e := call("help", mf)
	if rc != 0 || e != "" {
		t.Fatalf("help rc=%d stderr=%q", rc, e)
	}
	if !strings.Contains(out, "Build\n") {
		t.Errorf("missing ##@ group header; got:\n%s", out)
	}
	if !strings.Contains(out, "build") || !strings.Contains(out, "Build the binary") {
		t.Errorf("missing documented target; got:\n%s", out)
	}
	if !strings.Contains(out, "clean") || !strings.Contains(out, "Remove artifacts") {
		t.Errorf("missing second target; got:\n%s", out)
	}
	if strings.Contains(out, "not-a-target") || strings.Contains(out, ".PHONY") {
		t.Errorf("non-target lines leaked into help; got:\n%s", out)
	}
	// A missing makefile is a hard error (exit 1), not silent.
	if rc, _, _ := call("help", filepath.Join(t.TempDir(), "nope")); rc != 1 {
		t.Errorf("help(missing file) rc=%d, want 1", rc)
	}
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"mkdir"},
		{"mkdir", ""},
		{"rmrf"},
		{"cp", "only-one"},
		{"require", "one-arg"},
		{"help"},
		{"bogus"},
	}
	for _, args := range cases {
		if rc, _, e := call(args...); rc != 2 {
			t.Errorf("run(%v) rc=%d, want 2 (usage); stderr=%q", args, rc, e)
		}
	}
}
