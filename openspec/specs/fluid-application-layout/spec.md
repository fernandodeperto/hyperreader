# fluid-application-layout Specification

## Purpose

Provide a document-management interface that uses the browser's available horizontal workspace while retaining consistent page gutters and usable narrow-screen behavior.

## Requirements

### Requirement: Available-width application shell
The application SHALL allow its primary content shell to use the available viewport width, subject only to the shell's page gutters. The shell SHALL NOT impose a fixed maximum content width that leaves unused horizontal space on wider viewports.

#### Scenario: Wide viewport uses available workspace
- **WHEN** a user opens the application in a viewport wider than the previous 960px shell limit
- **THEN** the header, search toolbar, document table, empty state, and error state extend across the available shell width
- **AND** the documents table has substantially more usable width than it had within the previous fixed-width shell

### Requirement: Responsive controls remain usable
The application SHALL preserve the existing responsive behavior at narrow viewport widths. Controls that are intentionally content-sized, including the theme toggle and live-status indicator, SHALL remain content-sized while remaining positioned within the expanded shell.

#### Scenario: Narrow viewport retains usable layout
- **WHEN** a user opens the application in a narrow viewport
- **THEN** the shell retains its inline gutters
- **AND** the shell itself does not introduce horizontal viewport overflow
