# reader-top-bar Specification

## Purpose

Defines the composition of HyperReader's persistent top bar: the single row that hosts the page title, the search field, and the live-connection indicator, in place of the previous split between a header row and a separate search toolbar row.

## Requirements

### Requirement: Top bar hosts title, search, and live indicator
The application SHALL present the page title, the search field, and the live-connection indicator together within a single persistent top bar element. The application SHALL NOT present a separate search toolbar row outside the top bar.

#### Scenario: Reader opens the document list
- **WHEN** a user opens the served root page
- **THEN** the top bar displays the page title, the search field, and the live-connection indicator together
- **AND** no separate search toolbar row exists outside the top bar

### Requirement: Top bar remains usable at narrow viewport widths
The top bar's search field SHALL remain operable and the live-connection indicator SHALL remain visible at narrow viewport widths, and the top bar SHALL NOT cause the page to overflow the viewport width.

#### Scenario: Reader opens the application on a narrow viewport
- **WHEN** a user opens the application in a narrow viewport
- **THEN** the top bar's title, search field, and live-connection indicator are all visible and operable
- **AND** the top bar does not cause the page to overflow the viewport width
