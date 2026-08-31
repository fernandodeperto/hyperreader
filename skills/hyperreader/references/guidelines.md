# HyperReader page guidelines

Authoritative rules for generating HTML pages sent to HyperReader. This skill is
self-contained: follow only these rules and `assets/template.html`. Do not fetch
external rulesets (including the bpages CLI), read other skills, or browse other
sources.

These rules adapt the Booking Pages (bpages) artifact ruleset for HyperReader.
The reader shows each page in a trusted, same-origin, unsandboxed iframe below a
fixed top bar. There is no Content Security Policy and no ingest sanitization:
HTML/JS runs as trusted code, so bpages' CSP restrictions and build-step
requirements do not apply. HyperReader renders every page in a single dark
theme; there is no light theme and no theme toggle.

## Dark theme (required)

- Root element is `<html lang="en">`; the reader always renders it in its fixed
  dark shell.
- Register the shell-matching palette on Tailwind:
  `tailwind.config = { theme: { extend: { colors: {...} } } };`. There is only
  one theme, so there is no `darkMode` key and no `dark:` variants.
- Style with the single dark palette directly: backgrounds, text, borders, and
  code all use the dark tokens below.
- Keep the minimal inline base in the template so the page paints the shell's
  exact dark color before Tailwind loads and if the CDN is unreachable:
  `html { background-color:#16181d; color:#e8e8e8; }`. These hex values match
  the reader shell's own `--bg`/`--fg` tokens (`web/app.css`) exactly — never
  substitute Tailwind's `gray-*` scale for page/body backgrounds; it is
  blue-tinted and visibly mismatches the shell.
- Do NOT add your own theme toggle. The reader renders dark-only and strips
  embedded elements with class `.theme`.
- Never use `prefers-color-scheme`; the page is always dark.

## Design principles (from bpages, non-negotiable)

1. **Container width**: the outer container is `max-w-5xl mx-auto px-4`.
   Prose-heavy pages may nest a narrower `max-w-2xl` reading column inside it.
2. **Tailwind is the only CSS framework** — via `https://cdn.tailwindcss.com`.
   No Bootstrap/Bulma/other frameworks, no custom stylesheet files. Inline
   `<style>` only for animations, the first-paint base, and states Tailwind
   cannot express. Allowed external script hosts: `aistudio.booking.com`,
   `cdn.tailwindcss.com`, `cdnjs.cloudflare.com`, `unpkg.com`,
   `cdn.jsdelivr.net`.
3. **Flat design — no depth**. Use `border` + `border-line` and solid `bg-*`
   fills instead of `shadow-*`, `drop-shadow-*`, decorative `ring-*`, elevation
   gradients, or `backdrop-blur-*`. Color, spacing, and typography carry
   hierarchy.
4. **Animations** — CSS transitions only, never JS animation libraries. Standard
   entrance is the `.reveal` fade-up (opacity + `translateY(10px)` -> 0).
   Duration 0.3s-0.5s (never over 0.6s), easing `ease`/`ease-out` (never
   `linear` for entrances), motion always upward. Stagger with `data-delay`
   (ms), triggered by adding `.visible` via IntersectionObserver.
5. **Typography** — system font stack (`font-sans`; no Google Fonts or font
   CDNs). Body and headings `text-ink leading-relaxed` (headings add
   `font-bold`); muted/secondary text `text-muted`; inline code
   `font-mono text-sm bg-surface-alt text-ink px-1.5 py-0.5 rounded`.
6. **Color palette** — Booking brand tokens plus shell-matching semantic
   tokens, all registered in `tailwind.config`: `brand #003B95`,
   `action #006CE4`, `action-hover #0057b8`, `accent #FEBB02`,
   `constructive #008009`, and `page #16181d`, `surface #1f2229`,
   `surface-alt #262a33`, `ink #e8e8e8`, `muted #9a9fa8`, `line #3a3f4a`
   (exact hex values in `assets/template.html`, mirroring `web/app.css`'s
   `--bg`/`--fg`/`--surface`/`--muted`/`--border`). Never use Tailwind's
   built-in `gray-*` scale for these roles — it is blue-tinted and visibly
   mismatches the reader shell's neutral dark background. Keep text contrast
   accessible.
7. **Structure** — wrap content in `<main>` with `<section>` children for clean
   semantics. Give each `<section>` its own surface classes rather than relying
   on a transparent background.

## Component patterns

- Card: `border border-line rounded-lg p-5 bg-surface`
- Section heading: `text-2xl font-bold text-ink mb-4`
- Body paragraph: `text-ink leading-relaxed`
- Link / CTA: `text-action hover:text-action-hover`

Example `{{CONTENT}}` section:

    <section class="mb-10">
      <h2 class="text-2xl font-bold text-ink mb-4">Section title</h2>
      <div class="border border-line rounded-lg p-5 bg-surface">
        <p class="text-ink leading-relaxed">Body copy.</p>
      </div>
    </section>

## MR review components

These components target `assets/mr-review-template.html`, which loads Prism.js
from `cdn.jsdelivr.net` (already an allowed script host per Design principles
#2) to syntax-highlight diffs. Inside `{{CONTENT}}`, order the pieces: verdict
banner → optional "Files reviewed" index → one change block per change.

### Diff authoring rules

- Put the raw unified diff as the *text content* of the `<code>` element,
  HTML-escaped: `&` → `&amp;`, `<` → `&lt;`, `>` → `&gt;`.
- Do not indent diff lines with template indentation — whitespace inside
  `<pre>` is literal. Keep the leading ` `/`+`/`-` diff column.
- Class is `language-diff-<lang> diff-highlight`, where `<lang>` is the Prism
  language id for the file (`go`, `typescript`/`ts`, `javascript`/`js`,
  `python`/`py`, `rust`, `java`, `ruby`, `bash`, `json`, `yaml`, `sql`,
  `markup` for HTML/XML, `css`). The autoloader fetches the grammar on demand;
  `diff-highlight` both tints +/- line backgrounds and highlights the
  underlying language.

### Verdict banner

Top of `{{CONTENT}}`; the overall recommendation plus totals:

    <section class="reveal mb-8 border border-line rounded-lg p-5 bg-surface">
      <div class="flex flex-wrap items-center gap-3 mb-3">
        <span class="inline-flex items-center text-xs font-semibold uppercase tracking-wide px-2.5 py-1 rounded border border-accent text-accent">Request changes</span>
        <span class="text-muted text-sm font-mono">3 files · +128 −44</span>
      </div>
      <p class="text-ink leading-relaxed">One-paragraph overall assessment of the MR.</p>
    </section>

Recommendation badge color: Approve → `border-constructive text-constructive`;
Request changes → `border-accent text-accent`; Comment → `border-action text-action`.

### Files reviewed index

Optional, recommended for complex MRs. Each entry links to its change block's `id`:

    <nav class="reveal mb-8 border border-line rounded-lg p-5 bg-surface">
      <h2 class="text-sm font-bold text-muted uppercase tracking-wide mb-3">Files reviewed</h2>
      <ul class="space-y-1">
      <li><a href="#change-1" class="text-action hover:text-action-hover font-mono text-sm">1. internal/api/handlers.go</a> <span class="text-muted text-xs">+42 −7 · blocking</span></li>
      </ul>
    </nav>

### Change block

Repeat per change. Each change block is a collapsible `<details>` card, open
by default; its `<summary>` is the file header and clicking it collapses the
card. Number the cards sequentially from 1 in document order, set
`id="change-N"` (so the anchor is `#change-N`), and show N in the summary
badge. `scroll-mt-4` offsets anchor jumps. The number is how a reviewer refers
to a card ("card 3"); the Files reviewed index links to `#change-N`.

    <details id="change-1" class="reveal scroll-mt-4 mb-8 border border-line rounded-lg overflow-hidden bg-surface" open>
      <summary class="flex flex-wrap items-center justify-between gap-2 px-4 py-3 bg-surface-alt cursor-pointer">
        <span class="flex items-center gap-3 min-w-0">
          <svg class="change-caret w-3 h-3 shrink-0 text-muted" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M4 2l4 4-4 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          <span class="inline-flex items-center justify-center shrink-0 w-6 h-6 rounded border border-line text-muted font-mono text-xs">1</span>
          <code class="font-mono text-sm text-ink truncate">internal/api/handlers.go</code>
        </span>
        <span class="font-mono text-xs text-muted shrink-0">+42 −7</span>
      </summary>
      <div class="border-t border-line overflow-x-auto">
        <pre><code class="language-diff-go diff-highlight">@@ -10,6 +10,7 @@ func handle() {
     ctx := r.Context()
    -    return doThing(ctx)
    +    v, err := doThing(ctx)
    +    if err != nil { return err }
     }</code></pre>
      </div>
      <div class="px-4 py-4 border-t border-line">
        <div class="flex items-center gap-2 mb-2">
          <span class="inline-flex items-center text-xs font-semibold uppercase tracking-wide px-2 py-0.5 rounded border border-accent text-accent">Blocking</span>
          <h3 class="text-sm font-bold text-muted uppercase tracking-wide">Analysis</h3>
        </div>
        <p class="text-ink leading-relaxed">The reviewer agent's analysis of this specific change.</p>
      </div>
    </details>

Differences from the previous `<section>`-based recipe:

- Root `<section id="file-handlers-go">` → `<details id="change-1" … open>`;
  the old `<header>` becomes `<summary>` (the clickable collapse handle).
- The `<summary>` gains, left-aligned in a `flex items-center gap-3 min-w-0`
  wrapper: a rotating chevron (`<svg class="change-caret">`), then a **number
  badge** (`<span … border border-line text-muted font-mono text-xs>1</span>`),
  then the filename `<code>` (now `truncate`). The stats `<span>` stays
  right-aligned and gains `shrink-0`.
- The old `<header>` carried `border-b border-line`; the `<summary>` carries no
  bottom border. Instead the diff `<div>` becomes
  `border-t border-line overflow-x-auto` so the divider appears under the
  summary only when the card is open and vanishes when collapsed. The analysis
  `<div>` keeps `px-4 py-4 border-t border-line`.

### Severity badge → token mapping

Fixed; the palette has no red, so severity is carried by label text plus these
brand tokens, while the diff line-background red/green carries add/remove:

- Blocking → `border-accent text-accent`
- Concern / Question → `border-action text-action`
- Nit → `border-line text-muted`
- Praise / LGTM → `border-constructive text-constructive`

### Suggested fix

Optional sub-block inside the analysis panel: a plain highlighted snippet, e.g.
`<pre><code class="language-go">...</code></pre>` (Prism highlights non-diff
`language-<lang>` blocks too).

## Format & limits

- A single self-contained HTML document, UTF-8, under 5 MB.
- Delivered inline via the `send_html` MCP tool's `html` argument — no file
  upload, zip, or multi-file project.
