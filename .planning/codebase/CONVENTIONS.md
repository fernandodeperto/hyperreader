# Coding Conventions

**Analysis Date:** 2026-08-05

## Naming Patterns

- Go packages and files use lowercase names: `api`, `storage`, `server`, and `handlers.go`.
- Exported Go identifiers use PascalCase; unexported functions and fields use camelCase.
- Constants use camelCase for package-private values such as `heartbeatInterval`; exported constants use PascalCase.
- Tests use `TestSubject_Behavior` names with focused subtests where useful.

## Code Style

- Format Go files with `gofmt`; use `make fmt` to check and `make fmt-fix` to apply formatting.
- Keep package documentation and detailed comments around non-obvious invariants, protocol contracts, and failure modes.
- JavaScript in `web/app.js` intentionally uses ES5-compatible `var` and function declarations, not a bundler or framework.
- UI strings originating from documents are inserted using `textContent`, not `innerHTML`.

## Import Organization

- Go imports group standard library packages first, then a blank line, then module-local or third-party imports.
- Imports are formatted and sorted by `gofmt`.
- Browser test imports place Playwright imports first, then Node built-ins where needed.

## Error Handling

- Return wrapped errors with an operation prefix, for example `fmt.Errorf("open sqlite %s: %w", ...)` in `internal/storage/storage.go`.
- Validate at boundaries and return early: CLI flags, HTTP JSON bodies, IDs, and MCP arguments.
- HTTP handlers use a stable JSON error envelope via `writeError`.
- MCP handler failures are tool results with `IsError: true`, not protocol-level Go errors.

## Logging and Comments

- The service has minimal logging: startup information goes to stdout from `internal/server/server.go`.
- The MCP process must not write diagnostics to stdout because it corrupts JSON-RPC framing.
- Comments explain design constraints such as non-blocking SSE fan-out, transaction/file-write ordering, and browser security boundaries.

## Function and Module Design

- Use small package-local helpers such as `parseID`, `toResponse`, and `buildFTSQuery` for boundary behavior.
- Define narrow interfaces at consumers, exemplified by `api.Store`, to keep handlers testable without concrete storage.
- Keep integration boundaries explicit. `internal/mcp` duplicates HTTP wire structs instead of importing internal API types.

---

*Convention analysis: 2026-08-05*
