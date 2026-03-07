package repogov_test

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempFile creates a temporary file with the given content and returns
// its path. The file is automatically removed when the test completes.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
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
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
