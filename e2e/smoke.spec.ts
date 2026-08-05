// Browser smoke for the html-mcp primary user loop (R002/R006).
//
// This test runs against the REAL `html-mcp serve` binary (see
// playwright.config.ts webServer). It proves the live end-to-end flow:
//   1. seed documents via the API (POST /api/documents)
//   2. the table renders them (name/description/tags)
//   3. search filters the table (real FTS5, GET /api/documents?q=)
//   4. clicking a row renders that document's HTML full-page + UNSANDBOXED
//      (the doc's inline <script> runs and leaves a marker — R006)
//   5. Back returns to the table with the row still present + searchable
//
// The detail iframe must have NO sandbox attribute (R006): inline scripts in
// the document must execute, proving unsandboxed rendering.
//
// Cross-test state: the webServer starts ONCE for the whole suite, so tests
// share the store. Test 1 runs first on the clean slate (webServer wipes the
// data dir on start) and asserts exact counts. Test 2 search-isolates its own
// uniquely-named doc so it is robust to whatever test 1 left behind.
import { test, expect, type Page } from "@playwright/test";

// A document whose HTML contains an inline <script> that writes a marker
// into the DOM. If the iframe is sandboxed without allow-scripts, the script
// is blocked and the marker never appears — so the marker's presence proves
// unsandboxed rendering (R006). A <p> static marker is also included so a
// totally-disabled-JS environment would still fail loudly rather than pass
// on a false negative. The inserted id uses double quotes inside a
// single-quoted JS string to avoid nested-quote escaping ambiguity.
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
  description: "proves inline scripts execute in the detail iframe",
  tags: "test,unsandboxed",
  html:
    "<!DOCTYPE html><html><body><p id='static-marker'>static</p>" +
    "<script>document.body.insertAdjacentHTML('beforeend'," +
    "'<p id=\"script-marker\">script-ran</p>');</script>" +
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

test("primary user loop: table -> search -> unsandboxed detail -> back", async ({ page }) => {
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

  // 4. Click the Deploy Runbook row -> unsandboxed detail view.
  await page.locator("#documents-table tbody tr", { hasText: "Deploy Runbook" }).click();

  // The table view hides, the detail view shows.
  await expect(page.locator("#table-view")).toBeHidden();
  await expect(page.locator("#detail-view")).toBeVisible();

  // R006: the iframe must NOT have a sandbox attribute — assert it directly.
  const iframeEl = page.locator("#detail-frame");
  await expect(iframeEl).not.toHaveAttribute("sandbox");

  const frame = page.frameLocator("#detail-frame");
  // The static marker proves the document's HTML rendered in the iframe.
  await expect(frame.locator("#static-marker")).toHaveText("static");
  // The script marker proves the inline <script> executed — i.e. the
  // iframe is unsandboxed (a sandboxed iframe without allow-scripts would
  // block this script and the marker would never appear).
  await expect(frame.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });

  // 5. Back -> table restored with the row still present + searchable.
  await page.locator("#back-button").click();
  await expect(page.locator("#detail-view")).toBeHidden();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(2);
  await expect(page.locator("#documents-table tbody tr", { hasText: "Deploy Runbook" })).toBeVisible();

  // The row is still searchable after returning from the detail detour.
  await search.fill("deploy");
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Deploy Runbook");
});

test("detail view renders document HTML unsandboxed via search-isolated row", async ({ page }) => {
  // A second, narrower test proving the unsandboxed render is reachable
  // independent of the click path's exact row. It search-isolates its own
  // uniquely-named doc so it is robust to whatever test 1 left in the shared
  // store (the webServer runs once for the whole suite).
  await seed(page, DOC_UNSANDBOXED_PROOF);

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  // Isolate this test's doc via the real FTS5 search so cross-test rows do
  // not affect the click target.
  await page.locator("#search").fill("Unsandboxed Proof");
  const rows = tableRows(page);
  await expect(rows).toHaveCount(1);
  await expect(rows.nth(0)).toContainText("Unsandboxed Proof Doc");

  await rows.nth(0).click();
  await expect(page.locator("#detail-view")).toBeVisible();

  // R006: no sandbox attribute on the detail iframe.
  await expect(page.locator("#detail-frame")).not.toHaveAttribute("sandbox");

  const frame = page.frameLocator("#detail-frame");
  await expect(frame.locator("#static-marker")).toHaveText("static");
  // The inline <script> ran — unsandboxed rendering proven.
  await expect(frame.locator("#script-marker")).toHaveText("script-ran", { timeout: 10_000 });
});
