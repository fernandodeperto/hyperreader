## ADDED Requirements

### Requirement: Page view uses the viewport without parent overflow
While a page is open, the application SHALL fit the HyperReader shell within the viewport. The page viewport SHALL use all space below the fixed top bar and SHALL extend to the viewport edges.

The HyperReader document and application shell SHALL have no vertical or horizontal overflow in page view. The displayed page document SHALL own the only viewport-level scroll container.

#### Scenario: A long page scrolls without a parent scrollbar
- **GIVEN** the user opened a page that is taller than its page viewport
- **WHEN** the user scrolls the page
- **THEN** the displayed page document scrolls
- **AND** the HyperReader document does not scroll vertically
- **AND** the top bar remains visible
- **AND** only one viewport-level vertical scrollbar is present

#### Scenario: Page view has no application-level horizontal overflow
- **WHEN** the user opens a page at a narrow or wide viewport width
- **THEN** the HyperReader document and application shell fit within the viewport width
- **AND** neither the HyperReader document nor the application shell has horizontal overflow

#### Scenario: Page view fills the space below the top bar
- **WHEN** the user opens a page
- **THEN** the page viewport starts below the top bar
- **AND** the page viewport fills the remaining viewport height
- **AND** the page viewport extends to both horizontal viewport edges