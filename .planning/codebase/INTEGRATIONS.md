# Integrations

> Last mapped: 2026-08-05

## MCP Protocol

- `internal/mcp/server.go` exposes one stdio MCP tool: `send_html`.
- It uses `github.com/modelcontextprotocol/go-sdk/mcp` and `mcp.StdioTransport` for JSON-RPC over stdin/stdout.
- The MCP process deliberately owns no persistent state. It forwards requests to the running `serve` process.
- `main.go` preserves stdout exclusively for the MCP transport; command errors are emitted on stderr.

## Local HTTP API

- The `serve` subcommand exposes an HTTP API through `internal/api/api.go`.
- `POST /api/documents` creates a document from `{name, description, tags, html}`.
- `GET /api/documents` lists documents, and `?q=` invokes FTS5 search.
- `GET /api/documents/{id}` returns metadata, while `GET /api/documents/{id}/content` returns raw HTML.
- `GET /api/events` is a server-sent events stream for newly created documents.
- `internal/mcp/server.go` forwards `send_html` to `http://localhost:<port>/api/documents` with a 10-second HTTP client timeout.

## Persistence

- `internal/storage/storage.go` persists metadata in SQLite and raw HTML in a `files/<id>.html` directory beside the database.
- `internal/storage/schema.go` maintains an FTS5 external-content index over document name, description, and tags.
- The SQLite driver is embedded through `modernc.org/sqlite`; there is no remote database or managed storage integration.

## Browser Platform APIs

- `web/app.js` subscribes to the local SSE endpoint with `EventSource` for live list updates.
- Theme preference is stored in browser `localStorage`; system color preference comes from `matchMedia`.
- Selecting a document opens its raw content endpoint in a top-level browser tab via `window.open`.

## Absent Integrations

- No authentication, authorization, external identity provider, webhook provider, message queue, analytics service, cloud SDK, or remote API integration is present.
- No CI configuration or deployment integration is present in the repository.
