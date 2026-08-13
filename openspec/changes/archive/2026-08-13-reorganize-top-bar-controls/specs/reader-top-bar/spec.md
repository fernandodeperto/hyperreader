## MODIFIED Requirements

### Requirement: Top bar hosts title, search, and live indicator
The application SHALL present the page title, one context field, the live-connection indicator, and the theme control in one persistent top bar. In the document list, the context field SHALL be the search field and SHALL appear on the right side of the top bar before the status and theme controls. In the page view, the application SHALL hide the search field and SHALL show the selected page slug in the same top-bar position. The application SHALL place the theme control immediately after the live-connection indicator. The application SHALL NOT present a separate search toolbar row outside the top bar.

#### Scenario: Reader opens the document list
- **WHEN** a user opens the served root page
- **THEN** the top bar displays the page title and the search field
- **AND** the search field appears before the live-connection indicator and the theme control on the right side
- **AND** no separate search toolbar row exists outside the top bar

#### Scenario: Reader opens a stored page
- **WHEN** a user opens a page from the document table
- **THEN** the top bar hides the search field
- **AND** the top bar shows the selected page slug in the search field's position
- **AND** the live-connection indicator and theme control remain visible after the slug

#### Scenario: Reader returns to the document list
- **WHEN** a user returns from a stored page to the document list
- **THEN** the top bar hides the selected page slug
- **AND** the top bar restores the search field

### Requirement: Top bar remains usable at narrow viewport widths
The top bar's context field SHALL remain readable or operable at narrow viewport widths. The live-connection indicator and theme control SHALL remain visible and operable. The top bar SHALL NOT cause the page to overflow the viewport width.

#### Scenario: Reader opens the application on a narrow viewport
- **WHEN** a user opens the document list in a narrow viewport
- **THEN** the top bar's title, search field, live-connection indicator, and theme control are visible and operable
- **AND** the top bar does not cause the page to overflow the viewport width

#### Scenario: Reader opens a stored page on a narrow viewport
- **WHEN** a user opens a stored page in a narrow viewport
- **THEN** the selected page slug remains readable
- **AND** the live-connection indicator and theme control remain visible and operable
- **AND** the top bar does not cause the page to overflow the viewport width

## ADDED Requirements

### Requirement: Live connection uses an icon
The application SHALL show the live-connection state as one icon without visible badge text. The icon SHALL be green while the connection is live. The icon SHALL be red while the connection is connecting or reconnecting. The application SHALL expose the current state as accessible text so color is not the only status signal.

#### Scenario: Live connection opens
- **WHEN** the live-update connection opens
- **THEN** the live-connection icon is green
- **AND** assistive technology receives the live state

#### Scenario: Live connection is not open
- **WHEN** the live-update connection is connecting or reconnecting
- **THEN** the live-connection icon is red
- **AND** assistive technology receives the current non-live state
