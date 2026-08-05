# Code Conventions

> Last mapped: 2026-08-05

## Go Style

- Code follows standard Go formatting, enforced locally with `gofmt -l .` from `Makefile`.
- Packages are small and role-based under `internal/`; names use conventional lower-case Go package names.
- Exported identifiers carry documentation, such as `config.Resolve`, `server.Run`, and `mcp.NewServer`.
- Comments explain protocol, lifecycle, and persistence invariants in detail where behavior is non-obvious.
- Errors are wrapped with operation context using `fmt.Errorf(... %w ...)` so callers retain the root cause.

## HTTP Conventions

- Handlers in `internal/api/handlers.go` use JSON for metadata and errors.
- Invalid client input returns 400, missing documents return 404, unsupported methods use ServeMux's 405 behavior, and storage failures return 500.
- API response structs intentionally omit internal file paths and raw HTML from list and metadata responses.
- Route methods are declared in Go 1.22 ServeMux patterns, for example `POST /api/documents`.

## Dependency Boundaries

- HTTP handlers depend on the narrow `api.Store` interface declared in `internal/api/api.go`.
- MCP-side HTTP payload structs are duplicated in `internal/mcp/server.go` to avoid coupling the MCP process to API implementation types.
- Concrete SQLite storage is assembled only in `internal/server/server.go`.

## Concurrency And Lifecycle

- Use request contexts for database calls and HTTP forwarding.
- SSE shared state is protected by a mutex in `internal/api/events.go`.
- Slow SSE subscribers must not delay document ingestion: the bounded event channel uses a non-blocking send and drops that subscriber's overflowing event.
- Long-lived SSE responses intentionally omit `http.Server.WriteTimeout`; header reads remain bounded by `ReadHeaderTimeout`.

## Browser Code

- `web/app.js` uses an IIFE, strict mode, ES5-compatible `var` and functions, and no build step.
- User-controlled metadata enters the document table through `textContent`, never `innerHTML`.
- DOM lookup uses the `byId` helper; interactive row events use delegation on the table body.
- Theme choice follows stored preference, then OS preference, then light mode, with storage access guarded for private or sandboxed contexts.

## Configuration

- Configuration precedence is explicit and documented in `internal/config/config.go`: command flag, environment variable, then default.
- The MCP path must not write diagnostic output to stdout, because stdout is the JSON-RPC transport.
