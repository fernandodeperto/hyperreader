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

## Format & limits

- A single self-contained HTML document, UTF-8, under 5 MB.
- Delivered inline via the `send_html` MCP tool's `html` argument — no file
  upload, zip, or multi-file project.
