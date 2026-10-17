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
)

// agentHintsKey is the top-level metadata key that carries agent guidance.
const agentHintsKey = "_agent_hints"

// UpgradeConfigHints adds the current "_agent_hints" block to the JSON config
// at path, or refreshes it when an older copy is present. It reports whether
// the file changed.
//
// The edit is textual, not a re-marshal: every other byte of the file (key
// order, compact arrays, comments in values, CRLF line endings, indentation)
// is preserved, so the diff is limited to the hints block. Before writing,
// the result is parsed and its effective [Config] compared with the original;
// if they differ, or the result is not valid JSON, nothing is written.
//
// YAML configs (.yaml/.yml) are supported too: the hints become a top-level
// "_agent_hints:" mapping of double-quoted strings, which repogov's YAML
// reader skips (unknown keys are ignored) and any YAML tool reads as plain
// data; comments and ordering elsewhere are preserved. Symlinks and special
// files are refused, in line with the rest of repogov's file-writing rules.
func UpgradeConfigHints(path string) (bool, error) {
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
	upgrade := upgradeHintsInJSON
	if isYAML(path) {
		upgrade = upgradeHintsInYAML
	}
	updated, err := upgrade(data)
	if err != nil {
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
// same directory and renaming it over the original, so a crash or full disk
// mid-write can never leave a truncated config behind. The original file's
// permission bits are preserved.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".repogov-config-*.tmp")
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
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
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
	if err := json.Unmarshal(data, &before); err != nil {
		return nil, err
	}

	nl := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		nl = "\r\n"
	}
	block := strings.TrimSuffix(defaultAgentHints, ",\n") // "  \"_agent_hints\": {...}"
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
	if !json.Valid(out) || json.Unmarshal(out, &after) != nil || !reflect.DeepEqual(before, after) {
		return nil, errors.New("internal check failed: hints edit would change the config; no changes made")
	}
	return out, nil
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

	idx := -1
	for i, l := range lines {
		if strings.HasPrefix(l, agentHintsKey+":") {
			idx = i
			break
		}
	}
	var out []string
	if idx >= 0 {
		// Replace the key line plus its indented/blank continuation lines,
		// but leave trailing blank lines (separators) in place.
		end := idx + 1
		for end < len(lines) && (lines[end] == "" || lines[end][0] == ' ' || lines[end][0] == '\t') {
			end++
		}
		for end > idx+1 && lines[end-1] == "" {
			end--
		}
		out = append(out, lines[:idx]...)
		out = append(out, block...)
		out = append(out, lines[end:]...)
	} else {
		// Insert after any leading comment/blank header and an optional "---".
		at := 0
		for at < len(lines) && (strings.TrimSpace(lines[at]) == "" || strings.HasPrefix(lines[at], "#")) {
			at++
		}
		if at < len(lines) && strings.TrimSpace(lines[at]) == "---" {
			at++
		}
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
