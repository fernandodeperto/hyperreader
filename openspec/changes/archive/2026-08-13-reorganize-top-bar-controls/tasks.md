## 1. Top Bar Structure

- [x] 1.1 Add one context slot that contains the search input and selected-slug text in `web/index.html`.
- [x] 1.2 Add the theme icon button immediately after the live-status element in `web/index.html`.
- [x] 1.3 Update `web/app.css` to right-align the context slot, bound its width, and keep both icon controls visible at narrow widths.
- [x] 1.4 Replace the live badge styles with green and red icon states, without visible badge text.

## 2. View and Control Behavior

- [x] 2.1 Update the view renderer in `web/app.js` to show search in the table view and the selected slug in the page view.
- [x] 2.2 Update live-status rendering to keep accessible state text for live, connecting, and reconnecting states.
- [x] 2.3 Add light theme tokens and implement the icon control that switches the root element between dark and light themes.
- [x] 2.4 Update the theme icon and accessible name after each theme change.

## 3. Contract Verification

- [x] 3.1 Update web tests for top-bar order, the search and slug swap, and return-to-list behavior.
- [x] 3.2 Update SSE tests for the green live icon, red non-live icon, and accessible state names.
- [x] 3.3 Update theme tests for the dark default, both color schemes, control order, and icon state.
- [x] 3.4 Update narrow-layout tests to cover both the document list and a stored page with no horizontal overflow.
- [x] 3.5 Run the focused Go web tests and Playwright scenarios, then verify the table and page views in a browser.
