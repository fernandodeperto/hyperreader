package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fmendonca/html-mcp/internal/storage"
)

// handlers holds the dependencies shared by all handler methods. It is
// constructed once by NewRouter and reused across requests. hub is the
// SSE broadcast fan-out (S04): create() publishes to it after a successful
// insert, and events() lets GET /api/events subscribers receive from it.
type handlers struct {
	store Store
	hub   *hub
}

// create handles POST /api/documents — ingests a document (metadata to
// SQLite, HTML to disk) and returns the created document's metadata.
//
// Request body: createRequest (JSON). name is required.
// Responses: 201 + documentResponse on success; 400 on malformed JSON or
// missing name; 500 on storage failure.
func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON body: %s", err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	id, err := h.store.Insert(r.Context(), storage.Doc{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		HTMLContent: req.HTML,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert document: %s", err)
		return
	}

	// Read back the persisted metadata so the response (and the SSE
	// broadcast below) reflects exactly what was stored (including the
	// DB-assigned created_at). If this fails (e.g. an immediate close
	// race) fall back to the input fields.
	resp := documentResponse{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
	}
	if doc, err := h.store.GetByID(r.Context(), id); err == nil {
		resp = toResponse(doc)
	}

	// Broadcast the exact same JSON representation to every GET /api/events
	// subscriber before replying to the ingest client. hub.broadcast is
	// non-blocking, so a slow or disconnected browser tab can never delay
	// or fail this response — the integration closure constraint for S04.
	if payload, err := json.Marshal(resp); err == nil {
		h.hub.broadcast(payload)
	}

	writeJSON(w, http.StatusCreated, resp)
}

// list handles GET /api/documents — returns documents most-recent-first.
// With no query it lists; with ?q= it runs an FTS5 search across
// name/description/tags. Results are capped at storage.DefaultLimit (100)
// so a degenerate query cannot pull the whole table into memory.
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
		writeError(w, http.StatusInternalServerError, "list documents: %s", err)
		return
	}
	resp := make([]documentResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, toResponse(d))
	}
	writeJSON(w, http.StatusOK, resp)
}

// get handles GET /api/documents/{id} — returns document metadata.
// 400 for a non-numeric id; 404 if the id does not exist.
func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	doc, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "document %d not found", id)
			return
		}
		writeError(w, http.StatusInternalServerError, "get document: %s", err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(doc))
}

// getContent handles GET /api/documents/{id}/content — returns the raw HTML
// payload with Content-Type text/html. A missing id is 404; an existing
// metadata row whose HTML file is unreadable on disk is 500 (internal
// inconsistency), not a client error.
func (h *handlers) getContent(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	doc, err := h.store.GetByIDContent(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "document %d not found", id)
			return
		}
		writeError(w, http.StatusInternalServerError, "get document content: %s", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(doc.HTMLContent))
}

// parseID extracts and validates the {id} path parameter. On failure it
// writes a 400 response and returns ok=false so the caller can return early.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid document id %q", idStr)
		return 0, false
	}
	return id, true
}

// toResponse converts a storage.Doc to its JSON API representation.
func toResponse(d storage.Doc) documentResponse {
	return documentResponse{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		Tags:        d.Tags,
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
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
