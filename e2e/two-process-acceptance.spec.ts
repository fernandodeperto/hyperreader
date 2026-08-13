// End-to-end proof for the real agent-to-browser path.
//
// Playwright drives the running serve process while a compiled `hyperreader
// mcp` process sends HTML over raw stdio JSON-RPC. The resulting SSE event
// updates the hidden table while another stored page remains open in the
// same-tab reader.
import { test, expect } from "@playwright/test";
import { type ChildProcessWithoutNullStreams, spawn, spawnSync } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

// Same port convention as playwright.config.ts: PORT env shared with the
// webServer that is already running when this test starts.
const port = Number(process.env.PORT) || 7421;


// buildHyperReaderBinary compiles the real hyperreader binary once into a fresh
// temp dir (outside the repo, so it never needs a .gitignore entry of its
// own) and returns its path. Building the shipped binary — not reusing a
// test binary — is what makes the subprocess below an honest proof of the
// actual CLI's subcommand dispatch, flag parsing, and stdout-purity
// contract, exactly as go's own main_mcp_e2e_test.go does for the
// mcp-to-serve half of this proof.
function buildHyperReaderBinary(): string {
  const dir = mkdtempSync(path.join(tmpdir(), "hyperreader-acceptance-"));
  const bin = path.join(dir, process.platform === "win32" ? "hyperreader.exe" : "hyperreader");
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

test("real mcp process updates the table during a same-tab page detour", async ({ page }) => {
  const timestamp = Date.now();
  const detourSlug = `two-process-detour-${timestamp}`;
  const detourName = `Two Process Detour ${timestamp}`;
  const uniqueSlug = `two-process-acceptance-page-${timestamp}`;
  const uniqueName = `Two Process Acceptance Page ${timestamp}`;
  const pageHTML =
    "<!DOCTYPE html><html><head><title>Acceptance</title></head>" +
    "<body><h1>Acceptance</h1><p id='static-marker'>static</p>" +
    "<script>document.body.insertAdjacentHTML('beforeend'," +
    "'<p id=\"script-marker\">script-ran</p>');</script>" +
    "</body></html>";

  const detourResponse = await page.request.post("/api/pages", {
    data: {
      slug: detourSlug,
      name: detourName,
      description: "kept open while the mcp process sends another page",
      html: pageHTML,
    },
  });
  expect(detourResponse.ok()).toBeTruthy();

  const bin = buildHyperReaderBinary();
  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await expect(page.locator("#live-status")).toHaveAttribute("data-state", "live", {
    timeout: 10_000,
  });

  const rows = page.locator("#pages-table tbody tr");
  const initialCount = await rows.count();
  await page.locator("#search").fill(detourName);
  await expect(rows).toHaveCount(1);
  await page.locator("#search").fill("");
  await expect(rows).toHaveCount(initialCount);
  await page.locator(`#pages-table tbody tr[data-slug="${detourSlug}"]`).click();
  await expect(page.locator("#page-view")).toBeVisible();
  await expect(page.frameLocator("#page-frame").locator("#script-marker")).toHaveText(
    "script-ran",
  );
  expect(page.context().pages()).toHaveLength(1);

  await page.evaluate(() => {
    Reflect.set(window, "__acceptanceSentinel", "still-here");
  });

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
        slug: uniqueSlug,
        name: uniqueName,
        html: pageHTML,
        description: "pushed by a real separate mcp OS process",
      },
    });
    if (callRes.error) {
      throw new Error(
        `tools/call(send_html) protocol-level error: ${JSON.stringify(callRes.error)}\nstderr:\n${client.stderr()}`,
      );
    }
    if (callRes.result?.isError) {
      throw new Error(
        `send_html returned isError=true: ${JSON.stringify(callRes.result)}\nstderr:\n${client.stderr()}`,
      );
    }

    await expect(rows).toHaveCount(initialCount + 1, { timeout: 10_000 });
    await expect(rows.first()).toContainText(uniqueName);
    await expect(page.locator("#page-view")).toBeVisible();
    await expect(page.locator("#table-view")).toBeHidden();
    await expect(page.frameLocator("#page-frame").locator("#static-marker")).toHaveText(
      "static",
    );
  } finally {
    client.close();
  }

  expect(await page.evaluate(() => Reflect.get(window, "__acceptanceSentinel"))).toBe(
    "still-here",
  );
  await page.locator("#home-link").click();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(initialCount + 1);
  await expect(rows.first()).toContainText(uniqueName);

  await page.locator("#search").fill(uniqueName);
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText(uniqueName);
  expect(page.context().pages()).toHaveLength(1);
});
