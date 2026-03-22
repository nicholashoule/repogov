package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- isAbsolute tests ---

func TestIsAbsolute(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", false},
		{"relative/path", false},
		{"file.json", false},
		{"/absolute/path", true},
		{"\\absolute\\path", true},
		{"C:\\Windows\\path", true},
		{"C:/unix/style", true},
		{"D:\\other", true}, //nolint:misspell // false positive: "ther" is a substring of "other", not a word
	}
	for _, tt := range tests {
		got := isAbsolute(tt.path)
		if got != tt.want {
			t.Errorf("isAbsolute(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFindGitRoot_Found(t *testing.T) {
	// Create a temp dir with a .git sub-directory and a nested sub-directory.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, repoRoot)
	}
}

func TestFindGitRoot_FoundAtSelf(t *testing.T) {
	// findGitRoot should return the directory itself when .git lives there.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(repoRoot)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", repoRoot, got, repoRoot)
	}
}

func TestFindGitRoot_GitFile(t *testing.T) {
	// .git as a plain file (Git worktree) should also be recognized.
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: ../.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != repoRoot {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, repoRoot)
	}
}

func TestFindGitRoot_NotFound(t *testing.T) {
	// A temp directory with no ancestor .git should return "".
	dir := t.TempDir()
	// Walk past the temp dir to a sub with no .git anywhere.
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	// We can't guarantee the test runner itself isn't inside a git repo, so
	// only verify that findGitRoot on a path with no .git returns some value
	// (may or may not be "" depending on the environment).  Instead, test the
	// behavior we care about: the returned path, if non-empty, must contain .git.
	got := findGitRoot(sub)
	if got != "" {
		if _, err := os.Stat(filepath.Join(got, ".git")); err != nil {
			t.Errorf("findGitRoot returned %q which has no .git: %v", got, err)
		}
	}
}

func TestResolveRoot_ExplicitPath(t *testing.T) {
	// An explicit absolute path that is NOT inside a git repo should be
	// returned as-is (no .git found → fallback to the resolved path).
	dir := t.TempDir()
	got := resolveRoot(dir)
	if got != dir {
		t.Errorf("resolveRoot(%q) = %q, want %q", dir, got, dir)
	}
}

func TestResolveRoot_ExplicitAgentSubdir(t *testing.T) {
	// An explicit path that points at an agent subdirectory inside a git repo
	// (e.g. -root .cursor or -root .github/rules) must be resolved up to the
	// git root, preventing double-nested scaffolding.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(repoRoot, ".cursor", "rules")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := resolveRoot(agentDir)
	gotInfo, err1 := os.Stat(got)
	wantInfo, err2 := os.Stat(repoRoot)
	if err1 != nil {
		t.Fatalf("resolveRoot returned unstat-able path %q: %v", got, err1)
	}
	if err2 != nil {
		t.Fatalf("stat repoRoot %q: %v", repoRoot, err2)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("resolveRoot(%q) = %q, want same dir as git root %q", agentDir, got, repoRoot)
	}
}

func TestResolveRoot_AutoDetect(t *testing.T) {
	// When root is "." and the CWD is inside a git repo, resolveRoot should
	// return the git root.
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	got := resolveRoot(".")
	// On macOS, t.TempDir() returns a symlinked path (/var/folders/…) while
	// os.Getwd() returns the real path (/private/var/folders/…). Use
	// os.SameFile for an inode-level comparison that is symlink-agnostic.
	gotInfo, err1 := os.Stat(got)
	wantInfo, err2 := os.Stat(repoRoot)
	if err1 != nil {
		t.Fatalf("resolveRoot returned unstat-able path %q: %v", got, err1)
	}
	if err2 != nil {
		t.Fatalf("stat repoRoot %q: %v", repoRoot, err2)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("resolveRoot(\".\") = %q, want same directory as %q", got, repoRoot)
	}
}

func TestResolveRoot_DotNonDefault(t *testing.T) {
	// An explicit non-"." path must not be processed by git-root detection.
	got := resolveRoot("/nonexistent/path")
	if got != "/nonexistent/path" {
		t.Errorf("resolveRoot returned %q, want /nonexistent/path", got)
	}
}

// TestRun_Init_NestedDir verifies that running init with root="." from a
// nested subdirectory (e.g. .github/) creates files in the git repo root,
// not in the subdirectory.
func TestRun_Init_NestedDir(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, ".github", "instructions")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := bufs()
	// No -root flag: should auto-detect repoRoot via git root resolution.
	code := run([]string{"-agent", "copilot", "init"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr: %s)", code, stderr.String())
	}

	// Files must be created under repoRoot, not under nested.
	if _, err := os.Stat(filepath.Join(repoRoot, ".github", "copilot-instructions.md")); err != nil {
		t.Errorf(".github/copilot-instructions.md not found under repo root: %v", err)
	}
	// Nothing should have been created inside the nested directory itself
	// (other than the .github/instructions dir that we pre-created).
	entries, err := os.ReadDir(nested)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("unexpected files created inside nested dir %s: %v", nested, names)
	}
}

// TestRunLayout_SkipFrontmatter_ConfigFile verifies the end-to-end path:
// a repogov-config.json with skip_frontmatter=true causes runLayout to skip
// frontmatter validation, so a file missing YAML frontmatter passes.
func TestRunLayout_SkipFrontmatter_ConfigFile(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/copilot-instructions.md":                     "# Copilot\n",
		".github/instructions/no-frontmatter.instructions.md": "# No frontmatter\n",
		".github/repogov-config.json":                         `{"skip_frontmatter": true}`,
	})

	stdout, stderr := bufs()
	// Without skip_frontmatter, the missing frontmatter would cause a FAIL.
	// With the config file setting skip_frontmatter=true, it should pass.
	if code := runLayout(root, "", "copilot", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0 with skip_frontmatter=true, got %d\nstdout: %s\nstderr: %s",
			code, stdout.String(), stderr.String())
	}

	// Confirm no frontmatter results appear in output.
	if strings.Contains(stdout.String(), "frontmatter") {
		t.Errorf("output should not mention frontmatter when skip_frontmatter=true:\n%s", stdout.String())
	}
}

// TestRunLayout_FrontmatterFails_WithoutConfig verifies that without
// skip_frontmatter in config, a file missing frontmatter does fail.
func TestRunLayout_FrontmatterFails_WithoutConfig(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/copilot-instructions.md":                     "# Copilot\n",
		".github/instructions/no-frontmatter.instructions.md": "# No frontmatter\n",
	})

	stdout, stderr := bufs()
	if code := runLayout(root, "", "copilot", "", false, false, stdout, stderr); code == 0 {
		t.Fatalf("expected non-zero exit for missing frontmatter, got 0\nstdout: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "frontmatter") {
		t.Errorf("expected frontmatter failure in output:\n%s", stdout.String())
	}
}

// TestRunLayout_SkipFrontmatter_ExplicitConfigPath verifies that -config
// flag correctly loads skip_frontmatter from an explicit config path.
func TestRunLayout_SkipFrontmatter_ExplicitConfigPath(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/copilot-instructions.md":          "# Copilot\n",
		".github/instructions/bad.instructions.md": "# Missing frontmatter\n",
		"custom-config.json":                       `{"skip_frontmatter": true}`,
	})

	stdout, stderr := bufs()
	cfgPath := filepath.Join(root, "custom-config.json")
	if code := runLayout(root, cfgPath, "copilot", "", false, false, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0 with explicit config skip_frontmatter=true, got %d\nstdout: %s\nstderr: %s",
			code, stdout.String(), stderr.String())
	}
}
