## Purpose

Defines how a page comes into existence and how it changes afterward: a single create-or-patch write path keyed by slug, full-body replacement with no history, and list ordering by recency of change.

## ADDED Requirements

### Requirement: A page is created or patched by one write operation, keyed by slug

Submitting a page write SHALL create a new page when the supplied slug does not already exist, and SHALL patch the existing page in place when the slug does already exist. There is no separate endpoint or parameter to distinguish the two cases; the slug's existence is what decides.

A create SHALL respond with a status distinct from a patch, so a caller can tell which one happened without comparing timestamps itself.

#### Scenario: First write with a new slug creates the page

- **WHEN** a write request supplies a slug that does not exist yet
- **THEN** a new page is stored under that slug
- **AND** the response indicates a creation, distinct from a patch response

#### Scenario: A second write with the same slug patches the page

- **GIVEN** a page already exists at a given slug
- **WHEN** a write request supplies that same slug again
- **THEN** the existing page's name, description, and HTML are replaced with the new request's values
- **AND** the response indicates a patch, distinct from a creation response
- **AND** no second page is created; exactly one page exists at that slug

### Requirement: A patch is a full-body replacement, not a partial update

A create-or-patch write request SHALL carry the complete state of the page: name, description, and HTML. There is no request shape that patches a subset of fields while leaving others at their previous stored value; the request's fields fully determine the result.

#### Scenario: A patch omitting a field does not preserve the prior value for it

- **GIVEN** a page exists with a non-empty description
- **WHEN** a patch request for that slug omits the description field
- **THEN** the page's stored description becomes the field's default for an omitted value, not the previously stored description

### Requirement: Creation time is fixed; change time advances on every patch

A page's creation timestamp SHALL be set once, when the page is first created, and SHALL NOT change on any subsequent patch. A page's change timestamp SHALL be set on creation and SHALL advance to the current time on every patch.

#### Scenario: Patching does not change the creation timestamp

- **GIVEN** a page was created at some past time
- **WHEN** the page is patched
- **THEN** the returned creation timestamp is unchanged from when the page was first created
- **AND** the returned change timestamp reflects the patch

### Requirement: Pages are listed by recency of change

Listing and searching pages SHALL order results by change time, most recent first. Patching a page SHALL move it to the front of that ordering, ahead of pages created more recently but not patched since.

#### Scenario: Patching an older page surfaces it above newer, untouched pages

- **GIVEN** page A was created before page B, and neither has been patched since creation
- **WHEN** page A is patched
- **THEN** a subsequent list places page A ahead of page B

### Requirement: A patch discards prior content irrecoverably

Patching a page SHALL overwrite its stored HTML, name, and description; the previous values SHALL NOT remain retrievable through any endpoint after the patch completes.

#### Scenario: Prior content is unavailable after a patch

- **GIVEN** a page has been patched with new HTML
- **WHEN** that page's content is fetched afterward
- **THEN** the response is the new HTML
- **AND** the HTML that existed before the patch is not obtainable through any endpoint
