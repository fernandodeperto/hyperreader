package storage

// schemaSQL is the idempotent migration applied on every Open. CREATE ... IF
// NOT EXISTS makes it safe to run repeatedly across restarts and tests.
//
// The docs table holds document metadata; the HTML blob lives on disk
// (files/<id>.html) and only its path is stored in file_path (R007 — no
// retention/cap logic by design).
//
// docs_fts is an FTS5 external-content table backed by docs, indexing the
// three searchable text fields (name, description, tags). Three triggers keep
// the FTS index in lockstep with docs on insert/update/delete so callers
// only ever write to docs and search "just works".
const schemaSQL = `
CREATE TABLE IF NOT EXISTS docs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL,
	description TEXT    NOT NULL DEFAULT '',
	tags        TEXT    NOT NULL DEFAULT '',
	file_path   TEXT    NOT NULL DEFAULT '',
	created_at  INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS docs_fts USING fts5(
	name, description, tags,
	content='docs', content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS docs_ai AFTER INSERT ON docs BEGIN
	INSERT INTO docs_fts(rowid, name, description, tags)
	VALUES (new.id, new.name, new.description, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS docs_ad AFTER DELETE ON docs BEGIN
	INSERT INTO docs_fts(docs_fts, rowid, name, description, tags)
	VALUES ('delete', old.id, old.name, old.description, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS docs_au AFTER UPDATE ON docs BEGIN
	INSERT INTO docs_fts(docs_fts, rowid, name, description, tags)
	VALUES ('delete', old.id, old.name, old.description, old.tags);
	INSERT INTO docs_fts(rowid, name, description, tags)
	VALUES (new.id, new.name, new.description, new.tags);
END;
`
