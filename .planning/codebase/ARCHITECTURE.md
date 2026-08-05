# Architecture

> Last mapped: 2026-08-05

## System Shape

The application is a single Go binary with two cooperating processes:

1. `html-mcp serve` is the long-lived HTTP server. It owns the database, document files, API, SSE hub, and embedded UI.
2. `html-mcp mcp` is a disposable stdio MCP server. An MCP client can launch it on demand and it forwards `send_html` calls to `serve` over localhost HTTP.

This process boundary is implemented by `main.go`, `internal/server/server.go`, and `internal/mcp/server.go`.

## Request Flow

1. An MCP client invokes `send_html` over stdio JSON-RPC.
2. `internal/mcp/server.go` validates the tool arguments and posts JSON to the local HTTP API.
3. `internal/api/handlers.go` validates the request and calls the injected storage interface.
4. `internal/storage/storage.go` creates a SQLite metadata row, writes `<id>.html`, records the file path, and commits the transaction.
5. The API returns metadata and broadcasts a document event through the in-memory SSE hub in `internal/api/events.go`.
6. `web/app.js` receives the event and prepends it to the active unfiltered table without reloading.

## Layers

- CLI and process lifecycle: `main.go`.
- Runtime configuration: `internal/config/config.go`.
- HTTP server composition and graceful shutdown: `internal/server/server.go`.
- API routing, validation, JSON responses, and SSE: `internal/api/`.
- SQLite schema, search, and raw-file persistence: `internal/storage/`.
- MCP tool protocol and HTTP forwarding: `internal/mcp/server.go`.
- Embedded static browser UI: `web/`.

## Boundaries And Abstractions

- `api.Store` in `internal/api/api.go` is the interface between HTTP handlers and concrete storage, enabling fake storage in API error-path tests.
- MCP request and response structs mirror HTTP payloads inside `internal/mcp/server.go` instead of importing API internals. The HTTP contract is the integration boundary.
- `server.composeRouter` mounts `/api/` before the catch-all embedded UI handler, relying on Go ServeMux longest-prefix routing.
- The SSE `hub` is in-memory and process-local. It has bounded per-subscriber buffers and drops events only for slow consumers.

## Lifecycle

- `server.Run` binds the configured port before opening SQLite, so a second server instance fails before touching persistent storage.
- `serve` reacts to SIGINT and SIGTERM with a five-second graceful HTTP shutdown.
- The MCP server maps expected forwarding failures to MCP tool results with `IsError=true`, rather than returning protocol-level errors.
