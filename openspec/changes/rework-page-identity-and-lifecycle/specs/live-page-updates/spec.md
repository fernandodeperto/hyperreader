## Purpose

Defines how a page write is reflected to a browser tab already subscribed to the live event stream: distinct signals for a new page versus a patched one, and how a connected client reconciles its list from those signals.

## ADDED Requirements

### Requirement: Creation and patch broadcast distinct event types

The live event stream SHALL emit a distinctly named event when a page is created and a distinctly named event when a page is patched, so a subscriber can tell the two apart without inspecting timestamps. Both event types SHALL carry the same page metadata shape a direct fetch of the page would return, excluding the HTML body.

#### Scenario: Creating a page broadcasts a creation event

- **WHEN** a page is created
- **THEN** every connected subscriber receives a creation-typed event carrying that page's metadata

#### Scenario: Patching a page broadcasts a patch event

- **WHEN** an existing page is patched
- **THEN** every connected subscriber receives a patch-typed event carrying that slug's updated metadata
- **AND** no subscriber receives a creation-typed event for that slug

### Requirement: A connected client reconciles by slug, not by appending

A client that has an unfiltered list open SHALL react to a creation-typed event by adding a new entry for that slug. It SHALL react to a patch-typed event by updating the entry already shown for that slug in place rather than adding a second entry, and SHALL move that entry to the front of the list, consistent with list ordering by recency of change.

#### Scenario: A patch updates the existing row instead of duplicating it

- **GIVEN** a client's list already shows a page at some position
- **WHEN** that page is patched and the client receives the patch-typed event
- **THEN** the list shows exactly one entry for that slug afterward
- **AND** that entry reflects the patched name and description
- **AND** that entry is at the front of the list

### Requirement: Live events are suppressed while a local filter is active

A client with an active search filter SHALL NOT inject a new entry or move an existing entry in response to a creation-typed or patch-typed event; the filtered view SHALL remain exactly what the last search returned until the filter is cleared or re-run.

#### Scenario: A patch event arrives while a search filter is active

- **GIVEN** a client has a non-empty search filter applied
- **WHEN** a patch-typed event arrives for a page
- **THEN** the client's currently displayed filtered results are unchanged
- **AND** the update is reflected only after the filter is cleared or the search is re-run
