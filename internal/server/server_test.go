package server

import (
	"context"
	"database/sql"

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

	"github.com/fernandodeperto/hyperreader/internal/config"
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
	url := "http://127.0.0.1:" + strconv(port) + "/api/pages"
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
		t.Fatalf("GET /api/pages status = %d (body %q), want 200", status, body)
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

// TestRun_ShutdownWithActiveEventStream proves an SSE subscription does not
// hold graceful shutdown open until shutdownTimeout expires.
func TestRun_ShutdownWithActiveEventStream(t *testing.T) {
	dataDir := t.TempDir()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, &config.Config{DataDir: dataDir, Port: port})
	}()

	requestCtx, cancelRequest := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRequest()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, "http://127.0.0.1:"+strconv(port)+"/api/events", nil)
	if err != nil {
		t.Fatalf("build event stream request: %v", err)
	}

	var response *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err = http.DefaultClient.Do(request)
		if err == nil {
			break
		}
		select {
		case runErr := <-errCh:
			t.Fatalf("server exited before serving events: %v", runErr)
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	if response == nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/events status = %d, want 200", response.StatusCode)
	}

	connected := make([]byte, len(": connected\n\n"))
	if _, err := io.ReadFull(response.Body, connected); err != nil {
		t.Fatalf("read event stream connection marker: %v", err)
	}
	if string(connected) != ": connected\n\n" {
		t.Fatalf("event stream connection marker = %q", connected)
	}

	cancel()
	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("Run returned on shutdown: %v", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return promptly with an active event stream")
	}

	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatalf("read closed event stream: %v", err)
	}
}

// TestRun_ShutdownDrainsActiveRequest proves shutdown does not cancel a finite
// request that was accepted before the listener closed.
func TestRun_ShutdownDrainsActiveRequest(t *testing.T) {
	dataDir := t.TempDir()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{DataDir: dataDir, Port: port}
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, cfg)
	}()

	base := "http://127.0.0.1:" + strconv(port)
	readyDeadline := time.Now().Add(3 * time.Second)
	ready := false

	for time.Now().Before(readyDeadline) {
		response, err := http.Get(base + "/api/pages")
		if err == nil {
			response.Body.Close()
			ready = true
			break
		}
		select {
		case runErr := <-errCh:
			t.Fatalf("server exited before serving: %v", runErr)
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatal("server did not start")
	}

	lockDB, err := sql.Open("sqlite", "file:"+cfg.DBPath())
	if err != nil {
		t.Fatalf("open database lock connection: %v", err)
	}
	defer lockDB.Close()
	lockConn, err := lockDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("open database lock connection: %v", err)
	}
	defer lockConn.Close()
	if _, err := lockConn.ExecContext(context.Background(), "BEGIN EXCLUSIVE"); err != nil {
		t.Fatalf("lock database: %v", err)
	}
	defer lockConn.ExecContext(context.Background(), "ROLLBACK")

	request, err := http.NewRequest(http.MethodPost, base+"/api/pages", strings.NewReader(`{"slug":"draining-request","name":"draining request"}`))
	if err != nil {
		t.Fatalf("build ingest request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	type responseResult struct {
		response *http.Response
		err      error
	}
	responseCh := make(chan responseResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		responseCh <- responseResult{response: response, err: err}
	}()

	select {
	case result := <-responseCh:
		if result.response != nil {
			result.response.Body.Close()
		}
		t.Fatalf("request completed before shutdown: %v", result.err)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	listenerCloseDeadline := time.Now().Add(time.Second)
	listenerClosed := false
	for time.Now().Before(listenerCloseDeadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv(port), 50*time.Millisecond)
		if err != nil {
			listenerClosed = true
			break
		}
		conn.Close()
		time.Sleep(20 * time.Millisecond)
	}
	if !listenerClosed {
		t.Fatal("server listener did not close for shutdown")
	}

	if _, err := lockConn.ExecContext(context.Background(), "COMMIT"); err != nil {
		t.Fatalf("unlock database: %v", err)
	}

	select {
	case result := <-responseCh:
		if result.err != nil {
			t.Fatalf("POST /api/pages: %v", result.err)
		}
		defer result.response.Body.Close()
		if result.response.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(result.response.Body)
			t.Fatalf("POST /api/pages status = %d (body %q), want 201", result.response.StatusCode, body)
		}
	case <-time.After(time.Second):
		t.Fatal("active request did not complete during shutdown")
	}

	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("Run returned on shutdown: %v", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after the active request completed")
	}
}

// TestRun_ComposesAPIAndUI proves the T01 composition must-have: a single
// server.Run process serves the embedded UI at GET / (200, text/html) and
// the S01 API at GET /api/pages (200, "[]") through the same composed
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
		resp, err := client.Get(base + "/api/pages")
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
	if !strings.Contains(string(body), "<title>HyperReader</title>") {
		t.Fatalf("GET / body missing expected <title>; got: %s", body)
	}

	// API surface: GET /api/pages still returns [] through the composed
	// server (no regression to S01). Proves the UI catch-all does not mask
	// the API route.
	resp, err = client.Get(base + "/api/pages")
	if err != nil {
		t.Fatalf("GET /api/pages: %v", err)
	}
	apiBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET /api/pages status = %d (body %q), want 200", resp.StatusCode, apiBody)
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
