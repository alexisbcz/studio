package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Node is a component call of the source: the instrumented program marks
// the element it renders with data-studio-src="ID".
type Node struct {
	ID    int    `json:"id"`
	Start int    `json:"start"` // byte offsets of the call in the source
	End   int    `json:"end"`
	Name  string `json:"name"` // "button.Button"
}

// srcAttr is the attribute instrumented calls add to their element.
const srcAttr = "studio-src"

// imports maps the file's import names to their paths.
func imports(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		name := pkgNameOf(p)
		if im.Name != nil {
			name = im.Name.Name
		}
		out[name] = p
	}
	return out
}

// pkgNameOf is the package name of an import path: the compiled
// package's (ui/select is selectui), or the path's last element.
func pkgNameOf(path string) string {
	if n, ok := packageNames()[path]; ok {
		return n
	}
	base := path[strings.LastIndex(path, "/")+1:]
	if len(base) > 1 && base[0] == 'v' && strings.Trim(base[1:], "0123456789") == "" {
		return pkgNameOf(path[:strings.LastIndex(path, "/")])
	}
	return base
}

// elementCall returns the "pkg.Func" name and import path of a call of a
// compiled element function, or "" when call is something else.
func elementCall(call *ast.CallExpr, names map[string]string) (string, string) {
	se, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	id, ok := se.X.(*ast.Ident)
	if !ok {
		return "", ""
	}
	path, ok := names[id.Name]
	if !ok {
		return "", ""
	}
	v, ok := Symbol(path, se.Sel.Name)
	key := path + "." + se.Sel.Name
	if !ok || notElements[key] || !isElementFunc(v) || !rendersElement(key, v) {
		return "", ""
	}
	return id.Name + "." + se.Sel.Name, path
}

// instrument adds a hgen__.Src(id) prop to every element call of src (a
// file with its package clause, whose first skip bytes are not the user's)
// and returns the new source with the calls found, in source order.
func instrument(src string, skip int) (string, []Node) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return src, nil
	}
	names := imports(f)
	type ins struct {
		off  int
		text string
	}
	var edits []ins
	var nodes []Node
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || call.Ellipsis.IsValid() {
			return true
		}
		name, _ := elementCall(call, names)
		if name == "" {
			return true
		}
		id := len(nodes) + 1
		nodes = append(nodes, Node{ID: id, Start: off(call.Pos()) - skip, End: off(call.End()) - skip, Name: name})
		prop := "hgen__.Src(" + strconv.Itoa(id) + ")"
		switch {
		case len(call.Args) == 0:
		case strings.Contains(src[off(call.Args[len(call.Args)-1].End()):off(call.Rparen)], ","):
			prop += ","
		default:
			prop = ", " + prop
		}
		edits = append(edits, ins{off(call.Rparen), prop})
		return true
	})
	sort.Slice(edits, func(i, j int) bool { return edits[i].off > edits[j].off })
	for _, e := range edits {
		src = src[:e.off] + e.text + src[e.off:]
	}
	return src, nodes
}
