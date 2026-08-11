## Context

See proposal.md - Why. Four facts about the delivery target shape every decision below.

**Reports do not travel.** `internal/api/handlers.go:140-142` writes `doc.HTMLContent` with a `Content-Type` header and nothing else: no CSP, no sanitisation. `web/app.js:400-402` opens each document full-page and unsandboxed in its own browser tab via `window.open(..., "_blank")`. External stylesheets, fonts and scripts will load. The "attached to a ticket, opened on a plane" premise of the current self-contained rule describes an artifact this one is not.

**The report is the whole surface.** There is no app chrome around an open document. Every reading affordance, navigation, theme control and print behaviour is the document's own responsibility, and nothing competes with it for the viewport.

**The reader has one theme preference, and the report ignores it.** `web/app.js:198` stores `hyperreader-theme`; `assets/template.html:222` reads `report-theme`. Same origin, shared `localStorage`, two keys.

**A prior overhaul of this skill exists only as a summary.** HyperReader document #10, `generate-html-readability-overhaul-2026-08-11`, describes an overhaul that was applied to the working tree and discarded: `git log -- skills/` has no commit for it, `git status` is clean, and all five skill files carry an identical mtime consistent with a checkout over the top. This change does not salvage it. One defect that document named, the one-shot `matchMedia` check, is included here because it was verified independently against the current file at `assets/template.html:81-82` and `:273`, not because that summary asserted it. Nothing else from that summary is carried, including its Booking-brand token set, its table scroll-cue gradient and its table caption support, none of which were verified and none of which are in scope.

## Goals / Non-Goals

**Goals:**

- A report that reads well to a person over twenty minutes in a browser, in either theme, at any width, and on paper.
- Every external resource is optional to the argument. Turning the network off degrades polish and never removes a claim.
- Prefer mechanisms that cannot drift over conventions an author must remember. This is the same principle the archived change applied to section numbering, extended.
- Every invariant in the spec is one that has already caught a real defect in this template.

**Non-Goals:**

- No change to HyperReader's server, storage, MCP or web app. The theme fix reads a key the app already writes.
- No sticky table headers. `2026-08-05-fix-generate-html-skill` removed them with a rationale that still holds, recorded under Decisions below.
- No serif body face, no Tailwind, no icon font, no external stylesheet for layout or colour.
- No alignment with any other artifact platform. This change is scoped to how HyperReader documents read.
- No salvage of the discarded overhaul described in Context.

## Decisions

### IBM Plex Sans with IBM Plex Mono, from one family

The current stack is `-apple-system` with `ui-monospace`, which resolves on macOS to SF Pro Text and SF Mono. That pairing is right about one thing and wrong about another. Right: sans and mono come from one family, so inline `<code>` sits in the line without a jump in x-height or stroke weight, and these reports are dense with inline identifiers. Wrong: SF Pro is a UI face, drawn for short strings at 11px in a toolbar, and it flattens across seven hundred words at 17px.

Take a text-grade humanist sans that brings its own matched mono. Plex Sans with Plex Mono. Source Sans 3 with Source Code Pro was the alternative and is equally defensible; Plex carries more character, Source is more invisible.

A serif body was considered at length and rejected. The catalog's own examples are the argument: a sentence like "the `ASEC_LNRS_UNKNOWN_SESSION` flag moved vanguard to full-on at 12:34 UTC" carries three typographic registers in one line. Serif prose against inline mono is a rough texture mix in which every code span reads as an intrusion. Publications that set technical prose with this code density are near-uniformly sans.

Inter was considered and rejected for the same reason as SF Pro: it is a UI face with a very large x-height, superb at 13px in an interface and characterless at reading size, and it brings no matched mono.

### The fallback is metric-matched, and that is the part that can go wrong

`font-display: swap` paints immediately in the fallback and swaps when Plex arrives. Without metric matching, that swap reflows the page under the reader.

Declare an `@font-face` for the fallback with `size-adjust`, `ascent-override` and `descent-override` tuned so the fallback occupies the same space as Plex, and put it between Plex and the generic in the stack. The override values must be measured against the fallback that actually resolves on the target platform, not copied from a table, and the acceptance test is visual: block the font, screenshot, unblock, screenshot, and compare. A numeric target here would be false precision.

`font-display: optional` was considered. It removes the swap entirely, at the cost of the webfont frequently never appearing on a first read, which for a document read once is the wrong trade.

### Four clauses, and clause 3 is the only one that is a test

1. Type always loads.
2. Three capabilities load conditionally: charts, syntax highlighting, mermaid.
3. The argument survives the network being off.
4. Nothing external for layout or colour.

Clauses 1, 2 and 4 are enumerations, which is deliberate: a permissive rule is much harder to hold than a prohibitive one, and an open-ended "external resources are fine when justified" drifts into Tailwind and a Google Font within two changes. Clause 4 is what keeps the visual identity in the file.

Clause 3 is the one that decides cases. Applied to the four capabilities it sorts them cleanly, and it is the reason clause 2's chart entry carries an extra obligation:

| capability | network off | verdict |
|---|---|---|
| type | system fallback, today's rendering | free |
| highlighting | plain mono code | free |
| mermaid | source text visible in `<pre class="mermaid">` | ugly, readable |
| chart | nothing, unless the data is in the HTML | needs the rule below |

### A chart is a figure containing both a rendering and its data

```html
<figure>
  <div data-chart="..."></div>
  <figcaption>The claim the chart supports.</figcaption>
  <details class="more"><summary>Data</summary><table>...</table></details>
</figure>
```

This is what makes a CDN chart acceptable rather than a regression. Offline the table is the evidence, online it is the appendix, and for a screen reader it is the accessible alternative to a picture. One structure, three payoffs.

The cost is two representations of the same numbers per figure, which is duplication and a new way to be subtly wrong. Accepted, because the alternative is a report whose central exhibit is an empty box.

### The chart library must emit SVG, and its defaults are accepted

Canvas is disqualified, not merely disfavoured. A canvas chart prints as a bitmap, cannot be selected, carries no text for assistive technology, and takes its colours from JavaScript configuration rather than from the palette variables, which means it needs bespoke handling on every theme change. These reports get exported to PDF and attached to tickets.

**Resolved during implementation: Frappe Charts, not Observable Plot.** Plot's UMD build does not bundle D3; it requires D3 loaded as a separate global first, so the real cost of "Observable Plot" is Plot (209KB) plus D3 (280KB), roughly 489KB, not Plot alone. That is not "Plot's weight with D3 bundled proving excessive" as a hypothetical, it consumes the entire non-mermaid share of the proposal's own worst-case budget by itself. Built one real line chart in each library against the same dataset: both rendered real, readable axis text (Plot: `30k`, `40k`, ...; Frappe: `0`, `100000`, ... and full ISO date labels), so the authoring-quality argument for Plot did not survive contact with a real comparison either. Frappe Charts, single script, 69KB, no companion CSS. Its one real gap is no log-scale axis; the catalog's worked example was chosen to not need one.

Library defaults, including gridlines and legends, are accepted rather than constrained. A house chart style was considered and rejected as a place to spend design effort that has no evidence behind it yet.

Static SVG generated at composition time was considered seriously and rejected on one fact: `figure svg { width: 100%; height: auto }` with a `viewBox` scales the entire coordinate system, so a 12px axis label in a `0 0 1000 300` viewBox renders at roughly 4px in a 340px column. That is why the catalog's current example has no `<text>` at all. Only a renderer that redraws on resize can keep labels at a constant size, and that rules out a static file.

### highlight.js gets a hand-written map onto the palette, not a stock theme

Around fifteen rules mapping `.hljs-comment` and friends onto the existing variables: comments to `--dim`, strings to a single hue that is neither green nor red, keywords by weight rather than colour.

A stock highlight.js theme would paint string literals green and numbers red inside a page whose entire discipline is that green means confirmed and red means failure. The reader has been trained by the banner, the badges and the stats; a rainbow code block spends that training. The archived change made the same kind of call when it rejected a capitals scan because `CONFIRMED` and `INFERRED` are legitimate content.

The cost is that restrained highlighting is worth less than full highlighting, and that a hand-written map drifts as the library adds token classes. Both accepted: the value here is modest either way, and the map failing to cover a class degrades to unstyled, which is exactly the offline behaviour.

### Mermaid sits beside `.steps` and ASCII, and guidance discourages it

`.steps` already covers causal chains, rollout timelines and remediation plans, which is most of what these reports diagram. Mermaid earns roughly a megabyte only for shapes `.steps` cannot linearise: dependency graphs, sequence diagrams, state machines.

The composer's instinct will be to reach for mermaid whenever it would draw a box, which would put that megabyte on reports that need three boxes and an arrow. The catalog entry therefore leads with when not to use it.

Its own convention, `<pre class="mermaid">`, gives the offline fallback for free: the source stays visible and a person can read it.

### One `themechange` event, three consumers

The toggle dispatches a single event. The chart re-renders, mermaid re-runs, and highlighting needs nothing because its map is expressed in palette variables that flip natively.

Without this, two of the three capabilities ship stale in dark mode, and nobody notices because reports get eyeballed in light. This is pure coupling cost on a template that currently has none, and it exists only because of the three capabilities above.

### Delimited `{{TOKEN}}` placeholders, overturning the archived decision on `DATE`

`2026-08-05-fix-generate-html-skill` decided to keep `DATE` duplicated across masthead and footer, reasoning that the collision is self-correcting: an author who writes the naive replace corrupts the footer, but the surviving text trips a different assert. That reasoning is correct about false negatives and silent about false positives. The same assert list runs `'DATE' not in html` against the entire composed document, so it fails a correct report about a SQL migration, or one with a `LAST UPDATED` column.

Delimited tokens remove both directions at once and generalise that change's own rejection of a capitals scan: the problem in both cases is a check whose vocabulary overlaps with legitimate content, and a delimiter is the fix.

Consequences: one mapping instead of eight ordered `.replace()` calls, one assertion instead of nine, the whitespace-sensitive contents-list match disappears, and the paragraph explaining the `DATE` hazard is deleted rather than rewritten. `{{DATE}}` may appear at both positions and resolve from one entry, which is correct because it is one date.

### Section wrappers, and a real scrollspy

Each section becomes `<section id aria-labelledby>`. The current spy runs `getBoundingClientRect().top <= 120` over every heading on every scroll event, with a separate `atBottom` special case so the final entry ever highlights. Section bounds plus an IntersectionObserver with a `rootMargin` biased to the top band replaces both the linear scan and the special case.

The cost is one more nesting level the composer must close, which is a new class of malformed output. It is mitigated by the catalog carrying the wrapper in every section snippet rather than describing it in prose.

### Two counters, one source, plus a composition check

Heading numbers come from a CSS counter over sections; contents numbers come from the existing counter over the contents list. These are two counters, and the spec requires one mechanism.

They are one mechanism: document order. They cannot disagree so long as the contents list has exactly one entry per section, which composition must already guarantee, because `references/components.md:7` requires section ids to match contents hrefs or the sidebar highlight goes dead. The check is therefore a composition-time assertion that the counts are equal and the nth entry targets the nth section, not a runtime mechanism.

Numbering only the headings and dropping numbers from the contents list was the alternative. Rejected: the sidebar is where a reader scans for a section, and a bare list is harder to scan than a numbered one.

### The report reads and writes the reader's theme key

The report reads `hyperreader-theme` before first paint and writes the same key when toggled, so the preference is one preference across the product: set it in the list, it holds in the document; toggle it in the document, the list agrees when you return.

This couples the document to a HyperReader implementation detail. Accepted. These documents are delivered to and read in HyperReader, and the portability premise that would have argued against the coupling is the same premise this change removes. A document opened outside HyperReader finds no key and falls back to `prefers-color-scheme`, which is the correct behaviour there.

Reports already delivered keep writing the old key until each is reopened. Not worth a migration: the population is small and the failure mode is one wrong theme on one old document.

### Sticky table headers stay removed

`2026-08-05-fix-generate-html-skill` removed `position: sticky` from `thead th` because `.scroll { overflow-x: auto }` computes `overflow-y` to `auto`, making `.scroll` the scroll container for the sticky header, and `.scroll` has no height cap. The fix is a cap or a deletion, and a cap creates a scroll region inside a page that already scrolls and truncates tables in print.

Reconsidered here and upheld. These reports are read top to bottom rather than filtered like a data grid, and the spec's existing scenario is satisfied by the catalog not claiming the pin.

### Print opens every disclosure, globally

`beforeprint` opens all disclosures, `afterprint` restores. Printed reports get longer, and someone printing for the summary now gets the raw dumps too.

A per-disclosure opt-out was considered and rejected as a knob nobody will set correctly. The composer already made the include-or-exclude decision when it chose to put something behind a disclosure rather than leaving it out; print should honour that decision rather than add a second one.

### The type scale carries the risk, and is sequenced last among the substantive work

Every other item here is additive or a localised fix. The type scale rewrites the basis every component sits on, it has the largest regression surface in the change, and it cannot be judged by measurement.

It is therefore sequenced behind the defect fixes and behind the font landing, so that it is tuned against Plex as actually rendered rather than against an intended face. The spec deliberately carries only the monotonic-heading invariant from this work: sizes, leading and measure are judged by eye, and a numeric target in a behaviour contract would encode taste as law and invite the next author to satisfy the number instead of the page.

## Risks / Trade-offs

- **Worst-case report weight rises from roughly 15KB to roughly 1.5MB, of which mermaid is two thirds** → Mitigated by conditional emission, which is why the `{{EXTERNAL}}` slot is not optional. A report with no chart, no code and no diagram carries only the two font files.
- **Fonts are the first always-on network dependency, so "loads nothing external" stops being true** → Accepted, and stated plainly in the rule rather than finessed. The degradation is the best of the four capabilities: no fetch means today's rendering.
- **Metric matching is fiddly, and getting it wrong produces a visible reflow on every first read** → Verified by blocking the font and comparing screenshots rather than by trusting override values.
- **Two of three external capabilities go stale on theme change** → The `themechange` event, and an explicit verification step that toggles the theme after everything has rendered. This is the defect most likely to ship, because reports are eyeballed in light mode.
- **The type scale cannot be validated by measurement, and the previous attempt at this work shipped a heading smaller than its body while its own verification table reported PASS on everything it thought to count** → The verification method is rendering at three widths in two themes and looking, with five invariants that each caught a real defect. The monotonic-heading invariant exists specifically because that defect has now survived two passes.
- **Delimited tokens change the composition idiom, so the worked example and the assertion in `SKILL.md` both change shape** → Contained: no delivered report changes, and the change is strictly a reduction in the number of moving parts.
- **A chart's rendering and its data table can drift apart** → Both are emitted from the same values at composition time. The catalog shows them emitted together rather than authored separately.

## Open Questions

- ~~Should the chart library be Observable Plot or Frappe Charts?~~ Resolved: Frappe Charts. See "The chart library must emit SVG" above.
- Scroll position memory per document is deferred rather than rejected. It is a genuine convenience on a long report, and the reason it is out is that it restores against a layout still settling behind a font swap and asynchronous chart rendering. If the font swap becomes imperceptible under metric matching, the objection weakens and it is worth revisiting.
