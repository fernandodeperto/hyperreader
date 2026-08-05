## Why

The web UI caps its application shell at 960px, leaving substantial unused horizontal space on wide browser windows. The search toolbar and documents table already size to their container, so the cap prevents otherwise responsive elements from using available workspace.

## What Changes

- Make the application shell fluid across the available viewport while preserving the existing page gutters.
- Allow the header, search toolbar, empty and error states, and documents table to occupy the expanded shell width.
- Preserve content-sized controls such as the theme toggle and live-status indicator rather than artificially stretching their hit areas.
- Preserve usable narrow-viewport behavior and prevent horizontal viewport overflow.

## Capabilities

### New Capabilities

- `fluid-application-layout`: A viewport-responsive application shell that lets the document-management UI use available horizontal space.

### Modified Capabilities

- None.

## Impact

- `web/app.css`: application-shell sizing and any narrow-viewport safeguards.
- Browser UI coverage: add or update viewport-based verification for the fluid shell and full-width table.
