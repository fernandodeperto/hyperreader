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

### 2. Compose, check, and send in one eval cell

In a single `eval` cell: read the template into memory, replace its markers with your composed content, assert no markers remain, then call `mcp__hyperreader_send_html`. The stylesheet is two hundred lines that never change; reading it in the `eval` scope keeps it out of conversation context entirely. No file touches disk.

Read [references/components.md](references/components.md) for the copy-paste snippets to compose inside the `replace`: banner, stats, cards, callouts, evidence badges, tables, diffs, steps, meters, SVG figures, appendices.

```python
import re

template = read('skill://generate-html/assets/template.html')

# Replace the CONTENT marker with your composed sections
html = template.replace('<!-- CONTENT -->', '<div class="banner bad">...</div>...')

# Fill the masthead, contents list, <title>, and footer
html = (html
    .replace('REPORT KIND', 'TROUBLESHOOTING')
    .replace('REPORT TITLE', 'The Capacity Ceiling')
    .replace('LEDE: one or two sentences stating the outcome, not the topic.', 'The 2026-07-08 flip pushed demand permanently over the consumer ceiling.')
    .replace('<li><a href="#section-1">Section one</a></li>',
             '<li><a href="#ceiling">The capacity ceiling</a></li>'
             '<li><a href="#trigger">What changed on 07-08</a></li>'
             '<li><a href="#options">Options</a></li>')
    .replace('<span>DATE</span><span>SCOPE</span><span>SOURCES</span>', '<span>2026-07-10</span><span>kafka-consumer</span><span>grafana, flog</span>')
    .replace('Generated DATE. METHOD AND CAVEATS.', 'Generated 2026-07-10. Prometheus queries only; no client logs.')
)

# Assert no shell marker survives. DATE catches either occurrence (see below).
for marker in ['REPORT KIND', 'REPORT TITLE', 'LEDE:', 'DATE', 'SCOPE', 'SOURCES',
               'Section one', '<!-- CONTENT -->', 'METHOD AND CAVEATS']:
    assert marker not in html, f"unfilled marker: {marker}"

result = tool.mcp__hyperreader_send_html({
    "name": "capacity-ceiling-2026-07-10",
    "html": html,
    "description": "The 2026-07-08 flip pushed demand permanently over the consumer ceiling.",
    "tags": "troubleshooting,kafka"
})
# Hand over the report's own permalink, not the list view
m = re.search(r'id=(\d+).*?port (\d+)', result["text"])
display(f"http://localhost:{m.group(2)}/api/documents/{m.group(1)}/content" if m else result["text"])
```

`DATE` appears twice in the shell: once in the masthead `<span>` run and once in the footer sentence. Fill each by replacing its whole enclosing string, as above, never the bare token, or filling one corrupts the other. The bare `DATE` in the assert list catches either occurrence being missed.

`send_html` arguments:

- `name` (required): the report title, kebab-case.
- `html` (required): the composed string from the `eval` scope.
- `description`: the lede sentence from the masthead. This is what appears under the title in HyperReader's list view, so it should summarize the outcome in one line.
- `tags`: the eyebrow (report kind) from the masthead, lowercased. Add more tags if useful, comma-separated.

`send_html` returns text of the form `Document "<name>" ingested (id=<id>) via serve on port <port>. View it at http://localhost:<port>/`. That trailing URL is the list view, not the report, so the cell above rebuilds the permalink from the `id` and port it carries. Hand over `http://localhost:<port>/api/documents/<id>/content`: it opens the report full-page, and it is the same URL HyperReader opens when a reader clicks a row. If the parse fails, hand over the returned text verbatim rather than a guessed URL. The report appears live in HyperReader's list via SSE the moment it is ingested.

If `send_html` fails (serve not running, connection refused), state the error. Optionally write the string to a file as a fallback so the user can open it directly.

## Rules

- **Self-contained.** One file. No CDN links, no external stylesheets, no image files, no chart libraries. It has to survive being attached to a ticket or opened on a plane.
- **Palette only.** `--accent --red --amber --green --violet` carry status and `--dim --border --surface` carry structure; all of them flip between light and dark, and a hard-coded hex breaks in one theme.
- **Colour carries meaning.** Red for failure and risk, amber for caution, green for confirmed and healthy, violet for inference. Decoration that says nothing is noise.
- **Every claim traceable.** For investigations and reviews, cite the file and line, the dashboard, the commit, the ticket. Everywhere, tag claims that are reasoned rather than observed with the `INFERRED` badge.
- **No emoji.** Badges and colour already carry status.
- **Dates ISO, numbers with units.** `2026-07-08`, `67 msgs/sec`, `p99 340 ms`.

## Updating a report

Re-run the workflow and send again. The shell is stable, so only `<main>`, the contents list and the footer change. HyperReader stores each send as a new document; re-sending an updated report creates a new entry, not an update to an old one.
