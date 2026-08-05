// Package mcp implements the "html-mcp mcp" subcommand: a thin stdio MCP
// server that exposes a single tool, send_html, which forwards an HTML
// document to the running serve process over localhost HTTP.
//
// The mcp process owns no storage of its own. It depends only on the S01
// HTTP ingest contract — POST /api/documents with a JSON body
// {name, description, tags, html} returning 201 + a documentResponse — and
// on the resolved serve port (config.DefaultPort + override chain). This
// preserves the two-process boundary: serve owns the SQLite DB and HTML
// files; the mcp process is a disposable forwarder an AI agent's MCP client
// launches and tears down at will.
//
// Error reporting follows the MCP convention (and R005): a tool failure —
// serve not running (connection refused), an HTTP 4xx/5xx, or a malformed
// response — is surfaced to the agent as a CallToolResult with IsError=true
// and explanatory text in Content, never as a returned Go error. A returned
// Go error becomes a protocol-level error response that the LLM cannot see,
// so it cannot self-correct; only exceptional conditions (the transport
// dying, the server failing to start) bubble as errors from Run.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverName/Version identify the MCP server to its clients.
const (
	serverName    = "html-mcp"
	serverVersion = "v0.1.0"
)

// httpClientTimeout bounds a single forward to serve: a hung serve should
// surface to the agent as an IsError result within a few seconds, not stall
// the agent's whole turn. The timeout covers the full request/response
// (dial + write + read) so a half-open connection cannot hang the tool.
const httpClientTimeout = 10 * time.Second

// sendHTMLArgs is the typed argument struct for the send_html tool. The
// go-sdk's generic AddTool derives the JSON input schema from these struct
// tags, so an agent sees name+html as required and description+tags as
// optional without a separate schema declaration.
type sendHTMLArgs struct {
	Name        string `json:"name" jsonschema:"the document name (required)"`
	HTML        string `json:"html" jsonschema:"the HTML document body (required)"`
	Description string `json:"description,omitempty" jsonschema:"optional human-readable description"`
	Tags        string `json:"tags,omitempty" jsonschema:"optional comma-separated tags"`
}

// forwardRequest is the JSON body posted to serve's POST /api/documents. Its
// shape mirrors internal/api.createRequest exactly — the mcp process must
// never import the serve package, so the struct is duplicated here to keep
// the dependency a pure HTTP contract (the integration closure).
type forwardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	HTML        string `json:"html"`
}

// documentResponse mirrors internal/api.documentResponse: the JSON serve
// returns on a successful ingest. Only the fields the tool surfaces to the
// agent are decoded.
type documentResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
}

// errorResponse mirrors internal/api.errorResponse: the JSON serve returns
// on a 4xx/5xx. It is a plain {"error": "..."} envelope.
type errorResponse struct {
	Error string `json:"error"`
}

// Run starts the stdio MCP server and blocks until ctx is cancelled or the
// transport closes. It constructs a single-tool server (send_html) whose
// handler forwards to the serve process at http://localhost:<port> and runs
// it over stdin/stdout. A transport/handler panic is caught at the Run
// boundary so a malformed request cannot crash the agent's MCP session.
//
// port selects the serve instance to forward to; the caller resolves it via
// config.ResolvePort so flag/env overrides apply. Run does not need a data
// directory — the mcp process owns no storage, preserving the two-process
// boundary (serve owns SQLite; mcp is a disposable forwarder).
func Run(ctx context.Context, port int) error {
	srv := newServer(port)
	// Run blocks until the client disconnects or ctx is cancelled. A panic
	// inside a tool handler is recovered by the sdk's Run loop, so it does
	// not crash the process; transport errors bubble up here.
	return srv.Run(ctx, &mcp.StdioTransport{})
}

// newServer builds the single-tool MCP server (send_html) wired to forward
// to the serve process on the given port. It is the one construction site
// shared by Run (stdio transport) and the in-memory transport tests, so the
// tool registration cannot drift between them.
func newServer(port int) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "send_html",
		Description: "Send an HTML document to the always-open html-mcp viewer. The document is persisted by the running serve process and appears in its list view. Returns the new document's id and name on success; returns an error (visible to the agent) if serve is not running or rejects the document.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args sendHTMLArgs) (*mcp.CallToolResult, any, error) {
		result, err := forward(ctx, port, args)
		if err != nil {
			// Tool-level failures (serve down, HTTP error, malformed
			// response) surface to the agent as IsError=true with text —
			// never as a returned Go error (see package doc + R005).
			return errorResult(err.Error()), nil, nil
		}
		return result, nil, nil
	})
	return srv
}

// forward posts args to the serve process's POST /api/documents endpoint
// and returns a success CallToolResult, or a descriptive error. The error
// is always meant to be wrapped into an errorResult by the caller — it
// carries a human-readable message naming the port/cause so a debugging
// agent can locate the intended serve instance.
func forward(ctx context.Context, port int, args sendHTMLArgs) (*mcp.CallToolResult, error) {
	if args.Name == "" {
		// The serve API rejects a missing name with 400; failing fast here
		// avoids a round-trip and gives a clearer message. Still surfaces
		// as IsError=true (the caller wraps it).
		return nil, fmt.Errorf("send_html: name is required")
	}

	body, err := json.Marshal(forwardRequest{
		Name:        args.Name,
		Description: args.Description,
		Tags:        args.Tags,
		HTML:        args.HTML,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/api/documents", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request to %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// A dedicated client with a timeout: the tool must not hang the
	// agent's turn on a wedged serve. The context (ctx) governs the
	// session; the client timeout is the backstop for a silent server.
	client := &http.Client{Timeout: httpClientTimeout}
	resp, err := client.Do(req)
	if err != nil {
		// Connection refused / serve not running is the expected case the
		// demo must handle. Name the port so a debugging agent can locate
		// the intended serve instance.
		return nil, fmt.Errorf("serve is not reachable at %s (is `html-mcp serve` running on port %d?): %w", url, port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Surface serve's error message verbatim so the agent can react
		// (e.g. a 400 "name is required" or a 500 storage failure).
		msg := readServeError(resp)
		return nil, fmt.Errorf("serve returned HTTP %d for %s: %s", resp.StatusCode, url, msg)
	}

	var doc documentResponse
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode serve response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Document %q ingested (id=%d) via serve on port %d. View it at http://localhost:%d/", doc.Name, doc.ID, port, port)},
		},
	}, nil
}

// readServeError extracts the human-readable message from a serve error
// response. Serve's error envelope is {"error": "..."}; if the body is
// empty or not JSON, fall back to the raw text so the agent always sees
// something actionable.
func readServeError(resp *http.Response) string {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return resp.Status
	}
	var er errorResponse
	if json.Unmarshal(raw, &er) == nil && er.Error != "" {
		return er.Error
	}
	return strings.TrimSpace(string(raw))
}

// errorResult builds the IsError=true CallToolResult used for all tool-level
// failures. Per the MCP convention (and R005), the message goes in Content
// so the LLM can read it and self-correct; IsError flags the failure.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

// NewServer is an exported test seam over newServer: it returns the
// single-tool server (send_html registered) without running a transport.
// Tests connect a real mcp.Client over mcp.NewInMemoryTransports to drive
// the tool end-to-end through the protocol. Production code calls Run.
func NewServer(port int) *mcp.Server { return newServer(port) }
