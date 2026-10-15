// Command build builds the studio into a static site (default ./site):
//
//	go run ./cmd/build [-o site]
//
// The site is the page (rendered with shadcn.go), its stylesheet (generated
// by the Go port of Tailwind CSS), the components' runtime, the engine
// compiled to WebAssembly, the canvas document and the examples.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alexisbcz/components/ui/runtime"
	h "github.com/alexisbcz/libhtml"
	"github.com/alexisbcz/studio/engine"
	_ "github.com/alexisbcz/studio/symbols" // every component, for the runtime modules
	"github.com/alexisbcz/tailwind"
)

func main() {
	out := flag.String("o", "site", "output directory")
	skipWasm := flag.Bool("skip-wasm", false, "do not rebuild studio.wasm")
	flag.Parse()
	start := time.Now()
	must(os.MkdirAll(*out, 0o755))

	if !*skipWasm {
		step("studio.wasm", func() {
			cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", filepath.Join(*out, "studio.wasm"), "./cmd/wasm")
			cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			must(cmd.Run())
		})
	}
	wasmInfo, err := os.Stat(filepath.Join(*out, "studio.wasm"))
	must(err)

	step("wasm_exec.js", func() {
		root, err := exec.Command("go", "env", "GOROOT").Output()
		must(err)
		dir := strings.TrimSpace(string(root))
		src := filepath.Join(dir, "lib", "wasm", "wasm_exec.js")
		if _, err := os.Stat(src); err != nil {
			src = filepath.Join(dir, "misc", "wasm", "wasm_exec.js")
		}
		copyFile(src, filepath.Join(*out, "wasm_exec.js"))
	})

	step("runtime", func() { exportRuntime(filepath.Join(*out, "_shadcn")) })

	var examples map[string]string
	step("examples", func() {
		b, err := os.ReadFile("examples/go-examples.json")
		must(err)
		must(json.Unmarshal(b, &examples))
		copyFile("examples/go-examples.json", filepath.Join(*out, "examples.json"))
		p, _ := json.Marshal(engine.Palette)
		write(filepath.Join(*out, "palette.json"), p)
	})

	step("web", func() {
		must(filepath.Walk("web", func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			copyFile(p, filepath.Join(*out, strings.TrimPrefix(p, "web/")))
			return nil
		}))
	})

	// The version busts caches: a hash of what the page loads.
	version := hashFiles(*out)
	names := make([]string, 0, len(examples))
	for n := range examples {
		names = append(names, n)
	}
	sort.Strings(names)

	step("index.html", func() {
		html, err := h.RenderString(page(names, version))
		must(err)
		write(filepath.Join(*out, "index.html"), []byte(html))
		manifest, _ := json.Marshal(map[string]any{"version": version, "wasmSize": wasmInfo.Size()})
		write(filepath.Join(*out, "studio.json"), manifest)
	})

	theme, err := os.ReadFile("web/theme.css")
	must(err)
	step("studio.css", func() {
		css := `@import "tailwindcss";
@import "tw-animate-css";
@import "shadcn/tailwind.css";
` + string(theme) + `
@source "./index.html";
@source "./studio.js";
@source "./preview.js";

/* Code colors (GitHub's). */
:root {
  --cm-keyword: #cf222e;
  --cm-string: #0a3069;
  --cm-comment: #6e7781;
  --cm-number: #0550ae;
  --cm-function: #8250df;
  --cm-type: #953800;
  --cm-property: #0550ae;
}
.dark {
  --cm-keyword: #ff7b72;
  --cm-string: #a5d6ff;
  --cm-comment: #8b949e;
  --cm-number: #79c0ff;
  --cm-function: #d2a8ff;
  --cm-type: #ffa657;
  --cm-property: #79c0ff;
}
`
		buildCSS(*out, css, "studio.css")
	})

	step("preview.css", func() {
		// Every class the components and the examples may render; classes
		// only a user's code has are compiled in the canvas by Tailwind's
		// browser build.
		mod, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/alexisbcz/components").Output()
		must(err)
		dir := strings.TrimSpace(string(mod))
		css := `@import "tailwindcss";
@import "tw-animate-css";
@import "shadcn/tailwind.css";
` + string(theme) + `
@source "` + filepath.ToSlash(dir) + `/**/*.go";
@source "` + filepath.ToSlash(dir) + `/style/css";
@source "./examples.json";
@source "./palette.json";
`
		buildCSS(*out, css, "preview.css")
		// The canvas's own compiler gets the theme and the CSS packages
		// inlined (it cannot import them).
		twMod, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/alexisbcz/tailwind").Output()
		must(err)
		var b bytes.Buffer
		for _, f := range []string{"tw-animate-css/tw-animate.css", "shadcn/tailwind.css"} {
			src, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(twMod)), "dist", "pkg", f))
			must(err)
			b.Write(src)
			b.WriteString("\n")
		}
		b.Write(theme)
		tpl, err := os.ReadFile("web/preview.html")
		must(err)
		html := strings.Replace(string(tpl), "/*THEME*/", b.String(), 1)
		html = strings.ReplaceAll(html, "{{version}}", version)
		write(filepath.Join(*out, "preview.html"), []byte(html))
	})
	log.Printf("built %s in %s", *out, time.Since(start).Round(time.Millisecond))
}

func buildCSS(dir, input, name string) {
	res, err := tailwind.Build(tailwind.Options{CSS: input, Dir: dir, Output: filepath.Join(dir, name), Minify: true})
	must(err)
	write(filepath.Join(dir, name), []byte(res.CSS))
}

// exportRuntime writes the runtime modules: the index, and every module it
// imports, recursively.
func exportRuntime(dir string) {
	must(os.MkdirAll(dir, 0o755))
	handler := runtime.Handler()
	importRe := regexp.MustCompile(`(?:from|import)\s*\(?\s*"(\./[^"]+\.js)"`)
	seen := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", runtime.Prefix+name, nil))
		if rec.Code != 200 {
			log.Fatalf("runtime %s: %d", name, rec.Code)
		}
		body := rec.Body.Bytes()
		write(filepath.Join(dir, name), body)
		for _, m := range importRe.FindAllSubmatch(body, -1) {
			visit(strings.TrimPrefix(string(m[1]), "./"))
		}
	}
	visit("index.js")
	for _, n := range runtime.Names() {
		visit(n + ".js")
	}
}

func hashFiles(dir string) string {
	sum := sha256.New()
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && !strings.HasSuffix(p, ".html") && !strings.HasSuffix(p, ".css") && !strings.HasSuffix(p, "studio.json") {
			f, err := os.Open(p)
			if err == nil {
				io.WriteString(sum, p)
				io.Copy(sum, f)
				f.Close()
			}
		}
		return nil
	})
	return hex.EncodeToString(sum.Sum(nil))[:10]
}

func step(name string, fn func()) {
	t := time.Now()
	fn()
	log.Printf("%-14s %s", name, time.Since(t).Round(time.Millisecond))
}

func copyFile(src, dst string) {
	b, err := os.ReadFile(src)
	must(err)
	write(dst, b)
}

func write(path string, b []byte) {
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	must(os.WriteFile(path, b, 0o644))
}

func must(err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("build: %w", err))
	}
}
