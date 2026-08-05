# Component catalog

Every class below is already defined in the copy you made. Use only these classes, and keep the single `<style>` block the shell ships with. If a report genuinely needs a one-off rule, add it at the end of that block in your copy, never in the skill's own `assets/template.html`, and leave a comment saying why.

## Section

Section ids must match the `href`s in the table of contents, or the sidebar highlight goes dead.

```html
<h2 id="findings">2. Findings</h2>
<p>Prose. Body text is capped at a 68-character measure for readability; cards, tables and figures span the full column.</p>
```

## Sub-labels

`<h4>` renders as a small uppercase dim label, a kicker above a card title or a group heading inside a section. Use it when a block needs a label above its `<h3>`.

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
  <thead><tr><th>Hour UTC</th><th class="n">07-07</th><th class="n">07-08</th><th>State</th></tr></thead>
  <tbody>
    <tr><td>11</td><td class="n">69.3</td><td class="n">67.1</td><td>baseline</td></tr>
    <tr><td><strong>12</strong></td><td class="n">70.0</td><td class="n"><strong>73.7</strong></td><td>flip lands mid-hour</td></tr>
  </tbody>
</table></div>
```

## Code and diffs

Escape `<` `>` `&` as `&lt;` `&gt;` `&amp;` inside `<pre>`. Unescaped generics or XML silently swallow the rest of the block. Colour diff lines with `.add` `.del` `.cm`.

```html
<pre><code><span class="del">- if (unknownSession) throw new ApiResponseUnknownSessionException();</span>
<span class="add">+ if (unknownSession) return LexisNexisErrorReason.UNKNOWN_SESSION;</span>
<span class="cm">// eighth per-message processor now runs</span></code></pre>
```

ASCII causal chains and topologies also belong in `<pre>`; they survive copy-paste better than boxes-and-arrows markup.

## Steps and timelines

Numbered sequence with a connecting rail. Use for causal chains, rollout timelines and remediation plans.

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

Inline SVG only. No chart libraries, no external images. The file must open with the network off.

Use a `viewBox` of `0 0 1000 300` and let the CSS scale it. Style strokes with the palette variables, and always give the `<svg>` a `role="img"` and an `aria-label`. Compute coordinates in a scratch script rather than by hand.

```html
<figure>
  <svg viewBox="0 0 1000 300" role="img" aria-label="Daily peak consumer lag, log scale">
    <line x1="64" y1="256" x2="984" y2="256" stroke="var(--border)"/>
    <polyline fill="none" stroke="var(--red)" stroke-width="2" points="64,240 300,232 520,120 984,96"/>
  </svg>
  <figcaption>Daily maximum summed consumer lag, log scale, 2026-05-01 to 2026-07-31.</figcaption>
</figure>
```

## Appendices

Long evidence dumps and raw output go behind a disclosure so they do not break the reading flow.

```html
<details class="more">
  <summary>Query appendix</summary>
  <pre><code>sum(kafka_consumergroup_lag{group="janus__data__prod"})</code></pre>
</details>
```
