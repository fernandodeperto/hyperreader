## Why

HyperReader opens each page in a new browser tab and removes the reader controls from view. Users need to read a page in the current tab, keep the top bar visible, and return to the live table without extra scrollbars.

## What Changes

- **BREAKING**: Activating a page row opens the page inside the current HyperReader tab instead of a new browser tab.
- Add an in-app page view that renders the stored HTML as a complete document in an unsandboxed iframe.
- Keep the top bar fixed while the stored page scrolls below it.
- Make the stored page document the only scroll container in page view. The HyperReader shell does not scroll or add horizontal overflow.
- Turn the HyperReader title into a home link that restores the table and stops the open page.
- Keep the table state, search value, and SSE connection active while a page is open.
- Keep page view active when search results or SSE events update the hidden table.
- Keep the current root URL during the page detour. Browser history and direct reader URLs are not added.
- Keep `GET /api/pages/{slug}/content` as the raw HTML source and as a direct endpoint.

## Capabilities

### New Capabilities

- `in-app-page-reader`: Defines same-tab page activation, complete HTML rendering, one-scroll-container behavior, state preservation, and return to the table.

### Modified Capabilities

- `reader-top-bar`: The top bar remains fixed in page view, and the HyperReader title becomes the home link.
- `fluid-application-layout`: Page view fills the viewport below the top bar without parent overflow or a second scrollbar.

## Impact

- `web/index.html`: Add the page view and iframe. Change the title to a home link.
- `web/app.js`: Replace `window.open` with explicit table and page state. Preserve that state during search and SSE renders.
- `web/app.css`: Add the fixed page-view shell, full-width iframe, title-link styles, and overflow rules for all page-view ancestors.
- `web/web_test.go`: Replace the new-tab and no-iframe assertions with the in-app reader contract.
- `e2e/smoke.spec.ts`, `e2e/sse-live.spec.ts`, and `e2e/two-process-acceptance.spec.ts`: Replace popup flows with same-tab page and home-link flows.
- `e2e/fluid-layout.spec.ts`: Verify the fixed top bar and the single scroll container at narrow and wide viewport widths.
- The API and storage contracts do not change.