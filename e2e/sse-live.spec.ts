// Browser proof that SSE keeps the hidden table current while a stored page
// remains open in the same-tab reader.
import { test, expect } from "@playwright/test";

test("connection states expose accessible icon status", async ({ page }) => {
  await page.route("**/api/events", () => new Promise<void>(() => {}));
  await page.goto("/");

  const status = page.locator("#live-status");
  await expect(status).toHaveAttribute("data-state", "connecting");
  await expect(status).toHaveAccessibleName("Connecting");
  await expect(status).toHaveCSS("background-color", "rgb(242, 135, 154)");
});

test("connection errors expose the reconnecting icon status", async ({ page }) => {
  await page.route("**/api/events", (route) => route.abort("failed"));
  await page.goto("/");

  const status = page.locator("#live-status");
  await expect(status).toHaveAttribute("data-state", "reconnecting");
  await expect(status).toHaveAccessibleName("Reconnecting");
  await expect(status).toHaveCSS("background-color", "rgb(242, 135, 154)");
});

test("live table updates survive a same-tab page detour", async ({ page }) => {
  const timestamp = Date.now();
  const openSlug = `sse-open-page-${timestamp}`;
  const createdSlug = `sse-created-page-${timestamp}`;
  const initialName = `SSE Open Page ${timestamp}`;
  const updatedName = `SSE Updated Page ${timestamp}`;
  const createdName = `SSE Created Page ${timestamp}`;
  const storedHTML =
    "<!DOCTYPE html><html><body><p id='static-marker'>static</p>" +
    "<script>document.body.dataset.scriptRan='yes'</script></body></html>";

  const seedResponse = await page.request.post("/api/pages", {
    data: {
      slug: openSlug,
      name: initialName,
      description: "opened before live updates",
      html: storedHTML,
    },
  });
  expect(seedResponse.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await expect(page.locator("#live-status")).toHaveAttribute("data-state", "live", {
    timeout: 10_000,
  });

  const liveStatus = page.locator("#live-status");
  await expect(liveStatus).toHaveAccessibleName("Live");
  await expect(liveStatus).toHaveCSS("background-color", "rgb(95, 212, 137)");

  const rows = page.locator("#pages-table tbody tr");
  const initialCount = await rows.count();
  await page.evaluate(() => {
    Reflect.set(window, "__sseLiveSentinel", "still-here");
  });

  await page.locator("#search").fill(initialName);
  await expect(rows).toHaveCount(1);

  const fullListResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "GET" &&
      new URL(response.url()).pathname === "/api/pages" &&
      new URL(response.url()).search === "",
  );
  await page.locator("#search").fill("");
  await fullListResponse;
  await page.locator(`#pages-table tbody tr[data-slug="${openSlug}"]`).click();

  await expect(page.locator("#page-view")).toBeVisible();
  await expect(page.frameLocator("#page-frame").locator("#static-marker")).toHaveText("static");
  await expect(page.frameLocator("#page-frame").locator("body")).toHaveAttribute(
    "data-script-ran",
    "yes",
  );
  expect(page.context().pages()).toHaveLength(1);
  const updateResponse = await page.request.post("/api/pages", {
    data: {
      slug: openSlug,
      name: updatedName,
      description: "updated while its old document stayed open",
      html: storedHTML,
    },
  });
  expect(updateResponse.ok()).toBeTruthy();

  const createResponse = await page.request.post("/api/pages", {
    data: {
      slug: createdSlug,
      name: createdName,
      description: "created while another page was open",
      html: "<!DOCTYPE html><html><body>created</body></html>",
    },
  });
  expect(createResponse.ok()).toBeTruthy();

  await expect(page.locator("#page-view")).toBeVisible();
  await expect(page.locator("#table-view")).toBeHidden();
  await expect(page.frameLocator("#page-frame").locator("#static-marker")).toHaveText("static");

  await page.locator("#home-link").click();
  await expect(page.locator("#table-view")).toBeVisible();
  await expect(rows).toHaveCount(initialCount + 1, { timeout: 10_000 });
  await expect(rows.first()).toContainText(createdName);
  await expect(page.locator("#pages-table tbody tr", { hasText: updatedName })).toBeVisible();

  const sentinel = await page.evaluate(() => Reflect.get(window, "__sseLiveSentinel"));
  expect(sentinel).toBe("still-here");
  expect(page.context().pages()).toHaveLength(1);
});
