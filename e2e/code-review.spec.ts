// Playwright coverage for the code-review template: change block renders inside
// the fixed dark shell, isolated from the count-asserting specs via a unique
// timestamped slug. Live Prism syntax highlighting (the CDN-dependent .token
// spans) is verified manually against the running reader, not asserted here —
// same rationale as dark-mode.spec.ts's own template fixture.
import { readFileSync } from "node:fs";
import path from "node:path";
import { test, expect } from "@playwright/test";

test("a page composed from the code-review template renders dark in the reader", async ({ page }) => {
  const slug = `code-review-dark-${Date.now()}`;
  const templatePath = path.join(__dirname, "..", "skills", "hyperreader", "assets", "code-review-template.html");
  const values: Record<string, string> = {
    TITLE: "Review: sample change",
    SUBTITLE: "jdoe · feature → main",
    CONTENT:
      '<section class="reveal mb-8 border border-line rounded-lg p-5 bg-surface">' +
      '<div class="flex flex-wrap items-center gap-3 mb-3">' +
      '<span class="inline-flex items-center text-xs font-semibold uppercase tracking-wide px-2.5 py-1 rounded border border-accent text-accent">Request changes</span>' +
      '<span class="text-muted text-sm font-mono">1 file · +2 −1</span></div>' +
      '<p class="text-ink leading-relaxed">One-paragraph overall assessment of the change.</p></section>' +
      '<details id="change-1" class="reveal scroll-mt-4 mb-8 border border-line rounded-lg overflow-hidden bg-surface" open>' +
      '<summary class="flex flex-wrap items-center justify-between gap-2 px-4 py-3 bg-surface-alt cursor-pointer">' +
      '<span class="flex items-center gap-3 min-w-0">' +
      '<svg class="change-caret w-3 h-3 shrink-0 text-muted" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M4 2l4 4-4 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
      '<span class="inline-flex items-center justify-center shrink-0 w-6 h-6 rounded border border-line text-muted font-mono text-xs">1</span>' +
      '<code class="font-mono text-sm text-ink truncate">internal/api/handlers.go</code></span>' +
      '<span class="font-mono text-xs text-muted shrink-0">+2 −1</span></summary>' +
      '<div class="border-t border-line overflow-x-auto"><pre><code class="language-diff-go diff-highlight">@@ -10,6 +10,7 @@ func handle() {\n' +
      ' ctx := r.Context()\n' +
      '-    return doThing(ctx)\n' +
      '+    v, err := doThing(ctx)\n' +
      '+    if err != nil { return err }\n' +
      ' }</code></pre></div>' +
      '<div class="px-4 py-4 border-t border-line">' +
      '<div class="flex items-center gap-2 mb-2">' +
      '<span class="inline-flex items-center text-xs font-semibold uppercase tracking-wide px-2 py-0.5 rounded border border-accent text-accent">Blocking</span>' +
      '<h3 class="text-sm font-bold text-muted uppercase tracking-wide">Analysis</h3></div>' +
      '<p class="text-ink leading-relaxed">The reviewer agent\'s analysis of this specific change.</p></div></details>',
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
  await expect(frame.locator("#change-1")).toBeVisible();
  await expect(frame.locator("#change-1")).toHaveJSProperty("open", true);
  await expect(frame.locator("#change-1 summary").getByText("1", { exact: true })).toBeVisible();

  await frame.locator("#change-1 summary").click();
  await expect(frame.locator("#change-1")).toHaveJSProperty("open", false);
  await expect(frame.locator("#change-1 pre")).toBeHidden();

  // Dark base comes from the template's own inline <style>, so this holds even
  // if the Tailwind/Prism CDNs are unreachable in CI.
  await expect(frame.locator("html")).toHaveCSS("background-color", "rgb(22, 24, 29)");
  await expect(page.locator("#theme-toggle")).toHaveCount(0);
});
