package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// typesFirst reorders the top-level declarations of src: imports, types,
// constants, variables, then functions (each with its doc comment).
func typesFirst(src string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return src
	}
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	groups := make([][]string, 5)
	for _, d := range f.Decls {
		start, doc := d.Pos(), (*ast.CommentGroup)(nil)
		rank := 4
		switch x := d.(type) {
		case *ast.GenDecl:
			doc = x.Doc
			switch x.Tok {
			case token.IMPORT:
				rank = 0
			case token.TYPE:
				rank = 1
			case token.CONST:
				rank = 2
			case token.VAR:
				rank = 3
			}
		case *ast.FuncDecl:
			doc = x.Doc
		}
		if doc != nil {
			start = doc.Pos()
		}
		groups[rank] = append(groups[rank], src[off(start):off(d.End())])
	}
	var b strings.Builder
	b.WriteString("package " + f.Name.Name + "\n\n")
	for _, g := range groups {
		for _, d := range g {
			b.WriteString(d + "\n\n")
		}
	}
	return b.String()
}
