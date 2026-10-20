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
// result, and only the sniff prefix is ever read from them. A byte limit
// still applies to them (checked from the file size). Text with a UTF-16 or
// UTF-32 byte order mark is not binary: its lines are counted in the
// decoded code units.
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
	// Only 0 exempts. A negative limit is invalid config and never passes:
	// skipping the count for it would turn a typo into a silent exemption.
	if limit < 0 || byteLimit < 0 {
		return Result{
			Path:      relPath,
			Limit:     limit,
			ByteLimit: byteLimit,
			Status:    Fail,
			Action:    negativeLimitAction(relPath, cfg, limit, byteLimit),
		}, nil
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
	warnThreshold := effectiveWarningThreshold(cfg)

	// UTF-16 and UTF-32 text (common from Windows tools) contains NUL bytes
	// but is text: recognize its byte order mark before the binary sniff so
	// it is line-counted instead of skipped.
	total := int64(n)
	if readErr == nil { // the sniff buffer filled: the file may be longer
		fi, statErr := f.Stat()
		if statErr != nil {
			return Result{}, statErr
		}
		total = fi.Size()
	}
	unit, bigEndian := wideEncoding(sniff[:n], total)
	if unit == 0 && bytes.IndexByte(sniff[:n], 0) >= 0 {
		// Binary file: never line-counted or parsed. Its size still counts
		// against a byte limit (from Stat, without reading further).
		if byteLimit > 0 {
			fi, statErr := f.Stat()
			if statErr != nil {
				return Result{}, statErr // never pass a byte check on an unknown size
			}
			size := clampInt(fi.Size())
			if bStatus, msg := checkByteLimit(size, byteLimit, warnThreshold); bStatus != Pass {
				return Result{
					Path: relPath, Limit: limit, Bytes: size, ByteLimit: byteLimit,
					Status: bStatus, Action: "binary file; " + msg,
				}, nil
			}
		}
		return Result{Path: relPath, Limit: limit, Status: Skip}, nil
	}

	status := Pass
	action := ""
	count, pct := 0, 0

	// Line limit. A limit of 0 exempts the file from the line check only; a
	// byte limit (below) may still apply.
	if limit > 0 {
		var err error
		if unit > 0 {
			count, err = countWideLinesFrom(f, sniff[:n], unit, bigEndian)
		} else {
			count, err = countLinesFrom(f, sniff[:n])
		}
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
		fi, statErr := f.Stat()
		if statErr != nil {
			return Result{}, statErr // never pass a byte check on an unknown size
		}
		size = clampInt(fi.Size())
		if bStatus, msg := checkByteLimit(size, byteLimit, warnThreshold); bStatus != Pass {
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

// negativeLimitAction explains a negative resolved line or byte limit and
// names the config entry it came from, so the fix is unambiguous.
func negativeLimitAction(path string, cfg Config, limit, byteLimit int) string { //nolint:gocritic // hugeParam: intentional value semantics
	var parts []string
	if limit < 0 {
		src := "default"
		if _, ok := cfg.Files[path]; ok {
			src = fmt.Sprintf("files[%q]", path)
		} else {
			for i, r := range cfg.Rules {
				if ruleGlobMatches(r.Glob, path) {
					src = fmt.Sprintf("rules[%d] (glob %q)", i, r.Glob)
					break
				}
			}
		}
		parts = append(parts, fmt.Sprintf("invalid line limit %d from %s", limit, src))
	}
	if byteLimit < 0 {
		src := "max_bytes"
		for i, r := range cfg.MaxBytes {
			if ruleGlobMatches(r.Glob, path) {
				src = fmt.Sprintf("max_bytes[%d] (glob %q)", i, r.Glob)
				break
			}
		}
		parts = append(parts, fmt.Sprintf("invalid byte limit %d from %s", byteLimit, src))
	}
	return strings.Join(parts, "; ") + " in repogov-config.json: limits must be >= 0 (0 exempts the file); fix the entry ('repogov validate' lists every invalid entry)"
}

// checkByteLimit classifies a file size against a positive byte limit and
// returns the status (Pass, Warn or Fail) with a remediation message for
// Warn and Fail.
func checkByteLimit(size, byteLimit, warnThreshold int) (Status, string) {
	bpct := int(100 * int64(size) / int64(byteLimit))
	switch {
	case size > byteLimit:
		return Fail, fmt.Sprintf("over byte limit by %d bytes (%d / %d); trim the file (agents truncate or ignore oversized instructions) or raise max_bytes in repogov-config.json", size-byteLimit, size, byteLimit)
	case bpct >= warnThreshold:
		return Warn, fmt.Sprintf("%d bytes is %d%% of the %d-byte limit; trim the file (agents truncate or ignore oversized instructions) or raise max_bytes in repogov-config.json", size, bpct, byteLimit)
	}
	return Pass, ""
}

// wideEncoding inspects the start of a file for a UTF-32 or UTF-16 byte
// order mark and returns the code-unit size (4 or 2; 0 when there is no such
// BOM) and whether the encoding is big-endian. size is the file's total
// length. FF FE 00 00 is ambiguous: a UTF-32LE BOM, or a UTF-16LE BOM
// followed by U+0000. It is read as UTF-32LE only when size is a whole
// number of 4-byte units, and as UTF-16LE otherwise.
func wideEncoding(prefix []byte, size int64) (unit int, bigEndian bool) {
	switch {
	case bytes.HasPrefix(prefix, []byte{0xFF, 0xFE, 0x00, 0x00}) && size%4 == 0:
		return 4, false
	case bytes.HasPrefix(prefix, []byte{0x00, 0x00, 0xFE, 0xFF}):
		return 4, true
	case bytes.HasPrefix(prefix, []byte{0xFF, 0xFE}):
		return 2, false
	case bytes.HasPrefix(prefix, []byte{0xFE, 0xFF}):
		return 2, true
	}
	return 0, false
}

// countWideLinesFrom counts lines in UTF-16/UTF-32 text (prefix, which starts
// with the BOM, followed by the rest of r) with the same semantics as
// [CountLines]: each U+000A code unit ends a line and a non-empty final line
// without one still counts. unit is 2 or 4 bytes. Reads are chunked, so
// memory use is constant.
func countWideLinesFrom(r io.Reader, prefix []byte, unit int, bigEndian bool) (int, error) {
	count := 0
	sawUnit := false
	lastNewline := true
	var pending []byte
	consume := func(b []byte) {
		pending = append(pending, b...)
		whole := len(pending) - len(pending)%unit
		for i := 0; i < whole; i += unit {
			u := pending[i : i+unit]
			var v uint32
			if bigEndian {
				for _, c := range u {
					v = v<<8 | uint32(c)
				}
			} else {
				for j := unit - 1; j >= 0; j-- {
					v = v<<8 | uint32(u[j])
				}
			}
			sawUnit = true
			lastNewline = v == '\n'
			if lastNewline {
				count++
			}
		}
		pending = append(pending[:0], pending[whole:]...)
	}
	consume(prefix[unit:]) // skip the BOM
	buf := make([]byte, 64*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			consume(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	if sawUnit && !lastNewline {
		count++
	}
	return count, nil
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
