# HyperReader

An always-open HTML viewer for AI agent output, exposed as an MCP tool.

`hyperreader` is a single Go binary with two subcommands:

- **`hyperreader serve`**: a long-lived HTTP server that stores HTML
  documents (SQLite + FTS5 full-text search) and serves a live web UI to
  browse, search, and view them.
- **`hyperreader mcp`**: a lightweight stdio [MCP](https://modelcontextprotocol.io)
  server exposing a single tool, `send_html`, that forwards a document to
  the running `serve` process over localhost HTTP.

The two-process split is intentional: `serve` owns storage and stays
running; `mcp` is disposable — an agent's MCP client launches and tears it
down at will, and it forwards to whichever `serve` instance is listening on
the resolved port.

## How it works

```
AI agent (MCP client)
     │  stdio (JSON-RPC)
     ▼
hyperreader mcp  ───────────►  hyperreader serve  ─────►  SQLite (docs.db)
 (forwards send_html)          (HTTP API +               + files/ (raw HTML)
                                 embedded web UI)
                                     │
                                     ▼
                              Browser at http://localhost:7420/
                              (table view, search, live SSE updates,
                               detail view for a given document)
```

- `send_html` posts to `serve`'s `POST /api/documents` and returns the new
  document's id/name, or an error if `serve` isn't reachable.
- The web UI polls/streams `GET /api/events` (SSE) so newly ingested
  documents appear live without a manual refresh.

## Requirements

- **Go** 1.26+ (see `go.mod`)
- **Node.js** 18+ and **npm** — only needed for the Playwright e2e suite
- SQLite is vendored via `modernc.org/sqlite` (pure Go, no cgo/system SQLite
  required)

## Getting started

```bash
git clone <this-repo>
cd hyperreader
go build ./...
```

### Run the server

```bash
go run . serve
# or, after building:
./hyperreader serve
```

By default `serve` listens on port `7420` and stores data under
`~/.local/share/hyperreader` (XDG-style). Open `http://localhost:7420/` to
see the web UI.

Flags and environment overrides:

| Setting  | Flag             | Env var                | Default                    |
|----------|------------------|-------------------------|-----------------------------|
| Data dir | `--data-dir PATH`| `HYPERREADER_DATA_DIR`  | `$XDG_DATA_HOME/hyperreader` or `~/.local/share/hyperreader` |
| Port     | `--port N`       | `HYPERREADER_PORT`      | `7420`                      |

Override priority (highest to lowest): flag > env var > default.

```bash
go run . serve --port 8080 --data-dir /tmp/hyperreader-data
```

### Run the MCP server

`hyperreader mcp` is meant to be launched by an MCP client (e.g. an AI
coding assistant), not run interactively — it speaks JSON-RPC over stdio
and forwards `send_html` calls to a running `serve` process. Point an MCP
client's config at the built binary:

```json
{
  "mcpServers": {
    "hyperreader": {
      "command": "/absolute/path/to/hyperreader",
      "args": ["mcp"]
    }
  }
}
```

If `serve` is running on a non-default port, pass `--port` (or set
`HYPERREADER_PORT`) so `mcp` forwards to the right instance:

```bash
hyperreader mcp --port 8080
```

`serve` must already be running for `send_html` calls to succeed; `mcp`
returns a tool-level error (visible to the agent) rather than crashing if
`serve` isn't reachable.

## Migrating from html-mcp

HyperReader is a clean rename of the former `html-mcp` project: the
binary name, MCP server name, environment variables (`HTML_MCP_DATA_DIR` /
`HTML_MCP_PORT` → `HYPERREADER_DATA_DIR` / `HYPERREADER_PORT`), and default
data directory (`html-mcp` → `hyperreader`) have all changed with **no
runtime fallback** to the old names or paths.

Existing documents are **not** picked up automatically. If you want to
keep them, move the old data directory into the new location before
starting the new binary, e.g.:

```bash
mv ~/.local/share/html-mcp ~/.local/share/hyperreader
```

(Adjust the paths above if you previously overrode `HTML_MCP_DATA_DIR` or
`XDG_DATA_HOME`.) Also update any MCP client configuration and scripts
that reference the old `html-mcp` binary name or environment variables.

## Development

Common dev commands are also available as `make` targets — run `make help` for
the full list (build, test, vet, fmt, check, e2e, clean). The Makefile only wraps
the raw commands documented below; either approach works.

### Project layout

```
main.go                    CLI entrypoint, subcommand dispatch (serve/mcp)
internal/config/           XDG data dir + port resolution (flag > env > default)
internal/server/           serve subcommand wiring: bind, storage, router, shutdown
internal/api/              HTTP API: POST/GET /api/documents, GET /api/events (SSE)
internal/storage/          SQLite storage layer + FTS5 search
internal/mcp/              stdio MCP server (send_html tool), forwards to serve's API
web/                       Embedded web UI (index.html, app.js, app.css) via go:embed
e2e/                       Playwright browser smoke tests against the real serve binary
skills/generate-html/      Agent skill: renders a long-form report and sends it to HyperReader
```

### Agent skills

`skills/generate-html/` is an agent skill that renders a long-form deliverable
(investigation, review, incident analysis) as a single self-contained HTML page
and delivers it through the `send_html` MCP tool, so the report opens in
HyperReader rather than scrolling past in a terminal.

It is registered by symlinking it into the agent's skill directory, so edits in
this repository take effect immediately with no copy to keep in sync:

```bash
ln -s "$PWD/skills/generate-html" ~/.agents/skills/generate-html
```

Unrelated to `.omp/skills/`, which holds vendored tooling managed by the harness.

### Build

```bash
go build ./...
```

### Run tests

Go unit and integration tests (includes a real subprocess MCP handshake
test in `main_mcp_e2e_test.go`):

```bash
go test ./...
```

Vet and format:

```bash
go vet ./...
gofmt -l .
```

Browser/e2e smoke tests (Playwright drives a real `hyperreader serve`
binary, not a mock):

```bash
npm install
npx playwright install chromium   # first run only, if browsers aren't cached
npm run test:e2e
```

The e2e suite builds and runs `go run . serve` on port `7421` against a
throwaway data dir (`./.e2e-data`, gitignored), so it can run alongside a
dev instance on the default port `7420` without conflict.

### HTTP API reference

All endpoints are served by `hyperreader serve`:

| Method | Path                          | Description                                   |
|--------|-------------------------------|------------------------------------------------|
| POST   | `/api/documents`               | Ingest a document (`{name, description, tags, html}`) |
| GET    | `/api/documents`               | List documents, most-recent-first; `?q=` searches name/description/tags |
| GET    | `/api/documents/{id}`          | Get document metadata                          |
| GET    | `/api/documents/{id}/content`  | Get raw HTML content (`Content-Type: text/html`) |
| GET    | `/api/events`                  | Server-Sent Events stream; broadcasts newly ingested documents |

`name` is required on ingest; `description`, `tags`, and `html` default to
empty strings.

### Data storage

`serve` creates its data directory on startup if it doesn't exist:

- `docs.db` — SQLite database (document metadata + FTS5 index)
- `files/` — raw HTML content per document

Only one `serve` instance can bind a given port at a time; a second
instance on the same port fails fast with a clear error instead of
silently queuing behind the first.

## License

No license file is currently present in this repository.
