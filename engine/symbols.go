package engine

import (
	"reflect"
	"strings"
	"sync"

	h "github.com/alexisbcz/libhtml"
	"github.com/alexisbcz/studio/symbols"
)

var nodeType = reflect.TypeOf((*h.Node)(nil)).Elem()

// packages indexes the exported symbols of the compiled packages by import
// path (the interpreter keys them "path/name").
var packages = sync.OnceValue(func() map[string]map[string]reflect.Value {
	out := map[string]map[string]reflect.Value{}
	for key, syms := range symbols.Symbols {
		if i := strings.LastIndex(key, "/"); i > 0 {
			out[key[:i]] = syms
		}
	}
	return out
})

// packageNames maps the import paths of the compiled packages to their
// package names (ui/select is package selectui).
var packageNames = sync.OnceValue(func() map[string]string {
	out := map[string]string{}
	for key := range symbols.Symbols {
		if i := strings.LastIndex(key, "/"); i > 0 {
			out[key[:i]] = key[i+1:]
		}
	}
	return out
})

// Symbol returns an exported symbol of a compiled package.
func Symbol(path, name string) (reflect.Value, bool) {
	v, ok := packages()[path][name]
	return v, ok
}

// isElementFunc reports whether fn builds a node from variadic children
// (func(...any) h.Node-like): extra props can be passed to it.
func isElementFunc(v reflect.Value) bool {
	if v.Kind() != reflect.Func {
		return false
	}
	t := v.Type()
	if !t.IsVariadic() || t.NumOut() != 1 || !t.Out(0).Implements(nodeType) {
		return false
	}
	last := t.In(t.NumIn() - 1).Elem()
	return last.Kind() == reflect.Interface && last.NumMethod() == 0
}

var (
	elementMu    sync.Mutex
	elementCache = map[string]bool{}
)

// rendersElement reports whether fn (an element function) resolves to a
// single element, so a prop passed to it lands on that element: it is
// called once without children to find out. Functions rendering fragments
// (several elements, or none) cannot take props of their own.
func rendersElement(key string, fn reflect.Value) (ok bool) {
	elementMu.Lock()
	defer elementMu.Unlock()
	if v, done := elementCache[key]; done {
		return v
	}
	defer func() {
		if recover() != nil {
			ok = false
		}
		elementCache[key] = ok
	}()
	out := fn.Call(nil)[0].Interface()
	switch n := out.(type) {
	case *h.Element:
		return n != nil
	case h.Elementer:
		return n.Element() != nil
	}
	return false
}

// notElements are node constructors whose props would land on the
// enclosing element instead of their own.
var notElements = map[string]bool{
	"github.com/alexisbcz/libhtml.Group": true, "github.com/alexisbcz/libhtml.Fragment": true,
	"github.com/alexisbcz/libhtml.Text": true, "github.com/alexisbcz/libhtml.Raw": true,
	"github.com/alexisbcz/libhtml.WithValue": true, "github.com/alexisbcz/libhtml.Doctype": true,
	"github.com/alexisbcz/components/style.Scope": true, "github.com/alexisbcz/components/style.WithBase": true,
}

// Alternatives returns the constants and variables of path that have type t
// (the alternatives of a variant: button.VariantOutline -> every
// button.Variant), sorted by name.
func Alternatives(path string, t reflect.Type) []string {
	var out []string
	for name, v := range packages()[path] {
		if v.Kind() != reflect.Func && v.Type() == t && !strings.HasSuffix(name, "Values") {
			out = append(out, name)
		}
	}
	sortStrings(out)
	return out
}
