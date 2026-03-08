package repogov

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CountLines returns the number of lines in the file at path. It uses
// buffered I/O and does not load the entire file into memory.
func CountLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// CheckFile checks a single file against its resolved line limit.
// It returns a Result with the line count and status.
func CheckFile(path string, cfg Config) (Result, error) { //nolint:gocritic // hugeParam: stable public API
	// Normalize to forward slashes for config lookup.
	relPath := filepath.ToSlash(path)

	limit := ResolveLimit(relPath, cfg)
	if limit == 0 {
		return Result{Path: relPath, Limit: 0, Status: Skip}, nil
	}

	count, err := CountLines(path)
	if err != nil {
		return Result{}, err
	}

	pct := 100 * count / limit
	status := Pass
	warnThreshold := effectiveWarningThreshold(cfg)
	switch {
	case count > limit:
		status = Fail
	case pct >= warnThreshold:
		status = Warn
	}

	action := ""
	switch status {
	case Fail:
		over := count - limit
		action = fmt.Sprintf("over limit by %d lines; reduce or exempt via repogov-config.json", over)
	case Warn:
		action = fmt.Sprintf("at %d%% of %d-line limit; consider refactoring", pct, limit)
	}

	return Result{
		Path:   relPath,
		Lines:  count,
		Limit:  limit,
		Pct:    pct,
		Status: status,
		Action: action,
	}, nil
}

// CheckDir walks a directory tree rooted at root, checks every file
// matching the given extensions (e.g., []string{".md"}), and returns
// results. An empty or nil extensions slice checks all files.
// Directories listed in cfg.SkipDirs are not entered.
func CheckDir(root string, exts []string, cfg Config) ([]Result, error) { //nolint:gocritic // hugeParam: stable public API
	return CheckDirContext(context.Background(), root, exts, cfg)
}

// CheckDirContext is like [CheckDir] but accepts a [context.Context]
// for cancellation support. When the context is canceled, the walk
// stops and CheckDirContext returns the context error along with any
// results collected before cancellation.
func CheckDirContext(ctx context.Context, root string, exts []string, cfg Config) ([]Result, error) { //nolint:gocritic // hugeParam: stable public API
	// Build a set for fast SkipDirs lookup.
	skipSet := make(map[string]bool, len(cfg.SkipDirs))
	for _, d := range cfg.SkipDirs {
		skipSet[d] = true
	}

	// Build a set for extension filtering.
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[strings.ToLower(e)] = true
	}
	filterByExt := len(extSet) > 0

	var results []Result

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if skipSet[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Extension filter.
		if filterByExt {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !extSet[ext] {
				return nil
			}
		}

		// Compute repo-relative path.
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		result, err := CheckFile(path, cfg)
		if err != nil {
			// Skip unreadable files rather than aborting the walk.
			return nil
		}
		result.Path = relPath
		results = append(results, result)
		return nil
	})

	return results, err
}
