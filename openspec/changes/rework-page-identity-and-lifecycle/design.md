## Context

See proposal.md for motivation. The constraints that shaped this design, verified against the working tree and the live data directory rather than assumed:

- **One existing spec constrains this domain, and this change deliberately breaks it.** `project-identity`'s "HyperReader MCP identity" requirement guarantees `send_html`'s tool name, arguments, and result contract are preserved. Dropping `Tags` and adding `Slug` violates the "arguments" and "result contract" clauses; see the `project-identity` delta narrowing that requirement to tool-name-and-identity only. No other existing spec (`html-report-skill`, `fluid-application-layout`, `graceful-server-shutdown`, `configuration-identity`, `binary-distribution`) documents the storage/API identity model itself — that part has been implementation-only since the project's start.
- **The write path is create-only today.** `grep` across `internal/` for `Update|PATCH|PUT ` returns nothing. `internal/storage/storage.go`'s `Insert` is the only mutation. `internal/storage/schema.go`'s migration comment explicitly says "no retention/cap logic by design" — that design predates this change's requirements.
- **Real data confirms the pain, not just the theory.** The live `~/.local/share/hyperreader/docs.db` (16 rows) contains a literal accidental duplicate (`full-template-size-test`, id 3 and 4, two seconds apart) and three rows that are the same evolving report hand-versioned by name (`improve-report-readability-changelog`, `-v2`, `-v3`). Description lengths in that data run 48–416 characters; most cluster 100–200.
- **Names are already slug-shaped.** `skills/generate-html/SKILL.md` already instructs "name (required): the report title, kebab-case." 15 of 16 real rows are already kebab-case. This change does not need the server to derive slugs from names, because the write contract requires the caller to supply one directly.
- **The SSE hub is already payload-agnostic.** `internal/api/events.go`'s `hub.broadcast(payload []byte)` does not inspect its argument. Only the event name written in `events()`'s `fmt.Fprintf(w, "event: document\ndata: %s\n\n", payload)` is hardcoded, and only the call site in `handlers.create` decides when to broadcast at all.
- **The client already keys its live-update de-dupe on an id field**, `web/app.js`'s `onDocumentEvent`, `isValidDocPayload`, and `render()` all reference `doc.id`; moving to slug-keyed reconciliation is a rename of what field drives the same logic shape, not a new logic shape.
- **Clean slate is the explicit direction**: the 16 real rows and their files are not migrated. No backfill code, no compatibility schema, no grandfathering existing data against new validation.

## Goals / Non-Goals

**Goals:**

- A slug that is safe to use directly as a SQL primary key, a filename component, and a URL path segment, with one validation rule serving all three uses.
- A single write endpoint whose request shape matches what the generate-html skill already builds in memory today (name, description, HTML) — creating and patching should feel like the same call.
- A live list that reflects a patch without a page reload, consistent with what a fresh fetch would show.
- Metadata trimmed to what the list view actually needs: name, a short description, nothing else.

**Non-Goals:**

- Reproducing or migrating the current 16 rows of real data. They are discarded.
- Partial-field patch semantics, page history, slug rename, and delete/archive — see proposal.md's Non-Goals; these are product-level exclusions, not omitted for lack of design.
- Concurrent-writer conflict detection. The assumed writer is a single local agent process; last-write-wins is acceptable.

## Decisions

### Full-body upsert, not a diff-shaped patch

The word "patch" suggests a partial edit, but the generate-html skill's workflow composes the entire HTML document in memory before sending it — there is no natural place in that workflow to compute or transmit a diff. A full-body replacement keyed by slug fits the existing call shape exactly: the same JSON body that creates a page today patches it tomorrow, differing only in whether the slug already exists.

*Alternative considered:* a true partial patch (a targeted section replace or a diff the server merges into stored HTML). Rejected: it requires the agent to fetch and inspect current content before editing it, and requires merge logic this codebase has never had. Revisit only if an agent workflow emerges that edits a page without first composing its full content.

### One endpoint, slug collision as the create/patch signal

`POST /api/pages` decides create-vs-patch by checking whether the slug exists, rather than exposing a separate `PATCH` verb. This mirrors exactly how the real duplicate-row and hand-versioned-name evidence arose: the agent already sends the same shape of request whether or not it "means" to update something, so the server, not the caller, is the natural place to decide.

*Alternative considered:* a dedicated `PATCH /api/pages/{slug}` endpoint, `POST` staying strictly create-only (`409` on collision). Rejected: it moves the create-vs-patch decision to the caller, who has to already know the answer to pick the verb, which is exactly the ambiguity that produced the `-v2`/`-v3` rows in the first place.

*Consequence accepted as a risk below:* a slug typo now silently patches the wrong page instead of returning "not found."

### Slug charset, length, and where it's enforced

Pattern: `^[a-z0-9]+(-[a-z0-9]+)*$`. Max length: 80 characters. This is not a style preference — the slug is the SQL primary key, the filename (`files/<slug>.html`), and the URL path segment, all three at once, so the charset is the complete input-validation boundary protecting all three. The pattern excludes `/`, `.`, `..`, whitespace, and every other filesystem- or URL-meaningful character by construction (only `[a-z0-9-]` survives), and the length cap keeps it well under filesystem filename limits.

Validation runs before any storage or filesystem call, in the HTTP handler, so a rejected slug never reaches `Store.Upsert` or a file write. `skills/generate-html/SKILL.md` states this pattern verbatim so the agent composing a slug and the server validating it can never disagree about what's legal.

*Alternative considered:* deriving the slug from `name` server-side (slugify), sidestepping the validation question by construction. Rejected for the write path per the earlier locked decision — the agent supplies the slug directly, because slug collision is how patching is triggered, and an auto-derived slug the agent didn't choose would make that trigger unpredictable.

### Slug as the sole primary key; no internal numeric id

With no legacy `files/<id>.html` layout to protect (clean slate) and no rename operation in the locked model (patch always targets the exact slug supplied, there is no "change this page's slug" call), a slug is already immutable in practice. Carrying a separate internal id would be protecting against a scenario, slug rename, that the write contract has already ruled out. `pages.slug` is the `TEXT PRIMARY KEY`; `files/<slug>.html` is the file path directly.

FTS5's external-content mechanism needs an integer `content_rowid`; SQLite tables retain an implicit rowid even with a `TEXT PRIMARY KEY` unless declared `WITHOUT ROWID` (which this schema does not do), so `pages_fts` can use that implicit rowid exactly as `docs_fts` used `docs.id` today. No functional loss from dropping the explicit column.

*Alternative considered:* keep a dual `id`/`slug` shape "for future-proofing" a possible later rename feature. Rejected per simplicity-first: add it back if a real rename requirement appears, not speculatively.

### Description cap: 200 characters, rejected not truncated

Calibrated against the real data's 48–416 character range, where descriptions above roughly 200 characters are already the outliers, not the norm. A `400` naming the limit, not silent truncation, because truncation would hide the fact that the agent's composed lede didn't fit — matching how `name` is already required rather than defaulted.

*Alternative considered:* a word-count limit ("a few phrases" read literally). Rejected: character length is what actually determines whether a description wraps to two lines in the list view, and it's simpler to enforce and to explain in an error message than a phrase-count heuristic.

### Sort by `updated_at desc`, not `created_at desc`

A page an agent just patched is, by definition, the one most likely to be relevant right now. Sorting by change time keeps the unfiltered list and the live SSE reconciliation (which moves a patched row to the front) in agreement — a hard reload and the live-updated view show the same order.

*Alternative considered:* keep `created_at desc`, leave patched pages wherever they were created. Rejected: it would make the live-update behavior in `live-page-updates` inconsistent with what a reload shows, since moving a patched row to the front only matches sort order if the sort key is change time.

### SSE: two named events, not one event with an action field

`internal/api/events.go` already discriminates message kinds by SSE event name (`event: document`), not by a field inside the JSON payload. Extending that same convention, `page-created` and `page-updated`, keeps the wire format's discriminator where the codebase already puts it, and lets the client register two small, single-purpose listeners instead of one listener that branches on payload content.

`hub.broadcast([]byte)` needs no change; only the string written by `events()`'s `fmt.Fprintf` and the handler call sites (now one per branch of the create-or-patch decision) change.

*Alternative considered:* one `event: page` with `{"action": "created"|"updated", ...}` in the payload. Rejected for consistency with the existing convention; either would work mechanically.

*Carried forward, not changed by this design:* `app.js` already drops live events entirely while a search filter is active (existing comment: it's a create-time broadcast, not a search result). That tradeoff applies unchanged to patch events — a page being edited under an active filter won't visibly update until the filter clears or reruns. Not solved here.

### Clean slate: delete, don't migrate

The existing `docs.db`/`files/` at `~/.local/share/hyperreader` are incompatible with the new schema (dropped `tags` column, new `slug` primary key with no historical values to backfill, new `updated_at` column) and are not migrated, per explicit direction. `schema.go`'s migration is rewritten to define `pages`/`pages_fts` directly; it does not need to handle an existing `docs` table at all.

### Narrow `project-identity`'s MCP contract-stability clause, rather than ignore it

`project-identity`'s "HyperReader MCP identity" requirement was written to guarantee `send_html`'s tool name, arguments, and result contract across the project's rename from `html-mcp`. That guarantee is not compatible with dropping `Tags`, adding `Slug`, and returning `slug` instead of `id` — all required by the identity model this change ships. Silently shipping the contract change without touching the requirement would leave the spec set claiming a stability guarantee the code no longer honors.

The requirement is narrowed, not deleted: it still guarantees the tool is named `send_html` and the server still identifies as HyperReader, which is the part of "identity" the requirement's own title is about. The argument/result shape guarantee is removed and reassigned to whatever spec governs the page model at the time (`page-identity`/`page-lifecycle` here).

*Alternative considered:* leave `project-identity` untouched and treat the contract break as an undocumented implementation detail. Rejected: `openspec validate --strict` checks delta structure, not cross-spec consistency, so nothing would have caught the spec set asserting two contradictory things about the same tool.

## Risks / Trade-offs

- **A slug typo patches the wrong page instead of failing.** → Response status (`201` vs `200`) tells the caller which happened, so an agent (and the person reading its output) can notice an unexpected patch. No stronger guard is proposed; a confirmation step would undercut the whole point of a single-call upsert.
- **No history: a patch is destructive.** → Accepted and stated plainly in the spec (`page-lifecycle`'s "discards prior content irrecoverably" requirement) so nothing downstream assumes recoverability.
- **Full-body replacement means fixing one typo still requires resending the whole composed page.** → Accepted; it costs nothing new relative to today's create-only workflow, which already always sends the whole page.
- **Slug charset is a hard requirement on both server and skill.** → `SKILL.md`'s slug guidance states the regex verbatim, not a paraphrase, so drift between what the agent composes and what the server accepts is not possible by construction (only by someone editing one copy and not the other, same risk any duplicated constant carries).
- **`event: document`-style two-event SSE is a breaking change for any other client subscribed to the old event name.** → Accepted; no external consumers exist for a single-user personal tool, and this change controls both server and client.
- **Deleting local data is irreversible.** → Explicit user direction; no rollback is provided or needed.

## Migration Plan

No data migration; this is a rewrite of the storage shape, not an upgrade of it. Implementation ordering matters because each layer depends on the one below:

1. `internal/storage`: schema rewrite (`pages`/`pages_fts`, drop `tags`, add `updated_at`), `Doc` struct changes, `Upsert`-by-slug, `GetBySlug`, `List`/`Search` re-sorted.
2. `internal/api`: routes, request/response shapes, create-or-patch handler logic, SSE event naming.
3. `internal/mcp`: `sendHTMLArgs`/forwarding structs follow the API shape.
4. `web/`: routes, SSE listener split, slug-keyed reconciliation, tags column removed from rendering.
5. `skills/generate-html/SKILL.md` and `README.md`: prose and examples updated to match the shipped behavior, not ahead of it.
6. Delete `~/.local/share/hyperreader/docs.db` and `files/` locally; verify a fresh `serve` starts clean.
7. Full-suite verification (see tasks.md) before considering this done.
