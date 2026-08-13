package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fernandodeperto/hyperreader/internal/storage"
)

// handlers holds the dependencies shared by all handler methods. It is
// constructed once by NewRouter and reused across requests. hub is the
// SSE broadcast fan-out: create() publishes to it after a successful
// write, and events() lets GET /api/events subscribers receive from it.
type handlers struct {
	store Store
	hub   *hub
}

// create handles POST /api/pages — creates a new page when the request's
// slug does not already exist, or patches (full-body-replaces) the
// existing page at that slug otherwise. Slug and description are
// validated before any storage or filesystem operation runs.
//
// Request body: pageRequest (JSON). slug and name are required.
// Responses: 201 + pageResponse on create; 200 + pageResponse on patch;
// 400 on malformed JSON, an invalid slug, a missing name, or an over-limit
// description; 500 on storage failure.
func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	var req pageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON body: %s", err)
		return
	}
	if err := storage.ValidateSlug(req.Slug); err != nil {
		writeError(w, http.StatusBadRequest, "%s", err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := storage.ValidateDescription(req.Description); err != nil {
		writeError(w, http.StatusBadRequest, "%s", err)
		return
	}

	created, err := h.store.Upsert(r.Context(), storage.Doc{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		HTMLContent: req.HTML,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "upsert page: %s", err)
		return
	}

	// Read back the persisted metadata so the response (and the SSE
	// broadcast below) reflects exactly what was stored (including the
	// DB-assigned created_at/updated_at). If this fails (e.g. an immediate
	// close race) fall back to the input fields.
	resp := pageResponse{Slug: req.Slug, Name: req.Name, Description: req.Description}
	if doc, err := h.store.GetBySlug(r.Context(), req.Slug); err == nil {
		resp = toResponse(doc)
	}

	status := http.StatusOK
	eventName := "page-updated"
	if created {
		status = http.StatusCreated
		eventName = "page-created"
	}

	// Broadcast the exact same JSON representation to every GET /api/events
	// subscriber before replying to the ingest client. hub.broadcast is
	// non-blocking, so a slow or disconnected browser tab can never delay
	// or fail this response.
	if payload, err := json.Marshal(resp); err == nil {
		h.hub.broadcast(eventName, payload)
	}

	writeJSON(w, status, resp)
}

// list handles GET /api/pages — returns pages most-recently-changed-first.
// With no query it lists; with ?q= it runs an FTS5 search across
// name/description. Results are capped at storage.DefaultLimit (100) so a
// degenerate query cannot pull the whole table into memory.
func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var (
		docs []storage.Doc
		err  error
	)
	if q != "" {
		docs, err = h.store.Search(r.Context(), q)
	} else {
		docs, err = h.store.List(r.Context(), 0)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list pages: %s", err)
		return
	}
	resp := make([]pageResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, toResponse(d))
	}
	writeJSON(w, http.StatusOK, resp)
}

// get handles GET /api/pages/{slug} — returns page metadata. 404 if the
// slug does not exist.
func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	doc, err := h.store.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "page %q not found", slug)
			return
		}
		writeError(w, http.StatusInternalServerError, "get page: %s", err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(doc))
}

// getContent handles GET /api/pages/{slug}/content — returns the raw HTML
// payload with Content-Type text/html. A missing slug is 404; an existing
// metadata row whose HTML file is unreadable on disk is 500 (internal
// inconsistency), not a client error.
func (h *handlers) getContent(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	doc, err := h.store.GetBySlugContent(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "page %q not found", slug)
			return
		}
		writeError(w, http.StatusInternalServerError, "get page content: %s", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(doc.HTMLContent))
}

// toResponse converts a storage.Doc to its JSON API representation.
func toResponse(d storage.Doc) pageResponse {
	return pageResponse{
		Slug:        d.Slug,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// errorResponse is the JSON body for an error reply.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response with a formatted message.
func writeError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	writeJSON(w, code, errorResponse{Error: fmt.Sprintf(format, args...)})
}
