## Why

The reports this skill produces are the deliverable, and they are read by people, in a browser, top to bottom. Two things stop them reading well.

First, five defects are live in the shipped template: card headings render smaller than the body text they head above a 987px viewport (`template.html:125` vs the body clamp at `:43`); the contents sidebar goes dead after a window resize across the 64rem breakpoint (`:273` checks `matchMedia` once and never listens for a change, while `:81-82` hides the disclosure marker at that width); a report opened from HyperReader ignores the reader's own theme, because the report reads `report-theme` (`:222`) while the app stores `hyperreader-theme` (`web/app.js:198`) on the same origin; diff colours never print (`:215` omits `pre` from the `print-color-adjust` list); and the pre-delivery placeholder check fails on any report whose body legitimately contains `UPDATE` or `VALIDATE`, because it substring-matches the bare token `DATE` against the entire composed document.

Second, the rule that governs everything else no longer matches the delivery target. "Self-contained. One file. No CDN links... It has to survive being attached to a ticket or opened on a plane" was written for a document that travels. These documents do not travel: `internal/api/handlers.go:140-142` serves stored HTML with a `Content-Type` header and nothing else, no CSP and no sanitisation, and `web/app.js:400-402` opens each one full-page and unsandboxed in its own browser tab. The prohibition buys portability the delivery model does not use, and it costs real reading quality. Charts have no axes at all, because a static `viewBox` scaled into a narrow column renders a 12px label at 4px, so the catalog's worked example (`references/components.md:141-148`) is a line and a polyline with zero `<text>`. Body text is set in a UI font at 16 to 17px. Code is unhighlighted. Diagrams are ASCII.

## What Changes

### Defects in the shipped template

- Read the reader's own theme preference, `hyperreader-theme`, rather than a private `report-theme` key nothing else writes.
- Add `pre` to the `print-color-adjust: exact` list so diff colours survive a PDF export.
- Raise `.card h3` above the body maximum, and hold that as an invariant: no heading may be smaller than the text it heads, at any viewport.
- Replace the one-shot `matchMedia` check with a `change` listener, so the contents sidebar does not die on a resize.
- Replace the English placeholder markers with delimited `{{TOKEN}}` placeholders, filled from one mapping and checked by a single assertion that no `{{` survives.

**BREAKING** for the composition workflow only: `.replace('REPORT TITLE', ...)` and the nine-item assert list are both removed. No delivered report changes.

### Reading experience, with no new dependency

- Rebuild the type and spacing scale against the chosen face: body size, leading, measure, heading ramp.
- Number section headings from the same CSS counter that numbers the contents list, so a reader deep in the page can still locate themselves.
- Wrap each section in `<section id aria-labelledby>`, which replaces the `getBoundingClientRect().top <= 120` scan running on every scroll event and the separate `atBottom` special case with a correct IntersectionObserver, and makes `break-inside` reliable in print.
- Open every `<details>` before printing and restore afterwards, so appendices reach the PDF.

### External resources

The "self-contained" rule is replaced by four clauses, in priority order:

1. Type always loads: IBM Plex Sans and IBM Plex Mono, variable woff2, `font-display: swap`, a metric-matched fallback so the swap does not reflow the page.
2. Three capabilities load conditionally, emitted into a `{{EXTERNAL}}` slot only when the report uses them: charts, syntax highlighting, mermaid.
3. The argument survives the network being off. A chart therefore ships the data it plots, as a table inside the same `<figure>`.
4. Nothing external for layout or colour.

Concretely: charts move from hand-computed SVG polylines to an SVG-emitting library with real axes; `highlight.js` arrives with a hand-written map from its token classes onto the existing palette variables rather than a stock theme, because a stock theme paints string literals green inside a page where green means confirmed; mermaid sits beside `.steps` and ASCII rather than replacing them, with `<pre class="mermaid">` as its own offline fallback; and one `themechange` event re-renders the chart and the diagram on toggle, which is the single most likely thing to ship broken because reports get eyeballed in light mode.

### Guidance

- Rewrite the self-contained rule into the four clauses above.
- State the three-depth contract the spine already implies: the report must work at masthead alone, at masthead plus banner plus stats, and in full, and a reader may stop after any section.
- Add a component selection table mapping situation to component, because component choice is most of what skimmability is and the default pull is toward prose and bullet lists.
- Keep the existing "three to five stats" budget and add only budgets that encode a perceptual limit, with the reason stated.
- Replace metric-counting verification with rendering at three widths in two themes and looking, plus five invariants that each caught a real defect.

### Reversals of prior decisions

- `2026-08-05-fix-generate-html-skill` decided to keep the duplicate `DATE` token, reasoning that the collision is self-correcting because a naive replace trips a different assert. That reasoning is sound for false negatives and does not address false positives: the same assert list fails a correct report containing `UPDATE`. Delimited tokens remove both directions, and generalise that change's own rejection of a capitals scan (which would have fired on `CONFIRMED` and `INFERRED` badge text).
- That change also decided to drop sticky table headers rather than cap `.scroll`, because `overflow-x: auto` computes `overflow-y` to `auto` and a height cap creates a scroll region inside a page that already scrolls and truncates tables in print. That reasoning still holds. Sticky headers were considered again here and are **not** part of this change.

### Considered and cut

- Read-time in the masthead: the estimate is wrong on exactly the chart-and-table-heavy reports that are hardest to judge, and a wrong number is worse than none.
- Hover anchor links on headings: hover-only, and each report already occupies its own tab with the URL in the address bar.
- Scroll position memory: genuinely useful, but it restores against a layout that is still settling behind a font swap and asynchronous chart rendering, so it lands in the wrong place often enough to annoy. Deferred, not rejected.
- A serif body face: reconsidered and rejected. This is technical prose with high inline-code density, where serif plus mono is a rough texture mix and every code span reads as an intrusion.

## Capabilities

### New Capabilities

<!-- None. Everything below is the same behavior contract for the same skill. -->

### Modified Capabilities

- `html-report-skill`: placeholders become delimited tokens and the pre-delivery check gains a false-positive prohibition; theme correctness extends past first paint to survive a theme change with externally rendered components present; the stored preference a report honours is the reader's, not a private key; section numbering permits one mechanism to produce more than one rendering. Adds requirements for conditional external resources and their offline degradation, for a chart shipping its data, for monotonic heading type, for navigation surviving a viewport change, and for a printed report carrying the evidence the screen report carries.

## Impact

- `skills/generate-html/assets/template.html`: token block and type scale, masthead, layout and contents, all component rules, print block, theme bootstrap and toggle, scrollspy, and a new `{{EXTERNAL}}` slot in `<head>`. This is a rewrite of the stylesheet against a new type basis, not a patch.
- `skills/generate-html/SKILL.md`: the self-contained rule, the composition example and its assertion, the three-depth contract, the verification method.
- `skills/generate-html/references/components.md`: section wrapper, figure structure with its data table, chart, highlighting and diagram entries, component selection table, budgets.
- `~/.agents/skills/generate-html`: symlink to the above, so the registered skill changes with no separate edit.
- New runtime dependencies, all external, none required for the argument to survive: IBM Plex Sans and IBM Plex Mono always; a chart library, `highlight.js` and mermaid conditionally. Worst-case report weight rises from roughly 15KB to roughly 1.5MB, of which mermaid is two thirds, which is why clause 2 is not optional.
- Read-only dependencies for verification: `internal/api/handlers.go`, `web/app.js`.
- No change to HyperReader's server, storage, MCP, or web app.
