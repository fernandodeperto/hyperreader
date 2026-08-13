package storage

// schemaSQL is the idempotent migration applied on every Open. CREATE ... IF
// NOT EXISTS makes it safe to run repeatedly across restarts and tests.
//
// The pages table holds page metadata, keyed by an agent-supplied slug (the
// sole primary key — no separate numeric id); the HTML blob lives on disk
// (files/<slug>.html) and only its path is stored in file_path.
//
// pages_fts is an FTS5 external-content table backed by pages, indexing the
// two searchable text fields (name, description). pages has no explicit
// rowid alias (its primary key is the TEXT slug column), but every rowid
// table retains an implicit "rowid" column unless declared WITHOUT ROWID,
// so pages_fts's content_rowid can reference it directly. Three triggers
// keep the FTS index in lockstep with pages on insert/update/delete so
// callers only ever write to pages and search "just works".
const schemaSQL = `
CREATE TABLE IF NOT EXISTS pages (
	slug        TEXT    PRIMARY KEY,
	name        TEXT    NOT NULL,
	description TEXT    NOT NULL DEFAULT '',
	file_path   TEXT    NOT NULL DEFAULT '',
	created_at  INTEGER NOT NULL,
	updated_at  INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
	name, description,
	content='pages', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS pages_ai AFTER INSERT ON pages BEGIN
	INSERT INTO pages_fts(rowid, name, description)
	VALUES (new.rowid, new.name, new.description);
END;

CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages BEGIN
	INSERT INTO pages_fts(pages_fts, rowid, name, description)
	VALUES ('delete', old.rowid, old.name, old.description);
END;

CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages BEGIN
	INSERT INTO pages_fts(pages_fts, rowid, name, description)
	VALUES ('delete', old.rowid, old.name, old.description);
	INSERT INTO pages_fts(rowid, name, description)
	VALUES (new.rowid, new.name, new.description);
END;
`
