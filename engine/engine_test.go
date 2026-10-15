package engine

import (
	"strings"
	"testing"
)

const demo = `import (
	"github.com/alexisbcz/components/ui/button"
	"github.com/alexisbcz/icons/lucide"
	h "github.com/alexisbcz/libhtml"
)

func Demo() h.Node {
	return h.Div(h.Class("flex flex-wrap items-center gap-2 md:flex-row"),
		button.Button(button.VariantOutline, "Button"),
		button.Button(button.VariantOutline, button.SizeIcon, h.Aria("label", "Submit"), lucide.ArrowUp()),
	)
}`

func TestRun(t *testing.T) {
	r, err := Run(demo, Options{Instrument: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Func != "Demo" || !strings.Contains(r.HTML, `data-slot="button"`) || !strings.Contains(r.HTML, "<svg") {
		t.Fatalf("%s: %s", r.Func, r.HTML)
	}
	if len(r.Nodes) != 4 || !strings.Contains(r.HTML, `data-studio-src="1"`) {
		t.Fatalf("nodes %+v", r.Nodes)
	}
	t.Log(r.HTML)
}
