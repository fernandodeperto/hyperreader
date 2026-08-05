# Codebase Structure

**Analysis Date:** 2026-08-05

## Directory Layout

```
html-mcp/
├── internal/
│   ├── api/          # HTTP document API and SSE hub
│   ├── config/       # Flag and environment configuration resolution
│   ├── mcp/          # stdio MCP send_html forwarder
│   ├── server/       # Serve bootstrap and router composition
│   └── storage/      # SQLite/FTS5 and raw HTML persistence
├── web/              # Go-embedded HTML, JavaScript, and CSS UI
├── e2e/              # Playwright browser acceptance tests
├── main.go           # CLI entry point and subcommand dispatch
├── main_test.go      # CLI unit tests
├── main_mcp_e2e_test.go # Real two-process Go integration test
├── go.mod            # Go module and dependencies
├── package.json      # Playwright dependency and test script
└── Makefile          # Local development commands
```

## Directory Purposes

**`internal/`:**
- Holds Go packages that are private to this module.
- Source and unit/integration tests are collocated, using `*_test.go` names.

**`web/`:**
- Holds the static browser application compiled into the Go binary through `//go:embed` in `web/embed.go`.
- Keep embeddable assets in this directory because `go:embed` cannot traverse parent directories.

**`e2e/`:**
- Holds Playwright specs that run against a real `serve` process started by `playwright.config.ts`.

## Key File Locations

**Entry points:**
- `main.go`: `serve` and `mcp` command dispatch.
- `internal/server/server.go`: server bootstrap, listener binding, router composition, graceful shutdown.
- `internal/mcp/server.go`: MCP server construction and HTTP forwarding.

**Core logic:**
- `internal/api/handlers.go`: HTTP request validation and response behavior.
- `internal/api/events.go`: SSE subscription and fan-out behavior.
- `internal/storage/storage.go`: metadata, FTS5 search, and HTML file persistence.

**Testing:**
- `internal/*/*_test.go`: Go package tests.
- `main_mcp_e2e_test.go`: real subprocess MCP-to-serve coverage.
- `e2e/*.spec.ts`: browser-level behavior.

## Naming Conventions

- Go source uses lowercase package directories and snake-free filenames such as `server.go` and `handlers_test.go`.
- Go test files are collocated with production code and end in `_test.go`.
- Browser E2E specifications use kebab-case names ending in `.spec.ts`.

## Where to Add New Code

- New HTTP endpoint: `internal/api/`, with tests in the same package.
- New persistence behavior: `internal/storage/`, plus storage and API integration coverage.
- New MCP tool: `internal/mcp/server.go`, with MCP package and subprocess tests.
- New browser behavior: `web/`, with a Playwright test in `e2e/` when it affects the primary flow.

---

*Structure analysis: 2026-08-05*
