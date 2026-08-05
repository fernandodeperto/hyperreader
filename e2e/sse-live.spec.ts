// Playwright proof for S04-T03: live SSE row append with no page refresh,
// then detail view + Back on that live-appended row (R002/R006 continuity).
//
// This runs against the REAL `html-mcp serve` binary (see
// playwright.config.ts webServer) — the same process the smoke suite uses.
// It proves the S04 slice's core UI contract:
//   1. the page opens and its EventSource connects (#live-status -> "live")
//   2. a document ingested through the running server's real API — while
//      the page stays open and idle — appears as a new top row via the SSE
//      broadcast, with NO fetch-driven re-render and NO page reload
//   3. that live-appended row behaves like any other row: clicking it
//      renders its HTML full-page unsandboxed (inline <script> executes)
//   4. Back returns to the table with the row still present and still
//      findable via the real FTS5 search
//
// "No reload" is proven with a window sentinel (per the slice's Must-Haves):
// a value stashed on `window` after the page loads that a navigation/reload
// would clear. If the sentinel is still present after the live row appears,
// the row arrived via the EventSource/DOM path, not a reload.
import { test, expect, type Page } from "@playwright/test";

// A document whose HTML contains an inline <script> that writes a marker
// into the DOM, plus a static marker — same proof shape as e2e/smoke.spec.ts
// (R006: unsandboxed rendering). The inserted id uses double quotes inside a
// single-quoted JS string to avoid nested-quote escaping ambiguity.
function liveDoc(uniqueName: string) {
  return {
    name: uniqueName,
    description: "pushed while the page was open, via SSE",
    tags: "live,sse",
    html:
      "<!DOCTYPE html><html><head><title>Live</title></head>" +
      "<body><h1>Live</h1><p id='static-marker'>static</p>" +
      "<script>document.body.insertAdjacentHTML('beforeend'," +
      "'<p id=\"script-marker\">script-ran</p>');</script>" +
      "</body></html>",
  };
}

function tableRows(page: Page) {
  return page.locator("#documents-table tbody tr");
}

test("live row append with no refresh, then detail and Back", async ({ page }) => {
  // A name unique to this test run so the FTS5 search step at the end can
  // isolate this exact row regardless of whatever else the shared store
  // (webServer runs once for the whole suite) already contains.
  const uniqueName = "SSE Live Doc " + Date.now();
  const doc = liveDoc(uniqueName);

  // 1. Open the UI and let the initial table fetch + EventSource connect.
  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  const liveStatus = page.locator("#live-status");
  // The stream flushes ": connected" immediately on subscribe, which the
  // browser surfaces as EventSource's "open" event -> app.js flips
  // #live-status to "live". Wait for that before pushing the document so
  // the test proves the *subscribed* connection delivered the event, not a
  // lucky race with the initial fetch.
  await expect(liveStatus).toHaveAttribute("data-state", "live", { timeout: 10_000 });

  const rows = tableRows(page);
  const initialCount = await rows.count();

  // Stash a sentinel on window now, with the page fully settled. A reload
  // or navigation would clear this; its continued presence after the live
  // row appears proves the update happened without one.
  await page.evaluate(() => {
    (window as unknown as { __sseLiveSentinel: string }).__sseLiveSentinel = "still-here";
  });

  // 2. Ingest a document through the running server's real API while the
  // page stays open and idle — this is the "agent pushes a doc" side of
  // the live-update contract (T04 proves it via a real separate mcp OS
  // process; this test proves the browser-visible half via the same real
  // HTTP endpoint the mcp server calls into).
  const createRes = await page.request.post("/api/documents", { data: doc });
  expect(createRes.ok()).toBeTruthy();
  const created = await createRes.json();
  expect(created.id).toBeGreaterThan(0);

  // 3. The new row appears live: no page.reload(), no manual re-fetch call
  // from the test — only the count growing and the top row matching proves
  // the SSE broadcast -> EventSource -> DOM prepend path worked.
  await expect(rows).toHaveCount(initialCount + 1, { timeout: 10_000 });
  await expect(rows.nth(0)).toContainText(uniqueName);

  // The sentinel survived: no reload occurred anywhere in this flow.
  const sentinel = await page.evaluate(
    () => (window as unknown as { __sseLiveSentinel?: string }).__sseLiveSentinel
  );
  expect(sentinel).toBe("still-here");

  // 4. Click the live-appended row -> unsandboxed detail view, same as any
  // other row (R006 continuity for live rows).
  await rows.nth(0).click();
  await expect(page.locator("#table-view")).toBeHidden();
  await expect(page.locator("#detail-view")).toBeVisible();

  const iframeEl = page.locator("#detail-frame");
  await expect(iframeEl).not.toHaveAttribute("sandbox");

  const frame = page.frameLocator("#detail-frame");
  await expect(frame.locator("#static-marker")).toHaveText("static");
  // The inline <script> ran — unsandboxed rendering proven even for a row
  // that arrived via SSE rather than the initial fetch.
  await expect(frame.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });

  // 5. Back -> table restored, live row still present.
  await page.locator("#back-button").click();
  await expect(page.locator("#detail-view")).toBeHidden();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(initialCount + 1);
  await expect(page.locator("#documents-table tbody tr", { hasText: uniqueName })).toBeVisible();

  // 6. The live row is searchable via the real FTS5 API, not just present
  // in the DOM from the prepend.
  await page.locator("#search").fill(uniqueName);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(uniqueName);

  // Clear search -> full list restored (sanity: search state didn't wedge).
  await page.locator("#search").fill("");
  await expect(rows).toHaveCount(initialCount + 1);
});
