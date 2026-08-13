## 1. Update the reader contract tests

- [x] 1.1 Replace the new-tab and no-iframe assertions in `web/web_test.go` with checks for the home link, page view, unsandboxed iframe, and same-tab client wiring
- [x] 1.2 Rewrite the popup flows in `e2e/smoke.spec.ts` to cover click and keyboard activation, inline script execution, one browser tab, home-link return, and preserved search state
- [x] 1.3 Rewrite the page flows in `e2e/sse-live.spec.ts` and `e2e/two-process-acceptance.spec.ts` to use the same-tab reader and verify live table updates survive the page detour
- [x] 1.4 Extend `e2e/fluid-layout.spec.ts` with a tall page that proves parent overflow is absent, iframe scrolling advances, and the top-bar rectangle stays fixed at narrow and wide widths

## 2. Add the page-view shell

- [x] 2.1 Update `web/index.html` with a semantic HyperReader home link, a sibling page-view section, and an unsandboxed iframe that has an accessible title
- [x] 2.2 Move the table error surface into the table view so hidden search errors cannot change page-view geometry
- [x] 2.3 Update `web/app.css` with home-link styles and page-view rules across `html`, `body`, `#app`, the top bar, the page-view section, and the iframe
- [x] 2.4 Keep table mode gutters and normal document scrolling unchanged while page mode uses a full-width `minmax(0, 1fr)` viewport below the top bar

## 3. Add explicit reader state

- [x] 3.1 Extend the client state in `web/app.js` with the selected view and selected slug
- [x] 3.2 Split table-data rendering from top-level view rendering so search, loading, errors, and SSE updates cannot replace an active page view
- [x] 3.3 Replace `window.open` with iframe navigation to `GET /api/pages/{slug}/content` for click, Enter, and Space activation
- [x] 3.4 Wire the HyperReader home link to restore the current table state, clear the selected slug, and reset the iframe document
- [x] 3.5 Update the web asset comments to describe the same-tab reader, fixed top bar, trusted iframe, and single scroll container

## 4. Verify the complete behavior

- [x] 4.1 Run `go test ./web` and confirm the embedded asset contract passes
- [x] 4.2 Run the changed Playwright files and confirm same-tab reading, script execution, live updates, home return, and scroll ownership pass
- [x] 4.3 Run `go test ./...` and `npm run test:e2e` to confirm the full Go and browser suites pass
- [x] 4.4 Open a tall stored page in the real server and confirm the top bar stays fixed while the stored page owns the only scrollbar