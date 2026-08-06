// Package api implements the HTTP ingest/list/get API for hyperreader.
//
// Endpoints (Go 1.22+ ServeMux method+path pattern matching):
//
//	POST /api/documents              ingest a document
//	GET  /api/documents              list (most-recent-first) or ?q= search
//	GET  /api/documents/{id}         get document metadata
//	GET  /api/documents/{id}/content get raw HTML content
//
// All handlers are backed by a Store (the storage layer from T02) injected
// via NewRouter. JSON is the wire format for metadata; the content endpoint
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
	Insert(ctx context.Context, doc storage.Doc) (int64, error)
	GetByID(ctx context.Context, id int64) (storage.Doc, error)
	GetByIDContent(ctx context.Context, id int64) (storage.Doc, error)
	List(ctx context.Context, limit int) ([]storage.Doc, error)
	Search(ctx context.Context, query string) ([]storage.Doc, error)
}

// createRequest is the JSON body accepted by POST /api/documents. Name is
// required; Description, Tags, and HTML default to empty strings.
type createRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	HTML        string `json:"html"`
}

// documentResponse is the JSON representation of a document returned by the
// API. It deliberately omits internal fields (FilePath) and the HTML payload
// (served separately via the content endpoint) to keep list/get responses
// small and to avoid leaking storage-internal paths to API consumers.
type documentResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
}

// NewRouter builds the HTTP handler tree backed by store. shutdownDone
// optionally closes when the owning HTTP server begins shutdown, causing
// long-lived event streams to return promptly. It uses the Go 1.22+ ServeMux
// with method-prefixed patterns so each route is bound to a single HTTP method;
// unsupported methods fall through to 405 Method Not Allowed automatically. The
// longer /content pattern wins over the {id} pattern for
// /api/documents/{id}/content.
//
// GET /api/events (S04) streams text/event-stream: a successful POST
// /api/documents broadcasts the created document's JSON to every subscriber
// connected to /api/events.
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
	mux.HandleFunc("POST /api/documents", h.create)
	mux.HandleFunc("GET /api/documents", h.list)
	mux.HandleFunc("GET /api/documents/{id}", h.get)
	mux.HandleFunc("GET /api/documents/{id}/content", h.getContent)
	mux.HandleFunc("GET /api/events", h.events)
	return mux, h.hub
}
