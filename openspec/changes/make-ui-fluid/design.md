## Context

The application shell is a block-level `#app` element with a 960px maximum width, centered margins, and shared inline padding. Its descendants already derive their width from that shell: the document table and search field fill their available width, and the header flex layout positions content-sized controls at the edge.

## Goals / Non-Goals

**Goals:**

- Remove the single shell constraint that prevents existing fluid descendants from using a wide viewport.
- Retain the established spacing tokens, inline gutters, and intrinsic sizing of the theme toggle and live-status indicator.
- Verify the resulting geometry at both wide and narrow browser viewports.

**Non-Goals:**

- Redesign table column allocation, typography, or control styling.
- Add arbitrary ultrawide breakpoints or component-specific widths.
- Change the document API or client-side data behavior.

## Decisions

- Remove the shell's fixed maximum-width constraint rather than adding a larger replacement cap. This directly matches the decision to use the browser's available width and leaves the existing block layout, flex toolbar, and full-width table rules intact.
- Preserve the shell's existing padding as the viewport gutter. The global border-box sizing rule keeps those gutters within the available viewport width.
- Use browser-level geometry assertions rather than style-source assertions. At a wide viewport, assert that the rendered table width exceeds the former constrained layout. At a narrow viewport, assert that the page does not horizontally overflow.

## Risks / Trade-offs

- On very wide displays, text columns can have longer line lengths. This is accepted because the UI is a searchable data table, and the requested behavior is to use available workspace.
- Browser geometry can vary by pixel rounding. Assertions should compare meaningful bounds rather than exact fractional dimensions.
