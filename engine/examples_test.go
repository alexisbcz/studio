package engine

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestExamples(t *testing.T) {
	b, err := os.ReadFile("../examples/go-examples.json")
	if err != nil {
		t.Skip(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	failed := 0
	reasons := map[string]int{}
	var slowest time.Duration
	var slowName string
	for _, n := range names {
		start := time.Now()
		_, err := runSafe(m[n])
		if d := time.Since(start); d > slowest {
			slowest, slowName = d, n
		}
		if err != nil {
			failed++
			msg := err.Error()
			if i := strings.Index(msg, ": "); i >= 0 && strings.Count(msg[:i], ":") >= 1 {
				msg = msg[i+2:]
			}
			if len(msg) > 90 {
				msg = msg[:90]
			}
			reasons[msg]++
			if os.Getenv("V") != "" {
				t.Logf("%s: %v", n, err)
			}
		}
	}
	type kv struct {
		k string
		v int
	}
	var rs []kv
	for k, v := range reasons {
		rs = append(rs, kv{k, v})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].v > rs[j].v })
	for i, r := range rs {
		if i > 25 {
			break
		}
		t.Logf("%4d %s", r.v, r.k)
	}
	t.Logf("%d/%d examples failed; slowest %s (%s)", failed, len(names), slowName, slowest)
}

func runSafe(src string) (r *Result, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = panicError{p}
		}
	}()
	return Run(src, Options{Instrument: true})
}

type panicError struct{ v any }

func (p panicError) Error() string {
	return "panic: " + strings.SplitN(strings.TrimSpace(fmtAny(p.v)), "\n", 2)[0]
}

func fmtAny(v any) string {
	if e, ok := v.(error); ok {
		return e.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	return "?"
}
