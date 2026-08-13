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
                               content view for a given page)
```

- `send_html` posts to `serve`'s `POST /api/pages`, keyed by an
  agent-supplied slug: a new slug creates a page (`201`) and an existing
  slug patches it in place (`200`), returning the page's slug/name, or an
  error if `serve` isn't reachable.
- The web UI polls/streams `GET /api/events` (SSE) so newly created or
  patched pages appear live without a manual refresh.

## Install

Supported platforms are macOS and Linux, on x86-64 and arm64. Windows is
not supported.

```bash
curl -fsSL https://raw.githubusercontent.com/fernandodeperto/hyperreader/main/install.sh | sh
```

The installer detects your platform, downloads the matching executable
from the latest release, verifies it against that release's
`SHA256SUMS`, and installs it to `~/.local/bin/hyperreader`. It then
prints an MCP client entry with the installed path already filled in,
and, if `~/.local/bin` is not on your `PATH`, the line that adds it.

Fetch it with `curl` rather than a browser. A browser download carries
`com.apple.quarantine` and macOS refuses to run it without a manual
override.

If you would rather read the script before running it:

```bash
curl -fsSL -O https://raw.githubusercontent.com/fernandodeperto/hyperreader/main/install.sh
less install.sh
sh install.sh
```

With a Go toolchain, installing from the module path works too:

```bash
go install github.com/fernandodeperto/hyperreader@v0.1.0
```

## Requirements

Installing a release needs only `curl` and a shell. Go is required just
for `go install` and for building from source.

- **Go** 1.26+ (see `go.mod`), for `go install` or a source build
- **Node.js** 18+ and **npm**, only needed for the Playwright e2e suite
- SQLite is vendored via `modernc.org/sqlite` (pure Go, no cgo/system SQLite
  required)

## Build from source

```bash
git clone https://github.com/fernandodeperto/hyperreader
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
and forwards `send_html` calls to a running `serve` process. The
installer prints this entry with the path already resolved; if you built
from source, use the absolute path of your own binary:

```json
{
  "mcpServers": {
    "hyperreader": {
      "command": "/Users/alice/.local/bin/hyperreader",
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

Existing pages are **not** picked up automatically. If you want to
keep them, move the old data directory into the new location before
starting the new binary, e.g.:

```bash
mv ~/.local/share/html-mcp ~/.local/share/hyperreader
```

(Adjust the paths above if you previously overrode `HTML_MCP_DATA_DIR` or
`XDG_DATA_HOME`.) Also update any MCP client configuration and scripts
that reference the old `html-mcp` binary name or environment variables.

## Development

Common dev commands are also available as `make` targets. Run `make help` for
the full list (build, test, vet, fmt, check, e2e, release, dist-clean, clean).
The Makefile only wraps the raw commands documented below; either
approach works.

### Project layout

```
main.go                    CLI entrypoint, subcommand dispatch (serve/mcp)
internal/config/           XDG data dir + port resolution (flag > env > default)
internal/server/           serve subcommand wiring: bind, storage, router, shutdown
internal/api/              HTTP API: POST/GET /api/pages, GET /api/events (SSE)
internal/storage/          SQLite storage layer + FTS5 search
internal/mcp/              stdio MCP server (send_html tool), forwards to serve's API
web/                       Embedded web UI (index.html, app.js, app.css) via go:embed
e2e/                       Playwright browser smoke tests against the real serve binary
skills/generate-html/      Agent skill: renders a long-form report and sends it to HyperReader
install.sh                 Platform-detecting installer for the latest published release
dist/                      `make release` output: per-platform executables + SHA256SUMS (gitignored)
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

### Releasing

Cross-compile the published targets and their checksum manifest:

```bash
make release
```

That writes `dist/hyperreader-<os>-<arch>` for `darwin/arm64`,
`darwin/amd64`, `linux/amd64` and `linux/arm64`, each built with
`-trimpath -ldflags="-s -w"`, alongside `dist/SHA256SUMS`. Unlike `make
build`, release executables are stripped, so keep using `make build` for
development where panic traces need symbol names.

Check the manifest before publishing. The entries are bare filenames, so
verify from inside `dist/`:

```bash
(cd dist && shasum -a 256 -c SHA256SUMS)
```

Publish the whole directory rather than a hand-typed file list, so
`SHA256SUMS` cannot be left out:

```bash
gh release create v0.1.0 dist/* --target "$(git rev-parse HEAD)" \
  --title v0.1.0 --notes-file notes.md
```

`gh` creates the tag as part of the release, so no separate `git tag` and
`git push --tags` is needed. Tags follow `vMAJOR.MINOR.PATCH`. Drop
`--notes-file` to have `gh` open an editor instead.

The install script always fetches whatever the newest release is, through
GitHub's `releases/latest/download/` redirect, so publishing a newer tag
is all that is required to roll an update out.

### HTTP API reference

All endpoints are served by `hyperreader serve`:

| Method | Path                          | Description                                   |
|--------|-------------------------------|------------------------------------------------|
| POST   | `/api/pages`                   | Create or patch a page by slug (`{slug, name, description, html}`); `201` on create, `200` on patch |
| GET    | `/api/pages`                   | List pages, most-recently-changed-first; `?q=` searches name/description |
| GET    | `/api/pages/{slug}`            | Get page metadata                              |
| GET    | `/api/pages/{slug}/content`    | Get raw HTML content (`Content-Type: text/html`) |
| GET    | `/api/events`                  | Server-Sent Events stream; broadcasts `page-created` and `page-updated` |

`slug` and `name` are required on write; `description` and `html` default
to empty strings. `slug` must match `^[a-z0-9]+(-[a-z0-9]+)*$` (max 80
characters) and `description` is capped at 200 characters; either
violation is rejected with `400` and no write.

### Data storage

`serve` creates its data directory on startup if it doesn't exist:

- `docs.db`: SQLite database (page metadata + FTS5 index)
- `files/`: raw HTML content per page

Only one `serve` instance can bind a given port at a time; a second
instance on the same port fails fast with a clear error instead of
silently queuing behind the first.

## License

No license file is currently present in this repository.
