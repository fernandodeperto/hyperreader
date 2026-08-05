# Codebase Concerns

**Analysis Date:** 2026-08-05

## Security Considerations

**Unsandboxed document rendering:**
- Risk: persisted HTML opens in a normal top-level browser tab and its scripts execute with the viewer origin.
- Files: `web/app.js`, `internal/api/handlers.go`, and `e2e/smoke.spec.ts` explicitly preserve this behavior.
- Current mitigation: documents are expected to originate from the local agent workflow; the API is intended for localhost use.
- Recommendation: do not bind the service to an untrusted network. Any remote or multi-user use needs origin isolation, sandboxing, or a new trust model.

**No authentication boundary:**
- Risk: any process that can reach the HTTP listener can create or read documents.
- Files: `internal/server/server.go` binds `:<port>` and `internal/api/api.go` exposes all routes without authentication.
- Recommendation: restrict deployment to a trusted local environment until an explicit auth and listener-address design exists.

## Fragile Areas

**MCP stdout contract:**
- Why fragile: any accidental stdout output from `html-mcp mcp` corrupts the JSON-RPC stream.
- Files: `main.go`, `internal/mcp/server.go`, and `main_test.go`.
- Safe modification: route diagnostics to stderr and retain the stdout-purity test.

**Storage transaction and file ordering:**
- Why fragile: document insertion spans SQLite and a filesystem write, so failures can leave an inert orphan file after a failed database commit.
- Files: `internal/storage/storage.go`.
- Safe modification: preserve rollback and cleanup behavior; add tests for every changed failure path.

**SSE subscriber behavior:**
- Why fragile: each subscriber has a bounded channel and events are dropped for slow readers by design.
- Files: `internal/api/events.go` and `internal/api/events_test.go`.
- Safe modification: preserve non-blocking broadcast. If delivery guarantees change, define replay, cursor, or reconciliation behavior rather than increasing the buffer alone.

## Scaling Limits

**Document listing and search:**
- Current capacity: both unfiltered list and FTS search are capped at 100 rows.
- Files: `internal/storage/storage.go`.
- Limitation: there is no pagination or retention policy, while raw HTML files accumulate indefinitely.
- Improvement path: add explicit pagination and a document lifecycle policy before targeting large stores.

**SQLite writer concurrency:**
- Current mitigation: the store sets `SetMaxOpenConns(1)` and a 30-second busy timeout.
- Files: `internal/storage/storage.go`.
- Limitation: the design favors simple local correctness over high concurrent write throughput.

## Test Coverage Gaps

**Network exposure and authorization:**
- What's not tested: listener binding policy and access control, because none exists.
- Risk: future deployment changes could accidentally expose unsandboxed content.
- Priority: High if non-local deployment is introduced.

**Browser support:**
- What's not tested: Firefox and WebKit behavior.
- Files: `playwright.config.ts` configures Chromium only.
- Priority: Low while the project remains a local developer tool.

---

*Concerns audit: 2026-08-05*
