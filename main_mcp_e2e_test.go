// Copyright 2025.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPSubprocess is the slice's integration proof (S03's Proof Level:
// "real runtime required: yes"): a real MCP client drives the *compiled
// binary's* mcp subcommand as a separate OS process over real stdio
// JSON-RPC, while a separately-launched serve process (also a real OS
// process) owns the SQLite store. Everything below the top-level subtests
// exercises actual TCP sockets and actual subprocess pipes — no in-process
// shortcuts (those are covered by internal/mcp's NewInMemoryTransports
// tests).
//
// It proves two things the in-process tests structurally cannot:
//  1. send_html forwarded through a real subprocess mcp reaches a real
//     subprocess serve, and the document becomes observable through
//     serve's own API (get, get-content, and an FTS search hit).
//  2. When no serve is listening, the real subprocess mcp still returns
//     IsError=true with an actionable message — never a hang, a crash, or
//     a silent success — closing the loop on the stdout-purity contract
//     (a stray byte on mcp's stdout would corrupt the JSON-RPC framing
//     that this very test depends on to talk to it at all).
func TestMCPSubprocess(t *testing.T) {
	bin := buildHyperReaderBinary(t)

	t.Run("success: send_html forwards through real subprocesses to serve", func(t *testing.T) {
		dataDir := t.TempDir()
		port := freeTCPPort(t)

		serveCmd := startServeProcess(t, bin, dataDir, port)
		waitForServeReady(t, port, serveCmd)

		session := connectMCPSubprocess(t, bin, port)

		// Distinctive tokens so the FTS search assertion below cannot
		// false-positive against unrelated rows.
		const docName = "E2E-Proof-Doc-9f3a"
		const docHTML = "<h1>hello from the e2e test</h1>"

		res := callSendHTMLTool(t, session, map[string]any{
			"name":        docName,
			"html":        docHTML,
			"description": "written by TestMCPSubprocess",
			"tags":        "e2e,proof",
		})
		if res.IsError {
			t.Fatalf("send_html over real subprocess mcp returned IsError=true: %s", toolResultText(res))
		}
		text := toolResultText(res)
		if !strings.Contains(text, docName) {
			t.Fatalf("success result text %q does not name the document", text)
		}
		t.Logf("send_html result: %s", text)

		id := findDocumentIDByName(t, port, docName)

		// Ground truth: serve's own API, not the tool's self-report.
		content := getDocumentContent(t, port, id)
		if content != docHTML {
			t.Fatalf("GET /api/documents/%d/content = %q, want %q", id, content, docHTML)
		}

		hits := searchDocuments(t, port, "E2E-Proof-Doc-9f3a")
		found := false
		for _, d := range hits {
			if d.Name == docName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("FTS search for %q found no match among %d result(s)", docName, len(hits))
		}
	})

	t.Run("failure: serve not running surfaces IsError=true without hanging or crashing", func(t *testing.T) {
		// A port nothing is listening on: freeTCPPort releases its
		// listener before returning, guaranteeing connection-refused.
		port := freeTCPPort(t)

		session := connectMCPSubprocess(t, bin, port)

		res := callSendHTMLTool(t, session, map[string]any{
			"name": "should not persist",
			"html": "<p>x</p>",
		})
		if !res.IsError {
			t.Fatalf("expected IsError=true with no serve listening, got success: %s", toolResultText(res))
		}
		text := toolResultText(res)
		if !strings.Contains(text, strconv.Itoa(port)) {
			t.Errorf("result text %q does not name the unreachable port %d", text, port)
		}
		if !strings.Contains(text, "hyperreader serve") {
			t.Errorf("result text %q does not tell the agent to start `hyperreader serve`", text)
		}
		t.Logf("expected failure result: %s", text)
	})
}

// buildHyperReaderBinary compiles the hyperreader binary once into a temp dir and
// returns its path. Building the real binary (rather than reusing the test
// binary) is what makes this an honest proof of the shipped CLI: it
// exercises main()'s actual subcommand dispatch, flag parsing, and the
// stdout-purity contract exactly as a real agent's MCP client would see it.
func buildHyperReaderBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "hyperreader")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build -o %s .: %v\n%s", bin, err, stderr.String())
	}
	return bin
}

// freeTCPPort returns a TCP port that is free at the moment of the call: it
// binds 127.0.0.1:0 (OS-assigned free port), records the port, then closes
// the listener before returning. A subsequent dial to it — if nothing else
// binds it first — fails with "connection refused", which is exactly the
// serve-not-running scenario the second subtest needs.
func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return port
}

// startServeProcess launches `<bin> serve --port <port> --data-dir <dataDir>`
// as a real, independent OS process (not a goroutine) — the two-process
// boundary the slice proves. It registers cleanup to terminate the process
// at test end regardless of outcome.
func startServeProcess(t *testing.T, bin, dataDir string, port int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(bin, "serve", "--port", strconv.Itoa(port), "--data-dir", dataDir)
	var stderr bytes.Buffer
	cmd.Stdout = &stderr // serve's one startup line is diagnostic, not protocol-bearing
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start serve process: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		if t.Failed() {
			t.Logf("serve process output:\n%s", stderr.String())
		}
	})
	return cmd
}

// waitForServeReady polls GET /api/documents on the given port until serve
// answers 200 or the deadline passes, failing the test on timeout. Polling
// the real API (rather than a fixed sleep) is what makes the two-process
// handshake deterministic across slow CI machines.
func waitForServeReady(t *testing.T, port int, serveCmd *exec.Cmd) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/documents", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if serveCmd.ProcessState != nil {
			t.Fatalf("serve process exited early (before becoming ready): %v", serveCmd.ProcessState)
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("serve on port %d did not become ready within the deadline", port)
}

// connectMCPSubprocess launches `<bin> mcp --port <port>` as a real OS
// process and connects a real mcp.Client to it over mcp.CommandTransport
// (real stdin/stdout pipes, real newline-delimited JSON-RPC framing) — the
// same mechanism a production MCP host uses to drive this binary. Cleanup
// closes the session, which closes stdin and lets the subprocess exit on
// EOF (falling back to SIGTERM/SIGKILL per CommandTransport's Close, so a
// hung subprocess cannot leak past the test).
func connectMCPSubprocess(t *testing.T, bin string, port int) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(bin, "mcp", "--port", strconv.Itoa(port))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr // mcp's stdout is JSON-RPC-only; stderr is diagnostics

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-test-client", Version: "v0.0.1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect real MCP client to subprocess `%s mcp --port %d`: %v\nstderr:\n%s", bin, port, err, stderr.String())
	}
	t.Cleanup(func() {
		_ = session.Close()
		if t.Failed() && stderr.Len() > 0 {
			t.Logf("mcp subprocess stderr:\n%s", stderr.String())
		}
	})
	return session
}

// callSendHTMLTool invokes send_html over the given session and fails the
// test if CallTool itself returns a Go error — per R005/MEM013, a
// protocol-level error here would mean a tool-level failure leaked past the
// IsError=true contract, which is a regression this test exists to catch.
func callSendHTMLTool(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "send_html", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(send_html) over real subprocess: protocol-level error (should never happen; tool failures must be IsError=true results): %v", err)
	}
	return res
}

func toolResultText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// apiDoc mirrors the JSON shape serve's API returns for a document
// (internal/api.documentResponse) — duplicated here deliberately, same as
// internal/mcp does, so this test file depends only on the wire contract.
type apiDoc struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
}

// findDocumentIDByName is the ground-truth check that send_html's HTTP
// forward actually landed in serve's store: it lists documents through
// serve's real API (GET /api/documents) and returns the id of the one
// matching name, failing the test if it is not there.
func findDocumentIDByName(t *testing.T, port int, name string) int64 {
	t.Helper()
	docs := listDocuments(t, port)
	for _, d := range docs {
		if d.Name == name {
			return d.ID
		}
	}
	t.Fatalf("document %q not found via GET /api/documents on port %d (got %d doc(s))", name, port, len(docs))
	return 0
}

func listDocuments(t *testing.T, port int) []apiDoc {
	t.Helper()
	return getDocumentsJSON(t, fmt.Sprintf("http://127.0.0.1:%d/api/documents", port))
}

func searchDocuments(t *testing.T, port int, q string) []apiDoc {
	t.Helper()
	return getDocumentsJSON(t, fmt.Sprintf("http://127.0.0.1:%d/api/documents?q=%s", port, q))
}

func getDocumentsJSON(t *testing.T, url string) []apiDoc {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", url, resp.StatusCode, body)
	}
	var docs []apiDoc
	if err := json.Unmarshal(body, &docs); err != nil {
		t.Fatalf("GET %s: decode response %s: %v", url, body, err)
	}
	return docs
}

// getDocumentContent fetches GET /api/documents/{id}/content — the ground
// truth for the HTML payload send_html forwarded, independent of what the
// tool result text claims.
func getDocumentContent(t *testing.T, port int, id int64) string {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/api/documents/%d/content", port, id)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", url, resp.StatusCode, body)
	}
	return string(body)
}
