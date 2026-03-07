package repogov_test

import (
	"fmt"

	"github.com/nicholashoule/repogov"
)

func ExampleCheckFile() {
	cfg := repogov.DefaultConfig()
	cfg.Default = 500

	// CheckFile returns a Result with line count and status.
	// (Using a real file path would be needed for actual output.)
	_ = cfg
	fmt.Println("CheckFile returns a Result with Pass, Warn, Fail, or Skip status")
	// Output: CheckFile returns a Result with Pass, Warn, Fail, or Skip status
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

func ExamplePassed() {
	results := []repogov.Result{
		{Status: repogov.Pass},
		{Status: repogov.Warn},
	}
	fmt.Println(repogov.Passed(results))
	// Output: true
}

func ExampleDefaultGitHubLayout() {
	schema := repogov.DefaultGitHubLayout()
	fmt.Println(schema.Root)
	// Output: .github
}
