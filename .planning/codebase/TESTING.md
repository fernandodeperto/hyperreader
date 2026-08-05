# Testing

> Last mapped: 2026-08-05

## Test Layers

- Go unit and integration tests use the standard `testing` package and run with `go test ./...`.
- Package tests are colocated with implementation, including `internal/config/config_test.go`, `internal/api/handlers_test.go`, and `internal/storage/storage_test.go`.
- `main_test.go` tests CLI dispatch, help text, invalid flags, configuration precedence, and the MCP stdout-purity requirement.
- `main_mcp_e2e_test.go` compiles the binary and drives real `serve` and `mcp` subprocesses through TCP, stdio JSON-RPC, SQLite, and the HTTP API.
- Playwright specs in `e2e/` test browser workflows against a real `go run . serve` process.

## Test Harness Patterns

- Go tests commonly use `t.TempDir()` and `t.Cleanup()` for isolated SQLite stores, temporary files, and process shutdown.
- API tests in `internal/api/handlers_test.go` exercise a real `storage.Store` via `httptest`, then use `fakeStore` for isolated failure paths.
- Storage tests use `newTestStore` in `internal/storage/storage_test.go` as their canonical temporary store fixture.
- Subprocess tests allocate a free TCP port, poll the real API for readiness, and use timeouts instead of fixed startup sleeps.

## Covered Behavior

- SQLite document persistence, rollback on file-write failure, FTS5 search, ordering, limits, and not-found behavior.
- HTTP creation, list/search, metadata, raw content, malformed input, missing resources, method handling, and backend failures.
- SSE hub subscription lifecycle, event delivery, heartbeat behavior, and slow-subscriber isolation in `internal/api/events_test.go`.
- MCP tool registration, forwarding success and failure behavior, protocol-level error avoidance, and real stdio framing.
- Browser search, live SSE row updates, dark-mode preference, top-level raw HTML rendering, keyboard activation, and two-process acceptance flows.

## Browser Test Configuration

- `playwright.config.ts` runs Chromium only, serially with one worker.
- The test server uses port `7421` unless `PORT` is set and wipes `./.e2e-data` before it starts.
- Browser specs share that fresh server for their suite and seed documents through the real HTTP API.

## Gaps

- There is no visible CI workflow to run the Go or Playwright suites automatically.
- The repository has no explicit coverage threshold or reporting configuration.
- Browser coverage is Chromium-only; Firefox and WebKit are deliberately deferred in `playwright.config.ts`.
