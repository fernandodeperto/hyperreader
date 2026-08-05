# Technology Stack

> Last mapped: 2026-08-05

## Runtime

- Go module `github.com/fmendonca/hyperreader`, declared in `go.mod`.
- Go 1.26.4 is the required language/runtime version.
- The shipped artifact is one Go binary, `hyperreader`, with `serve` and `mcp` subcommands in `main.go`.
- Node.js 18+ and npm are development-only requirements for browser tests.

## Go Dependencies

- `github.com/modelcontextprotocol/go-sdk` supplies the stdio MCP server in `internal/mcp/server.go`.
- `modernc.org/sqlite` supplies the pure-Go SQLite driver used by `internal/storage/storage.go`; cgo and a system SQLite installation are not required.
- The storage schema uses SQLite FTS5, defined in `internal/storage/schema.go`.

## Frontend

- The browser UI is plain HTML, JavaScript, and CSS in `web/`.
- `web/embed.go` compiles `web/index.html`, `web/app.js`, and `web/app.css` into the Go binary with `go:embed`.
- The client uses browser-native `fetch`, `AbortController`, `EventSource`, `localStorage`, and `matchMedia`; there is no frontend framework or bundler.

## Test Tooling

- Go's built-in `testing` package covers unit and integration tests throughout the repository.
- `@playwright/test` 1.62.1 runs Chromium browser tests from `e2e/`.
- `playwright.config.ts` starts a real `go run . serve` process for browser tests rather than mocking the backend.

## Build And Local Commands

- `go build ./...` validates and builds all Go packages.
- `go test ./...` runs Go unit and integration tests.
- `go vet ./...` and `gofmt -l .` provide static and formatting checks.
- `npm run test:e2e` runs the Playwright suite.
- `Makefile` provides wrappers for build, test, vet, format, checks, and e2e commands.

## Runtime Configuration

- `internal/config/config.go` resolves `--data-dir`, `HTML_MCP_DATA_DIR`, `XDG_DATA_HOME`, and the home-directory fallback.
- The HTTP port priority is `--port`, then `HTML_MCP_PORT`, then default port `7420`.
- Data is stored outside the repository by default, under the resolved XDG-style app directory.
