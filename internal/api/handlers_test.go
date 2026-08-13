package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fernandodeperto/hyperreader/internal/storage"
)

// newRouter opens a real storage.Store against a temp dir and returns a
// router backed by it plus the store (so tests can assert on storage state
// directly). The DB is closed via t.Cleanup; t.TempDir is auto-removed.
func newRouter(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "pages.db"), filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return NewRouter(store, nil), store
}

// doJSON sends a JSON request and returns the decoded body plus status code.
// It fails the test on transport error; callers assert on status/body.
func doJSON(t *testing.T, h http.Handler, method, target, body string) (int, map[string]interface{}) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out map[string]interface{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec.Code, out
}

// write is a helper that POSTs a page create-or-patch request and returns
// the decoded body plus status code.
func write(t *testing.T, h http.Handler, slug, name, desc, html string) (int, map[string]interface{}) {
	t.Helper()
	body := `{"slug":` + jsonString(slug) +
		`,"name":` + jsonString(name) +
		`,"description":` + jsonString(desc) +
		`,"html":` + jsonString(html) + `}`
	return doJSON(t, h, http.MethodPost, "/api/pages", body)
}

// create is a helper that POSTs a page expected to create (201) and returns
// the created slug.
func create(t *testing.T, h http.Handler, slug, name, desc, html string) string {
	t.Helper()
	code, resp := write(t, h, slug, name, desc, html)
	if code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%v)", code, resp)
	}
	got, _ := resp["slug"].(string)
	if got != slug {
		t.Fatalf("create: response slug = %q, want %q", got, slug)
	}
	return got
}

// jsonString marshals s to a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// --- happy-path integration tests against a real storage.Store ---

func TestCreate_List_Get_RoundTrip(t *testing.T) {
	router, store := newRouter(t)

	// create then list shows the page
	create(t, router, "deploy-runbook", "Deploy Runbook", "production deploy", "<h1>Deploy</h1>")
	create(t, router, "on-call-guide", "On-call Guide", "rotation schedule", "<h1>OnCall</h1>")

	// JSON arrays unmarshal to []interface{}; decode explicitly.
	got := decodeList(t, router, "/api/pages")
	if len(got) != 2 {
		t.Fatalf("list: expected 2 pages, got %d (%+v)", len(got), got)
	}
	// most-recently-changed-first: "on-call-guide" was written second
	if got[0]["slug"] != "on-call-guide" {
		t.Errorf("list[0].slug = %v, want on-call-guide", got[0]["slug"])
	}
	_ = store // store available for direct storage assertions; unused here.
}

func TestCreate_ReturnsCreatedMetadata(t *testing.T) {
	router, _ := newRouter(t)

	create(t, router, "my-page", "My Page", "a desc", "<p>body</p>")
	code, resp := doJSON(t, router, http.MethodPost, "/api/pages",
		`{"slug":"second-page","name":"Second","description":"d","html":"<b/>"}`)
	if code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%v)", code, resp)
	}
	if resp["slug"] != "second-page" {
		t.Errorf("created slug = %v, want second-page", resp["slug"])
	}
	if resp["name"] != "Second" {
		t.Errorf("created name = %v, want Second", resp["name"])
	}
	if resp["description"] != "d" {
		t.Errorf("created description = %v, want d", resp["description"])
	}
	if _, ok := resp["created_at"].(string); !ok {
		t.Errorf("created_at should be a string, got %T", resp["created_at"])
	}
	if _, ok := resp["updated_at"].(string); !ok {
		t.Errorf("updated_at should be a string, got %T", resp["updated_at"])
	}
	// internal FilePath must not leak to API consumers; tags no longer exist.
	if _, ok := resp["file_path"]; ok {
		t.Errorf("file_path leaked into API response: %v", resp["file_path"])
	}
	if _, ok := resp["tags"]; ok {
		t.Errorf("tags leaked into API response: %v", resp["tags"])
	}
}

func TestPatch_SameSlug_Returns200AndReplacesContent(t *testing.T) {
	router, _ := newRouter(t)

	create(t, router, "changelog", "Changelog", "v1", "<p>v1</p>")
	firstCode, first := doJSON(t, router, http.MethodGet, "/api/pages/changelog", "")
	if firstCode != http.StatusOK {
		t.Fatalf("get after create: expected 200, got %d", firstCode)
	}

	code, resp := write(t, router, "changelog", "Changelog v2", "v2", "<p>v2</p>")
	if code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d (%v)", code, resp)
	}
	if resp["name"] != "Changelog v2" || resp["description"] != "v2" {
		t.Errorf("patched metadata = %v, want name/description updated", resp)
	}
	if resp["created_at"] != first["created_at"] {
		t.Errorf("created_at changed across a patch: %v -> %v", first["created_at"], resp["created_at"])
	}
	if resp["updated_at"] == first["updated_at"] {
		t.Errorf("updated_at did not advance across a patch: still %v", resp["updated_at"])
	}

	// Exactly one page exists at this slug; no duplicate created.
	got := decodeList(t, router, "/api/pages")
	if len(got) != 1 {
		t.Fatalf("list after a patch returned %d page(s), want 1", len(got))
	}

	// Prior content is not retrievable after the patch.
	req := httptest.NewRequest(http.MethodGet, "/api/pages/changelog/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Body.String() != "<p>v2</p>" {
		t.Errorf("content after patch = %q, want %q", rec.Body.String(), "<p>v2</p>")
	}
}

func TestSearch_ByEachField(t *testing.T) {
	router, _ := newRouter(t)

	// Seed one page per field so each search term is unique to that field.
	create(t, router, "alpha-page", "AlphaName", "filler", "a")
	create(t, router, "beta-page", "zzz2", "BetaDescription", "b")

	tests := []struct {
		name     string
		query    string
		wantSlug string
	}{
		{"name", "AlphaName", "alpha-page"},
		{"description", "BetaDescription", "beta-page"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := decodeList(t, router, "/api/pages?q="+tc.query)
			if len(got) != 1 {
				t.Fatalf("search %q: expected 1 result, got %d (%+v)", tc.query, len(got), got)
			}
			if got[0]["slug"] != tc.wantSlug {
				t.Errorf("search %q: slug = %v, want %v", tc.query, got[0]["slug"], tc.wantSlug)
			}
		})
	}
}

func TestSearch_MultiFieldMatch(t *testing.T) {
	router, _ := newRouter(t)

	// "blueprint" appears in name and description across two pages.
	create(t, router, "page-a", "blueprint filler", "x", "a")
	create(t, router, "page-b", "x", "blueprint filler", "b")
	create(t, router, "page-c", "unrelated", "nope", "c")

	got := decodeList(t, router, "/api/pages?q=blueprint")
	if len(got) != 2 {
		t.Fatalf("search blueprint: expected 2 results, got %d (%+v)", len(got), got)
	}
}

func TestGetBySlug_ReturnsCorrectMetadata(t *testing.T) {
	router, _ := newRouter(t)
	create(t, router, "the-page", "The Page", "its desc", "<x/>")

	code, resp := doJSON(t, router, http.MethodGet, "/api/pages/the-page", "")
	if code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d (%v)", code, resp)
	}
	if resp["slug"] != "the-page" {
		t.Errorf("get slug = %v, want the-page", resp["slug"])
	}
	if resp["name"] != "The Page" {
		t.Errorf("get name = %v", resp["name"])
	}
	if resp["description"] != "its desc" {
		t.Errorf("get description = %v", resp["description"])
	}
	if _, ok := resp["file_path"]; ok {
		t.Errorf("get: file_path should not leak: %v", resp["file_path"])
	}
	if _, ok := resp["html"]; ok {
		t.Errorf("get: html should not be in metadata response: %v", resp["html"])
	}
}

func TestGetContent_ReturnsStoredHTML(t *testing.T) {
	router, _ := newRouter(t)
	html := "<h1>Title</h1><p>Body with <em>markup</em></p>"
	create(t, router, "doc-page", "Doc", "d", html)

	req := httptest.NewRequest(http.MethodGet, "/api/pages/doc-page/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get content: expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if rec.Body.String() != html {
		t.Errorf("content body = %q, want %q", rec.Body.String(), html)
	}
}

// --- negative tests ---

func TestGetBySlug_MissingSlug_Returns404(t *testing.T) {
	router, _ := newRouter(t)
	code, resp := doJSON(t, router, http.MethodGet, "/api/pages/nonexistent", "")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing slug, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field in 404 body, got %v", resp)
	}
}

func TestGetContent_MissingSlug_Returns404(t *testing.T) {
	router, _ := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/pages/nonexistent/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing content slug, got %d", rec.Code)
	}
}

func TestCreate_MissingName_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, resp := doJSON(t, router, http.MethodPost, "/api/pages", `{"slug":"no-name","description":"no name"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field in 400 body, got %v", resp)
	}
}

func TestCreate_InvalidSlug_Returns400_NoFileNoRow(t *testing.T) {
	router, store := newRouter(t)

	tests := []struct {
		name string
		slug string
	}{
		{"path separator", "not/a-slug"},
		{"traversal", "..-etc"},
		{"uppercase", "Not-Ok"},
		{"leading dash", "-not-ok"},
		{"over length", strings.Repeat("a", 81)},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			code, resp := doJSON(t, router, http.MethodPost, "/api/pages",
				`{"slug":`+jsonString(tc.slug)+`,"name":"x","html":"x"}`)
			if code != http.StatusBadRequest {
				t.Fatalf("expected 400 for invalid slug %q, got %d (%v)", tc.slug, code, resp)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("expected error field naming the allowed pattern, got %v", resp)
			}
		})
	}

	docs, err := store.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("List after rejected slugs returned %d row(s), want 0", len(docs))
	}
}

func TestCreate_OverLongDescription_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, resp := doJSON(t, router, http.MethodPost, "/api/pages",
		`{"slug":"too-long","name":"x","description":`+jsonString(strings.Repeat("d", 201))+`,"html":"x"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-limit description, got %d (%v)", code, resp)
	}
	if !strings.Contains(resp["error"].(string), "200") {
		t.Errorf("error %v does not name the 200-character limit", resp["error"])
	}
}

func TestCreate_MalformedJSON_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, _ := doJSON(t, router, http.MethodPost, "/api/pages", `{not json`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", code)
	}
}

func TestSearch_NoMatch_ReturnsEmptyArrayNotError(t *testing.T) {
	router, _ := newRouter(t)
	create(t, router, "real-page", "real page", "real desc", "a")
	got := decodeList(t, router, "/api/pages?q=nonexistentterm")
	if len(got) != 0 {
		t.Fatalf("expected empty result for no match, got %d (%+v)", len(got), got)
	}
}

func TestList_EmptyStore_ReturnsEmptyArrayNotNull(t *testing.T) {
	router, _ := newRouter(t)
	got := decodeList(t, router, "/api/pages")
	if got == nil {
		t.Fatal("expected empty array, got nil (null in JSON)")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 pages, got %d", len(got))
	}
}

func TestWrongMethod_Returns405(t *testing.T) {
	router, _ := newRouter(t)
	// DELETE is not registered on /api/pages
	req := httptest.NewRequest(http.MethodDelete, "/api/pages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for DELETE, got %d", rec.Code)
	}
}

// --- error-path isolation via fake store ---

func TestCreate_StoreError_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{upsertErr: errors.New("disk full")}, nil)
	code, resp := doJSON(t, router, http.MethodPost, "/api/pages", `{"slug":"x","name":"x"}`)
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field, got %v", resp)
	}
}

func TestGet_StoreError_NotErrNoRows_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{getErr: errors.New("connection lost")}, nil)
	code, _ := doJSON(t, router, http.MethodGet, "/api/pages/x", "")
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", code)
	}
}

func TestList_StoreError_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{listErr: errors.New("boom")}, nil)
	code, _ := doJSON(t, router, http.MethodGet, "/api/pages", "")
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", code)
	}
}

// --- helpers ---

// decodeList performs a GET and decodes the JSON array body.
func decodeList(t *testing.T, h http.Handler, target string) []map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d (%s)", target, rec.Code, rec.Body.String())
	}
	var out []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("GET %s: decode: %v (body=%q)", target, err, rec.Body.String())
	}
	return out
}

// fakeStore implements Store for error-path isolation. Each method returns
// its configured error (or zero value), so handlers can be exercised
// against a failing backend without touching SQLite.
type fakeStore struct {
	upsertErr error
	getErr    error
	listErr   error
	searchErr error
}

func (f *fakeStore) Upsert(ctx context.Context, doc storage.Doc) (bool, error) {
	return false, f.upsertErr
}
func (f *fakeStore) GetBySlug(ctx context.Context, slug string) (storage.Doc, error) {
	if f.getErr != nil {
		return storage.Doc{}, f.getErr
	}
	return storage.Doc{}, sql.ErrNoRows
}
func (f *fakeStore) GetBySlugContent(ctx context.Context, slug string) (storage.Doc, error) {
	if f.getErr != nil {
		return storage.Doc{}, f.getErr
	}
	return storage.Doc{}, sql.ErrNoRows
}
func (f *fakeStore) List(ctx context.Context, limit int) ([]storage.Doc, error) {
	return nil, f.listErr
}
func (f *fakeStore) Search(ctx context.Context, query string) ([]storage.Doc, error) {
	return nil, f.searchErr
}
