// Package repogov enforces repository governance conventions: line-count
// limits on text files and directory-structure validation for GitHub and
// AI-agent repositories.
//
// # Line-Count Limits
//
// [Config] defines per-file, glob-based, and default line-count limits.
// [LoadConfig] reads a JSON configuration file; [DefaultConfig] provides
// sensible defaults when no file is present.
//
//   - [CheckFile] checks a single file against its resolved limit.
//   - [CheckDir] walks a directory tree and checks every file matching
//     the given extensions.
//   - [CheckDirContext] is like [CheckDir] but accepts a [context.Context]
//     for cancellation support.
//   - [ResolveLimit] returns the effective limit for a path given the config.
//   - [CountLines] counts lines using buffered I/O without loading the
//     entire file into memory.
//
// # Layout Governance
//
// [LayoutSchema] declares the expected directory structure for a repository's
// platform directory (e.g., .github/ or .cursor/).
//
//   - [DefaultCopilotLayout] returns the built-in schema matching
//     each platform's conventions.
//   - [CheckLayout] validates a directory against a schema and returns
//     [LayoutResult] entries for required, optional, and unexpected files.
//   - [CheckLayoutContext] is like [CheckLayout] with cancellation support.
//
// # Output Helpers
//
//   - [Passed] and [LayoutPassed] report whether all checks succeeded.
//   - [Summary] and [LayoutSummary] produce human-readable reports.
//
// Typical usage:
//
//	cfg, _ := repogov.LoadConfig(".github/repogov-config.json")
//	results, _ := repogov.CheckDir(".", []string{".md"}, cfg)
//	if !repogov.Passed(results) {
//	    fmt.Fprint(os.Stderr, repogov.Summary(results))
//	    os.Exit(1)
//	}
package repogov
