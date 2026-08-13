## Why

The top bar gives equal space to controls that have different roles. The page view also keeps list-only search visible instead of showing page context.

## What Changes

- Move the search field to the right side of the top bar in the document list.
- Replace the text live-status badge with one status icon. Use green for a live connection and red for a disconnected connection.
- Keep a non-color status name for assistive technology.
- Move the light and dark theme control into the top bar, after the live-status icon.
- Hide the search field when a user opens a page from the table.
- Show the selected page slug in the search field's top-bar position while the page is open.
- Restore the search field when the user returns to the document list.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `reader-top-bar`: Change the order and form of the top-bar controls, and show either list search or the selected page slug.
- `single-dark-theme`: Replace the no-control rule with a light and dark theme control in the top bar.

## Impact

- The change affects the top-bar markup, styles, view-state rendering, live-status rendering, and theme control in `web/`.
- Browser tests for the top bar, page view, live status, narrow layouts, and theme behavior must change.
- The current `single-dark-theme` specification conflicts with the requested theme control. This change updates that requirement instead of preserving dark-only behavior.
