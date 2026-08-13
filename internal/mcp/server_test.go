// Copyright 2025.
package mcp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connectClient wires a real mcp.Client to the send_html server over an
// in-memory transport pair (mcp.NewInMemoryTransports), exercising the same
// protocol path a real subprocess-driven agent would use, minus the actual
// stdio pipe. It returns the connected client session and a cleanup func.
func connectClient(t *testing.T, port int) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()

	srv := NewServer(port)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}

	return clientSession, func() {
		clientSession.Close()
		serverSession.Close()
	}
}

// freeUnreachablePort returns a TCP port that is guaranteed to have nothing
// listening on it: it opens a listener, records the port, then closes the
// listener before returning, so a subsequent dial to it fails with
// "connection refused" rather than racing a real listener's lifecycle.
func freeUnreachablePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return port
}

// portOf extracts the numeric TCP port an httptest.Server is bound to, so
// the mcp server under test can be told to forward to it via the same
// port-based addressing the real serve/mcp process pair uses.
func portOf(t *testing.T, ts *httptest.Server) int {
	t.Helper()
	addr := ts.Listener.Addr().(*net.TCPAddr)
	return addr.Port
}

func callSendHTML(t *testing.T, cs *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "send_html", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(send_html): protocol-level error (should never happen for tool failures): %v", err)
	}
	return res
}

func resultText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// TestNewServer_IdentifiesAsHyperReader proves the MCP initialization
// identity moved with the project rename: a connected client's
// InitializeResult must report the server as "hyperreader", never the
// legacy "html-mcp" name, since an MCP client's host UI and logs surface
// this value directly to the user.
func TestNewServer_IdentifiesAsHyperReader(t *testing.T) {
	cs, cleanup := connectClient(t, 0)
	defer cleanup()

	info := cs.InitializeResult()
	if info == nil || info.ServerInfo == nil {
		t.Fatal("InitializeResult().ServerInfo is nil")
	}
	if info.ServerInfo.Name != "hyperreader" {
		t.Errorf("ServerInfo.Name = %q, want %q", info.ServerInfo.Name, "hyperreader")
	}
}

// TestSendHTML_Success proves the happy path end-to-end through a real MCP
// client: send_html forwards to a fake serve backend and the tool result
// reports success (not IsError) naming the page and slug.
func TestSendHTML_Success(t *testing.T) {
	var gotBody forwardRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/pages" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(pageResponse{
			Slug:      gotBody.Slug,
			Name:      gotBody.Name,
			CreatedAt: "2025-01-01T00:00:00Z",
			UpdatedAt: "2025-01-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{
		"slug":        "my-doc",
		"name":        "My Doc",
		"html":        "<h1>hi</h1>",
		"description": "a test doc",
	})

	if res.IsError {
		t.Fatalf("expected success, got IsError=true: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "My Doc") || !strings.Contains(text, "my-doc") {
		t.Errorf("result text %q does not name the page/slug", text)
	}
	if !strings.Contains(text, "created") {
		t.Errorf("result text %q does not say the page was created", text)
	}
	if gotBody.Slug != "my-doc" || gotBody.Name != "My Doc" || gotBody.HTML != "<h1>hi</h1>" || gotBody.Description != "a test doc" {
		t.Errorf("forwarded body mismatch: %+v", gotBody)
	}
}

// TestSendHTML_Success_Patch proves a second write to the same slug is
// reported as an update (HTTP 200 from serve), not a creation, so the
// agent can tell the two apart from the result text alone.
func TestSendHTML_Success_Patch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body forwardRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(pageResponse{Slug: body.Slug, Name: body.Name})
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "changelog", "name": "Changelog v2", "html": "<p/>"})

	if res.IsError {
		t.Fatalf("expected success, got IsError=true: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "updated") {
		t.Errorf("result text %q does not say the page was updated", text)
	}
}

// TestSendHTML_ServeNotRunning proves failure path 1: no listener on the
// target port. forward() must return a connection-refused-flavored error
// that the tool surfaces as IsError=true, never a protocol-level Go error.
func TestSendHTML_ServeNotRunning(t *testing.T) {
	port := freeUnreachablePort(t)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true when serve is not running, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "not reachable") {
		t.Errorf("result text %q does not explain serve is unreachable", text)
	}
}

// TestSendHTML_HTTPError proves failure path 2: serve is up but rejects the
// request (4xx/5xx). forward() must surface serve's status + error body as
// IsError=true text, not a Go error.
func TestSendHTML_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: "html is required"})
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true on HTTP 400, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "400") || !strings.Contains(text, "html is required") {
		t.Errorf("result text %q does not surface status + serve error body", text)
	}
}

// TestSendHTML_MalformedResponse proves failure path 3: serve returns 201
// but a body that is not valid pageResponse JSON. forward()'s decode
// step must fail and surface as IsError=true, not a Go error or a panic.
func TestSendHTML_MalformedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true on malformed response body, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "decode") {
		t.Errorf("result text %q does not explain the decode failure", text)
	}
}

// TestSendHTML_MissingName proves failure path 4: a client-supplied
// argument that fails the tool's own precondition (empty name) is rejected
// before any network call, and surfaces as IsError=true — not a schema
// validation error the agent cannot act on, and not a call to serve at all.
func TestSendHTML_MissingName(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true when name is empty, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "name is required") {
		t.Errorf("result text %q does not explain the missing name", text)
	}
	if called {
		t.Errorf("expected no request to serve when name is empty; forward() should fail fast")
	}
}

// TestSendHTML_MissingSlug proves the slug-required precondition: an empty
// slug is rejected before any network call, surfacing as IsError=true, not
// a call to serve at all. Slug is the page's sole identifier, so this fails
// exactly as fast as the pre-existing missing-name check.
func TestSendHTML_MissingSlug(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true when slug is empty, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "slug is required") {
		t.Errorf("result text %q does not explain the missing slug", text)
	}
	if called {
		t.Errorf("expected no request to serve when slug is empty; forward() should fail fast")
	}
}

// TestSendHTML_HTTPError_EmptyBody proves readServeError's first fallback:
// when serve returns a non-2xx status with no body at all (no {"error":...}
// envelope to parse), the tool result must still say something actionable
// — the HTTP status text — rather than an empty or missing message.
func TestSendHTML_HTTPError_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true on HTTP 500 with an empty body, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "500") {
		t.Errorf("result text %q does not name the HTTP status", text)
	}
	if !strings.Contains(text, http.StatusText(http.StatusInternalServerError)) {
		t.Errorf("result text %q does not fall back to the HTTP status text when the body is empty", text)
	}
}

// TestSendHTML_HTTPError_NonJSONBody proves readServeError's second
// fallback: when serve (or some intermediary) returns a non-2xx status with
// a body that isn't the {"error":...} envelope — plain text, an HTML error
// page, etc. — the raw body text must still surface verbatim rather than
// being silently dropped because it failed to unmarshal as JSON.
func TestSendHTML_HTTPError_NonJSONBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("  upstream timed out  \n"))
	}))
	defer ts.Close()

	port := portOf(t, ts)
	cs, cleanup := connectClient(t, port)
	defer cleanup()

	res := callSendHTML(t, cs, map[string]any{"slug": "x", "name": "x", "html": "<p>x</p>"})

	if !res.IsError {
		t.Fatalf("expected IsError=true on HTTP 502 with a non-JSON body, got success: %s", resultText(res))
	}
	text := resultText(res)
	if !strings.Contains(text, "502") {
		t.Errorf("result text %q does not name the HTTP status", text)
	}
	if !strings.Contains(text, "upstream timed out") {
		t.Errorf("result text %q does not surface the raw non-JSON body", text)
	}
}
