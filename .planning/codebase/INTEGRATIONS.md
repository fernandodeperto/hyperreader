# External Integrations

**Analysis Date:** 2026-08-05

## APIs & External Services

- No third-party network APIs or hosted services are integrated.
- The local MCP client/server boundary uses the Model Context Protocol over stdio in `internal/mcp/server.go`.
- The MCP process forwards `send_html` to the local serve process via `POST http://localhost:<port>/api/documents`.

## Data Storage

**Database:**
- SQLite with FTS5 stores document metadata and search indexes.
- The database path is `<data-dir>/docs.db`, resolved in `internal/config/config.go` and opened in `internal/storage/storage.go`.
- Schema setup is idempotent SQL in `internal/storage/schema.go`; migrations run on every store open.

**File Storage:**
- Raw HTML is stored locally as `<data-dir>/files/<id>.html`.
- SQLite retains each file's absolute path in `docs.file_path`.

## Authentication & Identity

- No authentication, authorization, OAuth, or multi-user identity layer exists.
- The HTTP API is intended for localhost use. Exposing the service beyond localhost requires adding an access-control design first.

## Monitoring & Observability

- `serve` writes its startup address and data directory to stdout in `internal/server/server.go`.
- The MCP stdio protocol reserves stdout for JSON-RPC. Diagnostics must not be written there (`main.go`, `internal/mcp/server.go`).
- No hosted error tracking, metrics, analytics, or centralized logging is configured.

## CI/CD & Deployment

- No CI workflow, container definition, deployment manifest, or hosting configuration was found.
- `Makefile` is the local development automation surface.

## Environment Configuration

- `HTML_MCP_DATA_DIR`: explicit application data directory.
- `XDG_DATA_HOME`: parent directory used when no explicit data directory is set.
- `HTML_MCP_PORT`: HTTP serve port and MCP forward target when flags are absent.
- Defaults are local: `~/.local/share/html-mcp` and port `7420`.

---

*Integration audit: 2026-08-05*
