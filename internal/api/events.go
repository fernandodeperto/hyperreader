package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// heartbeatInterval is how often a ": ping" comment is sent to each
// subscriber. SSE has no application-level keepalive of its own; without a
// periodic write a client whose TCP connection dropped without a clean
// FIN/RST (a pulled network cable, a sleeping laptop) would look identical
// to a healthy idle connection until the OS eventually times it out. The
// heartbeat gives clients (and operators watching raw output) a concrete,
// short-interval signal that the stream is still alive.
const heartbeatInterval = 15 * time.Second

// eventBufferSize bounds each subscriber's outgoing channel. broadcast is
// non-blocking (see below), so this only controls how many events a
// momentarily-slow subscriber can fall behind by before further events for
// it are dropped — it does not bound memory unboundedly and it never lets
// one slow reader stall the publisher.
const eventBufferSize = 8

// hub is a broadcast fan-out for server-sent events. A single instance is
// shared by GET /api/events (subscribers register a channel) and
// POST /api/documents (the publisher broadcasts the created document's
// JSON to every current subscriber). All methods are safe for concurrent
// use from multiple goroutines, matching net/http's one-goroutine-per-request
// model.
type hub struct {
	mu           sync.Mutex
	subs         map[chan []byte]struct{}
	shutdownDone <-chan struct{}
}

// newHub returns an empty hub ready for use. shutdownDone closes when the
// owning server begins shutdown; a nil channel disables that lifecycle signal.
func newHub(shutdownDone <-chan struct{}) *hub {
	return &hub{
		subs:         make(map[chan []byte]struct{}),
		shutdownDone: shutdownDone,
	}
}

// subscribe registers a new subscriber and returns its event channel plus
// an unsubscribe function. The caller (the /api/events handler) must call
// unsubscribe exactly once, typically via defer, when the connection ends
// so the hub never leaks a channel or a slot in subs for a client that has
// gone away.
func (h *hub) subscribe() (ch chan []byte, unsubscribe func()) {
	ch = make(chan []byte, eventBufferSize)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

// broadcast sends payload to every currently-registered subscriber without
// blocking. If a subscriber's buffer is full — a slow consumer that isn't
// draining its channel as fast as events arrive — that message is dropped
// for that subscriber only; broadcast never waits. This is what guarantees
// a slow or dead browser tab can never delay or fail a POST /api/documents
// request: the publisher's call into broadcast always returns immediately.
func (h *hub) broadcast(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- payload:
		default:
			// Slow subscriber: drop this event for it rather than block
			// the publisher. The next heartbeat/event still reaches it
			// once it catches up (or its connection is eventually reaped
			// on disconnect).
		}
	}
}

// subscriberCount reports how many subscribers are currently registered.
// It is deliberately unexported — it exists only so tests in this package
// can assert on unsubscribe-on-disconnect without the hub's internals
// becoming part of the API package's public surface.
func (h *hub) subscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// events handles GET /api/events: a long-lived text/event-stream response.
// It writes a ": connected" comment immediately after the headers so a
// client (or a test) can observe that the response has actually been
// flushed rather than buffered, then relays broadcast document events and
// periodic ": ping" heartbeats until the client disconnects (request
// context cancellation) or the underlying connection is closed by the
// server shutting down.
//
// Failure modes:
//   - Client never reads (slow/hung browser tab): handled by hub.broadcast's
//     non-blocking send above; this goroutine may block on the write to the
//     socket, but that only affects this one connection, never ingest.
//   - Client disconnects mid-stream: r.Context() is cancelled by net/http,
//     the select's ctx.Done() case fires, and the deferred unsubscribe runs.
//   - ResponseWriter doesn't support flushing (should not happen with the
//     standard net/http server, but defensively checked): responds 500
//     rather than silently sending unflushed, effectively-buffered data.
func (h *handlers) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch, unsubscribe := h.hub.subscribe()
	defer unsubscribe()

	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.hub.shutdownDone:
			return
		case <-r.Context().Done():
			return
		case payload, ok := <-ch:
			if !ok {
				// Channel closed by unsubscribe (shouldn't happen while
				// this same goroutine owns the deferred unsubscribe, but
				// guarded defensively).
				return
			}
			if _, err := fmt.Fprintf(w, "event: document\ndata: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
