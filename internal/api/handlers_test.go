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

	"github.com/fmendonca/hyperreader/internal/storage"
)

// newRouter opens a real storage.Store against a temp dir and returns a
// router backed by it plus the store (so tests can assert on storage state
// directly). The DB is closed via t.Cleanup; t.TempDir is auto-removed.
func newRouter(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "docs.db"), filepath.Join(dir, "files"))
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

// ingest is a helper that POSTs a document and returns the created doc's id.
func ingest(t *testing.T, h http.Handler, name, desc, tags, html string) int64 {
	t.Helper()
	body := `{"name":` + jsonString(name) +
		`,"description":` + jsonString(desc) +
		`,"tags":` + jsonString(tags) +
		`,"html":` + jsonString(html) + `}`
	code, resp := doJSON(t, h, http.MethodPost, "/api/documents", body)
	if code != http.StatusCreated {
		t.Fatalf("ingest: expected 201, got %d (%v)", code, resp)
	}
	id, _ := resp["id"].(float64)
	if id <= 0 {
		t.Fatalf("ingest: no positive id in response: %v", resp)
	}
	return int64(id)
}

// jsonString marshals s to a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// --- happy-path integration tests against a real storage.Store ---

func TestCreate_List_Get_RoundTrip(t *testing.T) {
	router, store := newRouter(t)

	// ingest then list shows the doc
	ingest(t, router, "Deploy Runbook", "production deploy", "ops,deploy", "<h1>Deploy</h1>")
	ingest(t, router, "On-call Guide", "rotation schedule", "ops,oncall", "<h1>OnCall</h1>")

	// JSON arrays unmarshal to []interface{}; decode explicitly.
	got := decodeList(t, router, "/api/documents")
	if len(got) != 2 {
		t.Fatalf("list: expected 2 docs, got %d (%+v)", len(got), got)
	}
	// most-recent-first: "On-call Guide" was inserted second
	if got[0]["name"] != "On-call Guide" {
		t.Errorf("list[0].name = %v, want On-call Guide", got[0]["name"])
	}
	_ = store // store available for direct storage assertions; unused here.
}

func TestCreate_ReturnsCreatedMetadata(t *testing.T) {
	router, _ := newRouter(t)

	id := ingest(t, router, "My Doc", "a desc", "t1,t2", "<p>body</p>")
	code, resp := doJSON(t, router, http.MethodPost, "/api/documents",
		`{"name":"Second","description":"d","tags":"t","html":"<b/>"}`)
	if code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%v)", code, resp)
	}
	createdID, _ := resp["id"].(float64)
	if int64(createdID) != id+1 {
		t.Errorf("created id = %v, want %d", createdID, id+1)
	}
	if resp["name"] != "Second" {
		t.Errorf("created name = %v, want Second", resp["name"])
	}
	if resp["description"] != "d" {
		t.Errorf("created description = %v, want d", resp["description"])
	}
	if resp["tags"] != "t" {
		t.Errorf("created tags = %v, want t", resp["tags"])
	}
	if _, ok := resp["created_at"].(string); !ok {
		t.Errorf("created_at should be a string, got %T", resp["created_at"])
	}
	// internal FilePath must not leak to API consumers
	if _, ok := resp["file_path"]; ok {
		t.Errorf("file_path leaked into API response: %v", resp["file_path"])
	}
}

func TestSearch_ByEachField(t *testing.T) {
	router, _ := newRouter(t)

	// Seed one doc per field so each search term is unique to that field.
	ingest(t, router, "AlphaName", "filler", "zzz1", "a")
	ingest(t, router, "zzz2", "BetaDescription", "filler", "b")
	ingest(t, router, "zzz3", "filler", "GammaTags", "c")

	tests := []struct {
		name     string
		query    string
		wantName string
	}{
		{"name", "AlphaName", "AlphaName"},
		{"description", "BetaDescription", "zzz2"},
		{"tags", "GammaTags", "zzz3"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := decodeList(t, router, "/api/documents?q="+tc.query)
			if len(got) != 1 {
				t.Fatalf("search %q: expected 1 result, got %d (%+v)", tc.query, len(got), got)
			}
			if got[0]["name"] != tc.wantName {
				t.Errorf("search %q: name = %v, want %v", tc.query, got[0]["name"], tc.wantName)
			}
		})
	}
}

func TestSearch_MultiFieldMatch(t *testing.T) {
	router, _ := newRouter(t)

	// "blueprint" appears in name, description, and tags across three docs.
	ingest(t, router, "blueprint filler", "x", "y", "a")
	ingest(t, router, "x", "blueprint filler", "y", "b")
	ingest(t, router, "x", "y", "blueprint", "c")
	ingest(t, router, "unrelated", "nope", "none", "d")

	got := decodeList(t, router, "/api/documents?q=blueprint")
	if len(got) != 3 {
		t.Fatalf("search blueprint: expected 3 results, got %d (%+v)", len(got), got)
	}
}

func TestGetByID_ReturnsCorrectMetadata(t *testing.T) {
	router, _ := newRouter(t)
	id := ingest(t, router, "The Doc", "its desc", "a,b", "<x/>")

	code, resp := doJSON(t, router, http.MethodGet, "/api/documents/"+itoa(id), "")
	if code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d (%v)", code, resp)
	}
	if resp["id"] != float64(id) {
		t.Errorf("get id = %v, want %d", resp["id"], id)
	}
	if resp["name"] != "The Doc" {
		t.Errorf("get name = %v", resp["name"])
	}
	if resp["description"] != "its desc" {
		t.Errorf("get description = %v", resp["description"])
	}
	if resp["tags"] != "a,b" {
		t.Errorf("get tags = %v", resp["tags"])
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
	id := ingest(t, router, "Doc", "d", "t", html)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+itoa(id)+"/content", nil)
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

func TestGetByID_MissingID_Returns404(t *testing.T) {
	router, _ := newRouter(t)
	code, resp := doJSON(t, router, http.MethodGet, "/api/documents/9999", "")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field in 404 body, got %v", resp)
	}
}

func TestGetContent_MissingID_Returns404(t *testing.T) {
	router, _ := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/documents/9999/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing content id, got %d", rec.Code)
	}
}

func TestCreate_MissingName_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, resp := doJSON(t, router, http.MethodPost, "/api/documents", `{"description":"no name"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field in 400 body, got %v", resp)
	}
}

func TestCreate_MalformedJSON_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, _ := doJSON(t, router, http.MethodPost, "/api/documents", `{not json`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", code)
	}
}

func TestGetByID_NonNumericID_Returns400(t *testing.T) {
	router, _ := newRouter(t)
	code, _ := doJSON(t, router, http.MethodGet, "/api/documents/abc", "")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric id, got %d", code)
	}
}

func TestSearch_NoMatch_ReturnsEmptyArrayNotError(t *testing.T) {
	router, _ := newRouter(t)
	ingest(t, router, "real doc", "real desc", "real", "a")
	got := decodeList(t, router, "/api/documents?q=nonexistentterm")
	if len(got) != 0 {
		t.Fatalf("expected empty result for no match, got %d (%+v)", len(got), got)
	}
}

func TestList_EmptyStore_ReturnsEmptyArrayNotNull(t *testing.T) {
	router, _ := newRouter(t)
	got := decodeList(t, router, "/api/documents")
	if got == nil {
		t.Fatal("expected empty array, got nil (null in JSON)")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 docs, got %d", len(got))
	}
}

func TestWrongMethod_Returns405(t *testing.T) {
	router, _ := newRouter(t)
	// DELETE is not registered on /api/documents
	req := httptest.NewRequest(http.MethodDelete, "/api/documents", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for DELETE, got %d", rec.Code)
	}
}

// --- error-path isolation via fake store ---

func TestCreate_StoreError_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{insertErr: errors.New("disk full")}, nil)
	code, resp := doJSON(t, router, http.MethodPost, "/api/documents", `{"name":"x"}`)
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d (%v)", code, resp)
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error field, got %v", resp)
	}
}

func TestGet_StoreError_NotErrNoRows_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{getErr: errors.New("connection lost")}, nil)
	code, _ := doJSON(t, router, http.MethodGet, "/api/documents/1", "")
	if code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on store error, got %d", code)
	}
}

func TestList_StoreError_Returns500(t *testing.T) {
	router := NewRouter(&fakeStore{listErr: errors.New("boom")}, nil)
	code, _ := doJSON(t, router, http.MethodGet, "/api/documents", "")
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

// itoa is a tiny strconv-free int64->string to keep imports lean.
func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

// fakeStore implements Store for error-path isolation. Each method returns
// its configured error (or zero value), so handlers can be exercised
// against a failing backend without touching SQLite.
type fakeStore struct {
	insertErr error
	getErr    error
	listErr   error
	searchErr error
}

func (f *fakeStore) Insert(ctx context.Context, doc storage.Doc) (int64, error) {
	return 0, f.insertErr
}
func (f *fakeStore) GetByID(ctx context.Context, id int64) (storage.Doc, error) {
	if f.getErr != nil {
		return storage.Doc{}, f.getErr
	}
	return storage.Doc{}, sql.ErrNoRows
}
func (f *fakeStore) GetByIDContent(ctx context.Context, id int64) (storage.Doc, error) {
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
