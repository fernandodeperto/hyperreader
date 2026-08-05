import { test, expect } from "@playwright/test";

const WIDE_VIEWPORT = { width: 1440, height: 900 };
const NARROW_VIEWPORT = { width: 320, height: 720 };

test("uses the available shell width for document management at wide viewports", async ({ page }) => {
  await page.setViewportSize(WIDE_VIEWPORT);

  await page.route("**/api/documents", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          id: 1,
          name: "Fluid layout proof",
          description: "Makes the documents table visible for geometry checks.",
          tags: "layout",
        },
      ]),
    }),
  );

  await page.goto("/");
  await expect(page.locator("#documents-table")).toBeVisible();

  const geometry = await page.locator("#app").evaluate((shell) => {
    const width = (selector: string) =>
      shell.querySelector<HTMLElement>(selector)?.getBoundingClientRect().width ?? 0;

    return {
      header: width("header"),
      toolbar: width(".toolbar"),
      table: width("#documents-table"),
      themeToggle: width("#theme-toggle"),
      liveStatus: width("#live-status"),
    };
  });

  expect(geometry.table).toBeGreaterThan(1200);
  expect(geometry.header).toBeCloseTo(geometry.table, 0);
  expect(geometry.toolbar).toBeCloseTo(geometry.table, 0);
  expect(geometry.themeToggle).toBeLessThan(200);
  expect(geometry.liveStatus).toBeLessThan(200);
});

test("keeps narrow shell gutters and state surfaces within the viewport", async ({ page }) => {
  await page.setViewportSize(NARROW_VIEWPORT);
  await page.route("**/api/documents", (route) =>
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
      toolbar: rect(".toolbar"),
      empty: rect("#empty-state"),
      error: rect("#error-message"),
      themeToggle: rect("#theme-toggle"),
      liveStatus: rect("#live-status"),
    };
  });

  expect(geometry.pageWidth).toBeLessThanOrEqual(geometry.viewportWidth);
  expect(geometry.shell.left).toBeGreaterThanOrEqual(-1);
  expect(geometry.shell.right).toBeGreaterThanOrEqual(geometry.viewportWidth - 1);
  for (const surface of [geometry.header, geometry.toolbar, geometry.empty, geometry.error]) {
    expect(surface.left).toBeGreaterThanOrEqual(23);
    expect(surface.right).toBeLessThanOrEqual(geometry.viewportWidth - 23);
  }
  expect(geometry.themeToggle.width).toBeLessThan(200);
  expect(geometry.liveStatus.width).toBeLessThan(200);
});
