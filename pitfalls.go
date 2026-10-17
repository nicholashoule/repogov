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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
	registry := readRulesRegistry(filepath.Join(filepath.Dir(dir), "rules.json"))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var results []LayoutResult
	checked := 0
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			results = append(results, nestedRuleResults(root, dir, path, registry)...)
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

// nestedRuleResults flags .md files below a rules/ subdirectory: Antigravity
// scans rules/ flat, so they are ignored unless registered in rules.json.
func nestedRuleResults(root, rulesDir, subDir, registry string) []LayoutResult {
	var results []LayoutResult
	_ = filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.ToLower(filepath.Ext(d.Name())) != ".md" || strings.EqualFold(d.Name(), "README.md") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		underRules, _ := filepath.Rel(rulesDir, path)
		if registry != "" && strings.Contains(registry, filepath.ToSlash(underRules)) {
			return nil
		}
		results = append(results, LayoutResult{
			Path:   filepath.ToSlash(rel),
			Status: Warn,
			Message: "Antigravity scans .agents/rules flat, so this nested file is ignored unless registered in .agents/rules.json -- " +
				"FIX: move it directly into .agents/rules/, or add it to .agents/rules.json entries",
		})
		return nil
	})
	return results
}

// readRulesRegistry returns the slash-normalized text of rules.json, or ""
// when it is absent.
func readRulesRegistry(path string) string {
	path = filepath.Clean(path)
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() || info.Size() > pitfallMaxFileBytes {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(data), `\\`, "/")
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
	fm, ok := readFrontmatter(absPath)
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
// complete frontmatter block.
func readFrontmatter(path string) (map[string]string, bool) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, false
	}
	defer f.Close()
	// Only the head of the file is ever parsed; cap the read so a huge or
	// endless file cannot stall the run.
	sc := bufio.NewScanner(io.LimitReader(f, pitfallMaxFileBytes))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	if !sc.Scan() || strings.TrimSpace(strings.TrimPrefix(sc.Text(), string(utf8BOM))) != "---" {
		return nil, false
	}
	out := map[string]string{}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			return out, true
		}
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue // nested or blank: top-level keys only
		}
		if k, v, ok := yamlSplitKV(strings.TrimSpace(yamlStripComment(line))); ok {
			out[k] = yamlUnquote(v)
		}
	}
	return nil, false // never closed
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
