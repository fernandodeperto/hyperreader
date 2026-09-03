// Browser smoke for the HyperReader primary user loop.
//
// The tests seed the real API, exercise FTS5 search, open stored HTML in the
// same-tab iframe, and return through the HyperReader home link. Inline script
// markers prove that the trusted iframe remains unsandboxed.
import { test, expect, type Page } from "@playwright/test";

type StoredPage = {
  slug: string;
  name: string;
  description: string;
  html: string;
};

const SCRIPT_HTML =
  "<!DOCTYPE html><html><body><p id='static-marker'>static</p>" +
  "<script>document.body.insertAdjacentHTML('beforeend'," +
  "'<p id=\"script-marker\">script-ran</p>');</script>" +
  "</body></html>";

async function seed(page: Page, doc: StoredPage): Promise<void> {
  const response = await page.request.post("/api/pages", { data: doc });
  expect(response.ok()).toBeTruthy();
}


async function expectStoredPage(page: Page): Promise<void> {
  await expect(page.locator("#page-view")).toBeVisible();
  await expect(page.locator("#table-view")).toBeHidden();
  await expect(page.locator("html")).toHaveAttribute("data-view", "page");
  await expect(page.frameLocator("#page-frame").locator("#static-marker")).toHaveText("static");
  await expect(page.frameLocator("#page-frame").locator("#script-marker")).toHaveText(
    "script-ran",
  );
  expect(page.context().pages()).toHaveLength(1);
}

test("click opens a trusted same-tab page and home restores filtered table state", async ({
  page,
}) => {
  await seed(page, {
    slug: "deploy-runbook",
    name: "Deploy Runbook",
    description: "production deploy steps",
    html: SCRIPT_HTML,
  });
  await seed(page, {
    slug: "on-call-guide",
    name: "On-call Guide",
    description: "rotation schedule",
    html: "<!DOCTYPE html><html><body><h1>On-call</h1></body></html>",
  });

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();

  const search = page.locator("#search");
  const selectedSlug = page.locator("#selected-slug");
  const rows = page.locator("#pages-table tbody tr");
  await expect(search).toBeVisible();
  await expect(selectedSlug).toBeHidden();
  await search.fill("deploy");
  await expect(rows).toHaveCount(1);
  await rows.first().click();

  await expect(page.locator("#page-frame")).toHaveAttribute(
    "src",
    "/api/pages/deploy-runbook/content",
  );
  await expectStoredPage(page);
  await expect(search).toBeHidden();
  await expect(selectedSlug).toHaveText("deploy-runbook");
  await expect(selectedSlug).toHaveAttribute("aria-label", "deploy-runbook");

  await page.locator("#home-link").click();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(page.locator("#page-view")).toBeHidden();
  await expect(page.locator("html")).toHaveAttribute("data-view", "table");
  await expect(page.locator("#page-frame")).toHaveAttribute("src", "about:blank");
  await expect(search).toBeVisible();
  await expect(selectedSlug).toBeHidden();
  await expect(search).toHaveValue("deploy");
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("Deploy Runbook");
  expect(page.context().pages()).toHaveLength(1);
});

for (const activation of [
  { name: "Enter", key: "Enter", slug: "keyboard-enter-page" },
  { name: "Space", key: " ", slug: "keyboard-space-page" },
]) {
  test(`${activation.name} opens a stored page in the current tab`, async ({ page }) => {
    const name = `Keyboard ${activation.name} Page`;
    await seed(page, {
      slug: activation.slug,
      name,
      description: `opened with ${activation.name}`,
      html: SCRIPT_HTML,
    });

    await page.goto("/");
    await expect(page.locator("#loading-state")).toBeHidden();
    await page.locator("#search").fill(name);

    const row = page.locator("#pages-table tbody tr").first();
    await expect(row).toContainText(name);
    await row.focus();
    await row.press(activation.key);

    await expect(page.locator("#page-frame")).toHaveAttribute(
      "src",
      `/api/pages/${activation.slug}/content`,
    );
    await expectStoredPage(page);
  });
}

// Older or externally-authored stored documents can carry their own
// unguarded floating theme button (the pre-fix generate-html report
// template shape: no host-iframe check, so it stays visible everywhere).
// HyperReader's top bar now owns that control, so the reader must strip
// this one from every stored document it loads, regardless of when or how
// that document was authored.
const LEGACY_REPORT_THEME_BUTTON_HTML =
  "<!DOCTYPE html><html><body>" +
  '<button class="theme" type="button" aria-label="Switch to dark theme" aria-pressed="false">\u25D0</button>' +
  "<p id=\"content-marker\">report body</p>" +
  "</body></html>";

test("removes a stored document's own embedded theme button", async ({ page }) => {
  await seed(page, {
    slug: "legacy-report-with-theme-button",
    name: "Legacy Report",
    description: "predates the reader-owned theme control",
    html: LEGACY_REPORT_THEME_BUTTON_HTML,
  });

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill("Legacy Report");
  await page.locator("#pages-table tbody tr").first().click();

  const frame = page.frameLocator("#page-frame");
  await expect(frame.locator("#content-marker")).toHaveText("report body");
  await expect(frame.locator(".theme")).toHaveCount(0);
});

test("reader URL carries the slug and survives a full reload", async ({ page }) => {
  await seed(page, {
    slug: "reload-runbook",
    name: "Reload Runbook",
    description: "reload target",
    html: SCRIPT_HTML,
  });

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill("Reload Runbook");
  await page.locator("#pages-table tbody tr").first().click();

  await expect(page).toHaveURL(/\/read\/reload-runbook$/);
  await expectStoredPage(page);
  await expect(page.locator("#selected-slug")).toHaveText("reload-runbook");

  // Full reload must restore the reader view, not fall back to the table.
  await page.reload();
  await expect(page).toHaveURL(/\/read\/reload-runbook$/);
  await expectStoredPage(page);
  await expect(page.locator("#selected-slug")).toHaveText("reload-runbook");

  // Home returns to the table and resets the URL to "/".
  await page.locator("#home-link").click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(page.locator("#page-view")).toBeHidden();
});
