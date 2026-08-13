## Context

The current shell is two stacked rows: `<header>` (title + `#theme-toggle`) and `.toolbar` inside `#table-view` (`#search` + `#live-status`). Theme resolution runs in three places: a blocking `<script>` in `<head>` (reads `localStorage["hyperreader-theme"]`, falls back to `prefers-color-scheme`, falls back to light, sets `data-theme` before paint), `app.js`'s dark-mode module (toggle handling, OS-change listener, persistence), and `app.css` (`:root` light palette + `[data-theme="dark"]` override block). See proposal.md - Why/Impact for the full file list.

`hyperreader-theme` is also read by the separate `generate-html` report template (`skills/generate-html/assets/template.html`) to decide a generated report's initial theme, and reports carry their own independent toggle that writes the same key. That report-side theme system is a different capability (`html-report-skill`) with its own light and dark palettes; this change does not touch it.

## Goals / Non-Goals

**Goals:**
- One DOM element (`<header>`) contains the title, search, and live indicator; `.toolbar` is removed.
- Exactly one color palette ships; no `data-theme` branching remains in CSS or JS.
- A visitor with a stored `"light"` preference from before this change sees the same dark UI as everyone else, with no error and no residual toggle.

**Non-Goals:**
- Changing the `generate-html` report template's own theme toggle or its light/dark palettes (`html-report-skill` capability, untouched).
- Changing search behavior, the live-update (SSE) mechanism, or the pages-table rendering - only their container and surrounding chrome move.

## Decisions

**Keep writing `hyperreader-theme = "dark"` once, rather than deleting the key.** The report template reads this key to pick a report's initial theme and still supports both palettes. Deleting the key outright means a report opened by a visitor who never previously set a preference falls back to `prefers-color-scheme` inside the report - drifting from the app, which is now always dark. Writing `"dark"` once (on `init()`, unconditionally) keeps the two in sync without adding a second app-side theme system. Alternative considered: leave any pre-existing stored value alone. Rejected - a visitor with a stored `"light"` value would then open reports in light while the app itself renders dark, reintroducing the mismatch this change removes from the app.

**Delete the `<head>` FOUC-prevention script rather than reduce it to a one-line `data-theme="dark"` set.** With one palette, `app.css` needs no `[data-theme]` selector at all - the dark values move directly into `:root`. A script that only ever writes one constant value is dead code the moment the CSS stops branching on it.

**Merge `.toolbar`'s children into `<header>` in the DOM, not just visually.** `e2e/fluid-layout.spec.ts` measures `header` and `.toolbar` as independent boxes; keeping two elements only to move controls between them with CSS would leave a redundant empty `.toolbar` wrapper with no content. Removing the element matches `reader-top-bar`'s requirement that no separate search toolbar row exists.

**`app.js`'s dark-mode module is deleted wholesale, not disabled.** `getStoredTheme`, `storeTheme`, `currentTheme`, `applyTheme`, `setTheme`, `toggleTheme`, `watchSystemTheme`, and their `init()` wiring have no caller once `#theme-toggle` is gone; keeping them as unreachable code would be dead weight with no reader.

## Risks / Trade-offs

- **A visitor's browser still has a stale `hyperreader-theme = "light"` entry from before this change** → The write-once-dark decision above overwrites it on next load; no migration script needed since the key is small and self-healing on first visit.
- **`e2e/fluid-layout.spec.ts` and `e2e/dark-mode.spec.ts` assert the removed structure/behavior directly (`#theme-toggle`, `.toolbar` as a distinct box, light/dark OS-preference resolution)** → Both need rewriting as part of this change's tasks: `fluid-layout.spec.ts`'s geometry checks move to `header` as the sole container plus `#search`/`#live-status` inside it, and `dark-mode.spec.ts` is replaced with coverage that the app renders dark under every OS-preference emulation and exposes no toggle.
