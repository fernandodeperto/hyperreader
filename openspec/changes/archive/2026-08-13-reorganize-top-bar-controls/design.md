## Context

`web/index.html` defines one persistent header with the home link, search field, and live-status element. `web/app.js` keeps the selected slug and the current table or page view in client state. `web/app.css` supplies one dark token set and renders the live state as a text badge.

The current source has no theme control. The main `single-dark-theme` specification also forbids one. The delta specifications replace that rule and make the requested icon a functional light and dark control.

## Goals / Non-Goals

**Goals:**

- Keep one stable top bar in the table and page views.
- Use the current view state to switch one context slot between search and the selected slug.
- Preserve accessible names when the live state and theme control use icons.
- Keep the top bar usable without horizontal page overflow.
- Add a light palette without changing stored page content in the iframe.

**Non-Goals:**

- Do not change the page API, search API, SSE protocol, or iframe navigation.
- Do not add a stored theme preference or operating-system theme detection.
- Do not change the layout or colors inside stored page content.

## Decisions

### Use one stable context slot

Place the search input and a slug text element in one flexible header container. Keep both elements in the DOM. The view renderer will show one element and hide the other.

This keeps focus behavior and search event wiring stable. Replacing nodes or moving the search input during navigation would add lifecycle work without changing behavior.

### Derive the slug display from existing view state

Use `state.view` and `state.selectedSlug` as the only source for the context slot. The page-opening path sets the selected slug before it renders the page view. The home path clears the selected slug and renders the table view.

A second header-specific state value would duplicate information and could become stale.

### Keep one visual status icon and one accessible status name

Keep the live-status element as the EventSource status target. Remove its visible badge text and background. Render one circular icon through CSS. Map `live` to green, and map `connecting` and `reconnecting` to red.

Update the element's accessible name for each EventSource state. Color alone cannot communicate the state to assistive technology.

### Add theme tokens and toggle the root theme attribute

Keep the existing dark tokens as the default. Add light token values under `html[data-theme="light"]`. Add one button after the live-status icon. The button toggles the root `data-theme` value and updates its icon and accessible name.

This design keeps the current dark first paint. It does not add storage or media-query behavior that the request does not require.

### Use flex order and bounded context width

Keep the home link first. Give the context slot flexible space with a bounded maximum width and a left auto margin. Keep the live and theme controls as fixed-size items after the context slot. Truncate a long slug inside the slot.

This puts search on the right side while it preserves the two icon controls. Fixed positioning or absolute positioning would make narrow layouts brittle.

## Risks / Trade-offs

- [A long slug can consume header space] → Truncate the visual text and keep the full slug in an accessible label and title.
- [Red can suggest an error while EventSource first connects] → Keep the accessible state name as `Connecting` or `Reconnecting`.
- [A light shell can surround dark stored content] → Limit theme tokens to the application shell and do not modify iframe content.
- [The theme resets after a reload] → Keep persistence out of scope and retain dark as the deterministic initial theme.

## Migration Plan

1. Deploy the updated embedded web assets with the server binary.
2. Verify the table view, page view, SSE states, both themes, and narrow layout.
3. Roll back the web asset changes if the header or page view fails.
