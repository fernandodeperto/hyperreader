// Playwright coverage for the explicit light and dark theme control.
import { readFileSync } from "node:fs";
import path from "node:path";
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

test("a report composed from the real template renders dark in the dark shell", async ({ page }) => {
  const slug = `template-dark-${Date.now()}`;
  const templatePath = path.join(__dirname, "..", "skills", "generate-html", "assets", "template.html");
  const values: Record<string, string> = {
    KIND: "TEST", TITLE: "Template dark fixture", LEDE: "proves the template honors the shell theme",
    DATE: "2026-08-31", SCOPE: "test", SOURCES: "test",
    CONTENTS: '<li><a class="hover:text-action-hover" href="#s">Section</a></li>',
    CONTENT: '<section id="s" class="mb-12"><h2 class="text-2xl font-bold text-gray-900 mb-4">1. Section</h2>'
           + '<div class="border border-gray-200 rounded-lg p-5 bg-white" id="card"><p class="text-gray-700">body</p></div></section>',
    CAVEATS: "none", EXTERNAL: "",
  };
  const html = readFileSync(templatePath, "utf8").replace(/\{\{(\w+)\}\}/g, (_, k) => values[k]);

  const seed = await page.request.post("/api/pages", { data: { slug, name: values.TITLE, description: values.LEDE, html } });
  expect(seed.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator("#loading-state")).toBeHidden();
  await page.locator("#search").fill(slug);
  await page.locator(`#pages-table tbody tr[data-slug="${slug}"]`).click();

  const frame = page.frameLocator("#page-frame");
  await expect(frame.locator("#card")).toBeVisible();
  const frameHtml = frame.locator("html");
  await expect(frameHtml).toHaveAttribute("data-theme", "dark");
  // Dark palette comes from the template's own inline <style>, so this holds
  // even if the Tailwind CDN is unreachable in CI.
  await expect(frame.locator("body")).toHaveCSS("background-color", "rgb(22, 24, 29)");
  await expect(frame.locator("#card")).toHaveCSS("background-color", "rgb(31, 34, 41)");

  // Toggling the shell flips the report instantly (pure CSS, no reload).
  await page.locator("#theme-toggle").click();
  await expect(frameHtml).toHaveAttribute("data-theme", "light");
  await expect(frame.locator("body")).not.toHaveCSS("background-color", "rgb(22, 24, 29)");
});
