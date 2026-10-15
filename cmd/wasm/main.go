//go:build js && wasm

// Command wasm is the studio's engine for the browser: it exposes
// globalThis.studioEngine, whose functions take and return JSON strings
// (see web/worker.js, which runs it in a Web Worker).
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/alexisbcz/studio/engine"
)

type runRequest struct {
	Code string `json:"code"`
	engine.Options
}

type inspectRequest struct {
	Code  string `json:"code"`
	Start int    `json:"start"`
}

type applyRequest struct {
	Code string      `json:"code"`
	Edit engine.Edit `json:"edit"`
}

// reply encodes a result, or {"error": "..."}.
func reply(v any, err error) any {
	if err != nil {
		v = map[string]string{"error": err.Error()}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// handler decodes the JSON argument into a T and calls fn with it.
func handler[T any](fn func(T) (any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		var req T
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			return reply(nil, err)
		}
		return reply(fn(req))
	})
}

func main() {
	js.Global().Set("studioEngine", js.ValueOf(map[string]any{
		"run": handler(func(r runRequest) (any, error) {
			return engine.Run(r.Code, r.Options)
		}),
		"inspect": handler(func(r inspectRequest) (any, error) {
			return engine.Inspect(r.Code, r.Start)
		}),
		"apply": handler(func(r applyRequest) (any, error) {
			return engine.Apply(r.Code, r.Edit)
		}),
		"format": handler(func(r runRequest) (any, error) {
			s, err := engine.Format(r.Code)
			return map[string]string{"src": s}, err
		}),
	}))
	select {}
}
