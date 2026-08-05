# Testing Patterns

**Analysis Date:** 2026-08-05

## Test Framework

**Runner:**
- Go's built-in `testing` package runs unit and integration coverage with `go test ./...`.
- Playwright runs browser E2E coverage through `npm run test:e2e`.

**Run Commands:**
```bash
go test ./...        # Go unit and integration tests
go vet ./...         # Static checks
gofmt -l .           # Formatting check
npm run test:e2e     # Browser tests against a real serve process
make check           # vet + format check + Go tests
```

## Test File Organization

- Go tests are collocated: `internal/api/handlers_test.go` tests API behavior next to handlers.
- CLI tests live in `main_test.go`; cross-process Go coverage lives in `main_mcp_e2e_test.go`.
- Browser tests are in `e2e/` and use `*.spec.ts` names.

## Test Structure

- Tests use `t.Helper()` for helpers and `t.TempDir()` plus `t.Cleanup()` for isolated storage and process cleanup.
- Subtests organize related scenarios with `t.Run`.
- Tests call real HTTP handlers, real SQLite, real network listeners, and real subprocesses when those integration boundaries matter.
- Expected error paths assert status codes and actionable message content, not only that an error occurred.

## Mocking

- Prefer real internal dependencies: storage/API tests create temporary SQLite stores.
- Use a small `fakeStore` in `internal/api/handlers_test.go` only to isolate handler responses for storage failure paths.
- Use `httptest.Server` for MCP HTTP error and malformed-response behavior.

## E2E Tests

- `playwright.config.ts` starts `go run . serve` on port 7421 with a fresh `.e2e-data/` directory.
- Browser tests verify the primary table/search/new-tab flow, SSE updates, theme persistence, and the real two-process MCP-to-browser path.
- Chromium is the only configured browser; retries run only under CI.

## Coverage

- No numeric coverage target or coverage-report command was found.
- High-value boundaries have explicit real-runtime coverage: SQLite/FTS5, HTTP/SSE, MCP stdio, subprocess interaction, and browser behavior.

---

*Testing analysis: 2026-08-05*
