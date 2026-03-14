package repogov

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

// mustReadTemplate reads and returns the content of an embedded template file
// by its name within the templates/ directory. Line endings are normalized to
// LF (\n) so that content is consistent across platforms regardless of how
// the file was checked out. It panics if the file is missing, since that
// indicates a broken build rather than a runtime error.
func mustReadTemplate(name string) string {
	data, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		panic(fmt.Sprintf("repogov: missing embedded template %q: %v", name, err))
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// mustRenderTemplate reads the named embedded template and renders it with
// text/template using the provided data value (pass nil for static templates).
// Unlike mustReadTemplate, this ensures that any future {{.Placeholder}}
// additions to a template are never silently emitted as literal text.
func mustRenderTemplate(name string, data any) string {
	tmpl := template.Must(template.New(name).Parse(mustReadTemplate(name)))
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		panic(fmt.Sprintf("repogov: template render error %q: %v", name, err))
	}
	return b.String()
}
