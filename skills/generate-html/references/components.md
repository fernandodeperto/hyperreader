# Component catalog

Every class below is already defined in the template's single `<style>` block. Use only these classes. The workflow composes the page as a string in memory and writes no file, so there is nothing on disk to edit: the template is read, its markers are replaced, and the result is sent. If a report genuinely needs a one-off rule, append it to the `<style>` block of that composed string, with a comment saying why. Never edit the skill's own `assets/template.html`.

Three capabilities here load an external resource, into the `{{EXTERNAL}}` slot, and only when the report actually uses them: charts, highlighted code, and mermaid diagrams. See SKILL.md's "The `{{EXTERNAL}}` slot" for the exact entries and the composition idiom that emits only the ones a report needs. All three degrade rather than disappear with the network off: a chart falls back to its own data table, code falls back to plain mono, and a diagram's source stays visible as text. Nothing about composing the components below changes because of this; the conditional loading is handled once, by inspecting the composed body for the markers noted in each section.

## Section

Wrap every top-level section, matching the id the table of contents `href` targets to the `<section>`, not to its heading:

```html
<section id="findings" aria-labelledby="findings-title">
<h2 id="findings-title">Findings</h2>
<p>Prose. Body text is capped at a 68-character measure for readability; cards, tables and figures span the full column.</p>
</section>
```

Do not number headings by hand. The shell numbers both the contents entry and the section heading from the same thing, document order, so a hand-written number is a second source that drifts the moment sections are reordered. The composer's own job is the check the shell cannot do for itself: confirm the contents list has exactly one entry per section, and that the nth entry's `href` targets the nth section's `id`. If those counts disagree, the sidebar highlight goes dead on whichever section fell out of step.

## Sub-labels

`<h4>` renders as a small uppercase dim label, a kicker above a card title or a group heading inside a section. Use it only paired immediately above an `<h3>` it labels; it is sized smaller than body text by design, which is correct for a kicker and wrong for a standalone sub-heading. A subsection that introduces its own content (a diagram, a code block, a paragraph with no `<h3>` following) needs an `<h3>`, not this. This is why the no-heading-smaller-than-its-body-text invariant doesn't apply to `<h4>`: it never heads body text directly, only ever the `<h3>` immediately below it, which does.

```html
<h4>Preconditions</h4>
<h3>Consumer capacity</h3>
```

## Banner (the verdict)

One per report, directly under `<main>`. First sentence carries the conclusion. Variants: `good` `warn` `bad` `neutral` (default is accent blue).

```html
<div class="banner bad">
  <p><strong>The 2026-07-08 flag flip pushed a demand curve that was already grazing the consumer ceiling permanently over it.</strong></p>
  <p>Nothing regressed. All three incoming hypotheses are falsified by direct measurement.</p>
</div>
```

## Stats (the numbers that matter)

Three to five. More than six stops being a summary. Variants: `good` `warn` `bad` `info`.

```html
<div class="grid">
  <div class="card stat info"><div class="v">12:34 UTC</div><div class="l">flag promoted to full-on</div></div>
  <div class="card stat warn"><div class="v">+12–18%</div><div class="l">step in produce rate, same hour of day</div></div>
  <div class="card stat bad"><div class="v">30K → 378K</div><div class="l">daily peak lag, 07-07 vs 07-09</div></div>
</div>
```

## Cards

`.grid` auto-fits to the viewport (min 15rem per card); `.grid.wide` uses 22rem for text-heavy cards. Use `.hd` when the card needs a badge beside its title.

```html
<div class="grid wide">
  <div class="card">
    <div class="hd"><h3>Consumer capacity</h3><span class="badge ok">CONFIRMED</span></div>
    <p>Ceiling is fixed at 67–76 msgs/sec across June and July.</p>
  </div>
  <div class="card">
    <h3>Cassandra latency</h3>
    <p>p99 unchanged across the window.</p>
  </div>
</div>
```

## Callouts

Inline asides. Variants: `tip` `warn` `danger` `note` (default accent).

```html
<div class="callout warn">
  <span class="t">Blast radius</span>
  <p>Partition count cannot be lowered again. This is a one-way door.</p>
</div>
```

## Badges

Inline status. Variants: `ok` `warn` `bad` `info` `note`.

For investigations and reviews, tag every material claim with its evidence level so a reader can tell measurement from reasoning:

- `<span class="badge ok">CONFIRMED</span>`: directly observed in logs, metrics, code or a document.
- `<span class="badge note">INFERRED</span>`: reasoned from evidence, not observed.
- `<span class="badge warn">UNVERIFIED</span>`: plausible, no evidence either way.

```html
<p><span class="badge ok">CONFIRMED</span> Flag <code>ASEC_LNRS_UNKNOWN_SESSION</code> moved vanguard → full-on in a 46-minute window.</p>
```

## Tables

Wrap every table in `.scroll` so narrow viewports scroll the table instead of the page. Right-align numeric columns with `class="n"` for tabular figures.

```html
<div class="scroll"><table>
  <thead><tr><th scope="col">Hour UTC</th><th scope="col" class="n">07-07</th><th scope="col" class="n">07-08</th><th scope="col">State</th></tr></thead>
  <tbody>
    <tr><td>11</td><td class="n">69.3</td><td class="n">67.1</td><td>baseline</td></tr>
    <tr><td><strong>12</strong></td><td class="n">70.0</td><td class="n"><strong>73.7</strong></td><td>flip lands mid-hour</td></tr>
  </tbody>
</table></div>
```

## Code, diffs, and highlighting

Escape `<` `>` `&` as `&lt;` `&gt;` `&amp;` inside `<pre>`. Unescaped generics or XML silently swallow the rest of the block.

A code block is either a hand-coloured diff or a highlighted language, never both: the two use the same `<pre><code>` element but are mutually exclusive, because the highlighter re-tokenizes the block's raw text and would discard hand-inserted spans.

**Diffs**, coloured by hand with `.add` `.del` `.cm`, no `language-*` class:

```html
<pre><code><span class="del">- if (unknownSession) throw new ApiResponseUnknownSessionException();</span>
<span class="add">+ if (unknownSession) return LexisNexisErrorReason.UNKNOWN_SESSION;</span>
<span class="cm">// eighth per-message processor now runs</span></code></pre>
```

**Highlighted code**, in a real language, worth reading for its own structure rather than as a diff. Give `<code>` a `language-*` class (any highlight.js language name; `language-plaintext` if none fits) and nothing else: the composition idiom in SKILL.md detects this class and loads `highlight.js` only then, and the template's own script highlights only elements carrying it, so a diff block beside it is untouched.

```html
<pre><code class="language-go">func (s *Server) handleUnknownSession(w http.ResponseWriter, r *http.Request) {
	if s.session.Unknown() {
		http.Error(w, "unknown session", http.StatusUnauthorized)
	}
}</code></pre>
```

The token colours are a hand-written map onto the palette, not a stock theme: comments dim, strings a hue that is neither green nor red, keywords by weight. With the script blocked, the block renders as plain mono, nothing lost.

ASCII causal chains and topologies belong in a plain `<pre>` with no `language-*` class; they survive copy-paste better than boxes-and-arrows markup, and small enough ones don't need mermaid (see Diagrams below).

## Steps and timelines

Numbered sequence with a connecting rail. Use for causal chains, rollout timelines and remediation plans, which is most of what these reports diagram: reach for this, or ASCII, before reaching for mermaid.

```html
<ol class="steps">
  <li><span class="st">Raise partitions 8 → 32</span>One-way door. Requires a keyed-consumer audit first.</li>
  <li><span class="st">Re-measure the ceiling</span>Expect proportional headroom; confirm before closing.</li>
</ol>
```

## Progress meters

Inline proportion bar. Set the colour with `--hue`.

```html
<div class="meter" style="--hue: var(--amber)"><span style="width: 62%"></span></div>
```

## Figures and charts

A figure is a rendering plus its evidence: the caption states the claim, and a disclosure underneath holds the data as a table. Offline, or for a screen reader, the table is what the figure is; online, it's the appendix to a chart that also draws itself. Emit both from the same values so they cannot drift:

```python
rows = [("2026-05-01", 68), ("2026-06-01", 71), ("2026-06-15", 69), ("2026-07-01", 74), ("2026-07-09", 96)]

table_rows = "".join(f"<tr><td>{d}</td><td class='n'>{v}</td></tr>" for d, v in rows)
chart_labels = ", ".join(f'"{d}"' for d, _ in rows)
chart_values = ", ".join(str(v) for _, v in rows)
```

```html
<figure>
  <div class="chart" id="chart-ceiling" role="img" aria-label="Daily peak consumer throughput, msgs/sec, 2026-05-01 to 2026-07-09."><p>Chart unavailable — see Data below.</p></div>
  <figcaption>Daily peak consumer throughput, msgs/sec, 2026-05-01 to 2026-07-09.</figcaption>
  <details class="more"><summary>Data</summary>
    <div class="scroll"><table>
      <thead><tr><th scope="col">Date</th><th scope="col" class="n">Peak msgs/sec</th></tr></thead>
      <tbody>{table_rows}</tbody>
    </table></div>
  </details>
</figure>
<script>
(() => {
  const labels = [{chart_labels}];
  const values = [{chart_values}];
  const el = document.getElementById("chart-ceiling");
  // Palette variables are declared with light-dark(), which
  // getComputedStyle().getPropertyValue() returns as unparsed function
  // text. Frappe's `colors` option expects a real colour string and
  // silently falls back to its own default palette on anything it can't
  // parse, so resolve through the cascade instead.
  const resolveColor = (name) => {
    const probe = document.createElement("span");
    probe.style.color = `var(${name})`;
    document.body.appendChild(probe);
    const value = getComputedStyle(probe).color;
    probe.remove();
    return value;
  };
  const render = () => {
    if (!el) return;
    el.replaceChildren();
    new frappe.Chart(el, {
      data: { labels, datasets: [{ values }] },
      type: "line",
      height: 260,
      colors: [resolveColor("--red")],
    });
  };
  if (window.frappe) { render(); document.addEventListener("themechange", render); }
})();
</script>
```

`div.chart` is the marker the composition idiom in SKILL.md looks for to decide whether to load the chart library; a report with no `class="chart"` figure loads no chart script at all. Give the container `role="img"` and an `aria-label` matching the caption, and leave a plain-text fallback inside it (styled by the template's own `figure .chart p` rule, not an inline style): `render()`'s own `el.replaceChildren()` clears it the moment the chart actually draws, so it is visible only when the script never ran. Resolve colours through the cascade as above, never a hard-coded hex and never the raw `getPropertyValue()` text (`light-dark(...)` is not a colour a chart library can parse), and re-render on `themechange` so the chart doesn't go stale when the reader toggles theme. Layout defaults, gridlines, legends and tick counts, are fine as they come; colour is not a default left to the library, it's repainted from the palette by the template's own `.chart-container` overrides, and a chart with an unlabelled axis is the one thing to fix by hand, per the Rules in SKILL.md.

**Inline SVG remains correct** for a figure that isn't a plot: a hand-drawn topology, a mechanism, anything with no axes. Give the `<svg>` a `role="img"` and an `aria-label`, and let the existing `figure svg` rule scale it:

```html
<figure>
  <svg viewBox="0 0 1000 300" role="img" aria-label="Request path from edge to consumer">
    <line x1="64" y1="150" x2="984" y2="150" stroke="var(--border)" stroke-width="2"/>
  </svg>
  <figcaption>Request path, edge to consumer, one hop per box.</figcaption>
</figure>
```

## Diagrams (mermaid)

Reach for this only when `.steps` cannot linearise the shape: a dependency graph, a sequence diagram, a state machine. Mermaid earns roughly a megabyte of external script for exactly those three; anything a causal chain or a rollout timeline could express belongs in `.steps` instead, and anything small enough belongs in ASCII inside a plain `<pre>`. The composer's instinct will be to reach for a box-drawing tool whenever the content would draw a box: resist it here first.

```html
<pre class="mermaid">
graph LR
  A[janus] --> B[device-reputation]
  B --> C[(mysql)]
</pre>
```

The `class="mermaid"` marker is what the composition idiom in SKILL.md looks for to load the library; a report with no such block loads nothing. The template's own script initializes it with theme variables read from the resolved palette and re-runs it on `themechange`, so no per-report script is needed. With the script blocked, the source text above stays visible and readable in place, which is mermaid's own offline fallback for free.

## Appendices

Long evidence dumps and raw output go behind a disclosure so they do not break the reading flow. Printing opens every disclosure and restores it afterward, so this content reaches a PDF export too.

```html
<details class="more">
  <summary>Query appendix</summary>
  <pre><code>sum(kafka_consumergroup_lag{group="janus__data__prod"})</code></pre>
</details>
```
