// Package hgen holds libhtml's generic helpers (If, Map, Range, ...) for
// the interpreter: compiled generic functions cannot be called from
// interpreted code, so programs use these interpreted copies instead (the
// engine rewrites h.Map to hgen.Map). Same semantics as libhtml's; MapSeq
// is missing (the interpreter's standard library predates package iter).
package hgen

import (
	"reflect"
	"sort"
	"strconv"

	h "github.com/alexisbcz/libhtml"
)

func If[T any](cond bool, v T) T {
	if cond {
		return v
	}
	var zero T
	return zero
}

func Unless[T any](cond bool, v T) T { return If(!cond, v) }

func Iff[T any](cond bool, fn func() T) T {
	if cond {
		return fn()
	}
	var zero T
	return zero
}

func IfElse[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func IfElseF[T any](cond bool, fa, fb func() T) T {
	if cond {
		return fa()
	}
	return fb()
}

func Ternary[T any](cond bool, a, b T) T { return IfElse(cond, a, b) }

// The Map family is not generic and works by reflection: the interpreter
// infers two-parameter generic signatures wrongly (R = T) and cannot
// instantiate generic functions with compiled types ([]bubble.Variant).

func Map(items any, fn any) h.Fragment {
	v, f := reflect.ValueOf(items), reflect.ValueOf(fn)
	out := make(h.Fragment, v.Len())
	for i := range out {
		out[i] = call(f, v.Index(i))
	}
	return out
}

func MapIdx(items any, fn any) h.Fragment {
	v, f := reflect.ValueOf(items), reflect.ValueOf(fn)
	out := make(h.Fragment, v.Len())
	for i := range out {
		out[i] = call(f, reflect.ValueOf(i), v.Index(i))
	}
	return out
}

func MapKV(m any, fn any) h.Fragment {
	mv, f := reflect.ValueOf(m), reflect.ValueOf(fn)
	keys := mv.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return less(keys[i], keys[j]) })
	out := make(h.Fragment, len(keys))
	for i, k := range keys {
		out[i] = call(f, k, mv.MapIndex(k))
	}
	return out
}

func Range(n int, fn any) h.Fragment {
	f := reflect.ValueOf(fn)
	if n < 0 {
		n = 0
	}
	out := make(h.Fragment, n)
	for i := range out {
		out[i] = call(f, reflect.ValueOf(i))
	}
	return out
}

func call(f reflect.Value, args ...reflect.Value) any {
	for i, a := range args {
		if t := f.Type().In(i); a.Type() != t && a.Type().ConvertibleTo(t) {
			args[i] = a.Convert(t)
		}
	}
	return f.Call(args)[0].Interface()
}

func less(a, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.String:
		return a.String() < b.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() < b.Uint()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	}
	return false
}

func Filter[T any](items []T, keep func(T) bool) []T {
	var out []T
	for _, it := range items {
		if keep(it) {
			out = append(out, it)
		}
	}
	return out
}

// Src marks the element of an instrumented call (see engine.instrument).
func Src(id int) h.Prop { return h.Data("studio-src", strconv.Itoa(id)) }
