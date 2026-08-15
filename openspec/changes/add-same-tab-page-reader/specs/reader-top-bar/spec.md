## MODIFIED Requirements

### Requirement: Top bar hosts title, search, and live indicator
The application SHALL present the HyperReader title, the search field, and the live-connection indicator together within one persistent top bar. The application SHALL NOT present a separate search toolbar outside the top bar.

The HyperReader title SHALL be a home link. Activating the home link SHALL display the table in the current tab.

#### Scenario: Reader opens the page list
- **WHEN** a user opens the served root page
- **THEN** the top bar displays the HyperReader home link, search field, and live-connection indicator together
- **AND** no separate search toolbar exists outside the top bar

#### Scenario: User activates the title from page view
- **GIVEN** the user has a page open in the reader
- **WHEN** the user activates the HyperReader title
- **THEN** the application displays the table in the current tab

## ADDED Requirements

### Requirement: Top bar remains fixed during page reading
The top bar SHALL remain visible and fixed in place while the displayed page scrolls. The page content SHALL start below the top bar and SHALL NOT scroll over or under it.

#### Scenario: User scrolls a long page
- **GIVEN** the user opened a page that is taller than its page viewport
- **WHEN** the user scrolls to a later part of the page
- **THEN** the top bar remains visible in its original viewport position
- **AND** the page content scrolls below the top bar