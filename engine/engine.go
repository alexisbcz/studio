// Package engine runs shadcn.go code: it interprets a Go source file with
// yaegi, calls its page function and renders the result to HTML. It also
// maps the rendered elements back to the calls of the source (instrument)
// and edits the source at those calls (edit.go), for the visual editor.
package engine

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing/fstest"

	"github.com/alexisbcz/components/style"
	h "github.com/alexisbcz/libhtml"
	"github.com/alexisbcz/studio/symbols"
	"github.com/traefik/yaegi/interp"
)

const hgenPath = "github.com/alexisbcz/studio/engine/hgen"

//go:embed hgen/hgen.go
var hgenSource string

// Options configures Run.
type Options struct {
	Func       string // the function to render; default: PageFunc
	Style      string // "nova", "vega", ...; default: the components' default
	Base       string // "radix", "base" or "aria"; default: radix
	Instrument bool   // mark elements with data-studio-src (see Node)
}

// Result is a rendered program.
type Result struct {
	HTML  string `json:"html"`
	Func  string `json:"func"`  // the function that was rendered
	Nodes []Node `json:"nodes"` // with Options.Instrument
}

// Run interprets src, a Go file (the package clause is optional), and
// renders its page function.
func Run(src string, opts Options) (*Result, error) {
	full, skip := withPackage(src)
	fn := opts.Func
	if fn == "" {
		var err error
		if fn, err = PageFunc(full); err != nil {
			return nil, err
		}
	}
	var nodes []Node
	if opts.Instrument {
		full, nodes = instrument(full, skip)
	}
	r, err := run(full, fn, opts)
	if err != nil {
		// The interpreter is sensitive to declaration order (a type used
		// before its declaration, aliases): retry with types first.
		if alt := typesFirst(full); alt != full {
			if r2, err2 := run(alt, fn, opts); err2 == nil {
				r, err = r2, nil
			}
		}
	}
	if err != nil {
		return nil, shiftError(err, skip, src)
	}
	r.Nodes = nodes
	return r, nil
}

func run(src, fn string, opts Options) (r *Result, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	src = rewrite(src)
	i := interp.New(interp.Options{GoPath: ".", SourcecodeFilesystem: fstest.MapFS{
		"src/" + hgenPath + "/hgen.go": &fstest.MapFile{Data: []byte(hgenSource)},
	}})
	if err := i.Use(symbols.Symbols); err != nil {
		return nil, err
	}
	if _, err := i.Eval(src); err != nil {
		return nil, err
	}
	v, err := i.Eval("main." + fn)
	if err != nil {
		return nil, err
	}
	var node h.Node
	switch f := v.Interface().(type) {
	case func() h.Node:
		node = f()
	case func(*http.Request) h.Node:
		node = f(httptest.NewRequest("GET", "/", nil))
	case func(...any) h.Node:
		node = f()
	default:
		return nil, fmt.Errorf("%s must be a func() h.Node, func(...any) h.Node or func(*http.Request) h.Node", fn)
	}
	if opts.Base != "" {
		node = style.WithBase(style.Base(opts.Base), node)
	}
	if st, ok := style.Parse(opts.Style); ok {
		node = style.Scope(st, node)
	}
	var b bytes.Buffer
	if err := node.Render(&b); err != nil {
		return nil, err
	}
	return &Result{HTML: b.String(), Func: fn}, nil
}

// prefix is the package clause added to sources without one, on their
// first line so that line numbers match the editor's.
const prefix = "package main; "

// withPackage adds a package clause when src has none, as for the snippets
// of the docs, which start with their imports. It returns the length of
// what it added.
func withPackage(src string) (string, int) {
	if _, err := parser.ParseFile(token.NewFileSet(), "", src, parser.PackageClauseOnly); err == nil {
		return src, 0
	}
	return prefix + src, len(prefix)
}

// shiftError corrects the column of errors on the first line for the
// added package clause.
func shiftError(err error, skip int, src string) error {
	if skip == 0 {
		return err
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "1:") {
		var col int
		if _, e := fmt.Sscanf(msg[2:], "%d", &col); e == nil && col > skip {
			rest := msg[2+len(fmt.Sprint(col)):]
			return errors.New(fmt.Sprintf("1:%d%s", col-skip, rest))
		}
	}
	return err
}

// PageFunc returns the first top-level function of src that renders a
// page: a func() h.Node, func(...any) h.Node or func(*http.Request) h.Node.
func PageFunc(src string) (string, error) {
	full, _ := withPackage(src)
	f, err := parser.ParseFile(token.NewFileSet(), "", full, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Type.Params.NumFields() > 1 || fd.Type.Results.NumFields() != 1 {
			continue
		}
		if fd.Type.Params.NumFields() == 1 && !isRequest(fd.Type.Params.List[0].Type) && !isVariadicAny(fd.Type.Params.List[0].Type) {
			continue
		}
		if sel, ok := fd.Type.Results.List[0].Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Node" {
			return fd.Name.Name, nil
		}
	}
	return "", errors.New("no page function: add a func() h.Node (or func(r *http.Request) h.Node)")
}

func isVariadicAny(e ast.Expr) bool {
	el, ok := e.(*ast.Ellipsis)
	if !ok {
		return false
	}
	id, ok := el.Elt.(*ast.Ident)
	return ok && id.Name == "any"
}

func isRequest(e ast.Expr) bool {
	st, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := st.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Request"
}
