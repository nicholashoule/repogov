package repogov_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicholashoule/repogov"
)

// Example demonstrates the typical end-to-end workflow: load a config,
// check a directory for line-count violations, and report results.
func Example() {
	// Write a small markdown file to a temp directory.
	dir, _ := os.MkdirTemp("", "repo-*")
	defer os.RemoveAll(dir)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Title\nBody text.\n"), 0644)

	cfg := repogov.DefaultConfig()
	results, _ := repogov.CheckDir(dir, []string{".md"}, cfg)

	fmt.Println(repogov.Passed(results))
	fmt.Println(results[0].Status)
	// Output:
	// true
	// PASS
}

func ExampleCheckFile() {
	f, _ := os.CreateTemp("", "*.md")
	defer os.Remove(f.Name())
	fmt.Fprintln(f, "# Hello")
	fmt.Fprintln(f, "World")
	f.Close()

	cfg := repogov.DefaultConfig()
	result, _ := repogov.CheckFile(f.Name(), cfg)
	fmt.Println(result.Status)
	fmt.Println(result.Lines)
	// Output:
	// PASS
	// 2
}

func ExampleLoadConfig() {
	f, _ := os.CreateTemp("", "repogov-*.json")
	defer os.Remove(f.Name())
	_, _ = f.WriteString(`{"default": 200, "warning_threshold": 90}`)
	f.Close()

	cfg, _ := repogov.LoadConfig(f.Name())
	fmt.Println(cfg.Default)
	fmt.Println(cfg.WarningThreshold)
	// Output:
	// 200
	// 90
}

func ExampleDefaultConfig() {
	cfg := repogov.DefaultConfig()
	fmt.Println(cfg.Default)
	fmt.Println(cfg.WarningThreshold)
	// Output:
	// 300
	// 85
}

func ExampleResolveLimit() {
	cfg := repogov.Config{
		Default: 500,
		Files:   map[string]int{"README.md": 1200},
	}
	fmt.Println(repogov.ResolveLimit("README.md", cfg))
	fmt.Println(repogov.ResolveLimit("other.md", cfg))
	// Output:
	// 1200
	// 500
}

func ExampleCheckDir() {
	dir, _ := os.MkdirTemp("", "repo-*")
	defer os.RemoveAll(dir)
	_ = os.WriteFile(filepath.Join(dir, "notes.md"), []byte("line1\nline2\n"), 0644)

	cfg := repogov.DefaultConfig()
	results, _ := repogov.CheckDir(dir, []string{".md"}, cfg)
	fmt.Println(len(results))
	fmt.Println(repogov.Passed(results))
	// Output:
	// 1
	// true
}

func ExamplePassed() {
	results := []repogov.Result{
		{Status: repogov.Pass},
		{Status: repogov.Warn},
	}
	fmt.Println(repogov.Passed(results))
	// Output: true
}

func ExampleSummary() {
	results := []repogov.Result{
		{Path: "README.md", Lines: 42, Limit: 300, Status: repogov.Pass},
	}
	fmt.Print(repogov.Summary(results))
	// Output:
	// [PASS] README.md (42 / 300, 14%)
	//
	// Total: 1 files | 1 pass | 0 warn | 0 fail | 0 skip
}

func ExampleCheckLayout() {
	dir, _ := os.MkdirTemp("", "repo-*")
	defer os.RemoveAll(dir)
	_ = os.MkdirAll(filepath.Join(dir, ".github"), 0755)
	_ = os.WriteFile(
		filepath.Join(dir, ".github", "copilot-instructions.md"),
		[]byte("# Copilot Instructions\n"),
		0644,
	)

	schema := repogov.DefaultCopilotLayout()
	results, _ := repogov.CheckLayout(dir, schema)
	fmt.Println(repogov.LayoutPassed(results))
	// Output:
	// true
}

func ExampleLayoutPassed() {
	results := []repogov.LayoutResult{
		{Status: repogov.Pass},
		{Status: repogov.Info},
	}
	fmt.Println(repogov.LayoutPassed(results))
	// Output: true
}

func ExampleLayoutSummary() {
	results := []repogov.LayoutResult{
		{
			Path:    ".github/copilot-instructions.md",
			Status:  repogov.Pass,
			Message: "required file present",
		},
	}
	fmt.Print(repogov.LayoutSummary(results))
	// Output:
	// [PASS] .github/copilot-instructions.md -- required file present
	//
	// Layout: 1 checks | 1 pass | 0 warn | 0 fail | 0 info
}

func ExampleDefaultCopilotLayout() {
	schema := repogov.DefaultCopilotLayout()
	fmt.Println(schema.Root)
	fmt.Println(schema.Required[0])
	// Output:
	// .github
	// copilot-instructions.md
}

func ExampleValidateConfig() {
	cfg := repogov.Config{Default: -1}
	violations := repogov.ValidateConfig(cfg)
	fmt.Println(violations[0].Field)
	fmt.Println(violations[0].Severity)
	// Output:
	// default
	// error
}
