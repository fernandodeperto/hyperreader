import { test, expect, type Page } from "@playwright/test";

const WIDE_VIEWPORT = { width: 1440, height: 900 };
const NARROW_VIEWPORT = { width: 320, height: 720 };

test("uses the available shell width for page management at wide viewports", async ({ page }) => {
  await page.setViewportSize(WIDE_VIEWPORT);
  await page.route("**/api/pages", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          slug: "fluid-layout-proof",
          name: "Fluid layout proof",
          description: "Makes the pages table visible for geometry checks.",
        },
      ]),
    }),
  );

  await page.goto("/");
  await expect(page.locator("#pages-table")).toBeVisible();

  const geometry = await page.locator("#app").evaluate((shell) => {
    const width = (selector: string) =>
      shell.querySelector<HTMLElement>(selector)?.getBoundingClientRect().width ?? 0;
    return {
      header: width("header"),
      table: width("#pages-table"),
      liveStatus: width("#live-status"),
    };
  });

  expect(geometry.table).toBeGreaterThan(1200);
  expect(geometry.header).toBeCloseTo(geometry.table, 0);
  expect(geometry.liveStatus).toBeLessThan(200);
});

test("keeps narrow document-list controls within the viewport", async ({ page }) => {
  await page.setViewportSize(NARROW_VIEWPORT);
  await page.route("**/api/pages", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          slug: "narrow-layout-proof",
          name: "Narrow layout proof",
          description: "Keeps top-bar controls operable.",
        },
      ]),
    }),
  );

  await page.goto("/");
  await expect(page.locator("#pages-table")).toBeVisible();
  await expect(page.locator("#search")).toBeVisible();
  await expect(page.locator("#live-status")).toBeVisible();
  await expect(page.locator("#theme-toggle")).toBeVisible();

  const geometry = await page.locator("#app").evaluate((shell) => {
    const rect = (selector: string) => {
      const element = shell.querySelector<HTMLElement>(selector);
      if (!element) throw new Error(`Missing ${selector}`);
      return element.getBoundingClientRect();
    };
    return {
      viewportWidth: window.innerWidth,
      pageWidth: document.documentElement.scrollWidth,
      shell: shell.getBoundingClientRect(),
      header: rect("header"),
      search: rect("#search"),
      liveStatus: rect("#live-status"),
      themeToggle: rect("#theme-toggle"),
    };
  });

  expect(geometry.pageWidth).toBeLessThanOrEqual(geometry.viewportWidth);
  expect(geometry.shell.left).toBeGreaterThanOrEqual(-1);
  expect(geometry.shell.right).toBeGreaterThanOrEqual(geometry.viewportWidth - 1);
  for (const surface of [
    geometry.header,
    geometry.search,
    geometry.liveStatus,
    geometry.themeToggle,
  ]) {
    expect(surface.left).toBeGreaterThanOrEqual(0);
    expect(surface.right).toBeLessThanOrEqual(geometry.viewportWidth);
  }
});

async function verifyPageScrollOwnership(page: Page): Promise<void> {
  const slug = `tall-layout-${Date.now()}`;
  const response = await page.request.post("/api/pages", {
    data: {
      slug,
      name: `Tall layout ${slug}`,
      description: "proves iframe scroll ownership",
      html:
        "<!DOCTYPE html><html><head><style>html,body{margin:0}#tall{height:2400px}</style></head>" +
        "<body><div id='tall'>tall stored page</div></body></html>",
    },
  });
  expect(response.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill(slug);
  await expect(page.locator("#pages-table tbody tr")).toHaveCount(1);
  await page.locator("#pages-table tbody tr").click();
  await expect(page.locator("#selected-slug")).toHaveText(slug);
  await expect(page.locator("#live-status")).toBeVisible();
  await expect(page.locator("#theme-toggle")).toBeVisible();
  await expect(page.frameLocator("#page-frame").locator("#tall")).toBeVisible();

  const before = await page.evaluate(() => {
    const header = document.querySelector("header")?.getBoundingClientRect();
    const pageView = document.querySelector("#page-view")?.getBoundingClientRect();
    if (!header || !pageView) throw new Error("Missing page-view geometry");
    return {
      viewport: { width: innerWidth, height: innerHeight },
      root: {
        scrollTop: document.documentElement.scrollTop,
        scrollHeight: document.documentElement.scrollHeight,
        clientHeight: document.documentElement.clientHeight,
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth,
      },
      body: { scrollWidth: document.body.scrollWidth, clientWidth: document.body.clientWidth },
      app: document.querySelector("#app")!.getBoundingClientRect(),
      header,
      pageView,
    };
  });

  const frame = page.frameLocator("#page-frame");
  const frameBefore = await frame.locator("html").evaluate((root) => ({
    scrollTop: root.scrollTop,
    scrollHeight: root.scrollHeight,
    clientHeight: root.clientHeight,
  }));
  await frame.locator("html").evaluate((root) => root.scrollTo(0, 600));
  await expect.poll(() => frame.locator("html").evaluate((root) => root.scrollTop)).toBeGreaterThan(0);

  const after = await page.evaluate(() => ({
    rootScrollTop: document.documentElement.scrollTop,
    header: document.querySelector("header")!.getBoundingClientRect(),
  }));

  expect(before.root.scrollHeight).toBeLessThanOrEqual(before.root.clientHeight);
  expect(before.root.scrollWidth).toBeLessThanOrEqual(before.root.clientWidth);
  expect(before.body.scrollWidth).toBeLessThanOrEqual(before.body.clientWidth);
  expect(before.app.left).toBeGreaterThanOrEqual(-1);
  expect(before.app.right).toBeLessThanOrEqual(before.viewport.width + 1);
  expect(before.pageView.left).toBeCloseTo(0, 0);
  expect(before.pageView.right).toBeCloseTo(before.viewport.width, 0);
  expect(before.pageView.top).toBeCloseTo(before.header.bottom, 0);
  expect(before.pageView.bottom).toBeCloseTo(before.viewport.height, 0);
  expect(frameBefore.scrollHeight).toBeGreaterThan(frameBefore.clientHeight);
  expect(after.rootScrollTop).toBe(before.root.scrollTop);
  expect(after.header.top).toBeCloseTo(before.header.top, 1);
  expect(after.header.bottom).toBeCloseTo(before.header.bottom, 1);
  expect(page.context().pages()).toHaveLength(1);
}

for (const viewport of [NARROW_VIEWPORT, WIDE_VIEWPORT]) {
  test(`stored page owns scrolling at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await verifyPageScrollOwnership(page);
  });
}
