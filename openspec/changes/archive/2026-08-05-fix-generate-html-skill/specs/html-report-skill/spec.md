## Purpose

Defines the behavior contract for the `generate-html` skill: that it can resolve its own assets, that the reports it delivers render every documented component correctly in both colour themes, that a report carrying unfilled shell placeholders can never be delivered, and that the handover addresses the report itself rather than a listing of reports.

## ADDED Requirements

### Requirement: Skill resolves its own assets

The skill SHALL reference its template and reference files under its own registered skill id. No step of the workflow SHALL depend on a skill id other than the one declared in the skill's own `name:` field.

#### Scenario: Template read during the workflow

- **WHEN** the workflow reads the report template
- **THEN** the read resolves against the skill's own id and returns the template contents

#### Scenario: Skill installed with no sibling skills present

- **WHEN** the skill is the only skill installed, with no other agent skill registered
- **THEN** every asset the workflow reads still resolves, and the workflow runs to completion

### Requirement: Unfilled shell placeholders cannot reach a delivered report

The template ships placeholder text in its `<title>`, masthead, contents list, body slot, and footer. The workflow SHALL verify that no placeholder remains before delivery, and SHALL abort delivery when any remains.

#### Scenario: A masthead placeholder is left unfilled

- **WHEN** composition leaves any shell placeholder unreplaced, including the scope and sources fields of the masthead and the body content slot
- **THEN** the pre-delivery check fails, naming the placeholder that remains
- **AND** the report is not sent

#### Scenario: Every placeholder filled

- **WHEN** composition replaces all shell placeholders
- **THEN** the pre-delivery check passes and the report is sent

### Requirement: Placeholders that occur more than once resolve unambiguously

A placeholder token appearing at more than one position in the shell SHALL be replaceable at each position independently, so that filling one occurrence cannot corrupt another.

#### Scenario: Date appears in both masthead and footer

- **WHEN** the report date is filled into the masthead
- **THEN** the footer sentence retains its intended wording, with its own date filled independently and no residual placeholder text

### Requirement: Documented components render as specified in both themes

Every component in the catalog SHALL produce its documented visual result in both light and dark themes. A declaration that a supported browser discards SHALL NOT appear in the stylesheet.

#### Scenario: Surfaces that document a shadow

- **WHEN** a delivered report is rendered in either theme
- **THEN** each element documented as raised computes a non-`none` shadow appropriate to that theme

#### Scenario: Tables presented as scrollable with pinned headers

- **WHEN** a table documented as having a pinned header row is scrolled
- **THEN** the header row remains visible, or the catalog does not claim the header pins

#### Scenario: Stylesheet contains no inert rules

- **WHEN** the stylesheet is audited against the catalog's documented usages
- **THEN** every rule applies to at least one documented usage

### Requirement: Saved theme applies before first paint

A reader's stored theme preference SHALL be applied before the page first paints, so no report opens in the wrong theme and then corrects itself.

#### Scenario: Reader with a stored dark preference opens a report

- **WHEN** a reader who previously selected dark opens any report
- **THEN** the first painted frame is dark, with no visible flash of the light theme

### Requirement: Section numbering has a single source

Section numbers SHALL be produced by exactly one mechanism. Guidance SHALL NOT instruct authors to number a section by hand where numbering is already generated.

#### Scenario: Report with several sections

- **WHEN** a multi-section report is rendered
- **THEN** each contents entry and its section agree on that section's number, with no number stated twice

### Requirement: Contents list demonstrates multiple sections

The template and the worked example SHALL each demonstrate a contents list of more than one entry, since a report short enough to hold one section is out of the skill's stated scope.

#### Scenario: Author follows the worked example

- **WHEN** an author composes a report by following the example's contents-list idiom
- **THEN** the idiom extends to an arbitrary number of sections without reinterpretation

### Requirement: Handover addresses the delivered report

On success the skill SHALL hand the user a URL that opens the report just delivered, not a listing of all reports.

#### Scenario: Report sent successfully

- **WHEN** delivery succeeds and returns an identifier for the stored report
- **THEN** the handover states a URL that resolves to that report's own content

#### Scenario: Delivery fails

- **WHEN** delivery fails because the reader service is unreachable
- **THEN** the skill states the error rather than a URL

### Requirement: Guidance describes the workflow the skill actually performs

Reference material SHALL describe the composition model the workflow uses. It SHALL NOT instruct the author to edit files on disk when the workflow composes in memory and writes no file.

#### Scenario: Author reads the component catalog before composing

- **WHEN** an author reads the catalog's opening guidance on where to put a one-off style rule
- **THEN** that guidance is consistent with in-memory composition and names no on-disk copy to edit

### Requirement: Skill is discoverable

The skill SHALL be discoverable from the repository's own documentation, and SHALL sit where this repository keeps its agent skills.

#### Scenario: Contributor looks for the repository's agent skills

- **WHEN** a contributor reads the repository README and browses the skills location
- **THEN** the skill is listed with a one-line description and is found alongside the repository's other agent skills
