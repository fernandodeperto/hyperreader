// Package web embeds the static single-page UI for HyperReader and serves it
// from the same origin and port as the HTTP API. The assets (index.html,
// app.js, app.css) live alongside this file in the web/ directory so
// go:embed can reference them directly — go:embed patterns cannot traverse
// parent directories, so the embed glue and the assets must share a
// directory. (The plan listed internal/web/embed.go, but internal/web/
// cannot embed repo-root web/ files; co-locating preserves every asset
// path that T02/T03 depend on.)
//
// Handler() returns an http.Handler mounted at "/" on the serve mux; the
// API is mounted at "/api/" alongside it (see server.composeRouter). Go's
// ServeMux longest-prefix matching routes /api/... to the API and
// everything else to the UI, so both share the single bound port.
package web

import (
	"embed"
	"net/http"
	"strings"
)

// assets holds the static UI files compiled into the binary at build time.
// go:embed pulls index.html, app.js, and app.css from this package's
// directory into the binary so the serve process can serve the UI with no
// external file dependencies.
//
//go:embed index.html app.js app.css
var assets embed.FS

// Handler returns an http.Handler that serves the embedded static UI. It is
// mounted at "/" on the serve mux.
//
// A GET under /read/ is a client-side reader route (/read/<slug>) with no
// embedded asset of its own; serve the index.html shell so app.js can
// restore the page view from the URL on a full reload. Every other unknown
// path still falls through to the file server's 404 so a typo'd route is
// not masked by the SPA shell.
func Handler() http.Handler {
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/read/") {
			serveIndex(w)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// serveIndex writes the embedded index.html shell as text/html. Used for
// SPA deep-links (/read/<slug>) that carry no asset of their own.
func serveIndex(w http.ResponseWriter) {
	data, err := assets.ReadFile("index.html")
	if err != nil {
		http.Error(w, "index unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
