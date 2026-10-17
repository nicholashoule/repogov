package repogov

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CountLines returns the number of lines in the file at path. It reads the
// file in fixed-size chunks and counts newline bytes, so it uses constant
// memory and -- unlike a bufio.Scanner -- has no maximum line length. A final
// line without a trailing newline still counts as a line, matching the
// semantics of [bufio.Scanner]. Files with lines longer than 64 KiB
// (minified assets, long tables, generated Markdown) previously failed with
// bufio.ErrTooLong and were silently dropped from directory checks; they are
// now counted like any other file.
func CountLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return countLinesFrom(f, nil)
}

// countLinesFrom counts lines in prefix followed by the remaining content
// of r, using the same semantics as [CountLines]. It is the shared counting
// core for CountLines (whole file) and checkFileAt (which has already
// consumed a sniff prefix from the file).
func countLinesFrom(r io.Reader, prefix []byte) (int, error) {
	count := bytes.Count(prefix, []byte{'\n'})
	lastByte := byte('\n') // empty input counts as 0 lines
	if len(prefix) > 0 {
		lastByte = prefix[len(prefix)-1]
	}
	buf := make([]byte, 64*1024)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			count += bytes.Count(buf[:n], []byte{'\n'})
			lastByte = buf[n-1]
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
	}
	// A non-empty final line without a trailing newline is still a line.
	if lastByte != '\n' {
		count++
	}
	return count, nil
}

// binarySniffSize is the number of leading bytes inspected for a NUL byte
// when detecting binary files. This mirrors the heuristic used by Git and
// other text-processing tools.
const binarySniffSize = 512

// sensitiveFileExts lists file extensions holding certificate and key
// material that repogov must never open, line-count, or parse. Files with
// these extensions are skipped before the file is opened, so their content
// is never read at all. Matching is case-insensitive.
var sensitiveFileExts = map[string]bool{
	".crt": true,
	".pem": true,
	".key": true,
	".p12": true,
	".pfx": true,
	".jks": true,
}

// isSensitivePath reports whether path has an extension listed in
// [sensitiveFileExts].
func isSensitivePath(path string) bool {
	return sensitiveFileExts[strings.ToLower(filepath.Ext(path))]
}

// CheckFile checks a single file against its resolved line limit.
// It returns a Result with the line count and status.
// path is used both as the filesystem path for reading the file and,
// after normalizing path separators to forward slashes, as the key for
// config lookups. The normalized path is used as-is; callers should pass
// repository-relative paths (using the same root as CheckDir) if they want
// Config.Files / Config.Rules patterns to match consistently. When called
// from [CheckDir], the internal checkFileAt variant is used instead so that
// the repo-relative path is computed correctly from the walk root.
func CheckFile(path string, cfg Config) (Result, error) { //nolint:gocritic // hugeParam: stable public API
	relPath := filepath.ToSlash(path)
	return checkFileAt(path, relPath, cfg)
}

// checkFileAt is the shared implementation for CheckFile and CheckDirContext.
// fsPath is the filesystem path read for counting; relPath is the
// slash-normalized path key used for config lookups. When invoked from
// CheckDir, relPath is computed as a repo-relative path from the walk root.
//
// Binary and compiled files (detected by a NUL byte within the first 512
// bytes) are never line-counted or otherwise parsed: they produce a Skip
// result, and only the sniff prefix is ever read from them.
func checkFileAt(fsPath, relPath string, cfg Config) (Result, error) { //nolint:gocritic // hugeParam: stable public API
	// Certificate and key material is skipped before the file is opened:
	// its content must never be read, counted, or parsed.
	if isSensitivePath(fsPath) {
		return Result{Path: relPath, Limit: 0, Status: Skip}, nil
	}

	limit := ResolveLimit(relPath, cfg)
	byteLimit := ResolveByteLimit(relPath, cfg)
	if limit == 0 && byteLimit == 0 {
		return Result{Path: relPath, Limit: 0, Status: Skip}, nil
	}

	// Only regular files are opened. A symlink could point at a secret
	// outside the repo (its content would be read to count lines) and a FIFO
	// or device would block the walk indefinitely.
	if info, err := os.Lstat(fsPath); err == nil && !info.Mode().IsRegular() {
		return Result{
			Path:   relPath,
			Limit:  limit,
			Status: Skip,
			Action: "not a regular file (symlink or special file); not opened",
		}, nil
	}

	f, err := os.Open(fsPath)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	sniff := make([]byte, binarySniffSize)
	n, readErr := io.ReadFull(f, sniff)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return Result{}, readErr
	}
	if bytes.IndexByte(sniff[:n], 0) >= 0 {
		// Binary file: report Skip without reading any further content.
		return Result{Path: relPath, Limit: limit, Status: Skip}, nil
	}

	status := Pass
	action := ""
	count, pct := 0, 0
	warnThreshold := effectiveWarningThreshold(cfg)

	// Line limit. A limit of 0 exempts the file from the line check only; a
	// byte limit (below) may still apply.
	if limit > 0 {
		var err error
		count, err = countLinesFrom(f, sniff[:n])
		if err != nil {
			return Result{}, err
		}
		pct = 100 * count / limit
		switch {
		case count > limit:
			status = Fail
			action = fmt.Sprintf("over limit by %d lines; reduce or exempt via repogov-config.json", count-limit)
		case pct >= warnThreshold:
			status = Warn
			action = fmt.Sprintf("at %d%% of %d-line limit; consider refactoring", pct, limit)
		}
	}

	// Byte limit: some agents truncate or drop instruction files by size, not
	// by lines (e.g. Antigravity truncates rule files over 24,000 bytes), so a
	// file inside its line limit can still be cut off.
	size := 0
	if byteLimit > 0 {
		if fi, statErr := f.Stat(); statErr == nil {
			size = clampInt(fi.Size())
		}
		bpct := int(100 * int64(size) / int64(byteLimit))
		var bStatus Status
		switch {
		case size > byteLimit:
			bStatus = Fail
		case bpct >= warnThreshold:
			bStatus = Warn
		}
		if bStatus != Pass {
			msg := fmt.Sprintf("%d bytes is %d%% of the %d-byte limit; trim the file (agents truncate or ignore oversized instructions) or raise max_bytes in repogov-config.json", size, bpct, byteLimit)
			if bStatus == Fail {
				msg = fmt.Sprintf("over byte limit by %d bytes (%d / %d); trim the file (agents truncate or ignore oversized instructions) or raise max_bytes in repogov-config.json", size-byteLimit, size, byteLimit)
			}
			if action != "" {
				action += "; "
			}
			action += msg
			if bStatus > status {
				status = bStatus
			}
		}
	}

	if limit == 0 && status == Pass {
		// Line-exempt and within its byte limit: nothing to report.
		return Result{Path: relPath, Status: Skip}, nil
	}

	return Result{
		Path:      relPath,
		Lines:     count,
		Limit:     limit,
		Pct:       pct,
		Bytes:     size,
		ByteLimit: byteLimit,
		Status:    status,
		Action:    action,
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

		result, err := checkFileAt(path, relPath, cfg)
		if err != nil {
			// Do not abort the walk, but never drop the file silently: an
			// unreadable file must surface so Passed() cannot be vacuously true.
			results = append(results, Result{
				Path:   relPath,
				Status: Fail,
				Action: fmt.Sprintf("unreadable: %v; fix permissions or add to skip_dirs/files with limit 0", err),
			})
			return nil
		}
		results = append(results, result)
		return nil
	})

	return results, err
}

// clampInt converts a file size to int without overflow on 32-bit platforms.
func clampInt(n int64) int {
	const maxInt = int64(^uint(0) >> 1)
	if n > maxInt {
		return int(maxInt)
	}
	return int(n)
}
