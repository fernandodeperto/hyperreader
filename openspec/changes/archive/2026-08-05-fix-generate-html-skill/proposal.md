## Why

A review of `skills/generate-html/` (recorded in `Claude Notes/hyperreader/2026-08-05-generate-html-skill-review.md`) found the skill is broken in two ways that make it unusable as shipped: its workflow reads `skill://html-report/assets/template.html`, a skill id that is no longer registered on this machine, so step 2 fails outright; and `--shadow` is invalid CSS, so every shadow in every report it produces is silently dropped. A further eleven findings cover unenforced markers, dead CSS, doc drift, and a handover URL that points at the list view instead of the report.

## What Changes

**Blocking bugs**

- Fix the template URI: read `skill://generate-html/assets/template.html`, matching the skill's own `name:`. The old `html-report` id resolves to nothing now that the dotfiles copy is gone.
- Split `--shadow` into `--shadow-color`, a real two-argument `light-dark()`, and move the offsets to the usage sites (`.card`, `.theme`). `light-dark()` takes two `<color>`s; handing it full shadow values is invalid-at-computed-value-time, so `box-shadow` falls back to `none`.

**Correctness of generated reports**

- Complete the marker assert list to cover all nine shell markers. It currently checks five, so a report can ship with a literal `DATE - SCOPE - SOURCES` masthead and pass.
- Make `DATE` unambiguous. It appears in both the masthead and the footer; a naive `.replace('DATE', ...)` corrupts the footer. Replace the whole enclosing strings and say why.
- Resolve the double-numbering conflict between `.toc a::before { content: counter(toc) "." }` and the hand-numbered `<h2 id="findings">2. Findings</h2>` idiom in the component catalog. One source of numbering, stated as a rule.
- Show a multi-section table of contents. Template and example both demonstrate exactly one `<li>`, which is the one case that never occurs.

**Dead CSS**

- `thead th { position: sticky; top: 0 }` never sticks: `.scroll { overflow-x: auto }` computes `overflow-y` to `auto`, scoping the sticky to a container with no height cap. Cap `.scroll` or drop the sticky.
- `.badge + * { margin-left: .1rem }` never fires: it is an adjacent-*element* selector and every documented usage follows a badge with a text node.

**Delivery and docs**

- Hand over the report permalink, `GET /api/documents/{id}/content` (`internal/api/api.go:81`), using the `id` already present in `send_html`'s result text (`internal/mcp/server.go:183`). The current `http://localhost:<port>/` is the list view.
- Eliminate the flash of wrong theme by moving the `localStorage` read into a blocking `<script>` in `<head>`; it currently runs at the end of `<body>`, after first paint.
- Rewrite the opening paragraph of `references/components.md`, which still describes editing "the copy you made" on disk. The workflow composes in memory and writes nothing.
- Make the skill discoverable: register it in the README and reconcile its location against the repo's other agent skills in `.omp/skills/`.
- Verify the `tool.mcp__hyperreader_send_html` return shape that `display(result["text"])` assumes, against a running `serve`.

**Already resolved, closed without action**

- The review's "verbatim fork" finding and its open question of which copy is canonical are settled: `~/.agents/skills/generate-html` is a symlink to `skills/generate-html` in this repo, so there is one copy and the repo is canonical.

**Accepted risk, no change**

- Every design token routes through `light-dark()` (Chrome 123+, Safari 17.5+, Firefox 120+). A fallback palette would double the token block to guard browsers this audience does not use.

## Capabilities

### New Capabilities

- `html-report-skill`: the contract the `generate-html` skill must satisfy - that it resolves its own template, that generated reports render every documented component correctly in both themes, that unfilled shell markers cannot reach a delivered report, and that the handover addresses the report itself.

### Modified Capabilities

<!-- None. The existing specs (configuration-identity, fluid-application-layout,
     graceful-server-shutdown, project-identity) describe the HyperReader server
     and app; none of their requirements change. -->

## Impact

- `skills/generate-html/SKILL.md`: template URI, marker assert list, `DATE` replacement idiom, multi-section TOC example, handover URL.
- `skills/generate-html/assets/template.html`: `--shadow-color` token and its two usages, `.scroll` height cap or sticky removal, `.badge + *` removal, theme bootstrap moved to `<head>`, TOC numbering.
- `skills/generate-html/references/components.md`: opening paragraph, heading-numbering idiom.
- `~/.agents/skills/generate-html`: symlink to the above; no separate edit, but the fixes take effect for the registered skill immediately.
- `README.md`: skill discoverability entry.
- Read-only dependencies for verification: `internal/api/api.go`, `internal/mcp/server.go`.
- No change to HyperReader server, storage, or MCP code.
