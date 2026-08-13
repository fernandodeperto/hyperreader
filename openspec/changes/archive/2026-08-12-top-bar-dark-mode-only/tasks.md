## 1. Merge header and toolbar into one top bar

- [x] 1.1 In `web/index.html`, move `#search` (with its `<label>`) and `#live-status` out of `.toolbar` into `<header>`, after the `<h1>`
- [x] 1.2 Remove the now-empty `.toolbar` wrapper element and its comment from `web/index.html`
- [x] 1.3 Remove the `#theme-toggle` button element from `<header>` in `web/index.html`
- [x] 1.4 In `web/app.css`, fold `.toolbar`'s layout rules (flex, gap, margin-bottom) into `header` so title, search, and live-status lay out in one row; remove the standalone `.toolbar` rule
- [x] 1.5 Update the `header h1` / `#search` flex sizing in `web/app.css` so the search field still grows to fill available space beside the fixed-size title and live-status indicator

## 2. Remove light theme and the theme toggle

- [x] 2.1 Delete the `<head>` FOUC-prevention `<script>` block from `web/index.html`
- [x] 2.2 In `web/app.css`, delete the light-palette values from `:root` and replace them with the current `[data-theme="dark"]` values directly; delete the `[data-theme="dark"]` override block
- [x] 2.3 Delete the `#theme-toggle` and `#theme-toggle:hover`/`:focus` rules from `web/app.css`
- [x] 2.4 In `web/app.js`, delete `THEME_KEY`, `getStoredTheme`, `storeTheme`, `currentTheme`, `applyTheme`, `setTheme`, `toggleTheme`, `watchSystemTheme`, and the "Dark mode (S05)" comment block
- [x] 2.5 In `web/app.js`'s `init()`, remove the `applyTheme(currentTheme())`, `#theme-toggle` lookup/listener, and `watchSystemTheme()` calls; add one unconditional write of `hyperreader-theme = "dark"` to `localStorage` (wrapped in try/catch, matching the removed module's error handling) so `generate-html` reports opened from this reader stay in sync with the app's single theme
- [x] 2.6 Grep `web/index.html`, `web/app.js`, and `web/app.css` for any remaining `data-theme`, `theme-toggle`, or light-palette reference and confirm none is left

## 3. Update tests

- [x] 3.1 Rewrite `e2e/dark-mode.spec.ts` to assert: the app renders `data-theme="dark"` regardless of `page.emulateMedia({ colorScheme: "light" | "dark" })`, no reload-persisted preference changes that, and `#theme-toggle` does not exist in the DOM
- [x] 3.2 In `e2e/fluid-layout.spec.ts`'s wide-viewport test, remove the `toolbar` and `themeToggle` geometry keys; assert `header` width is close to `table` width, and that `#search` and `#live-status` are both visible inside `header`
- [x] 3.3 In `e2e/fluid-layout.spec.ts`'s narrow-viewport test, remove the `toolbar` and `themeToggle` geometry keys; assert `header`'s rect stays within the same gutters previously asserted for `header`/`toolbar`, and that `#live-status` width stays under 200px
- [x] 3.4 Run the full Playwright suite (`npm test` / configured e2e command) and confirm every spec passes, including `e2e/smoke.spec.ts`, `e2e/sse-live.spec.ts`, and `e2e/two-process-acceptance.spec.ts`, none of which reference `.toolbar` or `#theme-toggle` but all of which locate `#search`/`#live-status` by id

## 4. Verify end to end

- [x] 4.1 Run `hyperreader serve` locally and open the root page: confirm the top bar shows title, search, and live indicator in one row, and no theme toggle is present
- [x] 4.2 With the OS set to light and to dark (via browser devtools emulation), confirm the page renders dark both times with no flash of a different palette
- [x] 4.3 Set `localStorage["hyperreader-theme"] = "light"` manually, reload, and confirm the app still renders dark and the key is now `"dark"`
- [x] 4.4 Resize to a narrow viewport and confirm search and live-status remain visible and operable with no horizontal overflow
