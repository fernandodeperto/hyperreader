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
	if !strings.Contains(string(body), "<title>HyperReader</title>") {
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
		{"/app.js", "text/javascript", "HyperReader"},
		{"/app.css", "text/css", "HyperReader"},
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

// TestHandler_TableAndSearchSurfaces is the build-time gate proving the
// embedded UI actually contains the page-table + live-FTS5-search surfaces
// this app claims to ship, so a regression that strips them fails the Go
// test suite (not just a later browser check). It asserts presence of the
// search input, the pages table skeleton, and the app.js wiring that
// drives GET /api/pages?q=<encoded> (the real FTS5 search, not a
// client-side filter). The live browser behavior is proven by the
// Playwright smoke suite; this gate only proves the surfaces are embedded.
func TestHandler_TableAndSearchSurfaces(t *testing.T) {
	// index.html must contain the search input and the pages table
	// skeleton (tbody is populated by app.js at runtime).
	html := serveBody(t, "/")
	for _, want := range []string{
		`id="search"`,
		`id="pages-table"`,
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
	// populated by a live fetch against /api/pages (not hardcoded) and
	// that search hits the same endpoint with ?q=.
	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/api/pages`,               // list/search endpoint
		`?q=`,                      // FTS5 search-param encoding
		`encodeURIComponent`,       // query is URL-encoded
		`AbortController`,          // last-write-wins guard for fast typing
		`addEventListener("input"`, // debounced search input handler
		`textContent`,              // rows built via textContent (no innerHTML injection)
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
}

// TestHandler_SameTabReaderSurfaces is the build-time gate for the embedded
// page reader. Browser tests cover behavior; this test keeps the semantic
// home link, trusted iframe, and same-tab client wiring in the Go suite.
func TestHandler_SameTabReaderSurfaces(t *testing.T) {
	html := serveBody(t, "/")
	for _, want := range []string{
		`<a id="home-link" href="/">HyperReader</a>`,
		`id="top-bar-context"`,
		`id="search"`,
		`id="selected-slug" hidden`,
		`id="live-status"`,
		`id="theme-toggle"`,
		`id="table-view"`,
		`id="page-view"`,
		`<iframe id="page-frame" title="Stored page" src="about:blank"></iframe>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
	for _, pair := range [][2]string{
		{`id="top-bar-context"`, `id="live-status"`},
		{`id="live-status"`, `id="theme-toggle"`},
	} {
		if strings.Index(html, pair[0]) >= strings.Index(html, pair[1]) {
			t.Errorf("index.html must place %q before %q", pair[0], pair[1])
		}
	}
	if strings.Contains(html, `id="page-frame" sandbox`) {
		t.Error("index.html must keep the trusted page iframe unsandboxed")
	}

	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`view: "table"`,
		`selectedSlug`,
		`document.documentElement.dataset.view`,
		`search.hidden = showingPage`,
		`selectedSlug.hidden = !showingPage`,
		`frame.src = API + "/" + encodeURIComponent(slug) + "/content"`,
		`frame.src = "about:blank"`,
		`addEventListener("click"`,
		`addEventListener("keydown"`,
		`addEventListener("load", onPageFrameLoad)`,
		`querySelectorAll(".theme")`,
		`doc.documentElement.dataset.theme = document.documentElement.dataset.theme`,
		`new Event("themechange")`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js missing %q", want)
		}
	}
	for _, gone := range []string{`window.open`, `"_blank"`} {
		if strings.Contains(js, gone) {
			t.Errorf("app.js must not contain new-tab wiring %q", gone)
		}
	}
}

// TestHandler_LiveUpdateSurfaces is the build-time gate proving the embedded
// UI contains the live-SSE-update surfaces: an icon with accessible state
// text, CSS colors for live and non-live states, and native EventSource
// wiring that handles page-created, page-updated, and malformed events.
func TestHandler_LiveUpdateSurfaces(t *testing.T) {
	html := serveBody(t, "/")
	for _, want := range []string{
		`id="live-status"`,
		`data-state="connecting"`,
		`aria-label="Connecting"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	css := serveBody(t, "/app.css")
	for _, want := range []string{
		`--status-live:`,
		`--status-offline:`,
		`#live-status[data-state="live"]`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("app.css missing %q", want)
		}
	}

	js := serveBody(t, "/app.js")
	for _, want := range []string{
		`/api/events`,                     // SSE subscribe endpoint
		`EventSource`,                     // native browser streaming client
		`addEventListener("page-created"`, // creation event listener
		`addEventListener("page-updated"`, // patch event listener
		`dataset.state`,                   // drives #live-status's data-state attribute
		`aria-label`,                      // state stays available without visible badge text
		`"Live"`,
		`"Connecting"`,
		`"Reconnecting"`,
		`JSON.parse`,    // decodes the event payload
		`catch`,         // malformed (non-JSON) payload is caught, not thrown
		`console.error`, // malformed payload is logged, not swallowed silently
		`unshift`,       // page prepended as the new top row
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
