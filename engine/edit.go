package engine

import (
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

const libhtmlPath = "github.com/alexisbcz/libhtml"

// Arg is an argument of an inspected call, classified for the inspector.
type Arg struct {
	Kind    string   `json:"kind"` // text, class, option, child, code
	Start   int      `json:"start"`
	End     int      `json:"end"`
	Value   string   `json:"value"`             // the text, the classes, the option's name or the code
	Options []string `json:"options,omitempty"` // option: the alternatives, qualified ("button.VariantGhost")
	Name    string   `json:"name,omitempty"`    // child: "card.Header"
}

// Choice is a variant the call could take but does not (button.Size when
// only a button.Variant is given).
type Choice struct {
	Type    string   `json:"type"` // "Size"
	Options []string `json:"options"`
}

// Inspection describes the call at an offset of the source.
type Inspection struct {
	Name    string   `json:"name"` // "button.Button"
	Path    string   `json:"path"` // import path of its package
	Start   int      `json:"start"`
	End     int      `json:"end"`
	Args    []Arg    `json:"args"`
	Choices []Choice `json:"choices,omitempty"`
	Line    int      `json:"line"`
}

// file is a parsed source; skip is the length of the package clause
// withPackage added (offsets given to and returned by the API are in the
// user's source).
type file struct {
	src  string // with the package clause
	skip int
	fset *token.FileSet
	f    *ast.File
}

func parse(src string) (*file, error) {
	full, skip := withPackage(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", full, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return &file{full, skip, fset, f}, nil
}

func (p *file) off(pos token.Pos) int { return p.fset.Position(pos).Offset }

// text is the source of a node.
func (p *file) text(n ast.Node) string { return p.src[p.off(n.Pos()):p.off(n.End())] }

// list is an argument list holding the call: a call's arguments or a
// composite literal's elements.
type list struct {
	elts          []ast.Expr
	open, close   token.Pos // ( ) or { }
	index         int       // of the call in elts
	isCall        bool
	ellipsisAfter bool
}

// find returns the call starting at user offset start, and the list it is
// an element of (nil at the top of an expression).
func (p *file) find(start int) (*ast.CallExpr, *list) {
	var call *ast.CallExpr
	var parent *list
	var stack []ast.Node
	ast.Inspect(p.f, func(n ast.Node) bool {
		if call != nil {
			return false
		}
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if c, ok := n.(*ast.CallExpr); ok && p.off(c.Pos())-p.skip == start {
			// The outermost call starting there is the element call; a
			// call of its result (f(x)(y)) is not ours.
			if _, isSel := c.Fun.(*ast.SelectorExpr); isSel {
				call = c
				for i := len(stack) - 1; i >= 0 && parent == nil; i-- {
					switch x := stack[i].(type) {
					case *ast.CallExpr:
						for j, a := range x.Args {
							if a == c {
								parent = &list{elts: x.Args, open: x.Lparen, close: x.Rparen, index: j, isCall: true, ellipsisAfter: x.Ellipsis.IsValid()}
							}
						}
					case *ast.CompositeLit:
						for j, a := range x.Elts {
							if a == c {
								parent = &list{elts: x.Elts, open: x.Lbrace, close: x.Rbrace, index: j}
							}
						}
					}
					break
				}
				return false
			}
		}
		stack = append(stack, n)
		return true
	})
	return call, parent
}

// Inspect describes the call starting at offset start of src.
func Inspect(src string, start int) (*Inspection, error) {
	p, err := parse(src)
	if err != nil {
		return nil, err
	}
	call, _ := p.find(start)
	if call == nil {
		return nil, errors.New("no call there")
	}
	names := imports(p.f)
	name, path := elementCall(call, names)
	if name == "" {
		se := call.Fun.(*ast.SelectorExpr)
		name = p.text(se)
	}
	in := &Inspection{Name: name, Path: path, Start: start, End: p.off(call.End()) - p.skip, Line: p.fset.Position(call.Pos()).Line}
	present := map[reflect.Type]bool{}
	for _, a := range call.Args {
		arg := Arg{Kind: "code", Start: p.off(a.Pos()) - p.skip, End: p.off(a.End()) - p.skip, Value: p.text(a)}
		switch x := a.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if s, err := strconv.Unquote(x.Value); err == nil {
					arg.Kind, arg.Value = "text", s
				}
			}
		case *ast.CallExpr:
			if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "Class" && len(x.Args) == 1 {
				if id, ok := se.X.(*ast.Ident); ok && names[id.Name] == libhtmlPath {
					if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						s, _ := strconv.Unquote(lit.Value)
						arg.Kind, arg.Value = "class", s
						arg.Start, arg.End = p.off(lit.Pos())-p.skip, p.off(lit.End())-p.skip
					}
				}
			} else if n, _ := elementCall(x, names); n != "" {
				arg.Kind, arg.Name = "child", n
			}
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				if v, ok := Symbol(names[id.Name], x.Sel.Name); ok && v.Kind() != reflect.Func && v.Type().PkgPath() == names[id.Name] {
					if opts := Alternatives(names[id.Name], v.Type()); len(opts) > 1 {
						arg.Kind = "option"
						for _, o := range opts {
							arg.Options = append(arg.Options, id.Name+"."+o)
						}
						present[v.Type()] = true
					}
				}
			}
		}
		in.Args = append(in.Args, arg)
	}
	// Variants of the call's own package it does not use yet, for the
	// package's main component (button.Button, not card.Title: card.Size
	// is the Card's).
	if path != "" && strings.EqualFold(name[strings.Index(name, ".")+1:], pkgNameOf(path)) {
		alias := name[:strings.Index(name, ".")]
		types := map[reflect.Type]bool{}
		for _, v := range packages()[path] {
			if v.Kind() != reflect.Func && v.Type().PkgPath() == path && v.Type().Kind() == reflect.String && !present[v.Type()] {
				types[v.Type()] = true
			}
		}
		for t := range types {
			opts := Alternatives(path, t)
			if len(opts) < 2 {
				continue
			}
			c := Choice{Type: t.Name()}
			for _, o := range opts {
				c.Options = append(c.Options, alias+"."+o)
			}
			in.Choices = append(in.Choices, c)
		}
		sort.Slice(in.Choices, func(i, j int) bool { return in.Choices[i].Type < in.Choices[j].Type })
	}
	return in, nil
}

// Import is an import a snippet needs.
type Import struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path"`
}

// Edit is a change to the source, applied by Apply.
type Edit struct {
	Op string `json:"op"` // replace, delete, move, insert, addArg, setClass
	// replace: the range [Start, End) of the source becomes Text.
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	// Call is the offset of the call delete, move, insert, addArg and
	// setClass act on.
	Call int `json:"call"`
	// move: -1 (before the previous sibling) or 1. insert: "inside" (last
	// child), "before" or "after".
	Delta   int      `json:"delta"`
	Where   string   `json:"where"`
	Imports []Import `json:"imports"`
}

// Applied is the result of an edit: the new, formatted source and the
// offset of the call to select (-1 for none).
type Applied struct {
	Src    string `json:"src"`
	Select int    `json:"select"`
}

// Apply applies e to src.
func Apply(src string, e Edit) (*Applied, error) {
	p, err := parse(src)
	if err != nil {
		return nil, err
	}
	s := p.src
	sel := -1 // offset in s of the call to select
	splice := func(from, to int, text string) {
		s = s[:from] + text + s[to:]
	}
	switch e.Op {
	case "replace":
		from, to := e.Start+p.skip, e.End+p.skip
		if from < p.skip || to > len(s) || from > to {
			return nil, errors.New("range out of bounds")
		}
		text, imps := requalify(e.Text, e.Imports, imports(p.f))
		e.Imports = imps
		splice(from, to, text)
		sel = -1
	case "delete":
		call, parent := p.find(e.Call)
		if call == nil || parent == nil {
			return nil, errors.New("only the children of a call or list can be deleted")
		}
		from, to := p.elementRange(parent, parent.index)
		splice(from, to, "")
		if parent.isCall || true {
			sel = -1
		}
	case "move":
		call, parent := p.find(e.Call)
		if call == nil || parent == nil {
			return nil, errors.New("only the children of a call or list can be moved")
		}
		j := parent.index + e.Delta
		if j < 0 || j >= len(parent.elts) {
			return nil, errors.New("cannot move further")
		}
		a, b := parent.elts[parent.index], parent.elts[j]
		at, bt := p.text(a), p.text(b)
		if j > parent.index {
			splice(p.off(b.Pos()), p.off(b.End()), at)
			splice(p.off(a.Pos()), p.off(a.End()), bt)
			sel = p.off(b.Pos()) + len(bt) - len(at)
		} else {
			splice(p.off(a.Pos()), p.off(a.End()), bt)
			splice(p.off(b.Pos()), p.off(b.End()), at)
			sel = p.off(b.Pos())
		}
	case "insert", "addArg":
		call, parent := p.find(e.Call)
		if call == nil {
			return nil, errors.New("no call there")
		}
		text, imps := requalify(e.Text, e.Imports, imports(p.f))
		e.Imports = imps
		switch {
		case e.Op == "addArg" || e.Where == "first":
			at := p.off(call.Lparen) + 1
			if len(call.Args) > 0 {
				text += ", "
			}
			splice(at, at, text)
			sel = p.off(call.Pos())
		case e.Where == "inside":
			if call.Ellipsis.IsValid() {
				return nil, errors.New("cannot add children to a call with a spread (xs...)")
			}
			at, ins := p.appendAt(call.Args, call.Rparen)
			splice(at, at, ins(text))
			sel = at + strings.Index(ins(text), text)
		case e.Where == "after", e.Where == "before":
			if parent == nil {
				return nil, errors.New("this element has no parent to insert into")
			}
			if parent.isCall && parent.ellipsisAfter {
				return nil, errors.New("cannot insert into a spread (xs...)")
			}
			if e.Where == "before" {
				at := p.off(call.Pos())
				splice(at, at, text+",\n")
				sel = at
			} else {
				at := p.off(call.End())
				splice(at, at, ",\n"+text)
				sel = at + 2
			}
		default:
			return nil, errors.New("insert: where must be inside, before, after or first")
		}
	case "setClass":
		call, _ := p.find(e.Call)
		if call == nil {
			return nil, errors.New("no call there")
		}
		names := imports(p.f)
		alias := ""
		for n, path := range names {
			if path == libhtmlPath {
				alias = n
			}
		}
		for _, a := range call.Args {
			if c, ok := a.(*ast.CallExpr); ok && alias != "" {
				if se, ok := c.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "Class" && len(c.Args) == 1 {
					if id, ok := se.X.(*ast.Ident); ok && id.Name == alias {
						splice(p.off(c.Args[0].Pos()), p.off(c.Args[0].End()), strconv.Quote(e.Text))
						return finish(s, p.skip, p.off(call.Pos()), nil)
					}
				}
			}
		}
		if alias == "" {
			alias = "h"
			e.Imports = append(e.Imports, Import{Name: "h", Path: libhtmlPath})
		}
		at := p.off(call.Lparen) + 1
		text := alias + ".Class(" + strconv.Quote(e.Text) + ")"
		if len(call.Args) > 0 {
			text += ", "
		}
		splice(at, at, text)
		sel = p.off(call.Pos())
	default:
		return nil, errors.New("unknown edit " + e.Op)
	}
	return finish(s, p.skip, sel, e.Imports)
}

// requalify rewrites the package names of a snippet to those the file
// imports the packages under (dm.Item when the snippet says
// dropdownmenu.Item), and returns the imports still missing.
func requalify(text string, imps []Import, names map[string]string) (string, []Import) {
	byPath := map[string]string{}
	for n, p := range names {
		byPath[p] = n
	}
	rename := map[string]string{}
	var missing []Import
	for _, im := range imps {
		alias := im.Name
		if alias == "" {
			alias = pkgNameOf(im.Path)
		}
		if have, ok := byPath[im.Path]; ok {
			if have != alias {
				rename[alias] = have
			}
			continue
		}
		missing = append(missing, im)
	}
	if len(rename) == 0 {
		return text, missing
	}
	var sc scanner.Scanner
	fset := token.NewFileSet()
	sc.Init(fset.AddFile("", -1, len(text)), []byte(text), nil, 0)
	type edit struct {
		off int
		old string
	}
	var edits []edit
	prevIdent, prevOff, prevSel := "", 0, false
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		off := fset.Position(pos).Offset
		if tok == token.PERIOD && prevIdent != "" && !prevSel {
			if _, ok := rename[prevIdent]; ok {
				edits = append(edits, edit{prevOff, prevIdent})
			}
		}
		prevSel = tok == token.PERIOD
		if tok == token.IDENT {
			prevIdent, prevOff = lit, off
		} else if tok != token.PERIOD {
			prevIdent = ""
		}
	}
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		text = text[:e.off] + rename[e.old] + text[e.off+len(e.old):]
	}
	return text, missing
}

// elementRange is the range to delete for element i of a list: the
// element and the comma (and space) after it, or before it for the last.
func (p *file) elementRange(l *list, i int) (int, int) {
	from, to := p.off(l.elts[i].Pos()), p.off(l.elts[i].End())
	if i+1 < len(l.elts) {
		return from, p.off(l.elts[i+1].Pos())
	}
	if i > 0 {
		from = p.off(l.elts[i-1].End())
	}
	// A trailing comma stays for the previous element.
	return from, to
}

// appendAt returns where to add an element after the last of args and
// how to write it there.
func (p *file) appendAt(args []ast.Expr, closing token.Pos) (int, func(string) string) {
	if len(args) == 0 {
		return p.off(closing), func(t string) string { return t }
	}
	end := p.off(args[len(args)-1].End())
	if strings.Contains(p.src[end:p.off(closing)], ",") {
		// A trailing comma: the list is one element per line.
		at := end + strings.Index(p.src[end:], ",") + 1
		return at, func(t string) string { return "\n" + t + "," }
	}
	return end, func(t string) string { return ",\n" + t + ",\n" }
}

// selMarker marks the selection while the source is formatted.
const selMarker = "/*studio:sel*/"

// finish adds imports, formats s and returns it without the package clause
// withPackage added, with the selection's offset.
func finish(s string, skip, sel int, imps []Import) (*Applied, error) {
	if sel >= 0 {
		s = s[:sel] + selMarker + s[sel:]
	}
	if len(imps) > 0 {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", s, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, im := range imps {
			if im.Name != "" && im.Name != pkgNameOf(im.Path) {
				astutil.AddNamedImport(fset, f, im.Name, im.Path)
			} else {
				astutil.AddImport(fset, f, im.Path)
			}
		}
		var b strings.Builder
		if err := format.Node(&b, fset, f); err != nil {
			return nil, err
		}
		s = b.String()
	}
	out, err := format.Source([]byte(s))
	if err != nil {
		return nil, err
	}
	s = string(out)
	if skip > 0 {
		s = strings.TrimPrefix(s, "package main\n\n")
		s = strings.TrimPrefix(s, "package main\n")
	}
	pos := -1
	if i := strings.Index(s, selMarker); i >= 0 {
		pos = i
		s = s[:i] + s[i+len(selMarker):]
		if strings.HasPrefix(s[i:], " ") {
			s = s[:i] + s[i+1:]
		}
	}
	return &Applied{Src: s, Select: pos}, nil
}

// Format formats src (gofmt).
func Format(src string) (string, error) {
	a, err := finish(mustPackage(src))
	if err != nil {
		return "", err
	}
	return a.Src, nil
}

func mustPackage(src string) (string, int, int, []Import) {
	full, skip := withPackage(src)
	return full, skip, -1, nil
}
