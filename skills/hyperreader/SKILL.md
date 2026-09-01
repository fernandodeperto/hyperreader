---
name: hyperreader
description: "Generate a Tailwind-based HTML page following the Booking Pages design ruleset (flat design, dark theme) and send it to the running HyperReader server via the send_html MCP tool. Use when the user wants to view, publish, send, or update an HTML page, report, or artifact in HyperReader — including code review pages that show each change as a syntax-highlighted diff with space for reviewer analysis."
metadata:
  tags:
    - html-artifacts
    - hyperreader
    - publishing
  version: "1.0.0"
allowed-tools:
  - Read
  - Bash(curl *)
  - Bash(command -v hyperreader)
  - Bash(hyperreader serve*)
  - Bash(./hyperreader serve*)
  - Bash(nohup *)
  - Bash(sleep *)
  - Bash(seq *)
  - mcp__hyperreader_send_html
---

# HyperReader

Render a single self-contained HTML page following the embedded Booking Pages
design ruleset and deliver it to the running HyperReader reader through the
`send_html` MCP tool, so it opens in HyperReader instead of scrolling past in
the terminal.

This skill is self-contained. Follow only the bundled `references/guidelines.md`,
`assets/template.html`, and `assets/code-review-template.html`. Do not fetch
external rulesets (including the bpages CLI), read other skills, or browse
other sources.

## Workflow

### 1. Prepare page identity

- **name** (required): the display title, plain language, sentence case.
- **slug** (required): kebab-case, must match `^[a-z0-9]+(-[a-z0-9]+)*$`, max 80
  characters. The slug is the identity key — reusing an existing slug patches
  that page in place (full-body replacement); a new slug creates a new page.
- **description** (optional, max 200 characters): plain language saying what the
  page is and what it helps with; keep the name out of it; one or two short
  sentences.

Show these to the user briefly, then proceed (the send is idempotent by slug and
local, so it is safe to run).

### 2. Generate the HTML

For a general page, read `skill://hyperreader/assets/template.html` and
`skill://hyperreader/references/guidelines.md`. For reviewing a
code change, read `skill://hyperreader/assets/code-review-template.html`
instead — it loads syntax highlighting and defines change-block + analysis
components. Both templates share the same `{{TITLE}}`/`{{SUBTITLE}}`/
`{{CONTENT}}` fill; for the review template, `{{CONTENT}}` is a verdict banner
plus one change block per change (diff via `language-diff-<lang>
diff-highlight`, plus an analysis panel; each change block is a numbered,
collapsible card) — see the "Code review components" section of
`references/guidelines.md`.

Fill `{{TITLE}}`, `{{SUBTITLE}}`, and `{{CONTENT}}`, authoring `<section>`
content that follows the guidelines: Tailwind via CDN, flat design, the
Booking brand palette, and `<main>`/`<section>` structure. Keep it a single self-contained document under 5 MB. Do not add a
theme toggle — the reader renders dark-only and strips embedded `.theme`
controls.

### 3. Ensure the reader is running

HyperReader `serve` must be listening (default port 7420) for `send_html` to
succeed. Probe it, and start it in the background if it is down:

    PORT="${HYPERREADER_PORT:-7420}"
    if ! curl -fsS -o /dev/null "http://127.0.0.1:${PORT}/api/pages" 2>/dev/null; then
      BIN="$(command -v hyperreader || echo ./hyperreader)"
      nohup "$BIN" serve >"${TMPDIR:-/tmp}/hyperreader-serve.log" 2>&1 & disown
      for _ in $(seq 1 50); do
        curl -fsS -o /dev/null "http://127.0.0.1:${PORT}/api/pages" 2>/dev/null && break
        sleep 0.2
      done
    fi

If neither `hyperreader` on PATH nor `./hyperreader` exists, tell the user the
reader binary is not installed and stop.

### 4. Send the page

Call the `send_html` MCP tool (`mcp__hyperreader_send_html`) with
`{ slug, name, html, description }`. If it returns a serve-unreachable error, run
step 3 and retry once.

### 5. Report

Tell the user the page name and slug, and that it is open in HyperReader at
`http://localhost:<PORT>/` (search by name or slug to view it). The raw HTML is
at `http://localhost:<PORT>/api/pages/<slug>/content`.
