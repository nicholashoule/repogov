package repogov

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// agentHintsKey is the top-level metadata key that carries agent guidance.
const agentHintsKey = "_agent_hints"

// UpgradeConfigHints adds the current "_agent_hints" block to the JSON config
// at path, or refreshes it when an older copy is present, and adds
// "language": "en-US" ([DefaultLanguage]) right after it when the config has
// no language key (an existing value is never changed). It reports whether
// the file changed.
//
// The edit is textual, not a re-marshal: every other byte of the file (key
// order, compact arrays, comments in values, CRLF line endings, indentation)
// is preserved, so the diff is limited to the hints block and the language
// line. Before writing, the result is parsed and its effective [Config]
// compared with the original (Language aside); if they differ, or the result
// is not valid JSON, nothing is written.
//
// YAML configs (.yaml/.yml) are supported too: the hints become a top-level
// "_agent_hints:" mapping of double-quoted strings, which repogov's YAML
// reader skips (_-prefixed keys are metadata) and any YAML tool reads as plain
// data; comments and ordering elsewhere are preserved. Symlinks and special
// files are refused, in line with the rest of repogov's file-writing rules.
func UpgradeConfigHints(path string) (bool, error) {
	return UpgradeConfigWithLanguage(path, "")
}

// UpgradeConfigWithLanguage is like [UpgradeConfigHints] and also manages
// the top-level "language" key, as a textual edit like the hints refresh.
// With a non-empty lang the key is set to lang's canonical tag (see
// [NormalizeLanguage]): added when absent, its value replaced when present.
// With an empty lang the key is added as [DefaultLanguage] ("en-US") when
// absent, and an existing value is never changed. A newly added key goes
// right after the "_agent_hints" block. An unknown lang is an error and
// nothing is written.
func UpgradeConfigWithLanguage(path, lang string) (bool, error) {
	tag, overwrite := DefaultLanguage, false
	if lang != "" {
		overwrite = true
		var ok bool
		if tag, ok = NormalizeLanguage(lang); !ok {
			msg := fmt.Sprintf("unknown language tag %q", lang)
			if near := closestLanguages(lang); len(near) > 0 {
				msg += "; closest known tags: " + strings.Join(near, ", ")
			} else {
				msg += "; use a BCP 47 tag such as en-US, en-GB, es-419 or pt-BR (full list: README language table)"
			}
			return false, errors.New(msg)
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file; refusing to modify", path)
	}
	if info.Size() > maxConfigBytes {
		return false, fmt.Errorf("%s is %d bytes; refusing to edit a config over %d bytes", path, info.Size(), maxConfigBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	upgrade, setLang := upgradeHintsInJSON, setLanguageInJSON
	if isYAML(path) {
		upgrade, setLang = upgradeHintsInYAML, setLanguageInYAML
	}
	// Hints first, so a newly added language key can sit right after them.
	updated, err := upgrade(data)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if updated, err = setLang(updated, tag, overwrite); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if bytes.Equal(updated, data) {
		return false, nil
	}
	if err := writeFileAtomic(path, updated, info.Mode().Perm()); err != nil {
		return false, err
	}
	return true, nil
}

// maxConfigBytes bounds the config files repogov will rewrite; real configs
// are a few KiB.
const maxConfigBytes = 4 << 20

// writeFileAtomic replaces path with data by writing a temporary file in the
// same directory, flushing it to disk, and renaming it over the original, so
// a crash or full disk mid-write can never leave a truncated config behind.
// The original file's permission bits are preserved. A rename that fails
// because another program briefly holds the file open (Windows) is retried;
// see [renameRetrying].
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".repogov-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(perm); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := renameRetrying(os.Rename, isTransientRenameError, time.Sleep, tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// renameRetryBudget bounds the total time renameRetrying waits for another
// program to release the destination file.
const renameRetryBudget = time.Second

// renameRetrying renames from to to with rename, retrying with a doubling
// backoff (10 ms, 20 ms, ...) for at most [renameRetryBudget] while
// transient reports the failure as temporary. On Windows, replacing a file
// that an editor, indexer or antivirus scanner holds open fails with "Access
// is denied" or a sharing violation until the handle closes. A failure that
// outlasts the budget is returned naming the cause and the next step. rename,
// transient and sleep are parameters so the retry logic is testable without
// real file locks.
func renameRetrying(rename func(from, to string) error, transient func(error) bool, sleep func(time.Duration), from, to string) error {
	delay := 10 * time.Millisecond
	var waited time.Duration
	for {
		err := rename(from, to)
		if err == nil || !transient(err) {
			return err
		}
		if waited+delay > renameRetryBudget {
			return fmt.Errorf("replace %s: %w; another program (an editor, indexer or antivirus scanner) is holding the file open -- close it and run the command again", to, err)
		}
		sleep(delay)
		waited += delay
		delay *= 2
	}
}

// upgradeHintsInJSON returns data with the current hints block inserted or
// refreshed. It is the pure core of [UpgradeConfigHints].
func upgradeHintsInJSON(data []byte) ([]byte, error) {
	// Preserve a leading UTF-8 BOM (common from Windows editors) byte-for-byte.
	if bytes.HasPrefix(data, utf8BOM) {
		out, err := upgradeHintsInJSON(bytes.TrimPrefix(data, utf8BOM))
		if err != nil {
			return nil, err
		}
		return append(append([]byte{}, utf8BOM...), out...), nil
	}
	if !json.Valid(data) {
		return nil, errors.New("not valid JSON; fix the syntax first (repogov validate)")
	}
	var before Config
	if err := decodeConfigJSON(data, &before); err != nil {
		return nil, err
	}

	nl := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		nl = "\r\n"
	}
	block := strings.TrimSuffix(agentHints(), ",\n") // "  \"_agent_hints\": {...}"
	if nl != "\n" {
		block = strings.ReplaceAll(block, "\n", nl)
	}

	start, end, found, otherKeys, err := locateTopLevelKey(data, agentHintsKey)
	if err != nil {
		return nil, err
	}

	var out []byte
	if found {
		// Replace "_agent_hints": {...} in place; keep the original indent.
		out = append(out, data[:start]...)
		out = append(out, strings.TrimLeft(block, " ")...)
		out = append(out, data[end:]...)
	} else {
		open := bytes.IndexByte(data, '{')
		insert := nl + block
		if otherKeys {
			insert += ","
		}
		out = append(out, data[:open+1]...)
		out = append(out, insert...)
		out = append(out, data[open+1:]...)
	}

	var after Config
	if !json.Valid(out) || decodeConfigJSON(out, &after) != nil || !reflect.DeepEqual(before, after) {
		return nil, errors.New("internal check failed: hints edit would change the config; no changes made")
	}
	return out, nil
}

// setLanguageInJSON returns data with the top-level "language" key set to
// tag. An absent key is inserted right after the "_agent_hints" member (or
// after the opening brace when there is none); a present one has its value
// replaced only when overwrite is true, and is otherwise left as is. Every
// other byte is preserved and the effective [Config] is verified to differ
// only in Language.
func setLanguageInJSON(data []byte, tag string, overwrite bool) ([]byte, error) {
	if bytes.HasPrefix(data, utf8BOM) {
		out, err := setLanguageInJSON(bytes.TrimPrefix(data, utf8BOM), tag, overwrite)
		if err != nil {
			return nil, err
		}
		return append(append([]byte{}, utf8BOM...), out...), nil
	}
	if !json.Valid(data) {
		return nil, errors.New("not valid JSON; fix the syntax first (repogov validate)")
	}
	var before Config
	if err := decodeConfigJSON(data, &before); err != nil {
		return nil, err
	}
	quoted, err := json.Marshal(tag)
	if err != nil {
		return nil, err
	}
	member := `"language": ` + string(quoted)

	start, end, found, otherKeys, err := locateTopLevelKey(data, "language")
	if err != nil {
		return nil, err
	}
	if found && !overwrite {
		return data, nil
	}
	var out []byte
	if found {
		out = append(out, data[:start]...)
		out = append(out, member...)
		out = append(out, data[end:]...)
	} else {
		nl := "\n"
		if bytes.Contains(data, []byte("\r\n")) {
			nl = "\r\n"
		}
		_, hintsEnd, hasHints, _, err := locateTopLevelKey(data, agentHintsKey)
		if err != nil {
			return nil, err
		}
		if hasHints {
			// "_agent_hints": {...}<here>[,] -- the hints keep their comma.
			out = append(out, data[:hintsEnd]...)
			out = append(out, ","+nl+"  "+member...)
			out = append(out, data[hintsEnd:]...)
		} else {
			insert := nl + "  " + member
			if otherKeys {
				insert += ","
			}
			open := bytes.IndexByte(data, '{')
			out = append(out, data[:open+1]...)
			out = append(out, insert...)
			out = append(out, data[open+1:]...)
		}
	}

	want := before
	want.Language = tag
	var after Config
	if !json.Valid(out) || decodeConfigJSON(out, &after) != nil || !reflect.DeepEqual(want, after) {
		return nil, errors.New("internal check failed: language edit would change other settings; no changes made")
	}
	return out, nil
}

// setLanguageInYAML returns data with a top-level "language: tag" line. An
// absent key is inserted right after the "_agent_hints:" block (or after any
// leading comment header and "---" when there is none); a present one is
// replaced only when overwrite is true, and otherwise left as is. The
// effective [Config] is verified to differ only in Language.
func setLanguageInYAML(data []byte, tag string, overwrite bool) ([]byte, error) {
	if bytes.HasPrefix(data, utf8BOM) {
		out, err := setLanguageInYAML(bytes.TrimPrefix(data, utf8BOM), tag, overwrite)
		if err != nil {
			return nil, err
		}
		return append(append([]byte{}, utf8BOM...), out...), nil
	}
	text := string(data)
	crlf := strings.Contains(text, "\r\n") && !strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\n")
	if crlf {
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	var before Config
	if err := unmarshalYAML([]byte(text), &before); err != nil {
		return nil, fmt.Errorf("not a readable config: %w", err)
	}
	lines := strings.Split(text, "\n")
	line := "language: " + yamlQuote(tag)
	if i := yamlTopLevelKeyLine(lines, "language"); i >= 0 {
		if !overwrite {
			return data, nil
		}
		lines[i] = line
	} else {
		at := yamlHeaderEnd(lines)
		if idx, end := yamlHintsRange(lines); idx >= 0 {
			at = end
		}
		lines = append(lines[:at], append([]string{line}, lines[at:]...)...)
	}
	result := strings.Join(lines, "\n")
	want := before
	want.Language = tag
	var after Config
	if err := unmarshalYAML([]byte(result), &after); err != nil || !reflect.DeepEqual(want, after) {
		return nil, errors.New("internal check failed: language edit would change other settings; no changes made")
	}
	if crlf {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return []byte(result), nil
}

// yamlTopLevelKeyLine returns the index of the unindented line holding key
// (quoted or not), or -1.
func yamlTopLevelKeyLine(lines []string, key string) int {
	for i, l := range lines {
		if strings.HasPrefix(l, " ") {
			continue
		}
		if k, _, ok := yamlSplitKV(strings.TrimRight(yamlStripComment(l), " ")); ok && yamlUnquote(k) == key {
			return i
		}
	}
	return -1
}

// yamlHintsRange returns the line range [idx, end) of the top-level
// "_agent_hints:" block -- the key line plus its indented continuation,
// without trailing blank separator lines -- or idx = -1 when there is none.
func yamlHintsRange(lines []string) (idx, end int) {
	idx = yamlTopLevelKeyLine(lines, agentHintsKey)
	if idx < 0 {
		return -1, 0
	}
	end = idx + 1
	for end < len(lines) && (lines[end] == "" || lines[end][0] == ' ' || lines[end][0] == '\t') {
		end++
	}
	for end > idx+1 && lines[end-1] == "" {
		end--
	}
	return idx, end
}

// yamlHeaderEnd returns the index of the first line after any leading
// comment/blank header and an optional "---" document marker.
func yamlHeaderEnd(lines []string) int {
	at := 0
	for at < len(lines) && (strings.TrimSpace(lines[at]) == "" || strings.HasPrefix(lines[at], "#")) {
		at++
	}
	if at < len(lines) && strings.TrimSpace(lines[at]) == "---" {
		at++
	}
	return at
}

// locateTopLevelKey finds the byte range [start,end) of `"key": value` for a
// top-level key of the JSON object in data. otherKeys reports whether the
// object has any top-level key other than key.
func locateTopLevelKey(data []byte, key string) (start, end int, found, otherKeys bool, err error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if _, err = dec.Token(); err != nil { // opening '{'
		return
	}
	for dec.More() {
		var tok json.Token
		if tok, err = dec.Token(); err != nil {
			return
		}
		name, _ := tok.(string)
		keyEnd := int(dec.InputOffset())
		var raw json.RawMessage
		if err = dec.Decode(&raw); err != nil && err != io.EOF {
			return
		}
		err = nil
		if name != key {
			otherKeys = true
			continue
		}
		quoted, _ := json.Marshal(key)
		start = bytes.LastIndex(data[:keyEnd], quoted)
		end = int(dec.InputOffset())
		found = true
	}
	return
}

// upgradeHintsInYAML returns data with the current "_agent_hints:" block
// inserted or refreshed. Only the hints block changes; the effective [Config]
// is verified identical before returning.
func upgradeHintsInYAML(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, utf8BOM) {
		out, err := upgradeHintsInYAML(bytes.TrimPrefix(data, utf8BOM))
		if err != nil {
			return nil, err
		}
		return append(append([]byte{}, utf8BOM...), out...), nil
	}
	text := string(data)
	crlf := strings.Contains(text, "\r\n") && !strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\n")
	if crlf {
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}

	var before Config
	if err := unmarshalYAML([]byte(text), &before); err != nil {
		return nil, fmt.Errorf("not a readable config: %w", err)
	}

	lines := strings.Split(text, "\n")
	block := strings.Split(agentHintsYAML(), "\n")

	var out []string
	if idx, end := yamlHintsRange(lines); idx >= 0 {
		// Replace the key line plus its indented/blank continuation lines,
		// but leave trailing blank lines (separators) in place.
		out = append(out, lines[:idx]...)
		out = append(out, block...)
		out = append(out, lines[end:]...)
	} else {
		// Insert after any leading comment/blank header and an optional "---".
		at := yamlHeaderEnd(lines)
		out = append(out, lines[:at]...)
		out = append(out, block...)
		out = append(out, lines[at:]...)
	}

	result := strings.Join(out, "\n")
	var after Config
	if err := unmarshalYAML([]byte(result), &after); err != nil || !reflect.DeepEqual(before, after) {
		return nil, errors.New("internal check failed: hints edit would change the config; no changes made")
	}
	if crlf {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	return []byte(result), nil
}
