package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandler_ServesIndex proves the composition gate for T01: the embedded
// static UI is reachable at GET / and returns the index.html shell as
// text/html. This is the asset surface T02/T03 build on, so it must be
// served before any UI behavior lands.
func TestHandler_ServesIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want text/html prefix", ct)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "<title>html-mcp</title>") {
		t.Fatalf("GET / body does not contain the expected <title>; got: %s", body)
	}
}

// TestHandler_ServesAssets proves the secondary static assets (app.js,
// app.css) are served by path with browser-usable content types. The page
// references them via absolute paths, so they must resolve on the same
// origin as the API.
func TestHandler_ServesAssets(t *testing.T) {
	cases := []struct {
		path        string
		wantPrefix  string
		wantSnippet string
	}{
		{"/app.js", "text/javascript", "html-mcp"},
		{"/app.css", "text/css", "html-mcp"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", c.path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, c.wantPrefix) {
			t.Fatalf("GET %s Content-Type = %q, want %s prefix", c.path, ct, c.wantPrefix)
		}
		body, _ := io.ReadAll(rec.Body)
		if !strings.Contains(string(body), c.wantSnippet) {
			t.Fatalf("GET %s body missing snippet %q", c.path, c.wantSnippet)
		}
	}
}

// TestHandler_UnknownPath404 proves unknown paths fall through to 404 rather
// than silently serving index.html (which would mask missing routes). This
// guards the API/UI boundary: a typo'd /api path must not be swallowed by
// the catch-all UI handler when composed.
func TestHandler_UnknownPath404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/no-such-asset", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /no-such-asset status = %d, want 404", rec.Code)
	}
}

// TestHandler_TableAndSearchSurfaces is the T02 build-time gate: it proves
// the embedded UI actually contains the document-table + live-FTS5-search
// surfaces T02 claims to ship, so a regression that strips them fails the
// Go test suite (not just a later browser check). It asserts presence of
// the search input, the documents table skeleton, and the app.js wiring
// that drives GET /api/documents?q=<encoded> (the real S01 FTS5 search,
// not a client-side filter). The live browser behavior is proven by T03's
// Playwright smoke; this gate only proves the surfaces are embedded.
func TestHandler_TableAndSearchSurfaces(t *testing.T) {
	// index.html must contain the search input and the documents table
	// skeleton (tbody is populated by app.js at runtime).
	html := serveBody(t, "/")
	for _, want := range []string{
		`id="search"`,
		`id="documents-table"`,
		`<tbody></tbody>`,
		`id="empty-state"`,
		`id="error-message"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	// app.js must drive the real FTS5 API: the list endpoint, the ?q=
	// search-param encoding, the AbortController last-write-wins guard, and
	// the debounced input handler. Their presence proves the table is
	// populated by a live fetch against /api/documents (not hardcoded) and
	// that search hits the same endpoint with ?q=.
	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/api/documents`, // list/search endpoint
		`?q=`,             // FTS5 search-param encoding
		`encodeURIComponent`, // query is URL-encoded
		`AbortController`, // last-write-wins guard for fast typing
		`addEventListener("input"`, // debounced search input handler
		`textContent`, // rows built via textContent (no innerHTML injection)
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
}

// TestHandler_NewTabSurfaces is the M002 / Branch B build-time gate: it
// proves the embedded UI opens a document's raw rendered HTML in a new
// browser tab via GET /api/documents/{id}/content with zero app chrome,
// and that the in-app detail-view/iframe/Back surfaces have been removed
// entirely (not hidden). It asserts the app.js wiring (window.open to the
// content endpoint with _blank, row click + keydown activation) and guards
// against a regression that silently reintroduces the removed iframe/view.
// The live browser behavior (click opens a new tab, script executes
// unsandboxed at top level) is proven by T03's Playwright specs; this gate
// only proves the surfaces are embedded, so a regression that strips them
// fails the Go test suite (not just a later browser check).
func TestHandler_NewTabSurfaces(t *testing.T) {
	html := serveBody(t, "/")
	// The removed detail-view surfaces must be GONE entirely (regression
	// guard for the Branch B removal — a reintroduction that hides rather
	// than deletes them would re-add these and fail here).
	for _, gone := range []string{
		`id="detail-view"`,
		`id="detail-frame"`,
		`id="back-button"`,
		`<iframe`,
	} {
		if strings.Contains(html, gone) {
			t.Errorf("index.html must not contain removed surface %q (Branch B deletes the detail view)", gone)
		}
	}

	// app.js must wire the new-tab open: the content endpoint, window.open
	// with _blank, and row activation via click + keydown delegation.
	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/content`,                  // GET /api/documents/{id}/content
		`window.open`,              // opens the content endpoint in a new tab
		`_blank`,                   // target is a new tab, not in-app
		`encodeURIComponent`,       // id is URL-encoded into the path
		`addEventListener("click"`,  // row click delegation
		`addEventListener("keydown"`, // row keyboard activation (Enter/Space)
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
}

// TestHandler_LiveUpdateSurfaces is the S04-T02 build-time gate: it proves
// the embedded UI contains the live-SSE-update surfaces this task claims to
// ship — the #live-status indicator (with its data-state attribute) in
// index.html, and the app.js wiring that subscribes to GET /api/events via
// a native EventSource, mirrors its lifecycle into #live-status, prepends
// broadcast "document" events as new rows, and guards against a malformed
// event payload (invalid JSON, or JSON that isn't a document-shaped object)
// by logging via console.error and skipping rather than throwing. The live
// browser behavior (a document appears with no refresh) is proven by T03's
// Playwright spec; this gate only proves the surfaces are embedded, so a
// regression that strips them fails the Go test suite (not just a later
// browser check).
func TestHandler_LiveUpdateSurfaces(t *testing.T) {
	html := serveBody(t, "/")
	for _, want := range []string{
		`id="live-status"`,
		`data-state="connecting"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/api/events`,                // SSE subscribe endpoint
		`EventSource`,                // native browser streaming client
		`addEventListener("document"`, // broadcast event name from internal/api
		`dataset.state`,              // drives #live-status's data-state attribute
		`JSON.parse`,                 // decodes the event payload
		`catch`,                      // malformed (non-JSON) payload is caught, not thrown
		`console.error`,              // malformed payload is logged, not swallowed silently
		`unshift`,                    // new document prepended as the new top row
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
}

// serveBody fetches a path from the embedded UI handler and returns its
// body as a string. Fails the test on a non-200 response.
func serveBody(t *testing.T, path string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read %s body: %v", path, err)
	}
	return string(body)
}
