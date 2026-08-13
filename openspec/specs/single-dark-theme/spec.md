# single-dark-theme Specification

## Purpose

Defines HyperReader's single-theme behavior: the reader always renders in a dark color scheme, with no light palette, no theme toggle, and no operating-system-preference-driven switching.

## Requirements

### Requirement: Reader always renders in dark theme
The application SHALL render using a single dark color scheme at all times, regardless of the operating system's color-scheme preference or any previously stored preference. The application SHALL NOT present a light color palette.

#### Scenario: Reader opens the application on any system color-scheme preference
- **WHEN** a user opens the served root page, on a system set to a light preference, a dark preference, or no preference
- **THEN** the page renders using the dark color scheme
- **AND** no flash of a different color scheme occurs before or during first paint

### Requirement: No theme control is presented
The application SHALL NOT present a theme toggle or any other user-facing control for changing the color scheme.

#### Scenario: Reader looks for a theme control
- **WHEN** a user inspects the top bar or any other part of the application
- **THEN** no control for switching the color scheme is present
