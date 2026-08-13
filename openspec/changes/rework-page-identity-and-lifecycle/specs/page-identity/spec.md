## Purpose

Defines what a page is: how it is identified, how its metadata is bounded, and why those constraints are the input-validation boundary for a value that also names a file on disk and a segment of a URL.

## ADDED Requirements

### Requirement: Pages are identified by a slug

A page SHALL be identified by a slug matching `^[a-z0-9]+(-[a-z0-9]+)*$`: lowercase letters and digits, grouped into dash-separated words, with no leading dash, no trailing dash, and no consecutive dashes. A slug SHALL NOT exceed 80 characters.

The slug is the page's sole identifier: no separate numeric id is exposed by the API, stored as a distinct column, or accepted as an alternative identifier. The slug is supplied by the caller creating or patching the page; it is never server-generated from another field.

A request whose slug fails this pattern or exceeds the length limit SHALL be rejected with a client error before any storage or filesystem operation runs, since the slug is also used to name a file and to build a URL path segment, and an unvalidated value in either position is an injection or traversal vector.

#### Scenario: Slug within the allowed pattern and length

- **WHEN** a request supplies a slug consisting of lowercase dash-separated words no longer than 80 characters
- **THEN** the request is accepted for identity purposes and processing continues

#### Scenario: Slug contains a path separator

- **WHEN** a request supplies a slug containing `/`
- **THEN** the request is rejected with a client error
- **AND** no file is written and no row is created or modified

#### Scenario: Slug contains traversal segments

- **WHEN** a request supplies a slug containing `..`
- **THEN** the request is rejected with a client error
- **AND** no file is written and no row is created or modified

#### Scenario: Slug uses disallowed characters

- **WHEN** a request supplies a slug containing an uppercase letter, an underscore, a space, or any character outside `[a-z0-9-]`
- **THEN** the request is rejected with a client error naming the allowed pattern

#### Scenario: Slug has a leading, trailing, or doubled dash

- **WHEN** a request supplies a slug starting or ending with `-`, or containing `--`
- **THEN** the request is rejected with a client error

#### Scenario: Slug exceeds the maximum length

- **WHEN** a request supplies a slug longer than 80 characters
- **THEN** the request is rejected with a client error naming the limit

### Requirement: Description is capped

A page's description SHALL NOT exceed 200 characters. A request whose description exceeds this limit SHALL be rejected with a client error naming the limit; the description SHALL NOT be silently truncated to fit.

#### Scenario: Description within the cap

- **WHEN** a request supplies a description of 200 characters or fewer
- **THEN** the request is accepted for this field and processing continues

#### Scenario: Description exceeds the cap

- **WHEN** a request supplies a description longer than 200 characters
- **THEN** the request is rejected with a client error naming the 200-character limit
- **AND** no row is created or modified

### Requirement: Pages carry no tags

A page's stored metadata SHALL NOT include a tags field. A write request that includes a `tags` field SHALL have no effect from it: the field is not persisted, not indexed, and not returned.

Full-text search SHALL index only a page's name and description; it SHALL NOT index any tags-like field.

#### Scenario: Request includes a tags field

- **WHEN** a create-or-patch request includes a `tags` field alongside valid slug, name, description, and HTML
- **THEN** the page is created or patched normally
- **AND** the stored and returned page metadata contains no tags value derived from that field

#### Scenario: Search matches on name and description only

- **WHEN** a search query is run against the page index
- **THEN** matches are determined only by each page's name and description
