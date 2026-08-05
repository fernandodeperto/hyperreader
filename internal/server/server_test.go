package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fmendonca/html-mcp/internal/config"
)

// TestRun_AlreadyRunningGuard_FailsFast exercises the core behavioral
// guarantee of T04: a second instance trying to bind an in-use port fails
// fast with a clear "already running" message, before opening the database.
//
// The first instance binds a free port directly; the second instance's
// bindPort must return the translated error rather than hanging or
// silently colliding.
func TestRun_AlreadyRunningGuard_FailsFast(t *testing.T) {
	// Bind the first instance exactly as production does (wildcard :port),
	// so the second bindPort hits the same local address and conflicts.
	// Binding 127.0.0.1 instead would NOT conflict with a wildcard bind
	// on macOS, masking the real two-instance behavior.
	ln, err := bindPort(0) // :0 -> OS assigns a free wildcard port
	if err != nil {
		t.Fatalf("bind first instance: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	_, err = bindPort(port)
	if err == nil {
		t.Fatal("second bind succeeded; expected address-already-in-use error")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected error to mention 'already running', got: %v", err)
	}
	if !strings.Contains(err.Error(), "in use") {
		t.Fatalf("expected error to mention 'in use', got: %v", err)
	}
}

// TestRun_BootstrapAndServe covers the full happy path: server.Run binds a
// free port, opens storage on a temp data dir, serves the API, and shuts
// down cleanly when the context is cancelled. It issues a real HTTP
// request against the live server to prove the API router is mounted and
// answering (empty-store list -> "[]").
func TestRun_BootstrapAndServe(t *testing.T) {
	dataDir := t.TempDir()

	// Find a free port, then release it so server.Run can rebind it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := &config.Config{DataDir: dataDir, Port: port}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, cfg) }()

	// Poll until the server is accepting connections.
	url := "http://127.0.0.1:" + strconv(port) + "/api/documents"
	client := &http.Client{Timeout: time.Second}
	var status int
	var body string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			select {
			case err := <-errCh:
				t.Fatalf("server exited before serving: %v", err)
			default:
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		body = string(b)
		break
	}
	if status != 200 {
		t.Fatalf("GET /api/documents status = %d (body %q), want 200", status, body)
	}
	if got := strings.TrimSpace(body); got != "[]" {
		t.Fatalf("empty store list body = %q, want []", got)
	}

	// Cancel -> graceful shutdown, Run returns nil.
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run returned error on shutdown: %v", err)
	}
}

// TestRun_ComposesAPIAndUI proves the T01 composition must-have: a single
// server.Run process serves the embedded UI at GET / (200, text/html) and
// the S01 API at GET /api/documents (200, "[]") through the same composed
// mux on one port. This guards against two regressions: (a) the UI handler
// masking the API via the catch-all "/", and (b) mounting the UI breaking
// the already-passing API empty-list response. It complements the existing
// bootstrap test by asserting both surfaces, not just the API.
func TestRun_ComposesAPIAndUI(t *testing.T) {
	dataDir := t.TempDir()

	// Find a free port, then release it so server.Run can rebind it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := &config.Config{DataDir: dataDir, Port: port}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, cfg) }()

	base := "http://127.0.0.1:" + strconv(port)
	client := &http.Client{Timeout: time.Second}

	// Poll until the server is accepting connections.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/api/documents")
		if err != nil {
			select {
			case err := <-errCh:
				t.Fatalf("server exited before serving: %v", err)
			default:
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		break
	}

	// UI surface: GET / serves the embedded index.html as text/html.
	resp, err := client.Get(base + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET / status = %d (body %q), want 200", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want text/html prefix", ct)
	}
	if !strings.Contains(string(body), "<title>html-mcp</title>") {
		t.Fatalf("GET / body missing expected <title>; got: %s", body)
	}

	// API surface: GET /api/documents still returns [] through the composed
	// server (no regression to S01). Proves the UI catch-all does not mask
	// the API route.
	resp, err = client.Get(base + "/api/documents")
	if err != nil {
		t.Fatalf("GET /api/documents: %v", err)
	}
	apiBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /api/documents status = %d (body %q), want 200", resp.StatusCode, apiBody)
	}
	if got := strings.TrimSpace(string(apiBody)); got != "[]" {
		t.Fatalf("empty store list body = %q, want []", got)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run returned error on shutdown: %v", err)
	}
}

// TestRun_DataDirFailure_ReleasesPort proves the bootstrap-order invariant
// documented on Run: if a step after bindPort fails (here,
// config.EnsureDataDir because cfg.DataDir collides with an existing
// regular file), the listener must be closed before Run returns — the
// port must not be leaked. It asserts this concretely by rebinding the
// exact same port immediately after Run returns its error.
func TestRun_DataDirFailure_ReleasesPort(t *testing.T) {
	// A path that cannot become a directory: a regular file already sits at
	// the exact path Run will try to os.MkdirAll.
	parent := t.TempDir()
	blockedDataDir := filepath.Join(parent, "not-a-dir")
	if err := os.WriteFile(blockedDataDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("pre-create blocking file: %v", err)
	}

	// Find a free port, then release it so Run can bind it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := &config.Config{DataDir: blockedDataDir, Port: port}

	runErr := Run(context.Background(), cfg)
	if runErr == nil {
		t.Fatal("Run: expected an error when DataDir collides with an existing file, got nil")
	}

	// The port must be free again immediately — Run's bindPort listener was
	// closed on this failure path, not held open.
	ln2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port %d still bound after Run's DataDir failure (listener leaked): %v", port, err)
	}
	ln2.Close()
}

// TestIsAddrInUse confirms the error-classifier distinguishes a real
// address-in-use bind failure from an unrelated listen error.
func TestIsAddrInUse(t *testing.T) {
	// Real in-use case: bind twice.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	_, err2 := net.Listen("tcp", ln.Addr().String())
	if err2 == nil {
		t.Fatal("second listen on same addr succeeded; expected error")
	}
	if !isAddrInUse(err2) {
		t.Fatalf("isAddrInUse should be true for: %v", err2)
	}

	// Unrelated error: a non-network error must not be classified as in-use.
	if isAddrInUse(errors.New("not a network error")) {
		t.Fatal("isAddrInUse should be false for a non-network error")
	}
}

// strconv is a tiny local int->string helper so the test file stays focused
// on server behavior; strconv.Itoa would do the same but keeping the
// helper local avoids pulling strconv solely for port formatting.
func strconv(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
