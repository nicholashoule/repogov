package repogov

// pitfalls.go detects agent configuration that is accepted silently but has
// no effect: files an agent discards without any error, so the author
// believes a rule is active when it is not. Every finding carries a
// "-- FIX:" hint. The checks are read-only and report only documented vendor
// behavior (see docs/compliance).
//
// Covered today:
//   - Cursor: project rules must be .mdc; a plain .md in .cursor/rules is
//     ignored.
//   - Antigravity: every .md in .agents/rules/ needs YAML frontmatter with a
//     valid snake_case trigger or the rule is silently discarded; the
//     directory is scanned flat, so nested files are ignored unless
//     registered in .agents/rules.json.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// antigravityTriggers are the only trigger values Antigravity accepts.
var antigravityTriggers = map[string]bool{
	"always_on":      true,
	"model_decision": true,
	"glob":           true,
	"manual":         true,
}

// pitfallSkipDirs are never walked when searching for rule directories:
// version-control internals and dependency/build output, which can be huge
// and never hold hand-written agent rules.
var pitfallSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "target": true,
	"dist": true, "build": true, "out": true, ".venv": true, "venv": true,
	"__pycache__": true, ".tox": true, ".gradle": true, ".next": true,
}

// pitfallMaxDepth bounds the search for .agents/rules directories so a deep
// or adversarial tree cannot make every layout run walk the whole disk.
const pitfallMaxDepth = 8

// pitfallMaxFileBytes caps how much of any one file the pitfall checks read.
const pitfallMaxFileBytes = 1 << 20

// CheckAgentPitfalls reports agent configuration that is silently ignored
// by the agent that owns it. Directories that do not exist produce no
// results. It never modifies the repository.
func CheckAgentPitfalls(root string) []LayoutResult {
	var results []LayoutResult
	results = append(results, cursorPlainMarkdownResults(root)...)
	results = append(results, antigravityRuleResults(root)...)
	return results
}

// cursorPlainMarkdownResults flags plain .md files under .cursor/rules.
// README.md is exempt: it documents the directory and is not a rule.
func cursorPlainMarkdownResults(root string) []LayoutResult {
	base := filepath.Join(root, ".cursor", "rules")
	if info, err := os.Lstat(base); err != nil || !info.IsDir() {
		return nil
	}
	var results []LayoutResult
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.EqualFold(name, "README.md") || strings.ToLower(filepath.Ext(name)) != ".md" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		results = append(results, LayoutResult{
			Path:   filepath.ToSlash(rel),
			Status: Warn,
			Message: "Cursor ignores plain .md files in .cursor/rules (project rules must use the .mdc extension) -- " +
				"FIX: rename to " + strings.TrimSuffix(name, filepath.Ext(name)) +
				".mdc with frontmatter (description, globs, alwaysApply), or move the guidance into AGENTS.md",
		})
		return nil
	})
	return results
}

// antigravityRuleResults validates every .agents/rules (and legacy
// .agent/rules) directory found under root.
func antigravityRuleResults(root string) []LayoutResult {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path != root {
			if pitfallSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(root, path); relErr == nil && strings.Count(filepath.ToSlash(rel), "/") >= pitfallMaxDepth {
				return filepath.SkipDir
			}
		}
		if d.Name() == "rules" {
			parent := filepath.Base(filepath.Dir(path))
			if parent == ".agents" || parent == ".agent" {
				dirs = append(dirs, path)
			}
		}
		return nil
	})
	sort.Strings(dirs)

	var results []LayoutResult
	for _, dir := range dirs {
		results = append(results, checkAntigravityRulesDir(root, dir)...)
	}
	return results
}

func checkAntigravityRulesDir(root, dir string) []LayoutResult {
	relDir, _ := filepath.Rel(root, dir)
	relDir = filepath.ToSlash(relDir)
	registryPath := filepath.Join(filepath.Dir(dir), "rules.json")
	registry, problem := readRulesRegistry(registryPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var results []LayoutResult
	if problem != "" {
		relReg, _ := filepath.Rel(root, registryPath)
		results = append(results, LayoutResult{
			Path:    filepath.ToSlash(relReg),
			Status:  Warn,
			Message: problem + ", so no nested rule counts as registered -- FIX: make it a regular JSON file with only inherits and entries (path, exclude, include_only)",
		})
	}
	checked := 0
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			nested, n := nestedRuleResults(root, dir, path, registry, problem != "")
			results = append(results, nested...)
			checked += n
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != ".md" || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		checked++
		results = append(results, checkAntigravityRuleFile(path, relDir+"/"+e.Name())...)
	}
	results = append(results, LayoutResult{
		Path:    relDir,
		Status:  Info,
		Message: fmt.Sprintf("%d Antigravity rule file(s) checked", checked),
	})
	return results
}

// nestedRuleResults checks .md files below a rules/ subdirectory. Antigravity
// scans rules/ flat, so an unregistered nested file is ignored and gets a
// location warning. A file registered in rules.json is loaded, so it is
// validated like a top-level rule (registration waives only the location
// warning). checked is the number of registered files validated.
// registryBroken notes that rules.json exists but could not be used.
func nestedRuleResults(root, rulesDir, subDir string, registry *rulesRegistry, registryBroken bool) (results []LayoutResult, checked int) {
	agentsDir := filepath.Dir(rulesDir)
	_ = filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(d.Name())) != ".md" || strings.EqualFold(d.Name(), "README.md") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if registry.registers(agentsDir, path) {
			checked++
			results = append(results, checkAntigravityRuleFile(path, rel)...)
			return nil
		}
		msg := "Antigravity scans .agents/rules flat, so this nested file is ignored unless registered in .agents/rules.json"
		if registryBroken {
			msg += " (rules.json could not be used; see its finding)"
		} else if registry != nil && len(registry.Inherits) > 0 {
			msg += " (inherited registries are not followed)"
		}
		underRules, _ := filepath.Rel(rulesDir, path)
		results = append(results, LayoutResult{
			Path:   rel,
			Status: Warn,
			Message: msg + " -- FIX: move it directly into .agents/rules/, or register it in .agents/rules.json " +
				`(e.g. {"entries": [{"path": "rules", "include_only": [` + jsonString(filepath.ToSlash(underRules)) + `]}]})`,
		})
		return nil
	})
	return results, checked
}

// rulesRegistry is Antigravity's .agents/rules.json: entries register rule
// directories (path, relative to .agents/) beyond the flat rules/ scan,
// optionally narrowed by include_only and exclude; inherits names other
// registries.
type rulesRegistry struct {
	Inherits []struct {
		Path string `json:"path"`
	} `json:"inherits"`
	Entries []rulesRegistryEntry `json:"entries"`
}

// rulesRegistryEntry is one entries item of a [rulesRegistry].
type rulesRegistryEntry struct {
	Path        string   `json:"path"`
	Exclude     []string `json:"exclude"`
	IncludeOnly []string `json:"include_only"`
}

// readRulesRegistry parses rules.json at path. It returns (nil, "") when the
// file is absent, and (nil, reason) when it exists but cannot be used: not a
// regular file, too large, unreadable, not valid JSON, or holding a key whose
// meaning repogov does not know (it could change what is registered). A
// registry that cannot be used registers nothing.
func readRulesRegistry(path string) (*rulesRegistry, string) {
	path = filepath.Clean(path)
	info, err := os.Lstat(path)
	switch {
	case os.IsNotExist(err):
		return nil, ""
	case err != nil:
		return nil, fmt.Sprintf("rules.json cannot be read: %v", err)
	case !info.Mode().IsRegular():
		return nil, "rules.json is not a regular file (symlink or special file); not read"
	case info.Size() > pitfallMaxFileBytes:
		return nil, fmt.Sprintf("rules.json is over %d bytes; not read", pitfallMaxFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Sprintf("rules.json cannot be read: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, utf8BOM)))
	dec.DisallowUnknownFields()
	var reg rulesRegistry
	if err := dec.Decode(&reg); err != nil {
		return nil, fmt.Sprintf("rules.json is not a valid registry (%v)", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, "rules.json is not a valid registry (data after the top-level object)"
	}
	return &reg, ""
}

// registers reports whether the registry loads the rule file at absPath.
// agentsDir is the .agents directory that holds rules.json; entry paths
// resolve against it. An entry registers the files below its directory,
// narrowed to its include_only items when it has any, minus its exclude
// items. Matching is literal (no globs) and fails secure: an include_only
// item must name the file or one of its parent directories exactly (an item
// ending in "/" is a directory), while an exclude item without a slash also
// excludes any file or directory of that name at any depth. An entry whose
// path names the file itself is not a documented form and registers nothing.
func (r *rulesRegistry) registers(agentsDir, absPath string) bool {
	if r == nil {
		return false
	}
	for _, e := range r.Entries {
		p := strings.TrimSpace(strings.ReplaceAll(e.Path, `\`, "/"))
		if p == "" {
			continue
		}
		base := filepath.FromSlash(p)
		if !filepath.IsAbs(base) {
			base = filepath.Join(agentsDir, base)
		}
		rel, err := filepath.Rel(base, absPath)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		rel = filepath.ToSlash(rel)
		if len(e.IncludeOnly) > 0 && !registryItemsMatch(e.IncludeOnly, rel, false) {
			continue
		}
		if registryItemsMatch(e.Exclude, rel, true) {
			continue
		}
		return true
	}
	return false
}

// registryItemsMatch reports whether any include_only or exclude item
// matches rel (slash-separated, relative to the entry's directory). An item
// matches the exact path or, as a directory, any path below it. With broad
// (used for exclude), an item without an inner slash also matches any path
// segment of that name.
func registryItemsMatch(items []string, rel string, broad bool) bool {
	segs := strings.Split(rel, "/")
	for _, it := range items {
		it = strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(it, `\`, "/")), "./")
		dir := strings.TrimSuffix(it, "/")
		if dir == "" {
			continue
		}
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
		if broad && !strings.Contains(dir, "/") && slices.Contains(segs, dir) {
			return true
		}
	}
	return false
}

// checkAntigravityRuleFile validates one rule file's frontmatter.
func checkAntigravityRuleFile(absPath, rel string) []LayoutResult {
	// Never open anything but a regular file: a symlink could point at a
	// secret outside the repo (its first lines would be parsed and echoed in
	// the finding), and a FIFO or device would block the run.
	if info, err := os.Lstat(absPath); err != nil || !info.Mode().IsRegular() {
		return []LayoutResult{{
			Path:    rel,
			Status:  Warn,
			Message: "not a regular file (symlink or special file); content not read -- FIX: replace with a regular rule file",
		}}
	}
	fm, ok, err := readFrontmatter(absPath)
	if err != nil {
		return []LayoutResult{{
			Path:    rel,
			Status:  Fail,
			Message: "frontmatter could not be read (" + err.Error() + "); the rule was not checked -- FIX: make the file readable, with no line over 1 MiB",
		}}
	}
	if !ok {
		return []LayoutResult{{
			Path:   rel,
			Status: Fail,
			Message: "no YAML frontmatter: Antigravity silently discards this rule -- " +
				"FIX: start the file with a '---' block containing trigger: always_on|model_decision|glob|manual",
		}}
	}
	trigger := fm["trigger"]
	switch {
	case trigger == "":
		return []LayoutResult{{
			Path:   rel,
			Status: Fail,
			Message: "frontmatter has no trigger: Antigravity silently discards this rule -- " +
				"FIX: add trigger: always_on|model_decision|glob|manual",
		}}
	case !antigravityTriggers[trigger]:
		hint := "use one of always_on, model_decision, glob, manual (snake_case)"
		if alt := snakeCase(trigger); antigravityTriggers[alt] {
			hint = "use " + alt
		}
		return []LayoutResult{{
			Path:    rel,
			Status:  Fail,
			Message: fmt.Sprintf("unrecognized trigger %q: Antigravity silently discards this rule -- FIX: %s", trigger, hint),
		}}
	}
	var results []LayoutResult
	if trigger == "model_decision" && fm["description"] == "" {
		results = append(results, LayoutResult{
			Path:    rel,
			Status:  Warn,
			Message: "trigger model_decision needs a description for the model to decide on -- FIX: add description: \"...\"",
		})
	}
	if trigger == "glob" && fm["globs"] == "" && fm["glob"] == "" {
		results = append(results, LayoutResult{
			Path:    rel,
			Status:  Fail,
			Message: "trigger glob requires globs (or glob) -- FIX: add globs: \"*.ts, *.tsx\"",
		})
	}
	return results
}

// readFrontmatter returns the top-level scalar keys of a file's leading YAML
// frontmatter block with quotes removed. ok is false when the file has no
// complete frontmatter block; err is set when the file could not be read to
// the end of that block (unopenable, a read error, a line over the 1 MiB
// scanner bound). A read that stops early is an error, never "no
// frontmatter": the rule must be reported as unchecked, with its cause.
func readFrontmatter(path string) (fm map[string]string, ok bool, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	// Only the head of the file is ever parsed; cap the read so a huge or
	// endless file cannot stall the run.
	return parseFrontmatter(io.LimitReader(f, pitfallMaxFileBytes))
}

// parseFrontmatter is readFrontmatter over an open reader.
func parseFrontmatter(r io.Reader) (fm map[string]string, ok bool, err error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	if !sc.Scan() {
		return nil, false, sc.Err()
	}
	if strings.TrimSpace(strings.TrimPrefix(sc.Text(), string(utf8BOM))) != "---" {
		return nil, false, nil
	}
	out := map[string]string{}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			return out, true, nil
		}
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue // nested or blank: top-level keys only
		}
		if k, v, kvOK := yamlSplitKV(strings.TrimSpace(yamlStripComment(line))); kvOK {
			out[yamlUnquote(k)] = yamlUnquote(v) // "trigger": is the key trigger
		}
	}
	if err := sc.Err(); err != nil {
		return nil, false, err
	}
	return nil, false, nil // never closed
}

// snakeCase converts camelCase (alwaysOn) to snake_case (always_on).
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}
