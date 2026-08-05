# Architecture

**Analysis Date:** 2026-08-05

## Pattern Overview

**Overall:** Single Go binary with two cooperating local processes: a long-lived HTTP server and a disposable stdio MCP forwarder.

**Key Characteristics:**
- `html-mcp serve` owns persistence, HTTP API, SSE fan-out, and the embedded browser UI.
- `html-mcp mcp` exposes one `send_html` MCP tool and forwards to the running HTTP server.
- The browser UI and API share a single HTTP listener, with `/api/` taking precedence over the static UI catch-all.

## Layers

**CLI and configuration:**
- Purpose: parse subcommands and resolve flags/environment settings.
- Contains: `main.go` and `internal/config/config.go`.
- Depends on: `internal/server` and `internal/mcp`.

**HTTP API and events:**
- Purpose: ingest, list, search, retrieve content, and stream document creation events.
- Contains: `internal/api/api.go`, `handlers.go`, and `events.go`.
- Depends on: the small `api.Store` interface rather than the concrete storage type.

**Persistence:**
- Purpose: store metadata in SQLite/FTS5 and raw HTML on local disk.
- Contains: `internal/storage/storage.go` and `schema.go`.
- Used by: the serve bootstrap and API handlers.

**Embedded web UI:**
- Purpose: list/search documents, receive SSE updates, and open raw document content in a separate tab.
- Contains: `web/embed.go`, `web/index.html`, `web/app.js`, and `web/app.css`.

## Data Flow

**MCP ingestion:**
1. An MCP host launches `html-mcp mcp`.
2. `send_html` in `internal/mcp/server.go` validates the name and posts JSON to the serve process.
3. `internal/api/handlers.go` persists metadata and HTML using `storage.Store.Insert`.
4. The API broadcasts the created response through the in-memory SSE hub.
5. `web/app.js` prepends the event to the currently unfiltered document table.

**Browser retrieval:**
1. The UI fetches `GET /api/documents`, optionally with `?q=`.
2. The API uses SQLite FTS5 for search or lists newest-first.
3. Clicking a row opens `GET /api/documents/{id}/content` in a new top-level tab.

## Key Abstractions

- `storage.Store`: concurrency-safe database and file storage handle.
- `api.Store`: test seam consumed by HTTP handlers.
- `api.hub`: mutex-protected, non-blocking fan-out for SSE subscribers.
- `config.Config`: resolved data directory and port shared by `serve` and `mcp` startup.

## Error Handling

- Functions return wrapped errors with operation context; `main()` prints and exits non-zero.
- HTTP errors use a JSON `{ "error": "..." }` envelope.
- MCP tool failures become `CallToolResult{IsError: true}` so the agent can read and recover from them.

---

*Architecture analysis: 2026-08-05*
