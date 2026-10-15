package engine

import (
	"strings"
	"testing"
)

// TestPalette inserts every snippet into an empty page and renders it.
func TestPalette(t *testing.T) {
	const page = "import h \"github.com/alexisbcz/libhtml\"\n\nfunc Page() h.Node {\n\treturn h.Div(h.Class(\"p-6\"))\n}\n"
	for _, s := range Palette {
		a, err := Apply(page, Edit{Op: "insert", Call: strings.Index(page, "h.Div("), Where: "inside", Text: s.Code, Imports: s.Imports})
		if err != nil {
			t.Errorf("%s: %v", s.Label, err)
			continue
		}
		r, err := Run(a.Src, Options{Instrument: true})
		if err != nil {
			t.Errorf("%s: %v\n%s", s.Label, err, a.Src)
			continue
		}
		if len(r.Nodes) < 2 {
			t.Errorf("%s: %d nodes", s.Label, len(r.Nodes))
		}
	}
}
