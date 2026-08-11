## 1. Delimited placeholders

- [x] 1.1 In `assets/template.html`, replace every shell placeholder with a delimited token: `{{KIND}}` (`:232`), `{{TITLE}}` (`:6` and `:233`), `{{LEDE}}` (`:234`), `{{DATE}}` `{{SCOPE}}` `{{SOURCES}}` (`:235`), `{{CONTENTS}}` (`:242`), `{{CONTENT}}` (`:247`), `{{DATE}}` and `{{CAVEATS}}` (`:252`)
- [x] 1.2 Add `{{EXTERNAL}}` in `<head>` immediately before the `<style>` block, as the slot for conditionally emitted resources
- [x] 1.3 Confirm `{{TITLE}}` at `:6` and `:233` both fill from one mapping entry, and `{{DATE}}` at `:235` and `:252` likewise
- [x] 1.4 In `SKILL.md`, replace the ordered `.replace()` chain in the worked example with a single mapping and one substitution pass over `{{...}}`
- [x] 1.5 Replace the nine-item assert list with `assert '{{' not in html`, reporting which tokens remain when it fails
- [x] 1.6 Delete the paragraph explaining that `DATE` occurs twice and must be filled by replacing its whole enclosing string
- [x] 1.7 Compose a report whose body contains `UPDATE`, `VALIDATE` and a table column headed `SOURCES`, and confirm the check passes
- [x] 1.8 Compose a report leaving `{{SCOPE}}` unfilled and confirm the check fails, naming it, and that nothing is sent

## 2. Template defects independent of type

- [x] 2.1 In the head bootstrap (`:222`), read `hyperreader-theme` instead of `report-theme`
- [x] 2.2 In the toggle handler (`:269`), write `hyperreader-theme`
- [x] 2.3 Replace the one-shot check at `:273` with a `matchMedia("(min-width: 64rem)")` `change` listener that sets `.toc.open` on entering the wide layout, keeping the initial evaluation
- [x] 2.4 Verify at 900px then resized to 1400px: the sidebar shows its links, with no reload
- [x] 2.5 Verify at 1400px then resized to 900px: the contents disclosure is operable and shows its marker

## 3. Typeface

- [x] 3.1 Add IBM Plex Sans and IBM Plex Mono, variable woff2, to the `{{EXTERNAL}}` slot as an always-emitted entry, with `preconnect`
- [x] 3.2 Set `--sans` and `--mono` (`:24-25`) to Plex first, the metric-matched fallback second, the generic last
- [x] 3.3 Declare the fallback `@font-face` with `size-adjust`, `ascent-override` and `descent-override` measured against the fallback that actually resolves, not copied values
- [x] 3.4 Set `font-display: swap`
- [x] 3.5 Verify no visible reflow: block the font in dev tools, screenshot, unblock, screenshot, compare masthead and first section positions
- [x] 3.6 Verify inline `<code>` sits in the line without a jump in x-height against Plex Sans body text

## 4. Type and spacing scale

- [x] 4.1 Retune the body size clamp (`:43`) and `line-height` (`:44`) for Plex Sans at long-form reading size
- [x] 4.2 Retune `--measure` (`:26`) against the new body size
- [x] 4.3 Retune the heading ramp (`:50-54`) so `h1` through `h4` step cleanly against the new body size
- [x] 4.4 Raise `.card h3` (`:125`) above the body maximum, resolving the live inversion at viewports past roughly 987px
- [x] 4.5 Check every other heading-like rule against the invariant: `.stat .l`, `.callout .t`, `.steps .st`, `.toc summary`, `figcaption`, `thead th`
- [x] 4.6 Reconcile the spacing rhythm with the new scale: `--gap` (`:28`), paragraph and list margins (`:58-61`), heading margins, `.card` and `.callout` padding
- [x] 4.7 Verify the monotonic-heading invariant at 320px, 768px, 1024px and 1600px, in both themes
- [x] 4.8 Verify no horizontal scroll at 320px

## 5. Section structure and navigation

- [x] 5.1 Wrap each section in `<section id aria-labelledby>` in the template's `<main>` and in every section snippet in `references/components.md`
- [x] 5.2 Number section headings from a CSS counter over sections, matching the existing contents counter at `:89`
- [x] 5.3 Add a composition-time assertion that the contents list has exactly one entry per section and that the nth entry targets the nth section
- [x] 5.4 Replace the scrollspy (`:275-288`) with an IntersectionObserver over section bounds, using a `rootMargin` biased to the top band
- [x] 5.5 Delete the `atBottom` special case, and confirm the final section still highlights when the page is scrolled to the end
- [x] 5.6 Verify contents entry and section heading show the same number, and that reordering sections renumbers both with no hand edit

## 6. Print

- [x] 6.1 Add `pre` to the `print-color-adjust: exact` list (`:215`)
- [x] 6.2 Add `beforeprint` and `afterprint` handlers that open every `<details>` and restore prior state
- [x] 6.3 Add `break-inside: avoid` for `section` to the print block (`:214`), now that sections are wrapped
- [x] 6.4 Verify a printed report includes appendix content that sits behind a disclosure on screen
- [x] 6.5 Verify a printed diff reproduces its added, removed and comment colours

## 7. Conditional external slot

- [x] 7.1 In `SKILL.md`, document the `{{EXTERNAL}}` slot: fonts always, the three capabilities only when the report uses them
- [x] 7.2 Show the composition idiom that inspects the composed body and emits only the needed entries
- [x] 7.3 Verify a report with no chart, no code block and no diagram emits nothing beyond the font entry
- [x] 7.4 Verify a report using all three emits exactly three entries

## 8. Charts

- [x] 8.1 Choose between Observable Plot and Frappe Charts by building one real chart with each and comparing composed output and weight
- [x] 8.2 Use the build that works from a plain `<script src>` with no module context, and pin a major version
- [x] 8.3 Rewrite the Figures entry in `references/components.md:135-149` around the `<figure>` structure: rendering target, `figcaption` carrying the claim, `<details>` holding the data table
- [x] 8.4 Delete the instruction to compute coordinates in a scratch script, and the axis-free `<line>` and `<polyline>` example
- [x] 8.5 State that inline SVG remains correct for hand-drawn figures that are not plots
- [x] 8.6 Emit the chart's data and its table from the same values, so the two cannot drift
- [x] 8.7 Verify a chart renders readable axis labels at a 380px viewport
- [x] 8.8 Verify that with the chart script blocked, the figure still presents its values and supports the caption's claim

## 9. Syntax highlighting

- [x] 9.1 Add `highlight.js` as a conditional entry, pinned, emitted only when the report contains a highlighted code block
- [x] 9.2 Write the token map onto palette variables: comments to `--dim`, strings to one hue that is neither green nor red, keywords by weight rather than colour
- [x] 9.3 Confirm the existing `.add` `.del` `.cm` diff classes (`:168`) still apply and are not overridden by the map
- [x] 9.4 Verify highlighting is correct in both themes without a second stylesheet fetch
- [x] 9.5 Verify that with the script blocked, code renders as plain mono with nothing lost

## 10. Mermaid

- [x] 10.1 Add mermaid as a conditional entry, pinned, emitted only when the report contains a `<pre class="mermaid">` block
- [x] 10.2 Initialise with `theme: 'base'` and theme variables derived from the resolved palette
- [x] 10.3 Add a catalog entry that leads with when not to use it: `.steps` for anything linear, ASCII for small topologies, mermaid only for dependency graphs, sequence diagrams and state machines
- [x] 10.4 Verify that with the script blocked, the diagram source stays visible and readable
- [x] 10.5 Verify a wide diagram does not force horizontal scroll on the page at 380px

## 11. Theme change

- [x] 11.1 Dispatch a `themechange` event from the toggle handler (`:266-271`)
- [x] 11.2 Re-render the chart on that event
- [x] 11.3 Re-run mermaid on that event
- [x] 11.4 Confirm highlighting needs no handler because its map is expressed in palette variables
- [x] 11.5 Verify: load a report containing all three in light, let everything render, toggle to dark, and confirm none of the three is left in the previous theme's colours

## 12. Guidance

- [x] 12.1 Replace the "Self-contained" rule in `SKILL.md` with the four clauses, in priority order
- [x] 12.2 State the three-depth contract: the report works at masthead alone, at masthead plus banner plus stats, and in full, and a reader may stop after any section
- [x] 12.3 Add the component selection table mapping situation to component
- [x] 12.4 Keep the existing three-to-five stats budget; add only budgets that encode a perceptual limit, each with its reason stated
- [x] 12.5 Update the opening of `references/components.md` for the `{{EXTERNAL}}` slot and the conditional capabilities
- [x] 12.6 Replace the verification paragraph in `SKILL.md` with the render-and-look method plus the five invariants

## 13. Verification

- [x] 13.1 Start `serve` and confirm it is reachable
- [x] 13.2 Compose a report exercising every catalog component at least once, including a chart with its data table, a highlighted code block, a mermaid diagram, a diff, steps, a meter, an inline SVG figure, a disclosure, and at least three sections
- [x] 13.3 Send it and open the returned permalink full-page
- [x] 13.4 Invariant: no heading renders smaller than the text it heads, at 320px, 768px, 1024px and 1600px, in both themes
- [x] 13.5 Invariant: no horizontal scroll at 320px
- [x] 13.6 Invariant: every chart in the report carries its data table
- [x] 13.7 Invariant: with the network blocked, no figure is empty, the page is not unstyled, and every claim, number and relationship is still present
- [x] 13.8 Invariant: after toggling to dark with everything rendered, no component is left in the previous theme's colours
- [x] 13.9 Set dark in HyperReader's document list, open a report, and confirm it opens dark with no flash
- [x] 13.10 Toggle to light inside the report, return to the list, and confirm the list is light
- [x] 13.11 Resize across the 64rem breakpoint in both directions and confirm the contents navigation stays operable
- [x] 13.12 Print to PDF and confirm appendices appear and diff colours are reproduced
- [x] 13.13 Read the whole report end to end in both themes and judge it by eye. This step is the point of the change and is not satisfied by the invariants above
