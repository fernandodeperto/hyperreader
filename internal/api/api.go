// Package api implements the HTTP ingest/list/get API for hyperreader.
//
// Endpoints (Go 1.22+ ServeMux method+path pattern matching):
//
//	POST /api/pages              create a new page, or patch an existing one (by slug)
//	GET  /api/pages              list (most-recently-changed-first) or ?q= search
//	GET  /api/pages/{slug}       get page metadata
//	GET  /api/pages/{slug}/content get raw HTML content
//
// All handlers are backed by a Store (the storage layer) injected via
// NewRouter. JSON is the wire format for metadata; the content endpoint
// returns the raw HTML bytes with Content-Type text/html so a detail view
// can render them directly.
package api

import (
	"context"
	"net/http"

	"github.com/fernandodeperto/hyperreader/internal/storage"
)

// Store is the subset of the storage layer the API depends on. *storage.Store
// satisfies it implicitly; defining it as an interface keeps handlers
// testable with a fake and decouples the API from storage internals.
type Store interface {
	Upsert(ctx context.Context, doc storage.Doc) (created bool, err error)
	GetBySlug(ctx context.Context, slug string) (storage.Doc, error)
	GetBySlugContent(ctx context.Context, slug string) (storage.Doc, error)
	List(ctx context.Context, limit int) ([]storage.Doc, error)
	Search(ctx context.Context, query string) ([]storage.Doc, error)
}

// pageRequest is the JSON body accepted by POST /api/pages. It carries the
// complete state of a page: there is no partial-patch shape, so create and
// patch use the exact same request body. Slug and Name are required;
// Description and HTML default to empty strings.
type pageRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HTML        string `json:"html"`
}

// pageResponse is the JSON representation of a page returned by the API. It
// deliberately omits internal fields (FilePath) and the HTML payload
// (served separately via the content endpoint) to keep list/get responses
// small and to avoid leaking storage-internal paths to API consumers.
type pageResponse struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// NewRouter builds the HTTP handler tree backed by store. shutdownDone
// optionally closes when the owning HTTP server begins shutdown, causing
// long-lived event streams to return promptly. It uses the Go 1.22+ ServeMux
// with method-prefixed patterns so each route is bound to a single HTTP method;
// unsupported methods fall through to 405 Method Not Allowed automatically. The
// longer /content pattern wins over the {slug} pattern for
// /api/pages/{slug}/content.
//
// GET /api/events streams text/event-stream: a successful POST /api/pages
// broadcasts the written page's JSON to every subscriber connected to
// /api/events, as a page-created event for a new slug or a page-updated
// event for a patch.
func NewRouter(store Store, shutdownDone <-chan struct{}) http.Handler {
	mux, _ := newRouterAndHub(store, shutdownDone)
	return mux
}

// newRouterAndHub builds the router exactly as NewRouter does but also returns
// the underlying event hub. It is unexported and exists solely so tests in this
// package can inspect subscriber counts (e.g. proving unsubscribe-on-disconnect)
// without the hub's internals becoming part of the API package's public surface.
func newRouterAndHub(store Store, shutdownDone <-chan struct{}) (http.Handler, *hub) {
	mux := http.NewServeMux()
	h := &handlers{store: store, hub: newHub(shutdownDone)}
	mux.HandleFunc("POST /api/pages", h.create)
	mux.HandleFunc("GET /api/pages", h.list)
	mux.HandleFunc("GET /api/pages/{slug}", h.get)
	mux.HandleFunc("GET /api/pages/{slug}/content", h.getContent)
	mux.HandleFunc("GET /api/events", h.events)
	return mux, h.hub
}
