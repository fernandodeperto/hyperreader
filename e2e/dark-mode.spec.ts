// Playwright coverage for dark mode (small-feature workflow: dark-mode).
//
// Runs against the REAL `html-mcp serve` binary (see playwright.config.ts
// webServer), same as the rest of the suite. Each `test()` gets its own
// browser context by default, so localStorage starts empty per test —
// that isolation is what lets tests 1 and 2 assert OS-preference
// auto-detection without a stored override leaking between them.
//
// Covers:
//   1. no stored preference + OS light -> resolves to light, no click
//      needed (proves the FOUC-prevention script's fallback chain).
//   2. no stored preference + OS dark -> resolves to dark, no click
//      needed (proves prefers-color-scheme auto-detection).
//   3. clicking the toggle overrides + persists across reload, even when
//      that choice disagrees with the OS scheme (proves the manual
//      override is sticky and localStorage-backed, not session-only).
import { test, expect } from "@playwright/test";

test("auto-detects light OS preference with no stored preference", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await expect(page.locator("#theme-toggle")).toHaveAttribute("aria-pressed", "false");
});

test("auto-detects dark OS preference with no stored preference", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  // No click anywhere in this test: the inline head script must resolve
  // data-theme="dark" purely from prefers-color-scheme, before app.js even
  // runs, so there is no FOUC flash of a light shell first.
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("#theme-toggle")).toHaveAttribute("aria-pressed", "true");
});

test("toggle overrides OS preference and persists across reload", async ({ page }) => {
  // Start on an OS-light system with no stored preference.
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  // Manual override to dark.
  await page.locator("#theme-toggle").click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("#theme-toggle")).toHaveAttribute("aria-pressed", "true");

  const stored = await page.evaluate(() => window.localStorage.getItem("html-mcp-theme"));
  expect(stored).toBe("dark");

  // Reload on the SAME OS-light emulation: if the override weren't sticky,
  // the reload would fall back to the OS light preference. It must not.
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("#theme-toggle")).toHaveAttribute("aria-pressed", "true");

  // Toggling back to light also persists.
  await page.locator("#theme-toggle").click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});
