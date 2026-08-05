package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStore opens a Store against a temp dir and returns it with a cleanup
// func. t.TempDir() is auto-cleaned by the test framework; we only close the
// DB. This is the canonical harness used by every test in this file.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "docs.db"), filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestInsert_RoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.Insert(ctx, Doc{
		Name:        "Deploy Runbook",
		Description: "Step-by-step production deploy",
		Tags:        "ops,deploy",
		HTMLContent: "<h1>Deploy</h1><p>Run me.</p>",
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id <= 0 {
		t.Fatalf("got non-positive id %d", id)
	}

	got, err := store.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID(%d): %v", id, err)
	}
	if got.Name != "Deploy Runbook" {
		t.Errorf("Name = %q, want %q", got.Name, "Deploy Runbook")
	}
	if got.Description != "Step-by-step production deploy" {
		t.Errorf("Description = %q", got.Description)
	}
	if got.Tags != "ops,deploy" {
		t.Errorf("Tags = %q", got.Tags)
	}
	if got.FilePath == "" {
		t.Error("FilePath is empty; should be <filesDir>/<id>.html")
	}
	if filepath.Base(got.FilePath) != "1.html" {
		t.Errorf("FilePath base = %q, want %q (first insert should be id 1)", filepath.Base(got.FilePath), "1.html")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// HTML payload round-trips via the content endpoint, not metadata.
	if got.HTMLContent != "" {
		t.Errorf("GetByID should not populate HTMLContent, got %q", got.HTMLContent)
	}
	full, err := store.GetByIDContent(ctx, id)
	if err != nil {
		t.Fatalf("GetByIDContent(%d): %v", id, err)
	}
	if full.HTMLContent != "<h1>Deploy</h1><p>Run me.</p>" {
		t.Errorf("HTMLContent = %q", full.HTMLContent)
	}

	// the file actually exists on disk
	if _, err := os.Stat(full.FilePath); err != nil {
		t.Errorf("html file missing on disk: %v", err)
	}
}

func TestInsert_IncrementingIDs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		id, err := store.Insert(ctx, Doc{Name: "doc", HTMLContent: "x"})
		if err != nil {
			t.Fatalf("Insert #%d: %v", i, err)
		}
		if id != int64(i) {
			t.Errorf("insert #%d got id %d, want %d", i, id, i)
		}
	}
}

func TestInsert_MissingNameRejected(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Insert(context.Background(), Doc{Name: "", HTMLContent: "x"})
	if err == nil {
		t.Fatal("expected error for empty Name, got nil")
	}
}

// TestInsert_FileWriteFailureRollsBackMetadataRow proves the documented
// invariant in Insert's doc comment: "If the file write fails the
// transaction rolls back (no dangling metadata row)". It forces
// os.WriteFile to fail by pre-creating a directory at the exact path Insert
// will try to write the HTML file to (id 1's first insert is always
// <filesDir>/1.html per TestInsert_RoundTrip), then asserts (a) Insert
// returns an error, (b) List sees zero rows — no orphan metadata row
// left behind by the aborted insert — and (c) a subsequent successful
// Insert reuses id 1, proving the AUTOINCREMENT sequence itself rolled back
// too, not just the row.
func TestInsert_FileWriteFailureRollsBackMetadataRow(t *testing.T) {
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	store, err := Open(filepath.Join(dir, "docs.db"), filesDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

	// Block the write: Insert's first (and, on a fresh store, only) attempt
	// will target <filesDir>/1.html. Pre-creating a directory there makes
	// os.WriteFile fail with "is a directory" instead of writing the file.
	blockedPath := filepath.Join(filesDir, "1.html")
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("pre-create blocking directory %s: %v", blockedPath, err)
	}

	_, err = store.Insert(ctx, Doc{Name: "Doomed Doc", HTMLContent: "<p>x</p>"})
	if err == nil {
		t.Fatal("Insert: expected an error when the html file write is blocked, got nil")
	}
	if !strings.Contains(err.Error(), "write html file") {
		t.Errorf("Insert error = %q, want it to name the file-write failure", err.Error())
	}

	docs, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("List after a failed Insert returned %d doc(s), want 0 (metadata row must roll back with the failed file write)", len(docs))
	}

	// Unblock the path and prove the sequence itself rolled back: the next
	// successful Insert must reuse id 1, not skip to 2.
	if err := os.RemoveAll(blockedPath); err != nil {
		t.Fatalf("remove blocking directory: %v", err)
	}
	id, err := store.Insert(ctx, Doc{Name: "Recovered Doc", HTMLContent: "<p>ok</p>"})
	if err != nil {
		t.Fatalf("Insert after unblocking: %v", err)
	}
	if id != 1 {
		t.Errorf("Insert after the rolled-back failure got id %d, want 1 (rollback must also revert the AUTOINCREMENT sequence)", id)
	}
}

func TestSearch_FTS5(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Seed documents with terms placed in known single fields plus one
	// cross-field term (blueprint) so we can assert per-field FTS5 indexing
	// and a genuine multi-field match. Doc 4 matches nothing.
	docs := []Doc{
		{Name: "Alpha Deploy Guide", Description: "filler one", Tags: "blueprint", HTMLContent: "a"},
		{Name: "blueprint filler", Description: "rollback plan beta", Tags: "filler", HTMLContent: "b"},
		{Name: "filler", Description: "blueprint notes", Tags: "canary release", HTMLContent: "c"},
		{Name: "totally unrelated", Description: "no keywords here", Tags: "none", HTMLContent: "d"},
	}
	ids := make([]int64, len(docs))
	for i, d := range docs {
		id, err := store.Insert(ctx, d)
		if err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
		ids[i] = id
	}

	tests := []struct {
		name      string
		query     string
		wantNames []string
	}{
		// per-field: each term is unique to exactly one field of one doc
		{"name match", "deploy", []string{"Alpha Deploy Guide"}},
		{"description match", "rollback", []string{"blueprint filler"}},
		{"tags match", "canary", []string{"filler"}},
		// blueprint appears in doc1 tags, doc2 name, doc3 description -> multi-field
		{"multi-field match", "blueprint", []string{"Alpha Deploy Guide", "blueprint filler", "filler"}},
		// prefix matching: "roll" should match "rollback" in doc2 description
		{"partial prefix match", "roll", []string{"blueprint filler"}},
		// metacharacter input must be quoted, not raise a syntax error
		{"no match returns empty not error", "zzz-nonexistent", []string{}},
		{"empty query returns empty not error", "", []string{}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.Search(ctx, tc.query)
			if err != nil {
				t.Fatalf("Search(%q): %v", tc.query, err)
			}
			if len(got) != len(tc.wantNames) {
				t.Fatalf("Search(%q) got %d results, want %d (%+v)", tc.query, len(got), len(tc.wantNames), got)
			}
			gotNames := make(map[string]bool, len(got))
			for _, d := range got {
				gotNames[d.Name] = true
			}
			for _, w := range tc.wantNames {
				if !gotNames[w] {
					t.Errorf("Search(%q): want name %q in results, got %+v", tc.query, w, got)
				}
			}
		})
	}
}

func TestList_MostRecentFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert 3 docs; List should return them newest-first.
	id1, _ := store.Insert(ctx, Doc{Name: "oldest", HTMLContent: "a"})
	id2, _ := store.Insert(ctx, Doc{Name: "middle", HTMLContent: "b"})
	id3, _ := store.Insert(ctx, Doc{Name: "newest", HTMLContent: "c"})

	got, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List got %d docs, want 3", len(got))
	}
	// created_at desc; ties broken by id desc
	if got[0].ID != id3 || got[1].ID != id2 || got[2].ID != id1 {
		t.Errorf("List order = %d,%d,%d; want %d,%d,%d (newest first)",
			got[0].ID, got[1].ID, got[2].ID, id3, id2, id1)
	}
}

func TestList_LimitCap(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := store.Insert(ctx, Doc{Name: "doc", HTMLContent: "x"}); err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
	}
	got, err := store.List(ctx, 2)
	if err != nil {
		t.Fatalf("List(2): %v", err)
	}
	if len(got) != 2 {
		t.Errorf("List(2) returned %d, want 2 (limit should cap)", len(got))
	}
}

func TestGetByID_Missing(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
	// sql.ErrNoRows sentinel must be unwrappable so callers can branch.
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected a sql.ErrNoRows-wrapped error, got: %v", err)
	}
}

func TestGetByIDContent_MissingID(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetByIDContent(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for missing id content, got nil")
	}
}
