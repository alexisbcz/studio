package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// generics are libhtml's generic functions, which the interpreter calls in
// package hgen instead.
var generics = map[string]bool{
	"If": true, "Unless": true, "Iff": true, "IfElse": true, "IfElseF": true, "Ternary": true,
	"Map": true, "MapIdx": true, "MapKV": true, "Range": true, "Filter": true,
}

// rewrite prepares src for the interpreter (nameParams, rewriteGenerics)
// and imports hgen when the result uses it.
func rewrite(src string) string {
	src = rewriteGenerics(nameParams(src))
	if !strings.Contains(src, "hgen__.") {
		return src
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.PackageClauseOnly)
	if err != nil {
		return src
	}
	end := fset.Position(f.Name.End()).Offset
	return src[:end] + `; import hgen__ "` + hgenPath + `"` + src[end:]
}

// rewriteGenerics turns calls of libhtml's generic helpers (h.Map) into
// calls of their interpreted copies (hgen.Map). Lines are not moved, so
// error positions keep their line numbers.
func rewriteGenerics(src string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return src
	}
	names := map[string]bool{}
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); p == "github.com/alexisbcz/libhtml" {
			name := "libhtml"
			if im.Name != nil {
				name = im.Name.Name
			}
			names[name] = true
		}
	}
	if len(names) == 0 {
		return src
	}
	var offs []int // offsets of the package names to replace
	var lens []int
	ast.Inspect(f, func(n ast.Node) bool {
		if se, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := se.X.(*ast.Ident); ok && names[id.Name] && generics[se.Sel.Name] {
				offs = append(offs, fset.Position(id.Pos()).Offset)
				lens = append(lens, len(id.Name))
			}
		}
		return true
	})
	if len(offs) == 0 {
		return src
	}
	idx := make([]int, len(offs))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return offs[idx[a]] > offs[idx[b]] })
	for _, i := range idx {
		src = src[:offs[i]] + "hgen__" + src[offs[i]+lens[i]:]
	}
	return src
}

// nameParams names the unnamed parameters of function literals and
// declarations (func(int) becomes func(_ int)): the interpreter mixes up
// the frames of closures with unnamed parameters. Only columns move.
func nameParams(src string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return src
	}
	var offs []int
	ast.Inspect(f, func(n ast.Node) bool {
		var ft *ast.FuncType
		switch x := n.(type) {
		case *ast.FuncLit:
			ft = x.Type
		case *ast.FuncDecl:
			ft = x.Type
		}
		if ft != nil && ft.Params != nil {
			for _, p := range ft.Params.List {
				if len(p.Names) == 0 {
					offs = append(offs, fset.Position(p.Type.Pos()).Offset)
				}
			}
		}
		return true
	})
	sort.Sort(sort.Reverse(sort.IntSlice(offs)))
	for _, o := range offs {
		src = src[:o] + "_ " + src[o:]
	}
	return src
}
