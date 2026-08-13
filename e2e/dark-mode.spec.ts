// Playwright coverage for the single dark theme (small-feature workflow:
// top-bar-dark-mode-only).
//
// Runs against the REAL `hyperreader serve` binary (see playwright.config.ts
// webServer), same as the rest of the suite. Each `test()` gets its own
// browser context by default, so localStorage starts empty per test.
//
// Covers:
//   1. the app renders data-theme="dark" under every OS color-scheme
//      preference (light, dark, no-preference), proving there is no
//      light palette and no preference-driven branching.
//   2. a stale "light" preference from before this change does not change
//      the rendered theme across a reload.
//   3. no theme toggle exists anywhere in the DOM.
import { test, expect } from "@playwright/test";

test("renders dark regardless of OS color-scheme preference", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("renders dark with no OS color-scheme preference set", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "no-preference" });
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});

test("a stale stored light preference does not change the rendered theme on reload", async ({
  page,
}) => {
  await page.goto("/");
  await page.evaluate(() => window.localStorage.setItem("hyperreader-theme", "light"));

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  const stored = await page.evaluate(() => window.localStorage.getItem("hyperreader-theme"));
  expect(stored).toBe("dark");
});

test("no theme control is presented anywhere in the app", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#theme-toggle")).toHaveCount(0);
  await expect(page.getByRole("button", { name: /theme|dark mode|light mode/i })).toHaveCount(0);
});
