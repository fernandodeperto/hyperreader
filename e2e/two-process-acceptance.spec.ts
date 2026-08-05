// Playwright proof for S04-T04 — the milestone's explicitly non-mockable
// acceptance criterion.
//
// `html-mcp serve` is already running as one real OS process (Playwright's
// webServer in playwright.config.ts). This spec opens a real Chromium page
// against it, then spawns the *compiled binary's* `mcp` subcommand as a
// SECOND, separate OS process — exactly as an agent's MCP client would
// launch it — and drives it over raw stdio JSON-RPC (newline-delimited
// jsonrpc2, the go-sdk's real CommandTransport wire framing; see
// go-sdk@v1.7.0's ioConn read/write in mcp/transport.go). No go-sdk client
// helper is used here: this test speaks the literal bytes an MCP host
// sends, which is what makes it non-mockable at the protocol boundary.
//
// It proves the full agent-to-browser loop: send_html over the second
// process's stdio -> serve's POST /api/documents -> serve's SSE broadcast
// -> the already-open page's EventSource -> a new top row, with no manual
// refresh — then that row's detail view renders unsandboxed and Back
// leaves it present and FTS5-searchable (same continuity contract as
// e2e/sse-live.spec.ts, now proven end-to-end with real processes at every
// hop).
import { test, expect, type Page } from "@playwright/test";
import { type ChildProcessWithoutNullStreams, spawn, spawnSync } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

// Same port convention as playwright.config.ts: PORT env shared with the
// webServer that is already running when this test starts.
const port = Number(process.env.PORT) || 7421;

function tableRows(page: Page) {
  return page.locator("#documents-table tbody tr");
}

// buildHTMLMCPBinary compiles the real html-mcp binary once into a fresh
// temp dir (outside the repo, so it never needs a .gitignore entry of its
// own) and returns its path. Building the shipped binary — not reusing a
// test binary — is what makes the subprocess below an honest proof of the
// actual CLI's subcommand dispatch, flag parsing, and stdout-purity
// contract, exactly as go's own main_mcp_e2e_test.go does for the
// mcp-to-serve half of this proof.
function buildHTMLMCPBinary(): string {
  const dir = mkdtempSync(path.join(tmpdir(), "html-mcp-acceptance-"));
  const bin = path.join(dir, process.platform === "win32" ? "html-mcp.exe" : "html-mcp");
  const repoRoot = path.resolve(__dirname, "..");
  const res = spawnSync("go", ["build", "-o", bin, "."], {
    cwd: repoRoot,
    encoding: "utf-8",
    timeout: 60_000,
  });
  if (res.status !== 0) {
    throw new Error(`go build -o ${bin} . failed (status ${res.status}):\n${res.stderr}`);
  }
  return bin;
}

// A minimal, dependency-free JSON-RPC 2.0 client over the mcp subprocess's
// stdio, speaking the same newline-delimited framing the go-sdk's
// CommandTransport uses on the wire (one JSON object per line, no
// Content-Length headers). This is deliberately NOT the
// @modelcontextprotocol/sdk client: raw framing is what proves the literal
// bytes an MCP host sends, not an SDK abstraction over them.
class RawStdioJSONRPCClient {
  private proc: ChildProcessWithoutNullStreams;
  private buffer = "";
  private nextID = 1;
  private pending = new Map<number, { resolve: (v: any) => void; reject: (e: Error) => void }>();
  private stderrBuf = "";

  constructor(bin: string, args: string[]) {
    this.proc = spawn(bin, args, { stdio: ["pipe", "pipe", "pipe"] });
    this.proc.stderr.on("data", (chunk: Buffer) => {
      this.stderrBuf += chunk.toString("utf-8");
    });
    this.proc.stdout.on("data", (chunk: Buffer) => this.onData(chunk));
    this.proc.on("error", (err) => {
      for (const { reject } of this.pending.values()) reject(err);
      this.pending.clear();
    });
  }

  private onData(chunk: Buffer) {
    this.buffer += chunk.toString("utf-8");
    let idx: number;
    // eslint-disable-next-line no-cond-assign
    while ((idx = this.buffer.indexOf("\n")) >= 0) {
      const line = this.buffer.slice(0, idx);
      this.buffer = this.buffer.slice(idx + 1);
      if (!line.trim()) continue;
      let msg: any;
      try {
        msg = JSON.parse(line);
      } catch (err) {
        // A stray non-JSON byte on stdout would corrupt this exact framing
        // — surface it loudly rather than silently dropping it.
        throw new Error(`mcp subprocess wrote non-JSON line on stdout: ${line}\n${err}`);
      }
      if (msg && typeof msg.id !== "undefined" && this.pending.has(msg.id)) {
        const waiter = this.pending.get(msg.id)!;
        this.pending.delete(msg.id);
        waiter.resolve(msg);
      }
      // Notifications (no id) from the server are ignored — this test only
      // needs request/response round trips.
    }
  }

  private write(obj: unknown) {
    this.proc.stdin.write(JSON.stringify(obj) + "\n");
  }

  request(method: string, params: unknown, timeoutMs = 10_000): Promise<any> {
    const id = this.nextID++;
    const p = new Promise<any>((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          reject(
            new Error(
              `timed out waiting for response to ${method} (id ${id}); stderr:\n${this.stderrBuf}`
            )
          );
        }
      }, timeoutMs);
    });
    this.write({ jsonrpc: "2.0", id, method, params });
    return p;
  }

  notify(method: string, params?: unknown) {
    this.write({ jsonrpc: "2.0", method, params });
  }

  stderr(): string {
    return this.stderrBuf;
  }

  close() {
    this.proc.stdin.end();
    // Give the subprocess a moment to exit on stdin EOF, then force-kill so
    // a hung process cannot leak past the test.
    setTimeout(() => {
      if (this.proc.exitCode === null && this.proc.signalCode === null) {
        this.proc.kill();
      }
    }, 500);
  }
}

test("milestone acceptance: real mcp OS process pushes send_html into a real open browser via SSE", async ({
  page,
}) => {
  const bin = buildHTMLMCPBinary();

  // 1. Open the real page against the already-running serve process and
  // let its EventSource subscribe.
  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  const liveStatus = page.locator("#live-status");
  await expect(liveStatus).toHaveAttribute("data-state", "live", { timeout: 10_000 });

  const rows = tableRows(page);
  const initialCount = await rows.count();

  // Sentinel: a reload/navigation would clear this. Its survival after the
  // live row appears proves the update arrived via SSE, not a refresh.
  await page.evaluate(() => {
    (window as unknown as { __acceptanceSentinel: string }).__acceptanceSentinel = "still-here";
  });

  const uniqueName = "Two-Process Acceptance Doc " + Date.now();
  const docHTML =
    "<!DOCTYPE html><html><head><title>Acceptance</title></head>" +
    "<body><h1>Acceptance</h1><p id='static-marker'>static</p>" +
    "<script>document.body.insertAdjacentHTML('beforeend'," +
    "'<p id=\"script-marker\">script-ran</p>');</script>" +
    "</body></html>";

  // 2. Spawn the compiled binary's `mcp` subcommand as a SECOND, separate
  // real OS process — pointed at the already-running serve process's port
  // — and speak raw newline-delimited JSON-RPC to it over its stdio, the
  // same mechanism a real MCP host uses.
  const client = new RawStdioJSONRPCClient(bin, ["mcp", "--port", String(port)]);
  try {
    const initRes = await client.request("initialize", {
      protocolVersion: "2025-06-18",
      capabilities: {},
      clientInfo: { name: "two-process-acceptance-test", version: "0.0.1" },
    });
    if (initRes.error) {
      throw new Error(`initialize failed: ${JSON.stringify(initRes.error)}`);
    }
    client.notify("notifications/initialized", {});

    const callRes = await client.request("tools/call", {
      name: "send_html",
      arguments: {
        name: uniqueName,
        html: docHTML,
        description: "pushed by a real separate mcp OS process",
        tags: "acceptance,two-process",
      },
    });

    if (callRes.error) {
      throw new Error(
        `tools/call(send_html) protocol-level error: ${JSON.stringify(callRes.error)}\nstderr:\n${client.stderr()}`
      );
    }
    const result = callRes.result;
    if (result?.isError) {
      throw new Error(
        `send_html returned isError=true: ${JSON.stringify(result)}\nstderr:\n${client.stderr()}`
      );
    }

    // 3. The row appears live in the already-open page — no reload, no
    // manual re-fetch — driven purely by the SSE broadcast that the
    // subprocess's forwarded POST /api/documents triggered.
    await expect(rows).toHaveCount(initialCount + 1, { timeout: 10_000 });
    await expect(rows.nth(0)).toContainText(uniqueName);

    const sentinel = await page.evaluate(
      () => (window as unknown as { __acceptanceSentinel?: string }).__acceptanceSentinel
    );
    expect(sentinel).toBe("still-here");
  } finally {
    client.close();
  }

  // 4. The live-appended row behaves like any other row: unsandboxed
  // detail rendering, then Back with the row still present and searchable.
  await rows.nth(0).click();
  await expect(page.locator("#table-view")).toBeHidden();
  await expect(page.locator("#detail-view")).toBeVisible();

  const iframeEl = page.locator("#detail-frame");
  await expect(iframeEl).not.toHaveAttribute("sandbox");

  const frame = page.frameLocator("#detail-frame");
  await expect(frame.locator("#static-marker")).toHaveText("static");
  await expect(frame.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });

  await page.locator("#back-button").click();
  await expect(page.locator("#detail-view")).toBeHidden();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(initialCount + 1);
  await expect(page.locator("#documents-table tbody tr", { hasText: uniqueName })).toBeVisible();

  await page.locator("#search").fill(uniqueName);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(uniqueName);

  await page.locator("#search").fill("");
  await expect(rows).toHaveCount(initialCount + 1);
});
