## ADDED Requirements

### Requirement: Reader supports light and dark themes
The application SHALL provide light and dark color schemes. The application SHALL provide one icon control that switches between the two color schemes.

#### Scenario: User switches to the light theme
- **WHEN** the dark theme is active and the user activates the theme control
- **THEN** the application uses the light color scheme
- **AND** the control shows that another activation will select the dark theme

#### Scenario: User switches to the dark theme
- **WHEN** the light theme is active and the user activates the theme control
- **THEN** the application uses the dark color scheme
- **AND** the control shows that another activation will select the light theme

### Requirement: Theme control follows the live indicator
The application SHALL show the theme control in the top bar immediately after the live-connection indicator. The control SHALL remain available in the document list and page view.

#### Scenario: User opens the document list
- **WHEN** a user opens the served root page
- **THEN** the top bar shows the theme control immediately after the live-connection indicator

#### Scenario: User opens a stored page
- **WHEN** a user opens a page from the document table
- **THEN** the top bar keeps the theme control immediately after the live-connection indicator

## REMOVED Requirements

### Requirement: Reader always renders in dark theme
**Reason**: The requested theme control requires both light and dark color schemes.

**Migration**: Use the theme control in the top bar to select the required color scheme.

### Requirement: No theme control is presented
**Reason**: The top bar now includes the light and dark theme control.

**Migration**: Use the new icon control immediately after the live-connection indicator.
