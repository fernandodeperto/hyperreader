// Package server wires the serve subcommand bootstrap: a resolved config ->
// storage layer -> API router -> bound HTTP listener. It is the single
// entry point for "hyperreader serve".
//
// The already-running guard binds the configured TCP port BEFORE opening the
// database, so a second instance fails fast with a clear message rather than
// racing on the SQLite file (no DB corruption, no silent port conflict).
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fernandodeperto/hyperreader/internal/api"
	"github.com/fernandodeperto/hyperreader/internal/config"
	"github.com/fernandodeperto/hyperreader/internal/storage"
	"github.com/fernandodeperto/hyperreader/web"
)

// shutdownTimeout bounds graceful shutdown when the context is cancelled,
// draining in-flight requests before closing the listener.
const shutdownTimeout = 5 * time.Second

// Run bootstraps and serves the hyperreader HTTP API until ctx is cancelled.
//
// Bootstrap order:
//  1. Bind the configured port (the already-running guard). A second
//     instance fails here with a clear error before touching the database.
//  2. Ensure the XDG data dir exists.
//  3. Open and migrate the SQLite+FTS5 storage layer (T02).
//  4. Mount the composed router (API + embedded UI) and serve HTTP on the bound listener.
//
// On ctx cancellation Run initiates a graceful shutdown (in-flight requests
// drain up to shutdownTimeout) and returns nil. A port-already-in-use bind
// failure is translated into an actionable "already running" error.
func Run(ctx context.Context, cfg *config.Config) error {
	// 1. Already-running guard: bind first so a second instance fails fast.
	ln, err := bindPort(cfg.Port)
	if err != nil {
		return err
	}

	// 2. Ensure the data dir exists (storage.Open also creates filesDir,
	// but the parent data dir must exist for the db file to be writable).
	if _, err := config.EnsureDataDir(cfg.DataDir); err != nil {
		ln.Close()
		return err
	}

	// 3. Open + migrate storage. The listener is held open across this so
	// that if migration fails the port is released on exit, not leaked.
	store, err := storage.Open(cfg.DBPath(), cfg.FilesDir())
	if err != nil {
		ln.Close()
		return err
	}
	defer store.Close()

	// 4. Mount the composed router (API + embedded UI on one mux) and serve
	// on the bound listener. Both share the single bound port: the API at
	// /api/ and the static UI at / (catch-all).
	// WriteTimeout is deliberately left unset (zero value = no timeout).
	// S04's GET /api/events is a long-lived text/event-stream response that
	// can legitimately stay open for the lifetime of a browser tab; a
	// non-zero WriteTimeout would truncate it mid-stream. ReadHeaderTimeout
	// only bounds how long a client has to finish sending request headers
	// (the slowloris mitigation) and is unrelated to response duration, so
	// it stays regardless of stream length.
	eventShutdown := make(chan struct{})

	srv := &http.Server{
		Handler:           composeRouter(store, eventShutdown),
		ReadHeaderTimeout: 10 * time.Second, // mitigate slowloris
	}

	fmt.Fprintf(os.Stdout, "hyperreader serve: listening on http://localhost:%d (data-dir=%s)\n", cfg.Port, cfg.DataDir)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case <-ctx.Done():
		// Event streams are intentionally unbounded, so signal them to end
		// before draining ordinary in-flight requests.
		close(eventShutdown)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		// Serve exited on its own. http.ErrServerClosed is expected when
		// Shutdown was called concurrently; treat it as a clean stop.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// bindPort binds the configured TCP port and returns the listener. A bind
// failure caused by "address already in use" — the already-running guard's
// signal — is translated into a clear, actionable error so the operator
// knows a second instance is (likely) already running or the port is taken.
func bindPort(port int) (net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		if isAddrInUse(err) {
			return nil, fmt.Errorf("hyperreader serve is already running (port %d in use; use --port to listen elsewhere)", port)
		}
		return nil, fmt.Errorf("bind port %d: %w", port, err)
	}
	return ln, nil
}

// composeRouter builds the single HTTP handler tree mounted by Run: the API at
// "/api/" and the embedded static UI at "/" (catch-all). Go's ServeMux
// longest-prefix matching routes any path under /api/ to the API and everything
// else to the UI, so both share the single bound port without the UI masking
// API routes. eventShutdown is passed only to the API's long-lived SSE handler;
// finite HTTP requests retain standard http.Server shutdown behavior.
func composeRouter(store api.Store, eventShutdown <-chan struct{}) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewRouter(store, eventShutdown))
	mux.Handle("/", web.Handler())
	return mux
}

// isAddrInUse reports whether err is the "address already in use" bind
// failure. The string check is deliberately cross-platform: on Unix the
// underlying errno is EADDRINUSE, on Windows WSAEADDRINUSE, but both
// surface through net's OpError with the same message, so a string match
// avoids platform-specific syscall imports and build tags.
func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	return strings.Contains(opErr.Err.Error(), "address already in use")
}
