import { test, expect } from "@playwright/test";

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

  await expect(page.locator("header #search")).toBeVisible();
  await expect(page.locator("header #live-status")).toBeVisible();
});

test("keeps narrow shell gutters and state surfaces within the viewport", async ({ page }) => {
  await page.setViewportSize(NARROW_VIEWPORT);
  await page.route("**/api/pages", (route) =>
    route.fulfill({ status: 500, body: "layout test failure" }),
  );

  await page.goto("/");
  await expect(page.locator("#empty-state")).toBeVisible();
  await expect(page.locator("#error-message")).toBeVisible();

  const geometry = await page.locator("#app").evaluate((shell) => {
    const rect = (selector: string) => {
      const element = shell.querySelector<HTMLElement>(selector);
      if (!element) {
        throw new Error(`Missing ${selector}`);
      }
      return element.getBoundingClientRect();
    };
    const shellRect = shell.getBoundingClientRect();

    return {
      viewportWidth: window.innerWidth,
      pageWidth: document.documentElement.scrollWidth,
      shell: { left: shellRect.left, right: shellRect.right },
      header: rect("header"),
      empty: rect("#empty-state"),
      error: rect("#error-message"),
      liveStatus: rect("#live-status"),
    };
  });

  expect(geometry.pageWidth).toBeLessThanOrEqual(geometry.viewportWidth);
  expect(geometry.shell.left).toBeGreaterThanOrEqual(-1);
  expect(geometry.shell.right).toBeGreaterThanOrEqual(geometry.viewportWidth - 1);
  for (const surface of [geometry.header, geometry.empty, geometry.error]) {
    expect(surface.left).toBeGreaterThanOrEqual(23);
    expect(surface.right).toBeLessThanOrEqual(geometry.viewportWidth - 23);
  }

  expect(geometry.liveStatus.width).toBeLessThan(200);
});
