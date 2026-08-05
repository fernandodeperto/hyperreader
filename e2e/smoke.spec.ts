// Browser smoke for the html-mcp primary user loop (R002/R006).
//
// This test runs against the REAL `html-mcp serve` binary (see
// playwright.config.ts webServer). It proves the live end-to-end flow:
//   1. seed documents via the API (POST /api/documents)
//   2. the table renders them (name/description/tags)
//   3. search filters the table (real FTS5, GET /api/documents?q=)
//   4. clicking a row opens that document's raw rendered HTML in a NEW
//      BROWSER TAB (window.open to GET /api/documents/{id}/content), with
//      zero app chrome — the doc's inline <script> runs and leaves a
//      marker (R006: unsandboxed rendering, now at the top level rather
//      than inside an in-app iframe)
//   5. the table view stays visible (Branch B never hides it); the row is
//      still present + searchable after the popup is closed
//
// M002 / Branch B removed the in-app detail view, iframe, and Back button
// entirely (not hidden). There is no sandboxed iframe anymore: the browser
// navigates the existing content endpoint directly in a new top-level tab,
// so an inline <script> in the document executes natively — its marker
// proves unsandboxed rendering (R006) in the new-tab context.
//
// Cross-test state: the webServer starts ONCE for the whole suite, so tests
// share the store. Test 1 runs first on the clean slate (webServer wipes the
// data dir on start) and asserts exact counts. Test 2 search-isolates its own
// uniquely-named doc so it is robust to whatever test 1 left behind.
import { test, expect, type Page, type Locator } from "@playwright/test";

// A document whose HTML contains an inline <script> that writes a marker
// into the DOM. If the new tab were sandboxed (it is not — it is a normal
// top-level browser navigation), the script would be blocked and the marker
// never appears, so the marker's presence proves unsandboxed rendering
// (R006). A <p> static marker is also included so a totally-disabled-JS
// environment would still fail loudly rather than pass on a false negative.
// The inserted id uses double quotes inside a single-quoted JS string to
// avoid nested-quote escaping ambiguity.
const DOC_WITH_INLINE_SCRIPT = {
  name: "Deploy Runbook",
  description: "production deploy steps",
  tags: "ops,deploy",
  html:
    "<!DOCTYPE html><html><head><title>Deploy</title></head>" +
    "<body><h1>Deploy Runbook</h1><p id='static-marker'>static</p>" +
    "<script>document.body.insertAdjacentHTML('beforeend'," +
    "'<p id=\"script-marker\">script-ran</p>');</script>" +
    "</body></html>",
};

const DOC_OTHER = {
  name: "On-call Guide",
  description: "rotation schedule",
  tags: "ops,oncall",
  html: "<!DOCTYPE html><html><body><h1>On-call</h1></body></html>",
};

// A uniquely-named doc used by test 2 so it can search-isolate its own row
// regardless of what test 1 seeded into the shared store.
const DOC_UNSANDBOXED_PROOF = {
  name: "Unsandboxed Proof Doc",
  description: "proves inline scripts execute in the new browser tab",
  tags: "test,unsandboxed",
  html:
    "<!DOCTYPE html><html><body><p id='static-marker'>static</p>" +
    "<script>document.body.insertAdjacentHTML('beforeend'," +
    "'<p id=\"script-marker\">script-ran</p>');</script>" +
    "</body></html>",
};

// A tall-content doc used by test 3 to prove the new browser tab renders the
// FULL document with browser-native scrolling (M002 / Branch B): there is no
// in-app iframe to cap or measure, so the new tab's document is the sole
// scroll surface and its 2000px content block renders completely. The 2000px
// fixed-height block is deliberately taller than the Playwright viewport
// (720px), so a documentElement.scrollHeight above 1000px proves the full
// tall content rendered in the new tab rather than being clipped. A static
// marker is included so a failure to render any content fails loudly rather
// than passing on an empty page.
const DOC_TALL_CONTENT = {
  name: "Tall Content Scroll Doc",
  description: "proves the new tab renders the full tall document",
  tags: "test,scroll",
  html:
    "<!DOCTYPE html><html><body>" +
    "<p id='static-marker'>static</p>" +
    "<div style='height:2000px'>tall content block</div>" +
    "</body></html>",
};

// seed POSTs a document via the real API and returns the created id. The
// table view is populated by GET /api/documents, so seeding through the API
// proves the UI reflects real storage — not test-injected DOM.
async function seed(page: Page, doc: typeof DOC_WITH_INLINE_SCRIPT): Promise<number> {
  const res = await page.request.post("/api/documents", { data: doc });
  expect(res.ok()).toBeTruthy();
  const body = await res.json();
  expect(body.id).toBeGreaterThan(0);
  return body.id as number;
}

// Wait for the table to finish loading (the loading state hides once the
// first fetch resolves) and return the row locator.
function tableRows(page: Page) {
  return page.locator("#documents-table tbody tr");
}

// openRowInNewTab clicks a document row and returns the popup Page that
// window.open produced (Branch B: GET /api/documents/{id}/content in a new
// top-level tab with zero app chrome). The popup promise is registered
// BEFORE the click so Playwright captures the popup event regardless of
// timing. waitForLoadState("domcontentloaded") ensures the document's HTML
// (and its inline <script>) has parsed before the caller asserts on markers.
async function openRowInNewTab(page: Page, row: Locator): Promise<Page> {
  const popupPromise = page.waitForEvent("popup");
  await row.click();
  const popup = await popupPromise;
  await popup.waitForLoadState("domcontentloaded");
  return popup;
}

test("primary user loop: table -> search -> unsandboxed new tab -> table intact", async ({ page }) => {
  // 1. Seed two real documents through the API.
  await seed(page, DOC_WITH_INLINE_SCRIPT);
  await seed(page, DOC_OTHER);

  // 2. Open the UI and wait for the table to render with both rows.
  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  const rows = tableRows(page);
  await expect(rows).toHaveCount(2);
  // most-recent-first: the second-seeded doc is on top
  await expect(rows.nth(0)).toContainText("On-call Guide");
  await expect(rows.nth(1)).toContainText("Deploy Runbook");

  // 3. Search filters the table via the real FTS5 API.
  const search = page.locator("#search");
  await search.fill("deploy");
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Deploy Runbook");

  // Clear search -> full list restored.
  await search.fill("");
  await expect(rows).toHaveCount(2);

  // Regression guard (Branch B): the in-app detail view, iframe, and Back
  // button were DELETED, not hidden. They must be absent from the DOM — a
  // passing-by-omission hole on dead code is explicitly disallowed, so
  // assert the absence directly.
  await expect(page.locator("#detail-view")).toHaveCount(0);
  await expect(page.locator("#detail-frame")).toHaveCount(0);
  await expect(page.locator("#back-button")).toHaveCount(0);

  // 4. Click the Deploy Runbook row -> its raw rendered HTML opens in a new
  //    browser tab (window.open to /api/documents/{id}/content).
  const deployRow = page.locator("#documents-table tbody tr", { hasText: "Deploy Runbook" });
  const popup = await openRowInNewTab(page, deployRow);

  // The popup navigated to the content endpoint for this document.
  await expect(popup).toHaveURL(/\/api\/documents\/\d+\/content$/);

  // Branch B: the table view is never hidden — it stays visible while the
  // document opens in a separate tab (no in-app view-switching).
  await expect(page.locator("#table-view")).toBeVisible();

  // The static marker proves the document's HTML rendered in the new tab.
  await expect(popup.locator("#static-marker")).toHaveText("static");
  // The script marker proves the inline <script> executed — i.e. the new
  // tab is an unsandboxed top-level page (R006). There is no iframe to
  // sandbox it; the browser navigated the content endpoint directly.
  await expect(popup.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });

  // 5. Closing the popup leaves the table intact: both rows still present
  //    and the Deploy Runbook row is still searchable. (Branch B has no
  //    Back button — the user simply closes the tab.)
  await popup.close();
  await expect(rows).toHaveCount(2);
  await expect(page.locator("#documents-table tbody tr", { hasText: "Deploy Runbook" })).toBeVisible();

  // The row is still searchable after the new-tab detour.
  await search.fill("deploy");
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Deploy Runbook");
});

test("new tab renders document HTML unsandboxed via search-isolated row", async ({ page }) => {
  // A second, narrower test proving the unsandboxed new-tab render is
  // reachable independent of the click path's exact row. It search-isolates
  // its own uniquely-named doc so it is robust to whatever test 1 left in the
  // shared store (the webServer runs once for the whole suite).
  await seed(page, DOC_UNSANDBOXED_PROOF);

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  // Isolate this test's doc via the real FTS5 search so cross-test rows do
  // not affect the click target.
  await page.locator("#search").fill("Unsandboxed Proof");
  const rows = tableRows(page);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Unsandboxed Proof Doc");

  const popup = await openRowInNewTab(page, rows.nth(0));
  await expect(popup).toHaveURL(/\/api\/documents\/\d+\/content$/);

  // Branch B: the table view stays visible (no view-switching).
  await expect(page.locator("#table-view")).toBeVisible();

  await expect(popup.locator("#static-marker")).toHaveText("static");
  // The inline <script> ran — unsandboxed rendering proven in the new
  // top-level tab (R006), not inside a sandboxed iframe.
  await expect(popup.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });
});

test("new tab renders the full tall document with browser-native scroll (M002)", async ({ page }) => {
  // Proves Branch B's full-document render: with the in-app iframe gone,
  // a tall document opens in a new top-level tab whose document is the
  // sole scroll surface. There is no iframe to cap or auto-resize; the
  // browser renders the complete 2000px content block and scrolls it
  // natively. Search-isolates its own tall doc so it is robust to whatever
  // tests 1 and 2 left in the shared store.
  await seed(page, DOC_TALL_CONTENT);

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  // Isolate this test's doc via the real FTS5 search.
  await page.locator("#search").fill("Tall Content Scroll");
  const rows = tableRows(page);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Tall Content Scroll Doc");

  const popup = await openRowInNewTab(page, rows.nth(0));
  await expect(popup).toHaveURL(/\/api\/documents\/\d+\/content$/);

  // Branch B: the table view stays visible.
  await expect(page.locator("#table-view")).toBeVisible();

  // Wait for the document's HTML to render in the new tab — the static
  // marker proves content loaded (and fails loudly on an empty page rather
  // than passing on a blank tab).
  await expect(popup.locator("#static-marker")).toHaveText("static");

  // The 2000px content block is taller than the Playwright viewport
  // (720px), so a documentElement.scrollHeight above 1000px proves the
  // FULL tall content rendered in the new tab (not clipped by a capped
  // iframe), and that the new tab's document is the sole scroll surface —
  // exactly the browser-native single-scrollbar behavior Branch B delivers
  // by removing the iframe entirely.
  const scroll = await popup.evaluate(() => ({
    scrollHeight: document.documentElement.scrollHeight,
    clientHeight: document.documentElement.clientHeight,
  }));
  expect(scroll.scrollHeight).toBeGreaterThan(1000);
  expect(scroll.scrollHeight).toBeGreaterThan(scroll.clientHeight);
});
