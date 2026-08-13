// Package storage implements the SQLite+FTS5 persistence layer for hyperreader.
//
// A Store wraps a *sql.DB backed by modernc.org/sqlite (pure Go, no cgo).
// Pages are stored as metadata in the pages table plus an FTS5 external-
// content index over name/description; the raw HTML payload lives on disk
// under <filesDir>/<slug>.html and only its path is recorded in SQLite.
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
	"regexp"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

// DefaultLimit caps unfiltered List results so a degenerate query cannot
// pull the whole table into memory.
const DefaultLimit = 100

// SlugMaxLength is the maximum length of a page slug. The slug is also a
// filename component and a URL path segment, so the cap keeps it well
// under filesystem filename limits.
const SlugMaxLength = 80

// DescriptionMaxLength is the maximum length of a page description,
// enforced server-side; an over-limit description is rejected, never
// silently truncated.
const DescriptionMaxLength = 200

// slugPattern is the sole legal shape for a page slug: lowercase letters
// and digits, grouped into dash-separated words, with no leading dash, no
// trailing dash, and no consecutive dashes. It excludes '/', '.', '..',
// whitespace, and every other filesystem- or URL-meaningful character by
// construction (only [a-z0-9-] survives), since the slug is also the SQL
// primary key, the filename (files/<slug>.html), and the URL path segment.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateSlug reports whether slug is a legal page identifier: matching
// slugPattern and no longer than SlugMaxLength. It returns a clear,
// user-facing error naming the violated rule; callers must run this before
// any storage or filesystem operation, since an unvalidated slug in either
// position is an injection or traversal vector.
func ValidateSlug(slug string) error {
	if len(slug) > SlugMaxLength {
		return fmt.Errorf("slug exceeds the maximum length of %d characters", SlugMaxLength)
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug %q must match %s (lowercase letters, digits, and single dashes between words)", slug, slugPattern.String())
	}
	return nil
}

// ValidateDescription reports whether description is within
// DescriptionMaxLength, returning a clear error naming the limit if not.
func ValidateDescription(description string) error {
	if len(description) > DescriptionMaxLength {
		return fmt.Errorf("description exceeds the maximum length of %d characters", DescriptionMaxLength)
	}
	return nil
}

// Doc is the in-memory representation of a row in pages. HTMLContent is the
// raw payload read back from disk; it is populated only by callers that
// need it (e.g. GetBySlugContent), never by list/search which return
// metadata.
type Doc struct {
	Slug        string
	Name        string
	Description string
	FilePath    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

// Upsert persists a page's metadata to SQLite and its HTML payload to disk,
// creating a new page when doc.Slug does not already exist and patching the
// existing one in place otherwise. It returns created=true for a new page.
//
// Slug and description are validated first, before any storage or
// filesystem operation. On creation created_at and updated_at are both set
// to now; on a patch, created_at is preserved and updated_at advances to
// now. The row (with its final file_path, deterministic from the slug) is
// written first, then the HTML file, then the transaction is committed —
// the same crash-safety ordering the original create-only Insert used: if
// the file write fails the transaction rolls back (no dangling or
// half-patched metadata row); if the commit fails after the file write, an
// orphan/overwritten .html file may remain — inert, and preferable to a
// metadata row that disagrees with the file on disk.
func (s *Store) Upsert(ctx context.Context, doc Doc) (created bool, err error) {
	if err := ValidateSlug(doc.Slug); err != nil {
		return false, fmt.Errorf("upsert page: %w", err)
	}
	if doc.Name == "" {
		return false, errors.New("upsert page: name is required")
	}
	if err := ValidateDescription(doc.Description); err != nil {
		return false, fmt.Errorf("upsert page: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin upsert tx: %w", err)
	}
	defer tx.Rollback() // safe no-op after a successful Commit

	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM pages WHERE slug = ?)`, doc.Slug).Scan(&exists); err != nil {
		return false, fmt.Errorf("check slug existence: %w", err)
	}

	now := time.Now().UTC().UnixNano()
	filePath := filepath.Join(s.filesDir, doc.Slug+".html")

	if exists {
		if _, err := tx.ExecContext(ctx,
			`UPDATE pages SET name = ?, description = ?, file_path = ?, updated_at = ? WHERE slug = ?`,
			doc.Name, doc.Description, filePath, now, doc.Slug); err != nil {
			return false, fmt.Errorf("update page row: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO pages (slug, name, description, file_path, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			doc.Slug, doc.Name, doc.Description, filePath, now, now); err != nil {
			return false, fmt.Errorf("insert page row: %w", err)
		}
	}

	if err := os.WriteFile(filePath, []byte(doc.HTMLContent), 0o644); err != nil {
		return false, fmt.Errorf("write html file %s: %w", filePath, err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit upsert tx: %w", err)
	}
	return !exists, nil
}

// GetBySlug loads a page's metadata by slug. Returns sql.ErrNoRows
// (wrapped) when the slug does not exist.
func (s *Store) GetBySlug(ctx context.Context, slug string) (Doc, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT slug, name, description, file_path, created_at, updated_at
		 FROM pages WHERE slug = ?`, slug)
	d, err := scanDoc(row)
	if err != nil {
		return Doc{}, err
	}
	return d, nil
}

// GetBySlugContent loads metadata and the raw HTML payload from disk.
func (s *Store) GetBySlugContent(ctx context.Context, slug string) (Doc, error) {
	d, err := s.GetBySlug(ctx, slug)
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

// List returns the most-recently-changed limit pages (by updated_at desc,
// ties broken by rowid desc for a stable, insertion-order-respecting
// tiebreak). limit<=0 falls back to DefaultLimit.
func (s *Store) List(ctx context.Context, limit int) ([]Doc, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT slug, name, description, file_path, created_at, updated_at
		 FROM pages ORDER BY updated_at DESC, rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// Search runs an FTS5 MATCH query across name/description and returns
// matching pages ranked by relevance (bm25), most-recently-changed first
// among ties. An empty/no-match query returns an empty slice and nil error
// (not an error).
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
		`SELECT p.slug, p.name, p.description, p.file_path, p.created_at, p.updated_at
		 FROM pages_fts f
		 JOIN pages p ON p.rowid = f.rowid
		 WHERE pages_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`, q, DefaultLimit)
	if err != nil {
		return nil, fmt.Errorf("search pages %q: %w", query, err)
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
	var createdTS, updatedTS int64
	err := row.Scan(&d.Slug, &d.Name, &d.Description, &d.FilePath, &createdTS, &updatedTS)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Doc{}, fmt.Errorf("page %w", sql.ErrNoRows)
		}
		return Doc{}, fmt.Errorf("scan page: %w", err)
	}
	d.CreatedAt = time.Unix(0, createdTS).UTC()
	d.UpdatedAt = time.Unix(0, updatedTS).UTC()
	return d, nil
}

func scanRows(rows *sql.Rows) ([]Doc, error) {
	var out []Doc
	for rows.Next() {
		var d Doc
		var createdTS, updatedTS int64
		if err := rows.Scan(&d.Slug, &d.Name, &d.Description, &d.FilePath, &createdTS, &updatedTS); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		d.CreatedAt = time.Unix(0, createdTS).UTC()
		d.UpdatedAt = time.Unix(0, updatedTS).UTC()
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
