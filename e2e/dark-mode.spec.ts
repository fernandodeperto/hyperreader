// Playwright coverage for HyperReader's fixed dark theme: no light mode, no toggle.
import { readFileSync } from "node:fs";
import path from "node:path";
import { test, expect } from "@playwright/test";

test("renders a fixed dark shell with no theme control", async ({ page }) => {
  await page.goto("/");

  await expect(page.locator("body")).toHaveCSS("background-color", "rgb(22, 24, 29)");
  await expect(page.locator("#theme-toggle")).toHaveCount(0);
});

test("a page composed from the real template renders dark in the reader", async ({ page }) => {
  const slug = `template-dark-${Date.now()}`;
  const templatePath = path.join(__dirname, "..", "skills", "hyperreader", "assets", "template.html");
  const values: Record<string, string> = {
    TITLE: "Template dark fixture",
    SUBTITLE: "proves the template renders dark",
    CONTENT:
      '<section id="s" class="mb-10"><h2 class="text-2xl font-bold text-ink mb-4">Section</h2>' +
      '<div id="card" class="border border-line rounded-lg p-5 bg-surface"><p class="text-ink leading-relaxed">body</p></div></section>',
  };
  const html = readFileSync(templatePath, "utf8").replace(/\{\{(\w+)\}\}/g, (_, k) => values[k]);

  const seed = await page.request.post("/api/pages", {
    data: { slug, name: values.TITLE, description: values.SUBTITLE, html },
  });
  expect(seed.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill(slug);
  await page.locator(`#pages-table tbody tr[data-slug="${slug}"]`).click();

  const frame = page.frameLocator("#page-frame");
  await expect(frame.locator("#card")).toBeVisible();
  // Dark base comes from the template's own inline <style>, so this holds even
  // if the Tailwind CDN is unreachable in CI.
  await expect(frame.locator("html")).toHaveCSS("background-color", "rgb(22, 24, 29)");
  await expect(page.locator("#theme-toggle")).toHaveCount(0);
});
