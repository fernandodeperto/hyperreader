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
// http.FileServer serves index.html for "/" (its directory-index behavior)
// and serves app.js / app.css by path with browser-usable content types.
// The embedded FS root is the web/ directory, so no prefix stripping is
// needed when mounted at "/". Unknown paths return 404.
func Handler() http.Handler {
	return http.FileServer(http.FS(assets))
}
