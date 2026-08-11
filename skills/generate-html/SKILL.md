---
name: generate-html
description: Render a long-form deliverable as a single self-contained HTML page styled for browser reading. Use when finishing an investigation write-up, MR or code review, incident analysis, research report, migration plan or document summary that runs longer than a few screens, and whenever the user asks for a report "as HTML", "as a page", "in the browser", or wants something nicer to read than terminal markdown.
metadata:
  tags:
    - reporting
    - docs
  author: fmendonca2
  version: "1.0.0"
---

# HTML Report

A single HTML page built from a fixed shell and a fixed component vocabulary, delivered to HyperReader so it appears in the always-open reader, so every report reads like it came from the same place.

## When to use

- A report, review, investigation, assessment or summary long enough that a reader will scroll and skim.
- Findings carry structure that markdown flattens: headline numbers, per-item verdicts, comparison tables, timelines, charts.

## When NOT to use

- A short answer. Answer in chat.
- A note headed for the Obsidian vault. Use `obsidian-notes`.
- A document that lives in the repo and gets diffed and reviewed: ADR, spec, plan, README. Keep those markdown.

Both wanted? Write the markdown first, then render this page from it.

## Workflow

### 1. Structure before writing

1. **Lede**: the conclusion in one or two sentences, in the masthead.
2. **Banner**: the verdict, expanded to a short paragraph.
3. **Stats**: the numbers that carry the argument.
4. **Sections**: numbered, one `id` each, ordered so a reader can stop after any one of them.
5. **Appendix**: raw evidence, queries and dumps, behind a disclosure.

Name sections for what they establish ("The capacity ceiling"), not for the activity that produced them ("Metrics analysis").

The report works at three depths, and a reader may stop at any of them:

- **Masthead alone**: the eyebrow, title, lede and meta line state the outcome. A reader who reads nothing else still has the conclusion.
- **Masthead plus banner plus stats**: the verdict expands to a paragraph, and the headline numbers are visible. A reader who stops here has the argument, not just the conclusion.
- **In full**: every section, in the order a reader can stop after any one of them.

Each depth must stand on its own. Don't put a fact only in a section that a reader skimming the first two depths would never see.

### 2. Compose, check, and send in one eval cell

In a single `eval` cell: read the template into memory, fill a mapping of placeholder to value, substitute every `{{TOKEN}}` in one pass, assert none survive, then call `mcp__hyperreader_send_html`. The stylesheet is the bulk of the file and never changes; reading it in the `eval` scope keeps it out of conversation context entirely. No file touches disk.

Read [references/components.md](references/components.md) for the copy-paste snippets to compose inside `{{CONTENT}}`: banner, stats, cards, callouts, evidence badges, tables, diffs, steps, meters, charts, highlighted code, diagrams, appendices.

```python
import re

template = read('skill://generate-html/assets/template.html')
body = '<div class="banner bad">...</div>...'  # sections, wrapped per components.md

values = {
    "KIND": "TROUBLESHOOTING",
    "TITLE": "The Capacity Ceiling",
    "LEDE": "The 2026-07-08 flip pushed demand permanently over the consumer ceiling.",
    "DATE": "2026-07-10",
    "SCOPE": "kafka-consumer",
    "SOURCES": "grafana, flog",
    "CONTENTS": (
        '<li><a href="#ceiling">The capacity ceiling</a></li>'
        '<li><a href="#trigger">What changed on 07-08</a></li>'
        '<li><a href="#options">Options</a></li>'
    ),
    "CONTENT": body,
    "CAVEATS": "Prometheus queries only; no client logs.",
    "EXTERNAL": external_resources(body),  # see "The {{EXTERNAL}} slot" below
}

html = re.sub(r"\{\{(\w+)\}\}", lambda m: values[m.group(1)], template)

# The only check a delimited placeholder needs: nothing shaped like one
# survives. Reports legitimately contain UPDATE, VALIDATE, a SOURCES
# column; none of that resembles `{{...}}`, so it never trips this.
remaining = re.findall(r"\{\{\w+\}\}", html)
assert not remaining, f"unfilled placeholder(s): {remaining}"

result = tool.mcp__hyperreader_send_html({
    "name": "capacity-ceiling-2026-07-10",
    "html": html,
    "description": values["LEDE"],
    "tags": "troubleshooting,kafka"
})
# Hand over the report's own permalink, not the list view
m = re.search(r'id=(\d+).*?port (\d+)', result["text"])
display(f"http://localhost:{m.group(2)}/api/documents/{m.group(1)}/content" if m else result["text"])
```

`{{DATE}}` appears at both the masthead and the footer and resolves from the single `values["DATE"]` entry: one date, filled once, in a mapping, means both positions agree by construction rather than by discipline. There is nothing left to get wrong here; the old failure mode was a `.replace()` on the DATE token's whole enclosing sentence, done twice, which this mapping has no equivalent step for.

`send_html` arguments:

- `name` (required): the report title, kebab-case.
- `html` (required): the composed string from the `eval` scope.
- `description`: the lede sentence from the masthead. This is what appears under the title in HyperReader's list view, so it should summarize the outcome in one line.
- `tags`: the eyebrow (report kind) from the masthead, lowercased. Add more tags if useful, comma-separated.

`send_html` returns text of the form `Document "<name>" ingested (id=<id>) via serve on port <port>. View it at http://localhost:<port>/`. That trailing URL is the list view, not the report, so the cell above rebuilds the permalink from the `id` and port it carries. Hand over `http://localhost:<port>/api/documents/<id>/content`: it opens the report full-page, and it is the same URL HyperReader opens when a reader clicks a row. If the parse fails, hand over the returned text verbatim rather than a guessed URL. The report appears live in HyperReader's list via SSE the moment it is ingested.

If `send_html` fails (serve not running, connection refused), state the error. Optionally write the string to a file as a fallback so the user can open it directly.

### 3. The `{{EXTERNAL}}` slot

Reports are opened full-page, unsandboxed, by HyperReader itself (see Rules below), so external resources are allowed, on four conditions. `{{EXTERNAL}}` is where they go: a slot in `<head>`, filled at composition time with only the entries the report actually needs.

One entry is always emitted:

```python
FONTS = '''<link rel="preconnect" href="https://cdn.jsdelivr.net" crossorigin>
<style>
@font-face {
  font-family: "IBM Plex Sans Variable";
  font-style: normal;
  font-display: swap;
  font-weight: 100 700;
  src: url(https://cdn.jsdelivr.net/npm/@fontsource-variable/ibm-plex-sans@5.3.0/files/ibm-plex-sans-latin-wght-normal.woff2) format("woff2-variations");
  unicode-range: U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD;
}
@font-face {
  font-family: "IBM Plex Mono";
  font-style: normal;
  font-display: swap;
  font-weight: 400;
  src: url(https://cdn.jsdelivr.net/npm/@fontsource/ibm-plex-mono@5.3.0/files/ibm-plex-mono-latin-400-normal.woff2) format("woff2");
  unicode-range: U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD;
}
</style>'''
```

Three more load conditionally, one per external capability, and only when the composed body actually uses that capability. Inspect the composed body string; do not ask the author to declare what they used:

```python
def external_resources(body: str) -> str:
    entries = [FONTS]
    if 'class="chart"' in body:
        entries.append('<script src="https://cdn.jsdelivr.net/npm/frappe-charts@1.6.2/dist/frappe-charts.min.umd.js"></script>')
    if "language-" in body:
        entries.append('<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.11.1/highlight.min.js"></script>')
    if 'class="mermaid"' in body:
        entries.append('<script src="https://cdn.jsdelivr.net/npm/mermaid@11.16.1/dist/mermaid.min.js"></script>')
    return "\n".join(entries)
```

A report with no chart, no highlighted code and no diagram emits only the font entry. A report using all three emits exactly three script entries plus the font entry. The template's own script already no-ops each capability's initialization when its library never loaded, so nothing beyond this list needs to change per report.

## Choosing a component

| Situation | Component |
|---|---|
| The conclusion, in one place | Banner |
| A handful of headline numbers that carry the argument | Stats |
| Several independent facts, each worth its own space | Cards |
| A short aside that isn't load-bearing | Callout |
| Evidence level of a claim (measured vs reasoned vs unverified) | Badge |
| Rows and columns of comparable values | Table |
| A change to code or config | Diff (`.add`/`.del`/`.cm`) |
| Code worth reading for its own structure | Highlighted code block |
| A causal chain, rollout, or remediation plan | Steps |
| A single proportion | Meter |
| Numbers that trend or compare across categories | Chart |
| A dependency graph, sequence diagram or state machine | Mermaid |
| Anything else drawn by hand: a topology, a mechanism | Inline SVG |
| Raw dumps, full query text, long output | Appendix disclosure |

The default pull is toward prose and bullet lists. Resist it: a table skims where a paragraph of numbers doesn't, and a card grid skims where a bullet list of the same facts doesn't. Component choice is most of what makes a report skimmable.

## Rules

- **External resources are conditional and never carry the argument**, in priority order:
  1. Type always loads: IBM Plex Sans and IBM Plex Mono, variable woff2, `font-display: swap`, a metric-matched fallback so the swap does not reflow the page.
  2. Three capabilities load conditionally, emitted into `{{EXTERNAL}}` only when the report uses them: charts, syntax highlighting, mermaid.
  3. The argument survives the network being off. A chart therefore ships the data it plots, as a table inside the same `<figure>`; highlighting degrades to plain mono; a diagram's source stays visible in `<pre class="mermaid">`.
  4. Nothing external for layout or colour. The palette, the type scale, and every non-typeface visual rule live in the shell's own `<style>` block.
- **Palette only.** `--accent --red --amber --green --violet` carry status and `--dim --border --surface` carry structure; all of them flip between light and dark, and a hard-coded hex breaks in one theme.
- **Colour carries meaning.** Red for failure and risk, amber for caution, green for confirmed and healthy, violet for inference. Decoration that says nothing is noise.
- **Every claim traceable.** For investigations and reviews, cite the file and line, the dashboard, the commit, the ticket. Everywhere, tag claims that are reasoned rather than observed with the `INFERRED` badge.
- **No emoji.** Badges and colour already carry status.
- **Dates ISO, numbers with units.** `2026-07-08`, `67 msgs/sec`, `p99 340 ms`.
- **Stats budget: three to five.** More than six stops being a summary.
- **Chart axes budget: label every axis.** A chart with an unlabelled axis asks the reader to guess units; the chosen chart library draws real axes for exactly this reason, so there is no excuse to suppress them.
- **Mermaid weighs roughly a megabyte: reach for `.steps` or ASCII first.** `.steps` covers causal chains, rollout timelines and remediation plans, which is most of what these reports diagram. Mermaid earns its weight only on shapes `.steps` cannot linearise: dependency graphs, sequence diagrams, state machines. See references/components.md for the full case against reaching for it by default.

## Verification

Render the composed report and look at it: no metric here substitutes for reading the page. Open it in a browser at three widths (320px, a mid-size tablet width, and a wide desktop width past the 64rem sidebar breakpoint), in both themes, and check five invariants that have each caught a real defect in this template:

1. **No heading renders smaller than the text it heads**, at every width checked, in both themes. This includes headings inside cards.
2. **No horizontal scroll at 320px.**
3. **Every chart carries its data table.** Open the disclosure and confirm the numbers are the ones the chart plots.
4. **With the network blocked, the report still reads.** No figure is empty, the page is not unstyled, and every claim, number and relationship it asserts is still present.
5. **After toggling to dark with every component rendered, nothing is left in the previous theme's colours.** Load a report with a chart, highlighted code and a diagram in light mode, let everything render, then toggle. Two of the three capabilities go stale on a theme change if the toggle handler doesn't tell them to re-render; this is the defect most likely to ship, because reports get eyeballed in light mode.

## Updating a report

Re-run the workflow and send again. The shell is stable, so only `{{CONTENT}}`, `{{CONTENTS}}` and `{{EXTERNAL}}` change. HyperReader stores each send as a new document; re-sending an updated report creates a new entry, not an update to an old one.
