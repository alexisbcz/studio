package engine

import (
	"strings"
	"testing"
)

func at(t *testing.T, src, sub string) int {
	t.Helper()
	i := strings.Index(src, sub)
	if i < 0 {
		t.Fatalf("%q not in source", sub)
	}
	return i
}

func TestInspect(t *testing.T) {
	in, err := Inspect(demo, at(t, demo, `button.Button(button.VariantOutline, "Button")`))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "button.Button" || len(in.Args) != 2 || in.Args[0].Kind != "option" || in.Args[1].Kind != "text" || in.Args[1].Value != "Button" {
		t.Fatalf("%+v", in)
	}
	if len(in.Args[0].Options) < 5 || len(in.Choices) != 1 || in.Choices[0].Type != "Size" {
		t.Fatalf("options %v choices %+v", in.Args[0].Options, in.Choices)
	}
	div, _ := Inspect(demo, at(t, demo, "h.Div("))
	if div.Args[0].Kind != "class" || div.Args[1].Kind != "child" || div.Args[1].Name != "button.Button" {
		t.Fatalf("%+v", div)
	}
}

func TestEdits(t *testing.T) {
	btn := at(t, demo, `button.Button(button.VariantOutline, "Button")`)
	in, _ := Inspect(demo, btn)
	a, err := Apply(demo, Edit{Op: "replace", Start: in.Args[0].Start, End: in.Args[0].End, Text: "button.VariantGhost"})
	if err != nil || !strings.Contains(a.Src, `button.Button(button.VariantGhost, "Button")`) {
		t.Fatalf("replace: %v\n%s", err, a.Src)
	}
	a, err = Apply(demo, Edit{Op: "delete", Call: btn})
	if err != nil || strings.Contains(a.Src, `"Button"`) {
		t.Fatalf("delete: %v\n%s", err, a.Src)
	}
	if _, err := Run(a.Src, Options{}); err != nil {
		t.Fatalf("delete result does not run: %v\n%s", err, a.Src)
	}
	a, err = Apply(demo, Edit{Op: "move", Call: btn, Delta: 1})
	if err != nil || strings.Index(a.Src, "lucide.ArrowUp") > strings.Index(a.Src, `"Button"`) || !strings.HasPrefix(a.Src[a.Select:], `button.Button(button.VariantOutline, "Button")`) {
		t.Fatalf("move: %v %d\n%s", err, a.Select, a.Src)
	}
	div := at(t, demo, "h.Div(")
	a, err = Apply(demo, Edit{Op: "insert", Call: div, Where: "inside", Text: `badge.Badge("New")`, Imports: []Import{{Path: "github.com/alexisbcz/components/ui/badge"}}})
	if err != nil || !strings.Contains(a.Src, `"github.com/alexisbcz/components/ui/badge"`) || !strings.HasPrefix(a.Src[a.Select:], `badge.Badge("New")`) {
		t.Fatalf("insert: %v %d\n%s", err, a.Select, a.Src)
	}
	if r, err := Run(a.Src, Options{}); err != nil || !strings.Contains(r.HTML, "New") {
		t.Fatalf("insert result: %v\n%s", err, a.Src)
	}
	a, err = Apply(demo, Edit{Op: "insert", Call: btn, Where: "after", Text: `button.Button("Second")`})
	if err != nil || !strings.HasPrefix(a.Src[a.Select:], `button.Button("Second")`) {
		t.Fatalf("after: %v\n%s", err, a.Src)
	}
	a, err = Apply(demo, Edit{Op: "setClass", Call: btn, Text: "w-full"})
	if err != nil || !strings.Contains(a.Src, `button.Button(h.Class("w-full"), button.VariantOutline`) {
		t.Fatalf("setClass: %v\n%s", err, a.Src)
	}
	a, err = Apply(a.Src, Edit{Op: "setClass", Call: a.Select, Text: "w-1/2"})
	if err != nil || !strings.Contains(a.Src, `h.Class("w-1/2")`) {
		t.Fatalf("setClass again: %v\n%s", err, a.Src)
	}
	a, err = Apply(demo, Edit{Op: "addArg", Call: btn, Text: "button.SizeLg"})
	if err != nil || !strings.Contains(a.Src, `button.Button(button.SizeLg, button.VariantOutline`) {
		t.Fatalf("addArg: %v\n%s", err, a.Src)
	}
}

func TestRequalify(t *testing.T) {
	src := "import (\n\tmenu \"github.com/alexisbcz/components/ui/dropdown-menu\"\n\th \"github.com/alexisbcz/libhtml\"\n)\n\nfunc Page() h.Node { return h.Div(menu.Separator()) }\n"
	a, err := Apply(src, Edit{Op: "insert", Call: strings.Index(src, "h.Div("), Where: "inside", Text: `dm.Item("Profile", dm.Shortcut("P"))`,
		Imports: []Import{{Name: "dm", Path: "github.com/alexisbcz/components/ui/dropdown-menu"}}})
	if err != nil || !strings.Contains(a.Src, `menu.Item("Profile", menu.Shortcut("P"))`) || strings.Contains(a.Src, "dm ") {
		t.Fatalf("%v\n%s", err, a.Src)
	}
}
