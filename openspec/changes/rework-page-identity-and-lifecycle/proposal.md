## Why

HyperReader's document model is create-only, numeric-id-addressed, tag-categorized, and has no cap on description length. Real usage has already outgrown it: the live database contains a literal duplicate row from an accidental re-send (`full-template-size-test`, twice, two seconds apart) and three separate rows (`improve-report-readability-changelog`, `-v2`, `-v3`) where the agent hand-suffixed a version number onto the name because there was no way to say "same page, new content." The tool is drifting from "one-shot report dump" toward long-lived reference pages an agent keeps current, and the identity model needs to support that directly instead of being worked around.

## What Changes

- **BREAKING**: documents are renamed to pages across the entire surface: `docs` table becomes `pages`, `/api/documents` becomes `/api/pages`, UI strings, skill prose, and README all say "page."
- **BREAKING**: pages are identified by an agent-supplied **slug** (`^[a-z0-9]+(-[a-z0-9]+)*$`, max 80 chars) instead of an autoincrement id. The slug is the sole primary key, the on-disk filename (`files/<slug>.html`), and the URL path segment (`/api/pages/{slug}`, `/api/pages/{slug}/content`). No separate internal id is retained.
- **BREAKING**: `POST /api/pages` becomes a create-or-patch endpoint keyed on slug: a new slug creates a page (`201`); an existing slug replaces its name, description, and HTML in place (`200`) and bumps `updated_at`. `created_at` never changes after creation. There is no partial/diff patch, no history, and no delete/archive operation — a patch is a full-body overwrite.
- **BREAKING**: `tags` is removed entirely — from storage, the FTS index, the API request/response shape, the UI, and the skill's `send_html` call.
- **BREAKING**: `description` is capped at 200 characters, enforced server-side (`400` on overflow, no silent truncation).
- List and search order changes from `created_at desc` to `updated_at desc`, so a page an agent just patched surfaces at the top, matching what patching is for.
- `GET /api/events` (SSE) gains a second named event: `page-created` (unchanged shape) and `page-updated` (same shape, fired on patch). The live list in `app.js` reconciles by slug: `page-created` unshifts a new row (as today); `page-updated` removes the existing row for that slug and re-inserts it at the top, matching the new sort order.
- `skills/generate-html/SKILL.md` drops its `tags` guidance, adds slug guidance matching the validation regex verbatim, and updates its description guidance to the 200-char cap.
- Clean slate: the existing `~/.local/share/hyperreader/docs.db` and `files/` are not migrated. No backfill, no compatibility schema, no grandfathering of historical data against the new validation rules.

## Capabilities

### New Capabilities

- `page-identity`: what a page is — slug format and length, description cap, the absence of tags, and how those constraints double as the input-validation boundary for a value that is also a filename and a URL segment.
- `page-lifecycle`: how a page changes over time — create-or-patch-by-slug semantics, full-body replacement, `created_at`/`updated_at` behavior, and list ordering.
- `live-page-updates`: how a patch is reflected to a connected browser tab in real time — the `page-created`/`page-updated` SSE contract and slug-keyed client reconciliation.

### Modified Capabilities

- `project-identity`: its "HyperReader MCP identity" requirement currently guarantees `send_html`'s tool name, arguments, *and* result contract are preserved. This change breaks that guarantee on purpose — `Tags` drops, `Slug` is added, and the result carries `slug` instead of `id` — so the requirement is narrowed to guarantee the tool name and server identity only, not argument/result shape. `html-report-skill`, `binary-distribution`, `configuration-identity`, `fluid-application-layout`, and `graceful-server-shutdown` are unaffected: none of their requirements govern the `tags`/`description`/id contract this change touches.

## Impact

- `internal/storage`: `schema.go` (rewrite `docs`/`docs_fts` as `pages`/`pages_fts`, drop `tags` column, add `updated_at`), `storage.go` (`Doc` struct, replace `Insert`-only with an upsert-by-slug path, add `GetBySlug`, re-sort `List`/`Search` by `updated_at`).
- `internal/api`: `api.go` (routes move to `/api/pages` and `{slug}`), `handlers.go` (`create` becomes create-or-patch, request/response structs drop `Tags` and `ID`, add `Slug`), `events.go` (event name varies by create vs patch).
- `internal/mcp/server.go`: `sendHTMLArgs` drops `Tags`, adds `Slug`; `forwardRequest`/`documentResponse` mirrors follow.
- `web/`: `app.js` (routes, SSE listener split, slug-keyed reconciliation, tags column removed), `index.html` (any "document" strings).
- `skills/generate-html/SKILL.md`: `send_html` call arguments and guidance prose.
- `README.md`: API table, terminology, data storage section.
- Test suites touching any of the above: `main_test.go`, `main_mcp_e2e_test.go`, `web/web_test.go`, `internal/api/*_test.go`, `internal/mcp/*_test.go` (implied by server.go changes), `e2e/*.spec.ts`.
- Local data: delete `~/.local/share/hyperreader/docs.db` and `~/.local/share/hyperreader/files/` before running the new version (incompatible schema, not upgraded).

## Non-Goals

- **Page history or an audit trail.** A patch discards the previous HTML unrecoverably. Revisit only if losing a prior version actually costs someone something.
- **Partial or diff-based patching.** Every write (create or patch) carries the complete page. This matches the skill's existing compose-in-memory workflow and avoids building merge logic the codebase has never needed.
- **Slug rename.** The slug is fixed at creation; there is no operation to change a page's slug independently of its content. Renaming means creating a new page.
- **Delete or archive.** Still absent, unchanged from today, out of scope here.
- **Migrating existing installs.** Clean slate, by explicit direction.
- **Multi-writer conflict resolution.** The write path is last-write-wins, consistent with today's single-local-agent usage pattern.
