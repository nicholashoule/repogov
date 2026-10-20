package repogov

import (
	"embed"
	"fmt"
	"os"
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

// writeNewFile creates path with data and fails if anything already exists
// there, including a symlink (dangling or not). O_EXCL makes the existence
// check and the creation one atomic step, so a scaffold write can neither
// race a concurrent creator nor be redirected through a planted symlink.
func writeNewFile(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
