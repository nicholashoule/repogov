package repogov_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestCheckLayout_MissingRoot(t *testing.T) {
	root := t.TempDir()
	schema := repogov.LayoutSchema{Root: ".nonexistent"}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for missing root")
	}
	if results[0].Status != repogov.Fail {
		t.Errorf("expected Fail for missing root, got %v", results[0].Status)
	}
}

func TestCheckLayout_RequiredFiles(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml": "name: CI\n",
	})

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Required: []string{"workflows/ci.yml"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, r := range results {
		if r.Path == ".github/workflows/ci.yml" {
			found = true
			if r.Status != repogov.Pass {
				t.Errorf("required file status = %v, want PASS", r.Status)
			}
		}
	}
	if !found {
		t.Error("required file not found in results")
	}
}

func TestCheckLayout_MissingRequiredFile(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/.gitkeep": "",
	})

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Required: []string{"workflows/ci.yml"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Path == ".github/workflows/ci.yml" {
			if r.Status != repogov.Fail {
				t.Errorf("missing required file status = %v, want FAIL", r.Status)
			}
			return
		}
	}
	t.Error("expected a result for missing required file")
}

func TestCheckLayout_OptionalFiles(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/CODEOWNERS": "* @owner\n",
	})

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Optional: []string{"CODEOWNERS"},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Path == ".github/CODEOWNERS" {
			if r.Status != repogov.Info {
				t.Errorf("optional file status = %v, want INFO", r.Status)
			}
			return
		}
	}
	t.Error("expected a result for optional file")
}

func TestCheckLayout_NamingViolation(t *testing.T) {
	// Create a file with uppercase name that is not in exceptions.
	root := writeTempDir(t, map[string]string{
		".github/MyConfig.yml": "test\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Naming: repogov.NamingRule{
			Case: "lowercase",
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFail := false
	for _, r := range results {
		if r.Status == repogov.Fail {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a FAIL result for naming violation")
	}
}

func TestCheckLayout_NamingException(t *testing.T) {
	// CODEOWNERS is uppercase but listed as exception.
	root := writeTempDir(t, map[string]string{
		".github/CODEOWNERS": "* @owner\n",
	})

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Optional: []string{"CODEOWNERS"},
		Naming: repogov.NamingRule{
			Case:       "lowercase",
			Exceptions: []string{"CODEOWNERS"},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Status == repogov.Fail {
			t.Errorf("unexpected FAIL: path=%s msg=%s", r.Path, r.Message)
		}
	}
}

func TestCheckLayoutContext_Cancellation(t *testing.T) {
	root := writeTempDir(t, map[string]string{
		".github/test.md": "test\n",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	schema := repogov.LayoutSchema{Root: ".github"}
	_, err := repogov.CheckLayoutContext(ctx, root, schema)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestDefaultCopilotLayout(t *testing.T) {
	schema := repogov.DefaultCopilotLayout()

	if schema.Root != ".github" {
		t.Errorf("Root = %q, want .github", schema.Root)
	}
	// Verify all expected agent-related directories are present.
	expectedDirs := []string{
		"instructions",
	}
	for _, d := range expectedDirs {
		if _, ok := schema.Dirs[d]; !ok {
			t.Errorf("missing dir rule for %q", d)
		}
	}

	if schema.Naming.Case != "lowercase" {
		t.Errorf("Naming.Case = %q, want lowercase", schema.Naming.Case)
	}
}

func TestCheckLayout_DirMinimum(t *testing.T) {
	// Create workflow files to satisfy Min=1.
	root := writeTempDir(t, map[string]string{
		".github/workflows/ci.yml": "name: CI\n",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"workflows": {Glob: "*.yml", Min: 1, Description: "CI workflows"},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasPass := false
	for _, r := range results {
		if r.Path == ".github/workflows" && r.Status == repogov.Pass {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("expected PASS for workflows dir meeting minimum")
	}
}

func TestCheckLayout_DirBelowMinimum(t *testing.T) {
	// Directory exists but has no matching files.
	root := writeTempDir(t, map[string]string{
		".github/workflows/.gitkeep": "",
	})

	schema := repogov.LayoutSchema{
		Root: ".github",
		Dirs: map[string]repogov.DirRule{
			"workflows": {Glob: "*.yml", Min: 2, Description: "CI workflows"},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	hasFail := false
	for _, r := range results {
		if r.Path == ".github/workflows" && r.Status == repogov.Fail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for workflows dir below minimum")
	}
}

func TestCheckLayout_GitkeepSkipped(t *testing.T) {
	// .gitkeep is created by 'repogov init' and should not produce a WARN.
	root := writeTempDir(t, map[string]string{
		".github/workflows/.gitkeep": "",
		".github/workflows/ci.yml":   "name: CI\n",
	})

	schema := repogov.LayoutSchema{
		Root:     ".github",
		Required: []string{"workflows/ci.yml"},
		Dirs: map[string]repogov.DirRule{
			"workflows": {Glob: "*.yml", Min: 1},
		},
	}

	results, err := repogov.CheckLayout(root, schema)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if strings.Contains(r.Path, ".gitkeep") {
			t.Errorf("unexpected result for .gitkeep: [%v] %s -- %s", r.Status, r.Path, r.Message)
		}
	}
}
