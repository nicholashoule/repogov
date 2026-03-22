package repogov_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempFile creates a temporary file with the given content and returns
// its path. The file is automatically removed when the test completes.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTempDir creates a temporary directory tree from a map of relative
// paths to file content. Returns the root directory path.
func writeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relPath, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// assertExists fails the test when path does not exist.
func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected path to exist: %s", path)
	}
}

// assertNotExists fails the test when path exists.
func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected path to not exist: %s", path)
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
		return // file doesn't exist — assertion trivially passes
	}
	if strings.Contains(string(data), substr) {
		t.Errorf("%s: should not contain %q", path, substr)
	}
}
