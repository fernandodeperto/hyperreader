// Playwright coverage for the explicit light and dark theme control.
import { test, expect } from "@playwright/test";

test("starts dark with a control that switches to light", async ({ page }) => {
  await page.goto("/");

  const root = page.locator("html");
  const toggle = page.locator("#theme-toggle");
  await expect(root).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("body")).toHaveCSS("background-color", "rgb(22, 24, 29)");
  await expect(toggle).toHaveText("☀");
  await expect(toggle).toHaveAccessibleName("Switch to light theme");
  await expect(page.locator("header #live-status + #theme-toggle")).toHaveCount(1);
});

test("switches between light and dark themes with matching icon and name", async ({ page }) => {
  await page.goto("/");

  const root = page.locator("html");
  const toggle = page.locator("#theme-toggle");
  await toggle.click();
  await expect(root).toHaveAttribute("data-theme", "light");
  await expect(page.locator("body")).toHaveCSS("background-color", "rgb(245, 247, 250)");
  await expect(toggle).toHaveText("☾");
  await expect(toggle).toHaveAccessibleName("Switch to dark theme");

  await toggle.click();
  await expect(root).toHaveAttribute("data-theme", "dark");
  await expect(toggle).toHaveText("☀");
  await expect(toggle).toHaveAccessibleName("Switch to light theme");
});

test("propagates theme changes into a stored document's iframe", async ({ page }) => {
  const slug = `theme-sync-${Date.now()}`;
  // Mirrors the generate-html report template's own pattern: palette
  // tokens on :root, overridden by html[data-theme="light"], consumed by
  // body's background. A real visual change, not just an attribute flip,
  // proves the shell's theme reaches stored content.
  const html =
    "<!DOCTYPE html><html><head><style>" +
    ":root { --bg: #101010; } " +
    'html[data-theme="light"] { --bg: #ffffff; } ' +
    "body { margin: 0; background: var(--bg); }" +
    "</style></head><body><p id='marker'>content</p></body></html>";
  const seedResponse = await page.request.post("/api/pages", {
    data: {
      slug,
      name: "Theme sync fixture",
      description: "proves the shell theme reaches the trusted iframe",
      html,
    },
  });
  expect(seedResponse.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill(slug);
  await page.locator(`#pages-table tbody tr[data-slug="${slug}"]`).click();
  await expect(page.frameLocator("#page-frame").locator("#marker")).toBeVisible();

  const frameHtml = page.frameLocator("#page-frame").locator("html");
  const frameBody = page.frameLocator("#page-frame").locator("body");
  await expect(frameHtml).toHaveAttribute("data-theme", "dark");
  await expect(frameBody).toHaveCSS("background-color", "rgb(16, 16, 16)");

  const toggle = page.locator("#theme-toggle");
  await toggle.click();
  await expect(frameHtml).toHaveAttribute("data-theme", "light");
  await expect(frameBody).toHaveCSS("background-color", "rgb(255, 255, 255)");

  await toggle.click();
  await expect(frameHtml).toHaveAttribute("data-theme", "dark");
  await expect(frameBody).toHaveCSS("background-color", "rgb(16, 16, 16)");
});
