package main

// scaffold_test.go exercises every agent and subcommand combination and
// writes the generated file trees to ./tmp so the output can be inspected
// after a run.  TestMain wipes and recreates ./tmp before each run.
//
// Test layout under ./tmp:
//   tmp/copilot/   -- repogov -root . -agent copilot init
//   tmp/cursor/    -- repogov -root . -agent cursor  init
//   tmp/windsurf/  -- repogov -root . -agent windsurf init
//   tmp/claude/    -- repogov -root . -agent claude   init
//   tmp/all/       -- repogov -root . -agent all      init
//
// Error-path tests use t.TempDir() since they test failure conditions
// rather than inspecting what was created.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tmpRoot is the persistent output directory, located next to this file so
// that generated scaffolding is easy to inspect after a run.
const tmpRoot = "tmp"

func TestMain(m *testing.M) {
	// Wipe and recreate ./tmp before every run so the on-disk state always
	// reflects the most recent test execution.
	if err := os.RemoveAll(tmpRoot); err != nil {
		panic("scaffold_test: cannot remove " + tmpRoot + ": " + err.Error())
	}
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		panic("scaffold_test: cannot create " + tmpRoot + ": " + err.Error())
	}
	os.Exit(m.Run())
}

// scaffoldDir returns the absolute path to the per-agent tmp subdirectory
// and creates it if it does not yet exist.
func scaffoldDir(t *testing.T, agent string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(tmpRoot, agent))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// assertFileExists fails the test when path does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

// assertDirExists fails the test when path is not a directory.
func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("expected directory to exist: %s", path)
		return
	}
	if err != nil || !info.IsDir() {
		t.Errorf("expected a directory at: %s", path)
	}
}

// assertFileContains fails when path does not contain substr.
func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("%s: expected to contain %q", path, substr)
	}
}

// assertFileNotContains fails when path contains substr.
func assertFileNotContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		// If the file doesn't exist the assertion trivially passes.
		return
	}
	if strings.Contains(string(data), substr) {
		t.Errorf("%s: should not contain %q", path, substr)
	}
}

// assertValidJSON fails when the file at path is not valid JSON.
func assertValidJSON(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("%s: invalid JSON: %v", path, err)
	}
}

// assertConfigHasDefault fails when the JSON config at path does not have a
// positive "default" field.
func assertConfigHasDefault(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	var v struct {
		Default int `json:"default"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("%s: invalid JSON: %v", path, err)
		return
	}
	if v.Default <= 0 {
		t.Errorf("%s: default must be > 0, got %d", path, v.Default)
	}
}
