// Package storage implements the SQLite+FTS5 persistence layer for hyperreader.
//
// A Store wraps a *sql.DB backed by modernc.org/sqlite (pure Go, no cgo).
// Documents are stored as metadata in the docs table plus an FTS5 external-
// content index over name/description/tags; the raw HTML payload lives on
// disk under <filesDir>/<id>.html and only its path is recorded in SQLite.
//
// The DB connection is opened with the shared cache disabled and a 30s busy
// timeout so concurrent readers/writers (the serve process plus the MCP
// forwarder writing through it) don't surface "database is locked" to callers.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

// DefaultLimit caps unfiltered List results so a degenerate query cannot
// pull the whole table into memory.
const DefaultLimit = 100

// Doc is the in-memory representation of a row in docs. HTMLContent is the
// raw payload read back from disk; it is populated only by callers that
// need it (e.g. GetByIDContent), never by list/search which return metadata.
type Doc struct {
	ID          int64
	Name        string
	Description string
	Tags        string
	FilePath    string
	CreatedAt   time.Time
	HTMLContent string
}

// Store is the storage handle. It owns one *sql.DB; all methods are safe
// for concurrent use because database/sql pools connections internally.
type Store struct {
	db       *sql.DB
	filesDir string
}

// Open opens (or creates) the SQLite database at dbPath, applies the schema
// migration, and ensures filesDir exists. The caller is responsible for
// Close. Set foreign keys / busy timeout via the connection DSN query params.
func Open(dbPath, filesDir string) (*Store, error) {
	// _pragma=busy_timeout(30000) avoids "database is locked" under light
	// concurrency; foreign_keys are on by default in modernc but stated
	// explicitly for clarity (the FTS5 external-content triggers don't
	// need them, but the project may grow FK constraints later).
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(30000)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}
	// single writer connection pool size avoids write contention deadlocks
	// in SQLite's serialized model; readers still multiplex.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite %s: %w", dbPath, err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema %s: %w", dbPath, err)
	}

	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		db.Close()
		return nil, fmt.Errorf("create files dir %s: %w", filesDir, err)
	}

	return &Store{db: db, filesDir: filesDir}, nil
}

// Close releases the database connection pool.
func (s *Store) Close() error { return s.db.Close() }

// Insert persists a document's metadata to SQLite and its HTML payload to
// disk, returning the new row id. The row is inserted first (letting the DB
// assign the id), then the HTML file is written as <id>.html, then the
// file_path column is updated and the transaction committed. If the file
// write fails the transaction rolls back (no dangling metadata row); if the
// commit fails after the file write, an orphan .html file may remain — that
// is inert and preferable to a metadata row pointing at a missing file.
func (s *Store) Insert(ctx context.Context, doc Doc) (int64, error) {
	if doc.Name == "" {
		return 0, errors.New("insert doc: name is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin insert tx: %w", err)
	}
	defer tx.Rollback() // safe no-op after a successful Commit

	res, err := tx.ExecContext(ctx,
		`INSERT INTO docs (name, description, tags, file_path, created_at)
		 VALUES (?, ?, ?, '', ?)`,
		doc.Name, doc.Description, doc.Tags, time.Now().UTC().Unix())
	if err != nil {
		return 0, fmt.Errorf("insert doc row: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert doc lastInsertId: %w", err)
	}

	filePath := filepath.Join(s.filesDir, fmt.Sprintf("%d.html", id))
	if err := os.WriteFile(filePath, []byte(doc.HTMLContent), 0o644); err != nil {
		return 0, fmt.Errorf("write html file %s: %w", filePath, err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE docs SET file_path = ? WHERE id = ?`, filePath, id); err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("update doc file_path: %w", err)
	}

	if err := tx.Commit(); err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("commit insert tx: %w", err)
	}
	return id, nil
}

// GetByID loads a document's metadata by id. Returns sql.ErrNoRows (wrapped)
// when the id does not exist.
func (s *Store) GetByID(ctx context.Context, id int64) (Doc, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, tags, file_path, created_at
		 FROM docs WHERE id = ?`, id)
	d, err := scanDoc(row)
	if err != nil {
		return Doc{}, err
	}
	return d, nil
}

// GetByIDContent loads metadata and the raw HTML payload from disk.
func (s *Store) GetByIDContent(ctx context.Context, id int64) (Doc, error) {
	d, err := s.GetByID(ctx, id)
	if err != nil {
		return Doc{}, err
	}
	b, err := os.ReadFile(d.FilePath)
	if err != nil {
		return Doc{}, fmt.Errorf("read html content %s: %w", d.FilePath, err)
	}
	d.HTMLContent = string(b)
	return d, nil
}

// List returns the most-recent limit documents (by created_at desc). limit<=0
// falls back to DefaultLimit.
func (s *Store) List(ctx context.Context, limit int) ([]Doc, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, tags, file_path, created_at
		 FROM docs ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list docs: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// Search runs an FTS5 MATCH query across name/description/tags and returns
// matching documents ranked by relevance (bm25). An empty/no-match query
// returns an empty slice and nil error (not an error).
//
// The raw query is sanitized by buildFTSQuery: bare alphanumeric tokens get a
// trailing '*' for prefix matching (so "roll" finds "rollback"), and tokens
// containing FTS5 metacharacters (hyphens, colons, quotes, parens) are
// double-quoted so arbitrary user input can never produce a syntax error or
// an unintended column-filter. This makes the search box safe to feed
// untrusted strings directly.
func (s *Store) Search(ctx context.Context, query string) ([]Doc, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Doc{}, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT d.id, d.name, d.description, d.tags, d.file_path, d.created_at
		 FROM docs_fts f
		 JOIN docs d ON d.id = f.rowid
		 WHERE docs_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`, q, DefaultLimit)
	if err != nil {
		return nil, fmt.Errorf("search docs %q: %w", query, err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// buildFTSQuery converts a free-text user query into a safe FTS5 MATCH
// expression. Each whitespace-separated token becomes either:
//   - <token>*  when the token is a bare word (letters/digits/_), giving
//     prefix matching ("roll" -> matches "rollback");
//   - "<token>" when the token contains any FTS5 metacharacter, which
//     disables syntax interpretation and treats it as an exact phrase,
//     so strings like "zzz-nonexistent" can never raise "no such column".
//
// Tokens are joined with spaces (FTS5 implicit AND). An all-empty input
// yields an empty string, which Search short-circuits to an empty result.
func buildFTSQuery(input string) string {
	tokens := strings.Fields(input)
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if isBareWord(tok) {
			parts = append(parts, tok+"*")
		} else {
			parts = append(parts, "\""+tok+"\"")
		}
	}
	return strings.Join(parts, " ")
}

// isBareWord reports whether s contains only letters, digits, or underscore
// — the safe set that needs no quoting in FTS5.
func isBareWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func scanDoc(row *sql.Row) (Doc, error) {
	var d Doc
	var ts int64
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Tags, &d.FilePath, &ts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Doc{}, fmt.Errorf("doc %w", sql.ErrNoRows)
		}
		return Doc{}, fmt.Errorf("scan doc: %w", err)
	}
	d.CreatedAt = time.Unix(ts, 0).UTC()
	return d, nil
}

func scanRows(rows *sql.Rows) ([]Doc, error) {
	var out []Doc
	for rows.Next() {
		var d Doc
		var ts int64
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Tags, &d.FilePath, &ts); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		d.CreatedAt = time.Unix(ts, 0).UTC()
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}
	if out == nil {
		out = []Doc{}
	}
	return out, nil
}
