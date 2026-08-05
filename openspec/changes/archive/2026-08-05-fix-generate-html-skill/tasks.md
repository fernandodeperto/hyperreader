## 1. Unblock the workflow

- [x] 1.1 In `SKILL.md:48`, change the template read to `skill://generate-html/assets/template.html`
- [x] 1.2 Confirm no other `skill://html-report/...` reference survives in `SKILL.md` or `references/components.md`
- [x] 1.3 Verify the read resolves: in an `eval` cell, read `skill://generate-html/assets/template.html` and assert the returned string contains `<!-- CONTENT -->`

## 2. Stylesheet fixes in `assets/template.html`

- [x] 2.1 Replace the `--shadow` token (line 22) with `--shadow-color: light-dark(rgb(0 0 0 / .08), rgb(0 0 0 / .5))`
- [x] 2.2 Update `.card` (line 123) to `box-shadow: 0 1px 3px var(--shadow-color)`
- [x] 2.3 Update `.theme` (line 200) to `box-shadow: 0 1px 3px var(--shadow-color)`
- [x] 2.4 Delete `position: sticky; top: 0` from `thead th` (line 153), leaving `.scroll { overflow-x: auto }` for wide tables
- [x] 2.5 Delete the `.badge + * { margin-left: .1rem }` rule (line 146)
- [x] 2.6 Grep the stylesheet for any remaining `var(--shadow)` reference and confirm none is left

## 3. Theme bootstrap in `assets/template.html`

- [x] 3.1 Add a blocking `<script>` in `<head>` (before `</head>`, line 220) that reads `localStorage.getItem("report-theme")` and sets `document.documentElement.dataset.theme` when present
- [x] 3.2 Remove those two lines from the end-of-body script (lines 252-254), keeping `const root = document.documentElement` there since the rest of the script uses it
- [x] 3.3 Confirm the button wiring, `aria` sync, TOC auto-open and scrollspy all remain in the end-of-body script and still run

## 4. Section numbering

- [x] 4.1 Keep `.toc a::before { content: counter(toc) "." }` (line 89) unchanged
- [x] 4.2 In `references/components.md:10`, drop the manual number: `<h2 id="findings">Findings</h2>`
- [x] 4.3 Add one line to the Section entry in `components.md` stating that contents numbering is generated and headings carry no manual number

## 5. Workflow guidance in `SKILL.md`

- [x] 5.1 Extend the marker assert list (line 64) to all nine placeholders: `REPORT KIND`, `REPORT TITLE`, `LEDE:`, `DATE`, `SCOPE`, `SOURCES`, `Section one`, `<!-- CONTENT -->`, `METHOD AND CAVEATS`
- [x] 5.2 Add a sentence explaining that `DATE` occurs in both masthead and footer, so each is filled by replacing its whole enclosing string, never the bare token
- [x] 5.3 Change the contents-list replacement in the example (line 58) to produce three `<li>` entries from the single template `<li>`, so the multi-section case is the demonstrated one
- [x] 5.4 Replace the template's single `<li><a href="#section-1">Section one</a></li>` (line 237) only if step 5.3 requires a different anchor idiom; otherwise leave the shell as the one-entry seed the example expands

## 6. Handover URL in `SKILL.md`

- [x] 6.1 Replace the `http://localhost:<port>/` handover text (line 83) with the report permalink `http://localhost:<port>/api/documents/<id>/content`
- [x] 6.2 Show parsing `id=` and the port from `send_html`'s result text, matching the format at `internal/mcp/server.go:183`
- [x] 6.3 State the fallback: if the parse fails, hand over the returned text verbatim rather than a guessed URL
- [x] 6.4 Keep the existing failure branch (serve not running) unchanged

## 7. Reference doc drift in `references/components.md`

- [x] 7.1 Rewrite the opening paragraph (line 3) for in-memory composition: no "the copy you made", no instruction to edit `assets/template.html`
- [x] 7.2 State where a one-off rule goes under the in-memory model, that is, appended to the `<style>` block of the composed string, with a comment saying why

## 8. Discoverability

- [x] 8.1 Add a README entry for the skill: what it does, that it lives at `skills/generate-html/`, and that `~/.agents/skills/generate-html` symlinks to it
- [x] 8.2 Leave the directory where it is; do not move it into `.omp/skills/`, which holds vendored openspec tooling and would break the registration symlink

## 9. Verification

- [x] 9.1 Start `serve` and confirm it is reachable
- [x] 9.2 Compose a verification report exercising every catalog component at least once: banner, stats, cards, callouts, evidence badges, table in `.scroll`, diff, steps, meter, SVG figure, appendix, and at least three sections
- [x] 9.3 Send it with `mcp__hyperreader_send_html`, and record the actual return shape to confirm or correct the `display(result["text"])` idiom in `SKILL.md`
- [x] 9.4 Open the permalink from task 6.1 and confirm it renders the report full-page
- [x] 9.5 In the browser, assert `getComputedStyle` on `.card` and `.theme` returns a non-`none` `box-shadow` in both light and dark themes
- [x] 9.6 Confirm no flash of the light theme when reloading with a stored dark preference
- [x] 9.7 Confirm contents numbering runs 1..n against the sections, with no number repeated in a heading
- [x] 9.8 Run the marker assert against a deliberately under-filled report and confirm it fails, naming the missing placeholder
- [x] 9.9 Confirm scrollspy, theme toggle and persistence, skip link, and print rules still work after the script split
