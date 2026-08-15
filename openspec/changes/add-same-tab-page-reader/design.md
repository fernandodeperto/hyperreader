## Context

See `proposal.md` for the user problem. The current client renders metadata in a table and calls `window.open` for `GET /api/pages/{slug}/content`. The root document owns the top bar, search state, table state, error surface, and SSE connection. The raw content endpoint serves a complete HTML document with no sanitization or CSP.

HyperReader had an in-app iframe before commit `2f1b7bc`. That version fetched HTML into `srcdoc` and used content measurements plus a `ResizeObserver` to grow the iframe. The new-tab design removed that code. This change must not restore the auto-size system because it adds a second layout owner and makes the top bar scroll away.

The existing `render()` function updates table visibility for initial loads, searches, and SSE events. Page view must remain active during those renders. The selected view must therefore be client state, not a one-time DOM hide operation.

## Goals / Non-Goals

**Goals:**

- Keep the existing table, search value, and SSE connection alive during page view.
- Render stored HTML with the same complete-document behavior as the raw content endpoint.
- Give the stored page the only scroll container while the top bar stays fixed.
- Keep table mode layout and scrolling unchanged.
- Make the scroll contract measurable in a real browser.

**Non-Goals:**

- Do not add client routes, browser-history entries, or direct reader URLs.
- Do not auto-reload an open page after a `page-updated` event.
- Do not sandbox, sanitize, rewrite, or inject markup into stored HTML.
- Do not auto-size the page frame from its document height.
- Do not change the API, storage model, or raw content endpoint.

## Decisions

### Use an iframe whose `src` is the raw content endpoint

Add a hidden page-view section beside the table-view section. The page view owns one iframe with no `sandbox` attribute. Opening a page sets the iframe source to `/api/pages/{slug}/content` and changes the client view state to `page`.

Direct iframe navigation preserves document parsing, inline scripts, document-level styles, relative URLs, viewport units, and the endpoint URL as the base URL. It also avoids copying the HTML into the application process.

Alternatives considered:

- Direct same-tab navigation was rejected because it replaces the HyperReader shell and removes the top bar.
- `innerHTML` was rejected because it cannot embed a complete HTML document and does not preserve script execution.
- `srcdoc` was rejected because it requires a fetch path, changes URL and base-URL behavior, and repeats the removed design.
- Server-side chrome injection was rejected because stored page CSS and scripts could collide with the shell markup.

### Make view selection explicit client state

Extend the existing client state with `view` and `selectedSlug`. The render path has two responsibilities:

1. Render table data inside the table view.
2. Apply the top-level view from `state.view`.

Table renders may change loading, empty, error, table, and row surfaces. They must not select the table view. Search responses and SSE handlers continue to update table state while page view remains selected.

Opening a page sets the selected slug, selects page view, and sets the iframe source. Activating the HyperReader home link selects table view, clears the selected slug, and resets the iframe to an empty document. This reset stops scripts, media, timers, and resource requests from the prior page.

Alternative considered: Directly toggle `hidden` only in row and home click handlers. Rejected because later table renders can violate the page-view invariant and make behavior depend on call order.

### Lock all page-view ancestors to the viewport

Use a page-view marker on the root document. In page view, apply the fixed-height and overflow contract across `html`, `body`, `#app`, the page-view section, and the iframe.

The root document and body use the viewport height and `overflow: hidden`. The application shell uses a two-row grid:

```text
┌─ top bar: automatic height ───────────────┐
├─ page view: minmax(0, 1fr) ──────────────┤
│  iframe: width 100%, height 100%          │
└───────────────────────────────────────────┘
```

The `minmax(0, 1fr)` row and `min-height: 0` on all grid and frame ancestors are required. Without them, the iframe can expand the grid and create a parent scrollbar.

The application shell removes its normal page padding in page view. The top bar receives equivalent padding, while the page view spans the full viewport width. This places the stored page scrollbar at the viewport edge and avoids the appearance of a nested reader.

Table view keeps the current document flow, gutters, and browser scrolling.

Alternatives considered:

- A sticky header above an auto-growing iframe was rejected because the parent becomes the scroll owner and requires content-height observation.
- A fixed-height iframe inside the current padded shell was rejected because its scrollbar would be inset and the shell could still overflow.
- Applying overflow rules only to the iframe was rejected because the current padded ancestors can still create vertical or horizontal parent overflow.

### Keep the complete top bar active in page view

Wrap the `HyperReader` heading text in a semantic link to `/`. Intercept an unmodified in-app activation to restore table view without a reload. The `href` remains a fallback and supports normal link behavior when the client script does not run.

Keep search and live status active. A search during page view updates the hidden table. The global table error surface belongs inside the table view so a hidden search error cannot change page-view geometry.

Alternative considered: Hide or disable search in page view. Rejected because the existing top-bar specification defines title, search, and live status as one persistent control row.

### Preserve the current trusted-content model

Do not add a `sandbox` attribute. A same-origin iframe with scripts can access its parent, so the iframe is a document boundary and not a security boundary. This follows the current product rule that stored page code is trusted and runs unsandboxed in the HyperReader origin.

A useful sandbox would require a separate decision about scripts, local storage, navigation, downloads, and same-origin access. That decision is outside this change.

### Verify scroll ownership through browser geometry

Use a tall stored page in the browser suite. In page view, verify all of these facts:

- The parent scrolling element has no vertical or horizontal overflow.
- The parent scroll position stays unchanged after page scrolling.
- The iframe document has vertical overflow and its scroll position advances.
- The top-bar rectangle stays unchanged after iframe scrolling.
- The browser context still has one tab.
- The page view has no application-level horizontal overflow at narrow and wide widths.

This check proves behavior. Static CSS assertions do not prove scroll ownership.

## Risks / Trade-offs

- [Stored page code can access the HyperReader parent] → Keep the trusted-writer boundary explicit and do not claim iframe isolation.
- [A stored page can create its own nested or horizontal overflow] → Guarantee only that HyperReader adds no scroll container. Keep report-level overflow checks in the HTML report contract.
- [The iframe load event does not expose HTTP status] → Let the raw endpoint render its existing error response inside the page viewport. The fixed home link remains available.
- [An open page stays stale after a page update] → Update hidden table state only. Reload the page when the user returns and opens it again.
- [Viewport units vary on mobile browser chrome changes] → Use the dynamic viewport unit for page mode and verify the supported browser viewport sizes.
- [Resetting the iframe can run one empty-document load] → Treat that load as teardown and keep it outside application state transitions.

## Migration Plan

1. Ship the web assets and changed browser tests together.
2. Keep the API and stored page data unchanged.
3. Roll back by restoring the prior web assets. No data rollback is required.