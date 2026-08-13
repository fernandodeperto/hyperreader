## Why

The header and the search toolbar currently split the reader's persistent controls across two stacked rows (`<header>` holds the title and theme toggle; a separate `.toolbar` below it holds search and the live-status indicator), and the app additionally carries a light theme, a toggle, and OS-preference detection that only add surface area to test and maintain. Consolidating the title, search, and live indicator into a single top bar removes a redundant row, and dropping light mode removes a whole axis of visual state (and its toggle, FOUC-prevention script, and light palette) that this reader does not need going forward.

## What Changes

- **BREAKING** Merge the `<header>` and `.toolbar` rows into one persistent top bar containing the page title, the search field, and the live-connection indicator.
- **BREAKING** Remove the theme toggle button, the light color palette, `prefers-color-scheme` detection, and the `<head>` FOUC-prevention script. The reader always renders in dark theme; there is no light mode and no user-facing theme control.
- Drop the `hyperreader-theme` localStorage read/write and the `data-theme` attribute resolution logic from `app.js` and `index.html`, since there is only one theme to apply.
- Update the existing narrow/wide viewport layout tests and the dark-mode test suite to match the new single top-bar, dark-only shell (`e2e/fluid-layout.spec.ts`, `e2e/dark-mode.spec.ts`).

## Capabilities

### New Capabilities
- `reader-top-bar`: the persistent top bar's composition, that it hosts the page title, the search field, and the live-connection indicator together, and that it stays usable at narrow viewport widths.
- `single-dark-theme`: the reader renders exclusively in dark theme, with no light palette, no theme toggle, and no OS-preference-driven theme switching.

### Modified Capabilities
- `fluid-application-layout`: the "Responsive controls remain usable" requirement currently names the theme toggle as a content-sized control that must remain usable at narrow widths; the theme toggle no longer exists, so this requirement is narrowed to the live-status indicator and the search field, and the "Available-width application shell" requirement's scenario is updated to describe the single top bar in place of a separate header and search toolbar.

## Impact

- `web/index.html`: remove the `<head>` FOUC-prevention script and the `#theme-toggle` button; move `#search` and `#live-status` (currently in `.toolbar` inside `#table-view`) into `<header>` alongside the `<h1>`; remove the now-empty `.toolbar` wrapper.
- `web/app.js`: remove the dark-mode module (`THEME_KEY`, `getStoredTheme`, `storeTheme`, `currentTheme`, `applyTheme`, `setTheme`, `toggleTheme`, `watchSystemTheme`) and its `init()` wiring.
- `web/app.css`: remove the light/dark `:root` split (keep a single dark palette), the `#theme-toggle` rules, and update `header`/`.toolbar` rules for the merged top bar.
- `e2e/dark-mode.spec.ts`: replaced by coverage that the app always renders dark and has no toggle.
- `e2e/fluid-layout.spec.ts`: update the geometry assertions that currently measure `header`, `.toolbar`, and `#theme-toggle` as separate elements.
- `openspec/specs/fluid-application-layout/spec.md`: delta above.
