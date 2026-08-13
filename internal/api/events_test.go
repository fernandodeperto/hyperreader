package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fernandodeperto/hyperreader/internal/storage"
)

// newEventsTestServer opens a real storage.Store against a temp dir, builds
// the real router (not a fake), and serves it over a real network listener
// via httptest.NewServer. A real server (rather than httptest.Recorder) is
// required here because SSE relies on genuine streaming semantics —
// flushed-but-unfinished responses, a live connection a client can cancel
// mid-read — that a ResponseRecorder does not faithfully reproduce.
func newEventsTestServer(t *testing.T) (*httptest.Server, *hub) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "pages.db"), filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	router, h := newRouterAndHub(store, nil)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, h
}

// subscribeSSE opens a GET /api/events request against srv and returns a
// buffered reader over the streaming body plus a cancel func that both
// cancels the request context and closes the body (simulating a client
// disconnect, e.g. a closed browser tab).
func subscribeSSE(t *testing.T, srv *httptest.Server) (*bufio.Reader, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("GET /api/events: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		t.Fatalf("GET /api/events: status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		resp.Body.Close()
		cancel()
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		resp.Body.Close()
		cancel()
		t.Fatalf("Cache-Control = %q, want no-cache", cc)
	}
	return bufio.NewReader(resp.Body), func() {
		cancel()
		resp.Body.Close()
	}
}

// readLine reads one line from r (with the trailing newline stripped),
// failing the test if no line arrives within timeout. Reading happens in a
// goroutine so a stalled stream fails the test deterministically instead of
// hanging the whole suite.
func readLine(t *testing.T, r *bufio.Reader, timeout time.Duration) string {
	t.Helper()
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadString('\n')
		ch <- result{line, err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("read line: %v", res.err)
		}
		return strings.TrimRight(res.line, "\n")
	case <-time.After(timeout):
		t.Fatal("timed out waiting for a line from the event stream")
		return ""
	}
}

// httpPost POSTs body to url and fails the test unless the response is a
// successful page write (201 create or 200 patch).
func httpPost(t *testing.T, url, body string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: status = %d, want 201 or 200", url, resp.StatusCode)
	}
}

// waitForSubscriberCount polls h.subscriberCount() until it equals want or
// a short deadline elapses. Subscriber registration happens asynchronously
// relative to the client-side HTTP call returning, so a poll (rather than a
// fixed sleep) avoids both flakiness and unnecessary slowness.
func waitForSubscriberCount(t *testing.T, h *hub, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.subscriberCount() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("subscriberCount = %d, want %d (timed out waiting)", h.subscriberCount(), want)
}

// --- happy-path SSE behavior ---

func TestEvents_ConnectedCommentFlushedImmediately(t *testing.T) {
	srv, _ := newEventsTestServer(t)

	r, closeStream := subscribeSSE(t, srv)
	defer closeStream()

	line := readLine(t, r, 2*time.Second)
	if !strings.HasPrefix(line, ": connected") {
		t.Fatalf("first line = %q, want a ': connected' comment", line)
	}
	// A blank line terminates every SSE frame.
	blank := readLine(t, r, 2*time.Second)
	if blank != "" {
		t.Fatalf("line after ': connected' = %q, want blank frame terminator", blank)
	}
}

func TestEvents_BroadcastOnCreate_CarriesPageCreatedEvent(t *testing.T) {
	srv, _ := newEventsTestServer(t)

	r, closeStream := subscribeSSE(t, srv)
	defer closeStream()

	// Drain the initial ": connected" frame before triggering the event
	// under test, so the assertions below only see the page event.
	readLine(t, r, 2*time.Second) // ": connected"
	readLine(t, r, 2*time.Second) // blank terminator

	httpPost(t, srv.URL+"/api/pages", `{"slug":"live-page","name":"Live Page","description":"d","html":"<p>hi</p>"}`)

	eventLine := readLine(t, r, 2*time.Second)
	if eventLine != "event: page-created" {
		t.Fatalf("event line = %q, want %q", eventLine, "event: page-created")
	}
	dataLine := readLine(t, r, 2*time.Second)
	if !strings.HasPrefix(dataLine, "data: ") {
		t.Fatalf("data line = %q, want prefix %q", dataLine, "data: ")
	}
	// The broadcast payload must be the exact same pageResponse JSON shape
	// POST/GET already return.
	if !strings.Contains(dataLine, `"name":"Live Page"`) {
		t.Errorf("data line missing page name: %q", dataLine)
	}
	if !strings.Contains(dataLine, `"description":"d"`) {
		t.Errorf("data line missing page description: %q", dataLine)
	}
	if !strings.Contains(dataLine, `"slug":"live-page"`) {
		t.Errorf("data line missing page slug: %q", dataLine)
	}
	if strings.Contains(dataLine, "\n") {
		t.Errorf("data line must be single-line JSON, got embedded newline: %q", dataLine)
	}
}

// TestEvents_BroadcastOnPatch_CarriesPageUpdatedEvent proves the
// live-page-updates contract: patching an existing slug broadcasts a
// page-updated event (not a second page-created), carrying that slug's
// updated metadata.
func TestEvents_BroadcastOnPatch_CarriesPageUpdatedEvent(t *testing.T) {
	srv, _ := newEventsTestServer(t)

	httpPost(t, srv.URL+"/api/pages", `{"slug":"changelog","name":"Changelog","description":"v1","html":"<p>v1</p>"}`)

	r, closeStream := subscribeSSE(t, srv)
	defer closeStream()
	readLine(t, r, 2*time.Second) // ": connected"
	readLine(t, r, 2*time.Second) // blank terminator

	httpPost(t, srv.URL+"/api/pages", `{"slug":"changelog","name":"Changelog v2","description":"v2","html":"<p>v2</p>"}`)

	eventLine := readLine(t, r, 2*time.Second)
	if eventLine != "event: page-updated" {
		t.Fatalf("event line = %q, want %q", eventLine, "event: page-updated")
	}
	dataLine := readLine(t, r, 2*time.Second)
	if !strings.Contains(dataLine, `"slug":"changelog"`) {
		t.Errorf("data line missing page slug: %q", dataLine)
	}
	if !strings.Contains(dataLine, `"name":"Changelog v2"`) {
		t.Errorf("data line missing patched name: %q", dataLine)
	}
}

func TestEvents_MultipleSubscribers_AllReceiveBroadcast(t *testing.T) {
	srv, h := newEventsTestServer(t)

	r1, close1 := subscribeSSE(t, srv)
	defer close1()
	r2, close2 := subscribeSSE(t, srv)
	defer close2()

	waitForSubscriberCount(t, h, 2)

	readLine(t, r1, 2*time.Second)
	readLine(t, r1, 2*time.Second)
	readLine(t, r2, 2*time.Second)
	readLine(t, r2, 2*time.Second)

	httpPost(t, srv.URL+"/api/pages", `{"slug":"fan-out-page","name":"Fan-out Page","html":"<p/>"}`)

	for i, r := range []*bufio.Reader{r1, r2} {
		eventLine := readLine(t, r, 2*time.Second)
		if eventLine != "event: page-created" {
			t.Fatalf("subscriber %d: event line = %q, want %q", i, eventLine, "event: page-created")
		}
		dataLine := readLine(t, r, 2*time.Second)
		if !strings.Contains(dataLine, `"name":"Fan-out Page"`) {
			t.Errorf("subscriber %d: data line missing page name: %q", i, dataLine)
		}
	}
}

// --- failure-mode / negative tests ---

// TestEvents_SlowSubscriberDoesNotBlockIngest proves the Must-Have that a
// slow or disconnected subscriber never blocks, fails, or delays ingest. A
// subscriber connects but never reads its response body at all; writing
// well past the hub's per-subscriber buffer size must still complete
// quickly because hub.broadcast drops for a full subscriber channel instead
// of blocking the publisher.
func TestEvents_SlowSubscriberDoesNotBlockIngest(t *testing.T) {
	srv, h := newEventsTestServer(t)

	_, closeStream := subscribeSSE(t, srv)
	defer closeStream()
	waitForSubscriberCount(t, h, 1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range eventBufferSize + 5 {
			httpPost(t, srv.URL+"/api/pages", `{"slug":"slow-sub-page","name":"Page","html":"<p/>"}`)
		}
	}()

	select {
	case <-done:
		// Ingest completed without blocking on the slow (non-reading)
		// subscriber, proving broadcast's non-blocking guarantee.
	case <-time.After(5 * time.Second):
		t.Fatal("ingest blocked on a slow subscriber; hub.broadcast must never block the publisher")
	}
}

// TestEvents_UnsubscribeOnDisconnect proves the Must-Have that a
// disconnected subscriber is cleaned up: cancelling the client request
// context (simulating a closed browser tab) must bring subscriberCount
// back down, so the hub never accumulates dead channels for gone clients.
func TestEvents_UnsubscribeOnDisconnect(t *testing.T) {
	srv, h := newEventsTestServer(t)

	_, closeStream := subscribeSSE(t, srv)
	waitForSubscriberCount(t, h, 1)

	closeStream() // cancels the context and closes the body.

	waitForSubscriberCount(t, h, 0)
}

// TestEvents_SubscribeAndDisconnectMultipleTimes proves repeated
// connect/disconnect cycles never leak subscribers, guarding against a
// unsubscribe implementation that only works once or leaks on repeat use.
func TestEvents_SubscribeAndDisconnectMultipleTimes(t *testing.T) {
	srv, h := newEventsTestServer(t)

	for range 3 {
		_, closeStream := subscribeSSE(t, srv)
		waitForSubscriberCount(t, h, 1)
		closeStream()
		waitForSubscriberCount(t, h, 0)
	}
}

func TestHub_BroadcastWithNoSubscribers_DoesNotPanicOrBlock(t *testing.T) {
	h := newHub(nil)
	done := make(chan struct{})
	go func() {
		h.broadcast("page-created", []byte(`{"slug":"x"}`))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("broadcast with zero subscribers blocked")
	}
}

func TestEvents_WrongMethod_Returns405(t *testing.T) {
	srv, _ := newEventsTestServer(t)
	resp, err := http.Post(srv.URL+"/api/events", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /api/events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/events: status = %d, want 405", resp.StatusCode)
	}
}
