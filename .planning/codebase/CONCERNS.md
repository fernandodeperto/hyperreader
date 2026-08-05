# Concerns

> Last mapped: 2026-08-05

## Security

- Document HTML is intentionally served and opened as a top-level, unsandboxed browser page. `web/app.js` calls `window.open` on `GET /api/documents/{id}/content`, and browser tests explicitly prove inline scripts execute. Any party able to submit documents can execute arbitrary HTML and JavaScript in the viewer origin.
- `internal/server/server.go` binds `:<port>`, which listens on all network interfaces rather than loopback only. The API has no authentication, authorization, CSRF defense, or origin restriction, so network exposure materially expands the trusted-writer boundary.
- The content endpoint returns user-supplied HTML with no content-security policy or sanitization. This matches the stated product behavior but should remain an explicit deployment constraint.

## Data Growth And Consistency

- There is no retention, deletion, quota, or document-size limit. The SQLite database and `files/` directory can grow without bound.
- `storage.Insert` cannot atomically commit SQLite metadata and the raw filesystem write. Its cleanup handles known file-write and commit failures, but the code documents that an orphan HTML file can remain if commit fails after a write.
- `List` and FTS search cap results at 100, but the UI has no pagination or indication that more stored documents may exist.

## Availability And Scalability

- SQLite is configured with one open connection in `internal/storage/storage.go`, favoring simple serialized writes over concurrent throughput.
- SSE is process-local and best-effort for slow consumers: `internal/api/events.go` drops events when a subscriber's eight-event buffer fills. A client can briefly become stale until another event or a full refresh.
- The server has no request body-size limit, rate limiting, or authenticated admission control. Large or frequent document submissions can consume memory, disk, and the single SQLite writer.

## Operational Gaps

- No CI, deployment, release, container, or observability configuration is present in the repository.
- There are no structured logs, health endpoint, metrics, tracing, or backup/restore tooling.
- Default storage is in a per-user XDG-style data directory, so operational ownership and backup behavior are local-machine concerns.

## Testing Gaps

- Browser acceptance tests exercise Chromium only.
- The broad happy paths and important failure paths are tested, but there is no load, durability-after-crash, multi-process concurrency, or adversarial security test suite.
- Tests prove MCP forwarding to localhost, but do not validate deployment exposure or access controls because none exist.
