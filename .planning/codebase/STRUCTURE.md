# Codebase Structure

> Last mapped: 2026-08-05

## Repository Layout

```text
main.go                       Command entry point and subcommand dispatch
main_test.go                  CLI and flag behavior tests
main_mcp_e2e_test.go          Real subprocess MCP-to-serve integration test
internal/config/              Data-directory and port resolution
internal/server/              HTTP server startup, shutdown, and router composition
internal/api/                 JSON document API and SSE event hub
internal/storage/             SQLite schema, FTS search, and raw HTML persistence
internal/mcp/                 Stdio MCP send_html tool and forwarding client
web/                          Embedded static browser UI
e2e/                          Playwright browser acceptance tests
go.mod, go.sum                Go module and dependency lock data
package.json, package-lock.json Playwright test dependency and lock data
playwright.config.ts          Browser test server lifecycle and Chromium config
Makefile                      Local development command wrappers
README.md                     User, API, and development documentation
```

## Entry Points

- `main.go` dispatches `html-mcp serve` and `html-mcp mcp`.
- `internal/server/server.go` is the `serve` subcommand entry point.
- `internal/mcp/server.go` is the MCP process entry point.
- `web/index.html` is the browser UI entry page embedded into the binary.
- `playwright.config.ts` is the e2e test runner entry configuration.

## Package Responsibilities

- `internal/config`: produce a fully resolved `Config` with absolute data directory and port.
- `internal/server`: compose API and UI into one `http.Server`.
- `internal/api`: translate HTTP requests to `storage.Doc` operations and JSON/SSE responses.
- `internal/storage`: own database setup, FTS query handling, and filesystem storage.
- `internal/mcp`: translate MCP tool invocations into the HTTP create-document contract.
- `web`: serve compile-time embedded assets and run the browser interaction logic.

## Naming And File Placement

- Go implementation and test files remain together by package, such as `internal/api/handlers.go` and `internal/api/handlers_test.go`.
- Package-level documentation appears in the primary implementation file for each package.
- Browser assets are colocated with `web/embed.go`, because `go:embed` cannot traverse parent directories.
- Browser acceptance specs use descriptive kebab-case names, such as `e2e/two-process-acceptance.spec.ts`.
