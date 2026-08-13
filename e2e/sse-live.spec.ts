// Playwright proof for the live-page-updates slice: live SSE row append
// with no page refresh, then that live-appended row opens its page in a
// new browser tab (R002/R006 continuity).
//
// This runs against the REAL `hyperreader serve` binary (see
// playwright.config.ts webServer) — the same process the smoke suite uses.
// It proves the live-page-updates slice's core UI contract:
//   1. the page opens and its EventSource connects (#live-status -> "live")
//   2. a page created through the running server's real API — while the
//      page stays open and idle — appears as a new top row via the
//      page-created SSE broadcast, with NO fetch-driven re-render and NO
//      page reload
//   3. that live-appended row behaves like any other row: activating it
//      opens its raw rendered HTML in a NEW BROWSER TAB via window.open to
//      GET /api/pages/{slug}/content, where the page's inline <script>
//      runs and leaves a marker (R006: unsandboxed rendering at the top
//      level, not inside an in-app iframe — M002 / Branch B removed that
//      surface entirely)
//   4. the table view stays visible throughout (Branch B never hides it),
//      and the live row is still present and findable via the real FTS5
//      search after the popup is closed (no Back button anymore — the user
//      just closes the tab)
//
// "No reload" is proven with a window sentinel (per the slice's Must-Haves):
// a value stashed on `window` after the page loads that a navigation/reload
// would clear. If the sentinel is still present after the live row appears,
// the row arrived via the EventSource/DOM path, not a reload.
import { test, expect, type Page, type Locator } from "@playwright/test";

// A page whose HTML contains an inline <script> that writes a marker into
// the DOM, plus a static marker — same proof shape as e2e/smoke.spec.ts
// (R006: unsandboxed rendering). The inserted id uses double quotes inside a
// single-quoted JS string to avoid nested-quote escaping ambiguity.
function livePage(slug: string, uniqueName: string) {
  return {
    slug,
    name: uniqueName,
    description: "pushed while the page was open, via SSE",
    html:
      "<!DOCTYPE html><html><head><title>Live</title></head>" +
      "<body><h1>Live</h1><p id='static-marker'>static</p>" +
      "<script>document.body.insertAdjacentHTML('beforeend'," +
      "'<p id=\"script-marker\">script-ran</p>');</script>" +
      "</body></html>",
  };
}

function tableRows(page: Page) {
  return page.locator("#pages-table tbody tr");
}

// openRowInNewTab clicks a page row and returns the popup Page that
// window.open produced (Branch B: GET /api/pages/{slug}/content in a new
// top-level tab with zero app chrome). The popup promise is registered
// BEFORE the click so Playwright captures the popup event regardless of
// timing. waitForLoadState("domcontentloaded") ensures the page's HTML
// (and its inline <script>) has parsed before the caller asserts on markers.
async function openRowInNewTab(page: Page, row: Locator): Promise<Page> {
  const popupPromise = page.waitForEvent("popup");
  await row.click();
  const popup = await popupPromise;
  await popup.waitForLoadState("domcontentloaded");
  return popup;
}

test("live row append with no refresh, then new-tab open on that live row", async ({ page }) => {
  // A slug/name unique to this test run so the FTS5 search step at the end
  // can isolate this exact row regardless of whatever else the shared store
  // (webServer runs once for the whole suite) already contains.
  const timestamp = Date.now();
  const slug = "sse-live-page-" + timestamp;
  const uniqueName = "SSE Live Page " + timestamp;
  const doc = livePage(slug, uniqueName);

  // 1. Open the UI and let the initial table fetch + EventSource connect.
  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  const liveStatus = page.locator("#live-status");
  // The stream flushes ": connected" immediately on subscribe, which the
  // browser surfaces as EventSource's "open" event -> app.js flips
  // #live-status to "live". Wait for that before pushing the page so the
  // test proves the *subscribed* connection delivered the event, not a
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

  // 2. Create a page through the running server's real API while the page
  // stays open and idle — this is the "agent pushes a page" side of the
  // live-update contract (the two-process acceptance spec proves it via a
  // real separate mcp OS process; this test proves the browser-visible half
  // via the same real HTTP endpoint the mcp server calls into).
  const createRes = await page.request.post("/api/pages", { data: doc });
  expect(createRes.ok()).toBeTruthy();
  const created = await createRes.json();
  expect(created.slug).toBe(slug);

  // 3. The new row appears live: no page.reload(), no manual re-fetch call
  // from the test — only the count growing and the top row matching proves
  // the page-created SSE broadcast -> EventSource -> DOM prepend path
  // worked.
  await expect(rows).toHaveCount(initialCount + 1, { timeout: 10_000 });
  await expect(rows.nth(0)).toContainText(uniqueName);

  // The sentinel survived: no reload occurred anywhere in this flow.
  const sentinel = await page.evaluate(
    () => (window as unknown as { __sseLiveSentinel?: string }).__sseLiveSentinel
  );
  expect(sentinel).toBe("still-here");

  // Regression guard (Branch B): the in-app detail view, iframe, and Back
  // button were DELETED, not hidden. They must be absent from the DOM —
  // passing by omission on dead code is explicitly disallowed, so assert
  // the absence directly before exercising the new-tab path.
  await expect(page.locator("#detail-view")).toHaveCount(0);
  await expect(page.locator("#detail-frame")).toHaveCount(0);
  await expect(page.locator("#back-button")).toHaveCount(0);

  // 4. Activate the live-appended row -> its raw rendered HTML opens in a
  //    new browser tab (window.open to /api/pages/{slug}/content), same as
  //    any other row (R006 continuity for live rows — now proven in the
  //    new-tab context, not an in-app iframe).
  const popup = await openRowInNewTab(page, rows.nth(0));
  await expect(popup).toHaveURL(/\/api\/pages\/[a-z0-9-]+\/content$/);

  // Branch B: the table view is never hidden — it stays visible while the
  // page opens in a separate tab (no in-app view-switching).
  await expect(page.locator("#table-view")).toBeVisible();

  await expect(popup.locator("#static-marker")).toHaveText("static");
  // The inline <script> ran — unsandboxed rendering proven even for a row
  // that arrived via SSE rather than the initial fetch, in the new
  // top-level tab (R006) rather than a sandboxed iframe.
  await expect(popup.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });

  // 5. Closing the popup leaves the table intact: the live row is still
  //    present. (Branch B has no Back button — the user simply closes the
  //    tab.)
  await popup.close();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(initialCount + 1);
  await expect(page.locator("#pages-table tbody tr", { hasText: uniqueName })).toBeVisible();

  // 6. The live row is searchable via the real FTS5 API, not just present
  // in the DOM from the prepend.
  await page.locator("#search").fill(uniqueName);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText(uniqueName);

  // Clear search -> full list restored (sanity: search state didn't wedge).
  await page.locator("#search").fill("");
  await expect(rows).toHaveCount(initialCount + 1);
});
