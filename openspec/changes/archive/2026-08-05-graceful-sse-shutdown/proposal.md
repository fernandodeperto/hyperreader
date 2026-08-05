## Why

The embedded UI keeps a long-lived `/api/events` SSE connection open. On Ctrl+C, `http.Server.Shutdown` waits for that active connection until its deadline and the process exits with an error instead of completing a graceful shutdown.

## What Changes

- Add a server-shutdown signal for long-lived event-stream handlers.
- Close active SSE handlers promptly when shutdown begins, while allowing finite in-flight HTTP requests to drain through `http.Server.Shutdown`.
- Return a successful process exit when the server is interrupted with the embedded UI connected.
- Add integration coverage for shutdown with an active SSE subscriber.

## Capabilities

### New Capabilities
- `graceful-server-shutdown`: Gracefully stop long-lived server-sent event connections during server shutdown without aborting ordinary in-flight requests.

### Modified Capabilities

## Impact

- Affected code: `internal/server`, `internal/api`, and server lifecycle tests.
- Affected behavior: Ctrl+C and SIGTERM for `html-mcp serve` with connected embedded UI clients.
- No API endpoint or browser-client contract changes.
