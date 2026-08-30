---
name: generate-html
description: Render a long-form deliverable as a single self-contained HTML page styled for browser reading. Use when finishing an investigation write-up, MR or code review, incident analysis, research report, migration plan or document summary that runs longer than a few screens, and whenever the user asks for a report "as HTML", "as a page", "in the browser", or wants something nicer to read than terminal markdown.
metadata:
  tags:
    - reporting
    - docs
  author: fmendonca2
  version: "2.0.0"
---

# HTML Report

A single HTML page built from a fixed shell in the Booking Pages design system and delivered to HyperReader, so it appears in the always-open reader and reads like a native Booking Pages page. A report built this way is also a valid Booking Pages artifact.

## When to use

- A report, review, investigation, assessment or summary long enough that a reader will scroll and skim.
- Findings carry structure that markdown flattens: headline numbers, per-item verdicts, comparison tables, timelines, charts.

## When NOT to use

- A short answer. Answer in chat.
- A note headed for the Obsidian vault. Use `obsidian-notes`.
- A document that lives in the repo and gets diffed and reviewed: ADR, spec, plan, README. Keep those markdown.

Both wanted? Write the markdown first, then render this page from it.

## Design system (adopted from Booking Pages)

The shell and every element follow the Booking Pages artifact design system. Compose the body with Tailwind utilities and these rules — there is no separate component catalog.

1. **Container.** `max-w-5xl mx-auto px-4` outer (matches the shell); prose in an inner `max-w-2xl` reading column; cards, tables and figures span the full width. Structure the body with `<main>`'s `<section id="...">` children.
2. **Tailwind only**, from `cdn.tailwindcss.com` (already in the shell). No other framework, no CSS file. Add to the composed string's `<style>` only for what Tailwind can't express, with a comment saying why.
3. **Flat.** Hierarchy from `border`/`border-gray-200`, `rounded-*` and solid `bg-*` fills. Never `shadow-*`, `ring-*` decoration, depth gradients, or `backdrop-blur-*`.
4. **Type.** Body `text-gray-700 leading-relaxed`; headings `font-bold text-gray-900`; muted `text-gray-500`; inline code `font-mono text-sm bg-gray-100 px-1.5 py-0.5 rounded`. System font stack; no external fonts.
5. **Palette.** Brand tokens are configured on the CDN — `brand action accent constructive` (+ `action-hover`) — and carry chrome. Status carries meaning on the built-in Tailwind scales: red failure/risk, amber caution, green (`constructive`) confirmed/healthy, violet inference, `action`/blue neutral. A hard-coded hex for status is noise. The template auto-themes this enumerated palette for dark, so composing with these neutral grays and the built-in status scales — rather than arbitrary hexes — is what lets a report render correctly in both themes.
6. **Animation.** `.reveal` fade-up (the shell defines the class and reveals on scroll via IntersectionObserver); stagger with `data-delay` in ms; upward only; 0.3–0.5s. Use sparingly on section-body elements (stat cards, steps) — never on the masthead or the verdict.
7. **External scripts: conditional, allowlisted, never load-bearing.** Only from `aistudio.booking.com`, `cdn.tailwindcss.com`, `cdnjs.cloudflare.com`, `unpkg.com`, `cdn.jsdelivr.net`. Only charts, highlight.js and mermaid load, only when the body uses them, emitted into `{{EXTERNAL}}`. The argument survives them being blocked: a chart ships its data table in the same `<figure>`, code degrades to plain mono, a diagram's source stays visible in `<pre class="mermaid">`. Styling itself needs the Tailwind CDN — inherent to this look; content and data stay in the DOM when it is unreachable.
8. **Security** (keeps the report a valid Booking Pages artifact): no inline event handlers (`onclick`, `onload`, …); no `javascript:` or HTML/SVG `data:` URIs; no `<form>`, `<base>`, `<object>`, `<embed>`; no `<meta http-equiv="refresh">`; no protocol-relative sources. Inline `<style>` and inline `<script>` blocks are fine.

Everywhere: no emoji (colour and labels carry status); dates ISO and numbers with units (`2026-07-08`, `67 msgs/sec`, `p99 340 ms`); cite file+line, dashboard, commit or ticket, and tag reasoned-not-observed claims `INFERRED`.

## Workflow

### 1. Structure before writing

1. **Lede**: the conclusion in one or two sentences, in the masthead.
2. **Verdict**: the outcome, expanded to a short paragraph, as the first thing under the masthead.
3. **Stats**: three to five headline numbers that carry the argument (more than six stops being a summary).
4. **Sections**: numbered, one `id` each, ordered so a reader can stop after any one of them.
5. **Appendix**: raw evidence, queries and dumps, behind a `<details>` disclosure.

Name sections for what they establish ("The capacity ceiling"), not the activity that produced them ("Metrics analysis").

The report works at three depths, and a reader may stop at any of them:

- **Masthead alone**: eyebrow, title, lede and meta state the outcome.
- **Masthead + verdict + stats**: the outcome expands to a paragraph and the headline numbers are visible.
- **In full**: every section.

Each depth must stand on its own. Don't put a fact only in a section a skimming reader would never reach.

### 2. Compose, check, and send in one eval cell

In a single `eval` cell: read the template into memory, fill a mapping of placeholder to value, substitute every `{{TOKEN}}` in one pass, assert none survive, then call `mcp__hyperreader_send_html`. Reading the template in the `eval` scope keeps it out of conversation context. No file touches disk.

```python
import re

template = read('skill://generate-html/assets/template.html')
body = '<div class="border-l-4 border-red-600 bg-red-50 rounded-r-lg p-5 mb-10">...</div>...'  # per "Composing the body"

def external_resources(body: str) -> str:
    entries = []
    if 'class="chart"' in body:
        entries.append('<script src="https://cdn.jsdelivr.net/npm/frappe-charts@1.6.2/dist/frappe-charts.min.umd.js"></script>')
    if "language-" in body:
        entries.append('<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/styles/github.min.css">')
        entries.append('<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/highlight.min.js"></script>')
    if 'class="mermaid"' in body:
        entries.append('<script src="https://cdn.jsdelivr.net/npm/mermaid@11.16.1/dist/mermaid.min.js"></script>')
    return "\n".join(entries)

values = {
    "KIND": "TROUBLESHOOTING",
    "TITLE": "The Capacity Ceiling",
    "LEDE": "The 2026-07-08 flip pushed demand permanently over the consumer ceiling.",
    "DATE": "2026-07-10",
    "SCOPE": "kafka-consumer",
    "SOURCES": "grafana, flog",
    "CONTENTS": (
        '<li><a class="hover:text-action-hover" href="#ceiling">The capacity ceiling</a></li>'
        '<li><a class="hover:text-action-hover" href="#trigger">What changed on 07-08</a></li>'
        '<li><a class="hover:text-action-hover" href="#options">Options</a></li>'
    ),
    "CONTENT": body,
    "CAVEATS": "Prometheus queries only; no client logs.",
    "EXTERNAL": external_resources(body),
}

html = re.sub(r"\{\{(\w+)\}\}", lambda m: values[m.group(1)], template)

# The only check a delimited placeholder needs: nothing shaped like one
# survives. Reports legitimately contain UPDATE, VALIDATE, a SOURCES
# column; none of that resembles `{{...}}`, so it never trips this.
remaining = re.findall(r"\{\{\w+\}\}", html)
assert not remaining, f"unfilled placeholder(s): {remaining}"

slug = "capacity-ceiling-2026-07-10"
result = tool.mcp__hyperreader_send_html({
    "slug": slug,
    "name": values["TITLE"],
    "html": html,
    "description": values["LEDE"],
})
# The permalink is deterministic once the slug is chosen; only the port
# varies, so pull that out of the result text rather than guessing it.
m = re.search(r"View it at (\S+)", result["text"])
display(m.group(1) if m else result["text"])
```

`{{DATE}}` appears at both the masthead and the footer and resolves from the single `values["DATE"]` entry: one date, filled once, so both positions agree by construction.

`send_html` arguments:

- `slug` (required): `^[a-z0-9]+(-[a-z0-9]+)*$`, max 80 chars. Sending the same slug again patches that exact page in place (see "Updating a report").
- `name` (required): the report title.
- `html` (required): the composed string.
- `description`: the lede, max 200 chars (rejected, not truncated, if over). Shown under the title in HyperReader's list.

`send_html` returns text of the form `Page "<name>" (slug=<slug>) created via serve on port <port>. View it at http://localhost:<port>/api/pages/<slug>/content` (`updated` when the slug existed). That trailing URL is the report's permalink; the cell extracts it. If the parse fails, hand over the returned text verbatim. If `send_html` fails (serve not running), state the error; optionally write the string to a file as a fallback.

### 3. The `{{EXTERNAL}}` slot

Reports are opened full-page, unsandboxed, by HyperReader, so external resources load. Tailwind is loaded unconditionally in the shell `<head>` (it is the framework, not a per-report capability), and fonts are the system stack, so no font entry is emitted. `{{EXTERNAL}}` carries only the conditional capability entries, chosen by inspecting the composed body — do not ask the author to declare them:

- `class="chart"` &rarr; frappe-charts UMD from `cdn.jsdelivr.net`.
- `language-` &rarr; highlight.js from `cdnjs.cloudflare.com` plus its `github.min.css` light theme.
- `class="mermaid"` &rarr; mermaid from `cdn.jsdelivr.net`.

A report using none emits an empty `{{EXTERNAL}}`. The shell's own script auto-highlights and runs mermaid only when their libraries actually loaded, so nothing else changes per report.

## Composing the body

Compose freely with Tailwind following the design system above. These are starting patterns, not an exhaustive catalog; a body is bpages-idiomatic Tailwind HTML. Colour variants are class swaps on the built-in scales (bad `red`, warn `amber`, good `green`, inference `violet`, neutral `action`/`blue`).

- **Section** (target of a table-of-contents `href`):
  ```html
  <section id="ceiling" class="mb-12 scroll-mt-6">
    <h2 class="text-2xl font-bold text-gray-900 mb-4">1. The capacity ceiling</h2>
    <p class="max-w-2xl leading-relaxed mb-4">Prose in a max-w-2xl reading column.</p>
  </section>
  ```
- **Verdict banner** (one, first under `<main>`; first sentence carries the conclusion):
  ```html
  <div class="border-l-4 border-red-600 bg-red-50 rounded-r-lg p-5 mb-10">
    <p class="font-semibold text-gray-900 mb-1">The 2026-07-08 flip pushed a demand curve already grazing the consumer ceiling permanently over it.</p>
    <p>Nothing regressed. All three incoming hypotheses are falsified by direct measurement.</p>
  </div>
  ```
- **Stat grid** (3–5; colour the top border by status):
  ```html
  <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 mb-10">
    <div class="border border-gray-200 border-t-4 border-t-red-600 rounded-lg p-4 bg-white">
      <div class="text-2xl font-bold text-gray-900">30K &rarr; 378K</div>
      <div class="text-sm text-gray-500 mt-1">daily peak lag, 07-07 vs 07-09</div>
    </div>
  </div>
  ```
- **Card with a status badge**:
  ```html
  <div class="border border-gray-200 rounded-lg p-5 bg-white">
    <div class="flex items-center justify-between gap-3 mb-2">
      <h3 class="text-lg font-semibold text-gray-900">Consumer capacity</h3>
      <span class="inline-block text-xs font-semibold px-2 py-0.5 rounded bg-green-100 text-green-800">CONFIRMED</span>
    </div>
    <p>Ceiling is fixed at 67&ndash;76 msgs/sec across June and July.</p>
  </div>
  ```
  Evidence badges: `bg-green-100 text-green-800` CONFIRMED · `bg-violet-100 text-violet-800` INFERRED · `bg-amber-100 text-amber-800` UNVERIFIED.
- **Callout** (aside; border + tint + label colour by status):
  ```html
  <div class="border border-amber-200 bg-amber-50 rounded-lg p-4 my-6">
    <div class="text-xs font-semibold uppercase tracking-wide text-amber-700 mb-1">Blast radius</div>
    <p>Partition count cannot be lowered again. This is a one-way door.</p>
  </div>
  ```
- **Table** (wrap in `overflow-x-auto`; numeric cells `text-right tabular-nums`):
  ```html
  <div class="overflow-x-auto my-6">
    <table class="w-full text-sm border-collapse">
      <thead><tr class="border-b border-gray-300 text-left">
        <th scope="col" class="py-2 pr-4 font-semibold text-gray-900">Hour UTC</th>
        <th scope="col" class="py-2 pr-4 font-semibold text-gray-900 text-right">07-08</th>
      </tr></thead>
      <tbody><tr class="border-b border-gray-200">
        <td class="py-2 pr-4">12</td><td class="py-2 pr-4 text-right tabular-nums">73.7</td>
      </tr></tbody>
    </table>
  </div>
  ```
- **Diff** (hand-coloured; no `language-*`) and **highlighted code** (exactly one `language-*` class — the marker that loads highlight.js + auto-highlights; a block is one or the other, never both, since the highlighter re-tokenizes and discards hand spans). Escape `<` `>` `&` inside `<pre>`.
  ```html
  <pre class="bg-gray-100 rounded-lg p-4 overflow-x-auto text-sm font-mono my-6"><code><span class="block bg-red-50 text-red-800">- throw new ApiResponseUnknownSessionException();</span>
  <span class="block bg-green-50 text-green-800">+ return LexisNexisErrorReason.UNKNOWN_SESSION;</span></code></pre>

  <pre class="border border-gray-200 rounded-lg overflow-x-auto text-sm my-6"><code class="language-go">func handle(w http.ResponseWriter) { http.Error(w, "unknown session", 401) }</code></pre>
  ```
- **Steps** (`list-decimal` with a styled marker; reach for this or ASCII before mermaid):
  ```html
  <ol class="list-decimal marker:text-action marker:font-bold pl-6 space-y-4 my-6">
    <li class="pl-1"><span class="block font-semibold text-gray-900">Raise partitions 8 &rarr; 32</span>One-way door; needs a keyed-consumer audit first.</li>
  </ol>
  ```
- **Chart + data table** (`class="chart"` is the marker that loads frappe-charts; always pair it with the data table, emitted from the same values so they cannot drift; label units in the caption and dataset name; render in a per-figure inline `<script>` guarded on `window.frappe`):
  ```html
  <figure class="my-8">
    <div class="chart border border-gray-200 rounded-lg p-4 bg-white" id="chart-ceiling" role="img" aria-label="Daily peak throughput, msgs/sec, 2026-05 to 2026-07.">
      <p class="text-sm text-gray-500">Chart unavailable &mdash; see Data below.</p>
    </div>
    <figcaption class="text-sm text-gray-500 mt-2">Daily peak throughput, msgs/sec.</figcaption>
    <details class="mt-2"><summary class="cursor-pointer text-sm text-action">Data</summary>
      <div class="overflow-x-auto mt-2"><table class="w-full text-sm border-collapse">
        <thead><tr class="border-b border-gray-300 text-left"><th scope="col" class="py-1 pr-4 font-semibold text-gray-900">Date</th><th scope="col" class="py-1 pr-4 font-semibold text-gray-900 text-right">Peak</th></tr></thead>
        <tbody><tr class="border-b border-gray-200"><td class="py-1 pr-4">2026-07-09</td><td class="py-1 pr-4 text-right tabular-nums">96</td></tr></tbody>
      </table></div>
    </details>
  </figure>
  <script>
  (() => {
    const el = document.getElementById("chart-ceiling");
    const labels = ["2026-05-01", "2026-06-01", "2026-07-09"];
    const values = [68, 71, 96];
    if (window.frappe && el) {
      el.replaceChildren();
      new frappe.Chart(el, { data: { labels, datasets: [{ name: "Peak msgs/sec", values }] }, type: "line", height: 260, colors: ["#006CE4"], axisOptions: { xIsSeries: true } });
    }
  })();
  </script>
  ```
  For a figure that isn't a plot (topology, mechanism), use inline `<svg viewBox=... class="w-full h-auto" role="img" aria-label="...">` with `#d1d5db` strokes.
- **Mermaid** (`class="mermaid"` is the marker that loads mermaid; the shell inits and runs it; the source is the offline fallback). Reach for it only when steps/ASCII can't linearise the shape (dependency graph, sequence diagram, state machine) — it loads ~1 MB.
  ```html
  <pre class="mermaid my-8 text-sm">
  graph LR
    A[janus] --> B[device-reputation] --> C[(mysql)]
  </pre>
  ```
- **Appendix** (evidence dumps behind a disclosure):
  ```html
  <details class="border border-gray-200 rounded-lg my-6">
    <summary class="cursor-pointer px-4 py-3 font-semibold text-gray-900">Query appendix</summary>
    <div class="px-4 pb-4"><pre class="bg-gray-100 rounded-lg p-4 overflow-x-auto text-sm font-mono"><code>sum(kafka_consumergroup_lag{group="janus__data__prod"})</code></pre></div>
  </details>
  ```

Keep the table of contents and the sections in step: one `{{CONTENTS}}` `<li>` per section, the nth `href` targeting the nth section `id`. If numbering, number the heading and the contents entry the same.

## Verification

Render the composed report and read the page; no metric substitutes for looking. Open it in a browser at 320px, a mid tablet width, and a wide desktop width, and check:

1. **No heading renders smaller than the text it heads**, at every width.
2. **No horizontal scroll at 320px** (tables scroll inside their `overflow-x-auto` wrapper, not the page).
3. **Every chart carries its data table**; open the disclosure and confirm the numbers match the plot.
4. **With the capability scripts blocked, the report still reads**: the chart shows its table, code shows plain mono, mermaid shows its source, and every claim, number and relationship is present. (Blocking the Tailwind CDN too leaves the page unstyled but with all content intact — expected for this look.)
5. **Flat design holds**: no `shadow`/`ring`/gradient depth anywhere.
6. **Every `.reveal` element ends visible**: scroll the whole report and confirm nothing stays at `opacity: 0`; the masthead and verdict are visible without scrolling.
7. **Both themes render correctly**: HyperReader is dark by default; toggle to light. Confirm the page, cards, tables, code, and any chart or diagram are legible in each, with no flash of the wrong theme when the report first opens.

## Updating a report

Re-run the workflow and send again with the **same slug**. HyperReader patches the existing page in place (full-body replacement, `200` not `201`) and the row moves to the top of the list via SSE. Only `{{CONTENT}}`, `{{CONTENTS}}` and `{{EXTERNAL}}` typically change; the shell is stable. Reach for a new slug only when the report is genuinely a different page — reusing an old slug for an unrelated report overwrites it irrecoverably.
