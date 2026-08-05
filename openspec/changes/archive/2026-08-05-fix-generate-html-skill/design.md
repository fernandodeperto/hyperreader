## Context

See proposal.md - Why. Three constraints shape every decision below.

**The skill is one copy, reached two ways.** `~/.agents/skills/generate-html` is a symlink to `skills/generate-html` in this repo. Edits land in the registered skill immediately; there is no sync step and no second copy to keep in step. This also closes the review's open question about which copy is canonical.

**`skill://html-report` resolves to nothing.** The dotfiles skill that used to answer that id is no longer installed (`~/.agents/skills/` has no `html-report`). The workflow's template read is dead as written, not merely misnamed.

**The delivery target already has a permalink, and the app already uses it.** `GET /api/documents/{id}/content` (`internal/api/api.go:81`) returns the stored HTML full-page; `web/app.js:407` opens documents with exactly that URL. The MCP result text already carries both the id and the port (`internal/mcp/server.go:183`).

## Goals / Non-Goals

**Goals:**

- Every fix lands in the three skill files. The change is verifiable by rendering one report and inspecting computed styles.
- The stylesheet contains no rule a supported browser discards or never matches.
- Prefer mechanisms that cannot drift over conventions an author must remember.

**Non-Goals:**

- No change to HyperReader's server, storage, MCP, or web app. The handover fix reads what `send_html` already returns rather than changing what it returns.
- No redesign of the visual language. Component vocabulary, palette, and evidence badges stay as they are; the review found them sound.
- No fallback palette for browsers without `light-dark()` (see Risks).

## Decisions

### Reference the skill by its own id, rather than renaming the skill

Change the template read to `skill://generate-html/assets/template.html`.

The alternative, renaming the skill to `html-report`, was rejected: `generate-html` is the name in the frontmatter, the symlink, and the harness registration. Renaming propagates to all three to fix a string in one.

### Split the shadow token into a colour, not into two full shadows

`--shadow-color: light-dark(rgb(0 0 0 / .08), rgb(0 0 0 / .5))`, with offsets moved to the two usage sites (`.card`, `.theme`).

`light-dark()` takes exactly two `<color>` arguments. Passing full shadow values makes the declaration invalid at computed-value time, so `box-shadow` falls back to `none` in both themes.

Alternative considered: keep one `--shadow` token and switch it under a `[data-theme]` selector. Rejected because it duplicates the offsets, and because every other token in the block flips through `light-dark()` - a single token switching by a different mechanism is the kind of inconsistency the next editor breaks.

### Assert the literal placeholder tokens, listed once

Extend the pre-delivery check to all nine shell placeholders: `REPORT KIND`, `REPORT TITLE`, `LEDE:`, `DATE`, `SCOPE`, `SOURCES`, `Section one`, `<!-- CONTENT -->`, `METHOD AND CAVEATS`.

Alternative considered: scan for any run of capitals. Rejected: the report body legitimately contains `CONFIRMED`, `INFERRED`, `UNVERIFIED` badge text, so a generic scan fires on correct reports.

### Let the assert carry the duplicate `DATE`, rather than renaming one occurrence

`DATE` appears in the masthead (`<span>DATE</span>`) and the footer (`Generated DATE.`). Keep the token identical in both places, keep the documented idiom of replacing the whole enclosing string, and let the bare token `DATE` in the assert list catch either occurrence being missed.

This makes the hazard self-correcting: an author who writes the naive `.replace('DATE', ...)` corrupts the footer, but the surviving `Generated <date>. METHOD AND CAVEATS.` still trips `METHOD AND CAVEATS`, and a skipped footer trips `DATE`. Renaming the footer token to something unique was considered; it removes the collision but leaves the naive replace silently half-working, which is worse than a failed assert.

### Drop the sticky table header rather than capping the scroll container

Delete `position: sticky; top: 0` from `thead th`. `.scroll` keeps `overflow-x: auto` for wide tables.

The rule cannot work as written: `overflow-x: auto` computes `overflow-y` to `auto`, which makes `.scroll` the scroll container for the sticky header, and `.scroll` has no height cap. The fix is either a cap or a deletion. A cap (`max-height: 70vh`) creates a scroll region inside a page that already scrolls, which fights the reading mode this skill exists to produce and truncates tables in print. These reports are read top to bottom, not filtered like a data grid, so the header pin buys little.

### Keep the generated TOC numbering, drop the hand-written heading numbers

`.toc a::before { content: counter(toc) "." }` stays. The catalog's `<h2 id="findings">2. Findings</h2>` idiom becomes `<h2 id="findings">Findings</h2>`.

Numbering that a CSS counter derives from document order cannot disagree with itself, and reordering sections costs nothing. The reverse choice - delete the counter, hand-number both - was rejected because it reintroduces exactly the drift the review found, and every reorder becomes a renumber.

### Split the theme bootstrap, moving only the two lines that must precede paint

A blocking `<script>` in `<head>` reads `localStorage` and sets `document.documentElement.dataset.theme`. The rest of the script - the toggle button wiring, `aria` sync, TOC open, scrollspy - stays at the end of `<body>`.

Only the theme application must precede first paint. The button handler cannot move: `.theme` does not exist while `<head>` is parsing, so `document.querySelector(".theme")` would return `null`.

### Build the handover URL from the id already in the result text

Parse `id=` and the port from `send_html`'s result text and hand over `http://localhost:<port>/api/documents/<id>/content`.

This matches what the app does when a user clicks a row (`web/app.js:407`), so the URL the agent states and the URL the reader reaches are the same. Changing `internal/mcp/server.go` to return the permalink directly would be cleaner, and is worth doing later, but it puts a Go change in a skill fix and the id is already available without it.

### Keep the skill at `skills/`, and document it there

`.omp/skills/` holds only vendored openspec tooling that the harness manages. `skills/` holds this repository's own authored skill, and the registration symlink points into it. The review read these as one convention split across two directories; they are two different things. Add a README entry pointing at `skills/generate-html/` rather than moving the directory and breaking the symlink.

### Delete `.badge + *` outright

An adjacent-*element* selector that never matches, because every documented badge usage is followed by a text node. Replacing it with `margin-right` on `.badge` was considered and rejected: it would add trailing space to badges that end a line, and the surrounding text nodes already carry their own spaces.

## Risks / Trade-offs

- **`light-dark()` has no fallback, and every token routes through it** (Chrome 123+, Safari 17.5+, Firefox 120+) → Accepted. An older browser loses the whole palette at once, which conflicts with the skill's "survives being attached to a ticket" claim. A fallback block would double the token list to serve browsers this audience does not run. Recorded here so the trade-off is deliberate rather than overlooked.
- **Dropping the sticky header is a small regression for long tables** → Accepted. Reports that need a filterable grid are the wrong deliverable for this skill.
- **The handover URL depends on the wording of `send_html`'s result text** → Parse defensively on `id=(\d+)` and the port, and fall back to stating the returned text verbatim if the parse fails. A later change to `internal/mcp/server.go` returning a structured permalink would remove the coupling.
- **The verification report must exercise every component, or a dead rule survives the audit** → The verification step composes a report that uses each catalog component at least once, and inspects computed styles in both themes rather than eyeballing a screenshot.

## Open Questions

- Should `send_html` return the permalink directly instead of the list URL? Deferred: it is a server-side improvement that would simplify the skill later, but it changes neither the specs nor any task here.
