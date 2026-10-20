package repogov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// conditionalAlternatives lists [LayoutSchema.Alternatives] pairings, keyed
// by repo-relative {required, alternative} paths, that satisfy the
// requirement only when the owning agent is configured to read the
// alternative. Each condition returns "" when it holds, or the reason it does
// not (quoted in the missing-file message, so the refusal says why).
var conditionalAlternatives = map[[2]string]func(root string) string{
	{"GEMINI.md", "AGENTS.md"}: geminiReadsAgentsMd,
}

// geminiSettingsMaxBytes caps how much of .gemini/settings.json is read.
const geminiSettingsMaxBytes = 1 << 20

// geminiReadsAgentsMd reports whether Gemini CLI is configured, for this
// repository, to load AGENTS.md as a context file: the project settings file
// .gemini/settings.json must set context.fileName (a string or a list of
// strings) to include "AGENTS.md". Gemini CLI's default context file is
// GEMINI.md only. Antigravity reads AGENTS.md without any setting, but repogov
// cannot tell which CLI a contributor runs, so the fallback is honored only
// when it holds for both. A user-level ~/.gemini/settings.json is not
// consulted: it is invisible to other contributors and to CI. A missing,
// non-regular, unreadable, oversized or malformed settings file means no
// fallback; the returned reason says which.
func geminiReadsAgentsMd(root string) string {
	const need = `Gemini CLI reads it only when .gemini/settings.json lists "AGENTS.md" in context.fileName`
	dir := filepath.Join(root, ".gemini")
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
		return need + " and there is no .gemini/ directory"
	}
	p := filepath.Join(dir, "settings.json")
	info, err := os.Lstat(p)
	switch {
	case os.IsNotExist(err):
		return need + " and there is no .gemini/settings.json"
	case err != nil:
		return fmt.Sprintf("%s and .gemini/settings.json cannot be read: %v", need, err)
	case !info.Mode().IsRegular():
		return need + " and .gemini/settings.json is not a regular file (symlink or special file); not read"
	case info.Size() > geminiSettingsMaxBytes:
		return fmt.Sprintf("%s and .gemini/settings.json is over %d bytes; not read", need, geminiSettingsMaxBytes)
	}
	f, err := os.Open(p)
	if err != nil {
		return fmt.Sprintf("%s and .gemini/settings.json cannot be read: %v", need, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, geminiSettingsMaxBytes))
	if err != nil {
		return fmt.Sprintf("%s and .gemini/settings.json cannot be read: %v", need, err)
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	var settings struct {
		Context *struct {
			FileName json.RawMessage `json:"fileName"`
		} `json:"context"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Sprintf("%s and .gemini/settings.json is not valid JSON (%v)", need, err)
	}
	if settings.Context == nil || len(settings.Context.FileName) == 0 || string(settings.Context.FileName) == "null" {
		return need + " and context.fileName is not set (the default is GEMINI.md only)"
	}
	var names []string
	var one string
	if err := json.Unmarshal(settings.Context.FileName, &one); err == nil {
		names = []string{one}
	} else if err := json.Unmarshal(settings.Context.FileName, &names); err != nil {
		return need + " and context.fileName is neither a string nor a list of strings"
	}
	for _, n := range names {
		if n == "AGENTS.md" {
			return ""
		}
	}
	return need + " and context.fileName does not list it"
}
