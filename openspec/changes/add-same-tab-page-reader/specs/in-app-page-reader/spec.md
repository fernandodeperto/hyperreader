## Purpose

Defines how HyperReader opens stored pages in the current tab, preserves the live table during reading, and returns the user to that table.

## ADDED Requirements

### Requirement: Page activation opens an in-app page view
The application SHALL open a page inside the current HyperReader tab when the user activates its table row. The application SHALL NOT open a new browser tab or replace the HyperReader application document.

#### Scenario: User clicks a page row
- **WHEN** the user clicks a page row
- **THEN** the application displays that page inside the current HyperReader tab
- **AND** the application does not open another browser tab

#### Scenario: User activates a page row with the keyboard
- **WHEN** the user focuses a page row and presses Enter or Space
- **THEN** the application displays that page inside the current HyperReader tab
- **AND** the application does not open another browser tab

### Requirement: Page view renders the stored HTML as a complete document
The application SHALL render the stored HTML as its own complete document. Inline scripts, document-level styles, relative resources, fixed elements, and viewport units SHALL keep the behavior they have at the raw page-content endpoint.

The page view SHALL remain unsandboxed. This document boundary is not a security boundary, and stored page code remains trusted code in the HyperReader origin.

#### Scenario: Stored page contains an inline script
- **WHEN** the user opens a stored page that contains an inline script
- **THEN** the browser executes the script in the displayed page document

#### Scenario: Stored page uses document-level layout
- **WHEN** the stored page defines styles on its `html` or `body` element or uses viewport units
- **THEN** those styles apply within the displayed page viewport

### Requirement: Returning to the table preserves live table state
The application SHALL keep the table data, search value, and live-event connection active while a page is open. Returning to the table SHALL show the current table state without a required reload.

#### Scenario: User returns after searching before page activation
- **GIVEN** the user opened a page from filtered search results
- **WHEN** the user returns to the table
- **THEN** the search value remains present
- **AND** the table shows the current results for that search

#### Scenario: A page event arrives during page view
- **GIVEN** the user has a page open
- **WHEN** the application receives a page-created or page-updated event
- **THEN** the application updates the hidden table state
- **AND** the open page remains visible
- **AND** the updated table state is visible after the user returns

### Requirement: Table rendering cannot replace the active page view
The application SHALL treat the selected view as independent state. A table render caused by initial loading, search, an error, or a live event SHALL NOT replace an active page view or change the page-view scroll layout.

#### Scenario: Search completes during page view
- **GIVEN** the user has a page open
- **WHEN** a search request completes and renders table data
- **THEN** the page remains open
- **AND** the HyperReader shell remains fixed

### Requirement: Leaving page view stops the displayed page
The application SHALL remove the displayed page document when the user returns to the table. Scripts, media, timers, and resource activity owned by that page document SHALL stop with its removal.

#### Scenario: User returns to the table
- **WHEN** the user activates the HyperReader home link while a page is open
- **THEN** the application displays the table
- **AND** the prior page document is no longer loaded