package repogov

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// failAfter yields its data and then a read error, like a disk or network
// file system failing partway through.
type failAfter struct {
	data *strings.Reader
	err  error
}

func (f *failAfter) Read(p []byte) (int, error) {
	if f.data.Len() > 0 {
		return f.data.Read(p)
	}
	return 0, f.err
}

// A read that fails inside the frontmatter block is an error, never "no
// frontmatter" (which would tell the user to add frontmatter the file has)
// and never a partial map read as complete. The scanner loop had no final
// sc.Err() check.
func TestParseFrontmatter_ReadErrorIsReported(t *testing.T) {
	boom := errors.New("input/output error")
	for name, data := range map[string]string{
		"inside the block": "---\ntrigger: always_on\n",
		"before any line":  "",
	} {
		fm, ok, err := parseFrontmatter(&failAfter{data: strings.NewReader(data), err: boom})
		if !errors.Is(err, boom) || ok || fm != nil {
			t.Errorf("%s: got %v, %v, %v; want the read error and no frontmatter", name, fm, ok, err)
		}
	}
	// A clean read still distinguishes a closed block from none at all.
	if fm, ok, err := parseFrontmatter(strings.NewReader("---\ntrigger: glob\n---\nbody\n")); err != nil || !ok || fm["trigger"] != "glob" {
		t.Errorf("closed block: %v, %v, %v", fm, ok, err)
	}
	if _, ok, err := parseFrontmatter(strings.NewReader("no frontmatter\n")); err != nil || ok {
		t.Errorf("no block: ok=%v err=%v; want false, nil", ok, err)
	}
	if _, ok, err := parseFrontmatter(io.LimitReader(strings.NewReader("---\ntrigger: glob\n"), 100)); err != nil || ok {
		t.Errorf("unclosed block: ok=%v err=%v; want false, nil", ok, err)
	}
}
