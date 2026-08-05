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

// TestHandler_DetailViewSurfaces is the T03 build-time gate: it proves
// the embedded UI contains the unsandboxed detail-view surfaces T03
// claims to ship — the detail-view section, the iframe with NO sandbox
// attribute, the Back button, and the app.js wiring that fetches the
// content endpoint and renders into the iframe via srcdoc. The live
// browser behavior (click opens detail, script executes unsandboxed, Back
// restores the table) is proven by T03's Playwright smoke; this gate only
// proves the surfaces are embedded, so a regression that strips them fails
// the Go test suite (not just a later browser check).
func TestHandler_DetailViewSurfaces(t *testing.T) {
	html := serveBody(t, "/")
	// The detail-view section and its iframe + Back button must be present.
	for _, want := range []string{
		`id="detail-view"`,
		`id="detail-frame"`,
		`id="back-button"`,
		`<iframe`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
	// R006: the iframe must NOT carry a sandbox attribute — its absence is
	// what makes agent-authored inline scripts and CDN references execute.
	// Assert the opening iframe tag has no sandbox.
	if hasSandbox(html) {
		t.Errorf("index.html: #detail-frame iframe must not have a sandbox attribute (R006); got iframe tag with sandbox")
	}

	// app.js must wire the detail view: the content endpoint, srcdoc
	// rendering, the Back handler, and the row click handler.
	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/content`,            // GET /api/documents/{id}/content
		`srcdoc`,              // iframe rendered via srcdoc (unsandboxed)
		`back-button`,        // Back button handler
		`addEventListener("click"`, // row click delegation
		`setView`,             // table/detail view toggle
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

// hasSandbox reports whether the iframe tag in the HTML carries a sandbox
// attribute. It locates the <iframe opening tag and checks for "sandbox"
// within it. Used by the R006 unsandboxed-rendering assertion.
func hasSandbox(html string) bool {
	idx := strings.Index(strings.ToLower(html), "<iframe")
	if idx < 0 {
		return false
	}
	// Scan from <iframe up to the closing '>' of that tag.
	rest := html[idx:]
	end := strings.Index(rest, ">")
	if end < 0 {
		return false
	}
	tag := strings.ToLower(rest[:end])
	return strings.Contains(tag, "sandbox")
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
