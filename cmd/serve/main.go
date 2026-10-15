// Command serve serves the built site for development:
//
//	go run ./cmd/build && go run ./cmd/serve   # http://localhost:8091
package main

import (
	"compress/gzip"
	"flag"
	"log"
	"mime"
	"net/http"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8091", "listen address")
	dir := flag.String("dir", "site", "site directory")
	flag.Parse()
	_ = mime.AddExtensionType(".wasm", "application/wasm")
	files := http.FileServer(http.Dir(*dir))
	log.Printf("serving %s on http://%s", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			files.ServeHTTP(w, r)
			return
		}
		// Compressed, like GitHub Pages serves it.
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		gz, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		defer gz.Close()
		files.ServeHTTP(gzipWriter{w, gz}, r)
	})))
}

type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g gzipWriter) Write(b []byte) (int, error) { return g.gz.Write(b) }
func (g gzipWriter) WriteHeader(code int) {
	g.ResponseWriter.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}
