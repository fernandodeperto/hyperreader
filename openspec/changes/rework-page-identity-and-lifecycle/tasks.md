## 1. Storage layer (`internal/storage`)

- [x] 1.1 Rewrite `schema.go`: `pages` table (`slug TEXT PRIMARY KEY`, `name`, `description`, `file_path`, `created_at`, `updated_at`), drop `tags`; `pages_fts` external-content FTS5 over `name`/`description` only; update the three sync triggers accordingly
- [x] 1.2 Update `Doc` struct in `storage.go`: replace `ID int64` with `Slug string`, drop `Tags`, add `UpdatedAt`
- [x] 1.3 Add slug validation: constant/regex `^[a-z0-9]+(-[a-z0-9]+)*$`, max length constant (80), a validation function returning a clear error for a failing slug
- [x] 1.4 Add description length validation: max length constant (200), a validation function returning a clear error for an over-limit description
- [x] 1.5 Replace `Insert` with `Upsert(ctx, doc Doc) (created bool, err error)`: validates slug and description first (before any file write), checks slug existence, on new slug inserts with `created_at = updated_at = now`, on existing slug overwrites `name`/`description`/`file_path` contents and sets `updated_at = now` while preserving `created_at`; file write follows the same crash-safety ordering `Insert` used (write row/placeholder, write file, finalize)
- [x] 1.6 Add `GetBySlug(ctx, slug string) (Doc, error)` and `GetBySlugContent(ctx, slug string) (Doc, error)`, replacing the `GetByID`/`GetByIDContent` id-keyed equivalents
- [x] 1.7 Update `List` and `Search` to order by `updated_at desc`
- [x] 1.8 Update `storage_test.go`: cover slug validation (valid, path separator, traversal, disallowed chars, leading/trailing/double dash, over-length), description cap (accepted at limit, rejected over), create vs patch (`Upsert` returns correct `created` flag both times), `created_at` fixed / `updated_at` advancing across a patch, list ordering after a patch, and that patched-away HTML is not retrievable

## 2. HTTP API (`internal/api`)

- [x] 2.1 Update `api.go`: `Store` interface methods follow storage's new names; routes become `POST /api/pages`, `GET /api/pages`, `GET /api/pages/{slug}`, `GET /api/pages/{slug}/content`, `GET /api/events` unchanged path
- [x] 2.2 Update `createRequest`/`documentResponse` (rename to page-shaped equivalents): drop `Tags`, drop `ID`, add `Slug` (required on write, present in every response)
- [x] 2.3 Rewrite `create` handler (`handlers.go`) as create-or-patch: validate slug and description (400 on either failure, before touching storage), call `Upsert`, respond `201` on create / `200` on patch
- [x] 2.4 Update `get`/`getContent` handlers to key on `{slug}` path parameter instead of parsed numeric id; drop `parseID`
- [x] 2.5 Update `events.go`: broadcast helper takes the event name as a parameter (or two call sites), `create` handler broadcasts `page-created` on create and `page-updated` on patch
- [x] 2.6 Update `handlers_test.go` and `events_test.go`: create-then-patch-same-slug round trip asserting `201`/`200`, slug and description validation error cases, SSE subscriber receives the right event name for each case

## 3. MCP forwarder (`internal/mcp`)

- [x] 3.1 Update `sendHTMLArgs`: drop `Tags`, add required `Slug` field with a doc comment stating the validation pattern
- [x] 3.2 Update `forwardRequest`/`documentResponse` mirrors to match the API's new shapes
- [x] 3.3 Update `server_test.go` for the new argument shape and any response-text assertions that referenced `id=` (now `slug=` or equivalent)

## 4. Web UI (`web/`)

- [x] 4.1 Update `app.js` routes: `API` constant to `/api/pages`; row activation opens `/api/pages/{slug}/content`
- [x] 4.2 Update `render()`: drop the tags cell; row keys on `data-slug` instead of `data-id`
- [x] 4.3 Split `onDocumentEvent` into two listeners: `page-created` (unshift if slug not already present) and `page-updated` (remove existing entry for that slug, then unshift at top); update `isValidDocPayload` to key on `slug`
- [x] 4.4 Update `index.html` and any remaining "document" strings (empty-state text, aria labels) to say "page"
- [x] 4.5 Update `web_test.go` for slug-keyed rows and the two SSE event listeners

## 5. Skill and documentation

- [x] 5.1 Update `skills/generate-html/SKILL.md`: drop the `tags` argument and its guidance; add a `slug` argument with the validation pattern stated verbatim and length limit; update the `description` guidance to the 200-character cap; update the worked example's `send_html` call and its result-parsing regex (currently keys on `id=`)
- [x] 5.2 Update `README.md`: API table (`/api/pages`, request/response shape), "Data storage" section, any "document" terminology

## 6. Cleanup and verification

- [x] 6.1 Delete `~/.local/share/hyperreader/docs.db` and `~/.local/share/hyperreader/files/` locally
- [x] 6.2 Run `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`; fix fallout
- [x] 6.3 Start `hyperreader serve` against the now-empty data dir; confirm a fresh `pages` schema is created with no errors
- [x] 6.4 Smoke test via `send_html` (or a direct `POST /api/pages`): create a page with a new slug, confirm `201` and the row appears live in an open browser tab via `page-created`
- [x] 6.5 Smoke test patch: send the same slug again with different content, confirm `200`, confirm the open browser tab's row updates in place (no duplicate) and moves to the top via `page-updated`
- [x] 6.6 Smoke test validation: attempt a slug containing `/` or `..`, and a description over 200 characters; confirm both are rejected with no file written and no row created
- [x] 6.7 Run the Playwright suite (`e2e/*.spec.ts`) updated for the new routes/terminology; fix fallout
- [x] 6.8 Re-read `README.md` and `SKILL.md` end to end and confirm every example command/call matches what was actually run in 6.4-6.7
