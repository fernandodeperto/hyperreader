## MODIFIED Requirements

### Requirement: Unfilled shell placeholders cannot reach a delivered report
The template ships placeholders in its `<title>`, masthead, contents list, external-resource slot, body slot, and footer. Every placeholder SHALL be written in a single delimited form that cannot occur in report content. The workflow SHALL verify that no placeholder remains before delivery, and SHALL abort delivery when any remains. The check SHALL NOT abort delivery of a report whose content merely resembles a placeholder.

#### Scenario: A masthead placeholder is left unfilled
- **WHEN** composition leaves any shell placeholder unreplaced, including the scope and sources fields of the masthead and the body content slot
- **THEN** the pre-delivery check fails, naming the placeholder that remains
- **AND** the report is not sent

#### Scenario: Every placeholder filled
- **WHEN** composition replaces all shell placeholders
- **THEN** the pre-delivery check passes and the report is sent

#### Scenario: Report content contains a word that resembles a placeholder
- **WHEN** a fully composed report contains body text such as `UPDATE`, `VALIDATE`, or a table column headed `SOURCES`
- **THEN** the pre-delivery check passes and the report is sent

### Requirement: Placeholders that occur more than once resolve unambiguously
A placeholder occurring at more than one position in the shell SHALL resolve to its intended value at every position. Filling any placeholder SHALL NOT alter shell text outside a placeholder.

#### Scenario: Date appears in both masthead and footer
- **WHEN** the report date is filled
- **THEN** the masthead and the footer each carry that date
- **AND** the footer sentence retains its intended wording, with no residual placeholder text

#### Scenario: A value that itself contains shell wording is filled
- **WHEN** a filled value happens to repeat wording that also appears in the surrounding shell
- **THEN** only the placeholder position changes, and the surrounding shell text is unaltered

### Requirement: Documented components render as specified in both themes
Every component in the catalog SHALL produce its documented visual result in both light and dark themes. A declaration that a supported browser discards SHALL NOT appear in the stylesheet. A component whose visual result is produced by an external resource SHALL match the active theme after a theme change, not only at first render.

#### Scenario: Surfaces that document a shadow
- **WHEN** a delivered report is rendered in either theme
- **THEN** each element documented as raised computes a non-`none` shadow appropriate to that theme

#### Scenario: Tables presented as scrollable with pinned headers
- **WHEN** a table documented as having a pinned header row is scrolled
- **THEN** the header row remains visible, or the catalog does not claim the header pins

#### Scenario: Stylesheet contains no inert rules
- **WHEN** the stylesheet is audited against the catalog's documented usages
- **THEN** every rule applies to at least one documented usage

#### Scenario: Theme changed after an externally rendered component has drawn
- **WHEN** a report containing a chart, a diagram, or highlighted code is switched from one theme to the other after those components have rendered
- **THEN** each of them matches the newly active theme, with no component left in the previous theme's colours

### Requirement: Saved theme applies before first paint
The theme preference a report honours SHALL be the one the reader set in the reader application, and SHALL be applied before the page first paints, so no report opens in the wrong theme and then corrects itself.

#### Scenario: Reader with a stored dark preference opens a report
- **WHEN** a reader who previously selected dark opens any report
- **THEN** the first painted frame is dark, with no visible flash of the light theme

#### Scenario: Reader sets the theme in the reader application, then opens a report
- **WHEN** a reader selects dark in the reader's document list and then opens a document
- **THEN** that document opens dark, without the reader setting the theme a second time

#### Scenario: Reader changes the theme from inside a report
- **WHEN** a reader toggles the theme while reading a report and then returns to the document list
- **THEN** the list is in the theme they just chose

### Requirement: Section numbering has a single source
Exactly one mechanism SHALL produce every rendering of a section's number. A number MAY be rendered in more than one place, and all renderings SHALL agree. Guidance SHALL NOT instruct authors to number a section by hand where numbering is already generated.

#### Scenario: Report with several sections
- **WHEN** a multi-section report is rendered
- **THEN** each contents entry and its section heading show the same number for that section

#### Scenario: Sections are reordered
- **WHEN** the order of sections in a composed report changes
- **THEN** every rendering of every section number follows the new order, with no hand edit

## ADDED Requirements

### Requirement: External resources are conditional and never carry the argument
The skill SHALL define a fixed, enumerated set of capabilities that may load an external resource. A report that does not use a capability SHALL NOT load that capability's resource. With every external resource unavailable, a delivered report SHALL remain readable and SHALL retain every claim, number and relationship it asserts. No external resource SHALL supply layout or colour.

#### Scenario: Report uses none of the external capabilities
- **WHEN** a report contains no chart, no highlighted code and no diagram
- **THEN** it loads no resource for those capabilities

#### Scenario: Every external resource is unavailable
- **WHEN** a delivered report is opened with no network access
- **THEN** it renders readably, and every claim, number and relationship it asserts is still present

#### Scenario: The typeface is unavailable
- **WHEN** the report's typefaces cannot be fetched
- **THEN** text renders in a fallback whose metrics match closely enough that no layout shift is visible when the fetch later succeeds

### Requirement: A chart ships the data it plots
Every chart SHALL carry, within the same figure, the data it plots in textual form.

#### Scenario: The chart's renderer is unavailable
- **WHEN** a report containing a chart is opened and the chart cannot render
- **THEN** the figure still presents the plotted values, and the reader can reach the same conclusion the chart was drawn to support

#### Scenario: Report read with assistive technology
- **WHEN** a reader reaches a chart using a screen reader
- **THEN** the plotted values are available as text

### Requirement: Heading type is never smaller than the text it heads
At every viewport width, in both themes, a heading SHALL render at a larger type size than the body text within the block it heads.

#### Scenario: Card heading beside body prose at a wide viewport
- **WHEN** a report is rendered at any width at which body text is at its maximum size
- **THEN** every heading, including headings inside cards, renders larger than that body text

#### Scenario: Narrow viewport
- **WHEN** a report is rendered at the narrowest supported width
- **THEN** the same relationship holds

### Requirement: Contents navigation survives a viewport change
The contents navigation SHALL be usable at every viewport width, and SHALL remain usable after the viewport changes width, without a reload.

#### Scenario: Window resized across the layout breakpoint after load
- **WHEN** a report is opened at a width below the sidebar breakpoint and the window is then widened past it
- **THEN** the contents navigation presents its links and is operable

#### Scenario: Window narrowed after load
- **WHEN** a report is opened at a width above the sidebar breakpoint and the window is then narrowed below it
- **THEN** the contents navigation presents its links and is operable

### Requirement: A printed report carries the evidence the screen report carries
A report printed or exported to PDF SHALL include the content held behind disclosures on screen, and SHALL reproduce the colour that carries meaning.

#### Scenario: Report with disclosed appendices is printed
- **WHEN** a report whose appendices sit behind disclosures is printed
- **THEN** the appendix content appears in the output

#### Scenario: Report with coloured code is printed
- **WHEN** a report containing a coloured diff or highlighted code is printed
- **THEN** the colours that distinguish added, removed and commented lines are reproduced rather than flattened
