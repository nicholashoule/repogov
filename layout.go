package repogov

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LayoutSchema defines the expected directory structure for a repository's
// platform directory (e.g., .github/ or .cursor/).
type LayoutSchema struct {
	// Root is the directory to validate, relative to the repo root
	// (e.g., ".github" or ".cursor").
	Root string

	// Required lists files that must exist within Root.
	// Paths are relative to Root using forward slashes.
	Required []string

	// Optional lists known files that may or may not exist.
	// Their absence does not produce a warning or failure.
	Optional []string

	// Dirs defines expectations for subdirectories within Root.
	// Keys are directory names; values specify glob patterns and
	// minimum file counts.
	Dirs map[string]DirRule

	// Naming enforces filename conventions within Root.
	Naming NamingRule
}

// DirRule defines expectations for a subdirectory within [LayoutSchema.Root].
type DirRule struct {
	// Glob is a filepath.Match pattern for valid filenames within the directory.
	Glob string `json:"glob"`

	// Min is the minimum required file count. Zero means no minimum.
	Min int `json:"min"`

	// Description is a human-readable purpose for the directory.
	Description string `json:"description"`

	// NoCreate prevents this directory from being created by 'repogov init'.
	// The directory is still recognized and checked by layout commands.
	NoCreate bool `json:"no_create,omitempty"`
}

// NamingRule enforces filename conventions within a [LayoutSchema].
type NamingRule struct {
	// Case is the enforced case rule: "lowercase", "uppercase", or "any".
	// Empty string defaults to "any".
	//
	// When Case is "lowercase", files are expected to use kebab-case or
	// snake_case for multi-word names (e.g., developer-guide.md). Paths
	// listed in Exceptions are exempt from this rule.
	Case string `json:"case"`

	// Exceptions lists paths (relative to Root) exempt from the case rule.
	// These are typically platform-mandated names like ISSUE_TEMPLATE or
	// PULL_REQUEST_TEMPLATE.md, or convention patterns like *_AUDIT.md.
	Exceptions []string `json:"exceptions"`
}

// LayoutResult holds the outcome of a layout check for a single item.
type LayoutResult struct {
	// Path is the repo-relative path using forward slashes.
	Path string

	// Status is the check outcome: Pass, Warn, Fail, or Info.
	Status Status

	// Message is a human-readable explanation of the result.
	Message string
}

// CheckLayout validates a repository's directory structure against a
// [LayoutSchema] and returns results for every required, optional, and
// unexpected file found. The root parameter is the path to the repo root
// (the parent of schema.Root).
func CheckLayout(root string, schema LayoutSchema) ([]LayoutResult, error) { //nolint:gocritic // hugeParam: LayoutSchema is part of the public API; changing to pointer would be a breaking change
	return CheckLayoutContext(context.Background(), root, schema)
}

// CheckLayoutContext is like [CheckLayout] but accepts a [context.Context]
// for cancellation support.
func CheckLayoutContext(ctx context.Context, root string, schema LayoutSchema) ([]LayoutResult, error) { //nolint:gocritic // hugeParam: LayoutSchema is part of the public API; changing to pointer would be a breaking change
	var results []LayoutResult

	layoutDir := filepath.Join(root, filepath.FromSlash(schema.Root))

	// Check if the root layout directory exists.
	if _, err := os.Stat(layoutDir); os.IsNotExist(err) {
		results = append(results, LayoutResult{
			Path:    schema.Root,
			Status:  Fail,
			Message: "directory does not exist: " + schema.Root + " -- FIX: run 'repogov init' to scaffold the structure",
		})
		return results, nil
	} else if err != nil {
		return nil, err
	}

	// Build exception set for naming checks.
	exceptionSet := make(map[string]bool, len(schema.Naming.Exceptions))
	for _, e := range schema.Naming.Exceptions {
		exceptionSet[e] = true
	}

	// Track which required files are found.
	requiredFound := make(map[string]bool, len(schema.Required))
	for _, r := range schema.Required {
		requiredFound[r] = false
	}

	// Build optional set for quick lookup.
	optionalSet := make(map[string]bool, len(schema.Optional))
	for _, o := range schema.Optional {
		optionalSet[o] = true
	}

	// Track files found per managed directory.
	dirFileCounts := make(map[string]int)

	// Collect all files under the layout directory.
	var allFiles []string
	err := filepath.WalkDir(layoutDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Never walk git internals regardless of schema, whether .git is a
		// directory or a file (e.g., worktree gitdir pointer).
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(layoutDir, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)
		allFiles = append(allFiles, relPath)
		return nil
	})
	if err != nil {
		return results, err
	}

	// Classify each file.
	for _, relPath := range allFiles {
		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = ""
		}
		name := filepath.Base(relPath)

		// Silently skip .gitkeep files in managed directories; they are
		// infrastructure created by 'repogov init' and should not trigger
		// unexpected-file warnings.
		if name == ".gitkeep" {
			continue
		}

		// Count toward managed directory regardless of required/optional status.
		if dir != "" {
			if rule, ok := schema.Dirs[dir]; ok {
				if rule.Glob == "" || matchGlob(rule.Glob, name) {
					dirFileCounts[dir]++
				}
			}
		}

		// Check required.
		if _, isRequired := requiredFound[relPath]; isRequired {
			requiredFound[relPath] = true
			results = append(results, LayoutResult{
				Path:    filepath.ToSlash(filepath.Join(schema.Root, relPath)),
				Status:  Pass,
				Message: "required file present",
			})
			continue
		}

		// Check optional.
		if optionalSet[relPath] {
			results = append(results, LayoutResult{
				Path:    filepath.ToSlash(filepath.Join(schema.Root, relPath)),
				Status:  Info,
				Message: "optional file present",
			})
			continue
		}

		// Check if it belongs to a managed directory.
		matched := false
		if dir != "" {
			if rule, ok := schema.Dirs[dir]; ok {
				if rule.Glob == "" || matchGlob(rule.Glob, name) {
					matched = true
				}
			}
		}

		if !matched {
			results = append(results, LayoutResult{
				Path:    filepath.ToSlash(filepath.Join(schema.Root, relPath)),
				Status:  Warn,
				Message: "unexpected file -- not in required, optional, or managed dirs",
			})
		}

		// Check naming conventions.
		if schema.Naming.Case != "" && schema.Naming.Case != "any" {
			if !exceptionSet[relPath] && !exceptionSet[dir] && !exceptionSet[name] {
				if violation := checkNaming(name, schema.Naming.Case); violation != "" {
					results = append(results, LayoutResult{
						Path:    filepath.ToSlash(filepath.Join(schema.Root, relPath)),
						Status:  Fail,
						Message: violation + " -- FIX: rename to " + strings.ToLower(name),
					})
				}
			}
		}
	}

	// Report missing required files.
	for _, r := range schema.Required {
		if !requiredFound[r] {
			results = append(results, LayoutResult{
				Path:    filepath.ToSlash(filepath.Join(schema.Root, r)),
				Status:  Fail,
				Message: "missing required file -- FIX: create " + filepath.ToSlash(filepath.Join(schema.Root, r)) + " or run 'repogov init'",
			})
		}
	}

	// Check subdirectory minimum counts.
	for dirName, rule := range schema.Dirs {
		if rule.Min > 0 {
			count := dirFileCounts[dirName]
			if count < rule.Min {
				results = append(results, LayoutResult{
					Path:    filepath.ToSlash(filepath.Join(schema.Root, dirName)),
					Status:  Fail,
					Message: formatDirMinMessage(dirName, rule, count),
				})
			} else {
				results = append(results, LayoutResult{
					Path:    filepath.ToSlash(filepath.Join(schema.Root, dirName)),
					Status:  Pass,
					Message: formatDirPassMessage(dirName, rule, count),
				})
			}
		}
	}

	return results, nil
}

// matchGlob wraps filepath.Match with forward-slash normalization.
func matchGlob(pattern, name string) bool {
	ok, _ := filepath.Match(pattern, name)
	return ok
}

// checkNaming returns a violation message if name does not conform to
// the given case rule ("lowercase" or "uppercase"). Returns empty string
// if the name conforms.
func checkNaming(name string, caseRule string) string {
	switch caseRule {
	case "lowercase":
		if name != strings.ToLower(name) {
			return "naming violation: expected lowercase"
		}
	case "uppercase":
		if name != strings.ToUpper(name) {
			return "naming violation: expected uppercase"
		}
	}
	return ""
}

// formatDirMinMessage returns a message for a directory that did not meet
// its minimum file count.
func formatDirMinMessage(dir string, rule DirRule, count int) string {
	desc := rule.Description
	if desc == "" {
		desc = dir
	}
	return strings.Join([]string{
		desc, " -- ",
		intToStr(count), " file(s) matching ", rule.Glob,
		" (min: ", intToStr(rule.Min), ")",
	}, "")
}

// formatDirPassMessage returns a message for a directory that met its
// minimum file count.
func formatDirPassMessage(dir string, rule DirRule, count int) string {
	desc := rule.Description
	if desc == "" {
		desc = dir
	}
	return strings.Join([]string{
		desc, " -- ",
		intToStr(count), " file(s) matching ", rule.Glob,
		" (min: ", intToStr(rule.Min), ")",
	}, "")
}

// intToStr converts an int to its string representation.
func intToStr(n int) string {
	return strconv.Itoa(n)
}
