# Technology Stack

**Analysis Date:** 2026-08-05

## Languages

**Primary:**
- Go 1.26.4 for the CLI, HTTP server, MCP server, storage, and Go tests (`go.mod`, `main.go`, `internal/`).

**Secondary:**
- JavaScript for the embedded browser UI (`web/app.js`) and CSS/HTML assets in `web/`.
- TypeScript for Playwright end-to-end tests and their configuration (`e2e/`, `playwright.config.ts`).

## Runtime

**Environment:**
- Go 1.26+ builds the single `html-mcp` binary.
- Node.js 18+ and npm are required only for browser E2E testing.

**Package Manager:**
- Go modules with `go.mod` and `go.sum`.
- npm with `package-lock.json` present for Playwright.

## Frameworks

**Core:**
- Go standard library `net/http` ServeMux provides HTTP routing.
- `github.com/modelcontextprotocol/go-sdk` v1.7.0 implements the stdio MCP server.
- `modernc.org/sqlite` v1.56.0 provides a pure-Go SQLite driver with FTS5 support.

**Testing:**
- Go's standard `testing` package for unit and integration tests.
- `@playwright/test` v1.62.1 for Chromium browser acceptance tests.

## Configuration

- `HTML_MCP_DATA_DIR`, `XDG_DATA_HOME`, and `HTML_MCP_PORT` configure runtime storage and port resolution in `internal/config/config.go`.
- CLI flags override environment values: `serve --data-dir`, `serve --port`, and `mcp --port`.
- `Makefile` wraps build, formatting, Go tests, vet, and E2E commands.

## Platform Requirements

- The application has no cgo or external SQLite requirement because `modernc.org/sqlite` is pure Go.
- Browser tests use cached Playwright Chromium and a temporary `.e2e-data/` store.

---

*Stack analysis: 2026-08-05*
