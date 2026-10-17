package repogov

import "strings"

// cursorRule converts a seeded template into a Cursor project rule. Cursor
// only loads .mdc files (a plain .md in .cursor/rules is ignored) and reads
// its own frontmatter (description, globs, alwaysApply) rather than Copilot's
// applyTo. It returns the .mdc filename and the converted content:
//
//   - applyTo "**" (applies everywhere) becomes alwaysApply: true;
//   - any other applyTo glob becomes globs with alwaysApply: false;
//   - description is the document's first heading.
func cursorRule(templateName, content string) (name, body string) {
	name = templateStem(templateName) + ".mdc"

	applyTo := "**"
	rest := content
	const fence = "\n---\n"
	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], fence); end >= 0 {
			for _, line := range strings.Split(content[4:4+end], "\n") {
				if k, v, ok := yamlSplitKV(strings.TrimSpace(line)); ok && k == "applyTo" {
					applyTo = yamlUnquote(v)
				}
			}
			rest = strings.TrimLeft(content[4+end+len(fence):], "\n")
		}
	}

	description := templateStem(templateName)
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "# ") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			break
		}
	}

	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString("description: " + yamlDoubleQuote(description) + "\n")
	if applyTo == "**" {
		fm.WriteString("alwaysApply: true\n")
	} else {
		fm.WriteString("globs: " + yamlDoubleQuote(applyTo) + "\n")
		fm.WriteString("alwaysApply: false\n")
	}
	fm.WriteString("---\n\n")
	return name, fm.String() + rest
}
