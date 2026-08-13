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
	store, err := Open(filepath.Join(dir, "pages.db"), filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// --- slug validation ---

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid short", "deploy", false},
		{"valid dash-separated words", "deploy-runbook-v2", false},
		{"valid at max length", strings.Repeat("a", SlugMaxLength), false},
		{"path separator", "deploy/runbook", true},
		{"traversal segment", "../etc-passwd", true},
		{"traversal segment embedded", "deploy..runbook", true},
		{"uppercase letter", "Deploy", true},
		{"underscore", "deploy_runbook", true},
		{"space", "deploy runbook", true},
		{"leading dash", "-deploy", true},
		{"trailing dash", "deploy-", true},
		{"doubled dash", "deploy--runbook", true},
		{"over length", strings.Repeat("a", SlugMaxLength+1), true},
		{"empty", "", true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSlug(tc.slug)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateSlug(%q) = nil, want an error", tc.slug)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateSlug(%q) = %v, want nil", tc.slug, err)
			}
		})
	}
}

// --- description cap ---

func TestValidateDescription(t *testing.T) {
	if err := ValidateDescription(strings.Repeat("d", DescriptionMaxLength)); err != nil {
		t.Errorf("ValidateDescription at the %d-char cap = %v, want nil", DescriptionMaxLength, err)
	}
	err := ValidateDescription(strings.Repeat("d", DescriptionMaxLength+1))
	if err == nil {
		t.Fatal("ValidateDescription over the cap = nil, want an error")
	}
	if !strings.Contains(err.Error(), "200") {
		t.Errorf("error %q does not name the 200-character limit", err.Error())
	}
}

// --- create vs patch ---

func TestUpsert_Create_RoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Upsert(ctx, Doc{
		Slug:        "deploy-runbook",
		Name:        "Deploy Runbook",
		Description: "Step-by-step production deploy",
		HTMLContent: "<h1>Deploy</h1><p>Run me.</p>",
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !created {
		t.Fatal("Upsert on a new slug: created = false, want true")
	}

	got, err := store.GetBySlug(ctx, "deploy-runbook")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.Name != "Deploy Runbook" {
		t.Errorf("Name = %q, want %q", got.Name, "Deploy Runbook")
	}
	if got.Description != "Step-by-step production deploy" {
		t.Errorf("Description = %q", got.Description)
	}
	if filepath.Base(got.FilePath) != "deploy-runbook.html" {
		t.Errorf("FilePath base = %q, want %q", filepath.Base(got.FilePath), "deploy-runbook.html")
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("CreatedAt/UpdatedAt is zero")
	}
	if !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("on creation CreatedAt (%v) should equal UpdatedAt (%v)", got.CreatedAt, got.UpdatedAt)
	}

	// HTML payload round-trips via the content endpoint, not metadata.
	if got.HTMLContent != "" {
		t.Errorf("GetBySlug should not populate HTMLContent, got %q", got.HTMLContent)
	}
	full, err := store.GetBySlugContent(ctx, "deploy-runbook")
	if err != nil {
		t.Fatalf("GetBySlugContent: %v", err)
	}
	if full.HTMLContent != "<h1>Deploy</h1><p>Run me.</p>" {
		t.Errorf("HTMLContent = %q", full.HTMLContent)
	}
	if _, err := os.Stat(full.FilePath); err != nil {
		t.Errorf("html file missing on disk: %v", err)
	}
}

func TestUpsert_Patch_ReplacesContentPreservesCreatedAtAdvancesUpdatedAt(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Upsert(ctx, Doc{
		Slug:        "changelog",
		Name:        "Changelog",
		Description: "v1",
		HTMLContent: "<p>v1</p>",
	})
	if err != nil || !created {
		t.Fatalf("initial Upsert: created=%v err=%v, want true, nil", created, err)
	}
	first, err := store.GetBySlug(ctx, "changelog")
	if err != nil {
		t.Fatalf("GetBySlug after create: %v", err)
	}

	created, err = store.Upsert(ctx, Doc{
		Slug:        "changelog",
		Name:        "Changelog v2",
		Description: "v2",
		HTMLContent: "<p>v2</p>",
	})
	if err != nil {
		t.Fatalf("patch Upsert: %v", err)
	}
	if created {
		t.Fatal("Upsert on an existing slug: created = true, want false")
	}

	second, err := store.GetBySlug(ctx, "changelog")
	if err != nil {
		t.Fatalf("GetBySlug after patch: %v", err)
	}
	if second.Name != "Changelog v2" || second.Description != "v2" {
		t.Errorf("patched metadata = %+v, want Name/Description updated", second)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("CreatedAt changed across a patch: %v -> %v, want unchanged", first.CreatedAt, second.CreatedAt)
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Errorf("UpdatedAt did not advance across a patch: %v -> %v", first.UpdatedAt, second.UpdatedAt)
	}

	// Only one row exists at this slug — exactly one page.
	all, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List after a patch returned %d row(s), want 1 (no second page created)", len(all))
	}

	// Prior content is not retrievable after the patch.
	content, err := store.GetBySlugContent(ctx, "changelog")
	if err != nil {
		t.Fatalf("GetBySlugContent after patch: %v", err)
	}
	if content.HTMLContent != "<p>v2</p>" {
		t.Errorf("content after patch = %q, want %q", content.HTMLContent, "<p>v2</p>")
	}
	if strings.Contains(content.HTMLContent, "v1") {
		t.Errorf("patched-away HTML is still retrievable: %q", content.HTMLContent)
	}
}

func TestUpsert_MissingNameRejected(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Upsert(context.Background(), Doc{Slug: "no-name", Name: "", HTMLContent: "x"})
	if err == nil {
		t.Fatal("expected error for empty Name, got nil")
	}
}

func TestUpsert_InvalidSlugRejected_NoFileNoRow(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Upsert(ctx, Doc{Slug: "not/a slug", Name: "x", HTMLContent: "x"})
	if err == nil {
		t.Fatal("expected error for an invalid slug, got nil")
	}

	docs, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("List after a rejected slug returned %d row(s), want 0", len(docs))
	}
}

func TestUpsert_OverLongDescriptionRejected_NoRowCreated(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Upsert(ctx, Doc{
		Slug:        "too-much-description",
		Name:        "x",
		Description: strings.Repeat("d", DescriptionMaxLength+1),
		HTMLContent: "x",
	})
	if err == nil {
		t.Fatal("expected error for an over-limit description, got nil")
	}

	docs, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("List after a rejected description returned %d row(s), want 0", len(docs))
	}
}

// TestUpsert_FileWriteFailureRollsBackMetadataRow proves the documented
// invariant in Upsert's doc comment for the create path: if the file write
// fails, the transaction rolls back (no dangling metadata row). It forces
// os.WriteFile to fail by pre-creating a directory at the exact path Upsert
// will try to write the HTML file to (deterministic from the slug), then
// asserts (a) Upsert returns an error and (b) List sees zero rows — no
// orphan metadata row left behind by the aborted create.
func TestUpsert_FileWriteFailureRollsBackMetadataRow(t *testing.T) {
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	store, err := Open(filepath.Join(dir, "pages.db"), filesDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	ctx := context.Background()

	blockedPath := filepath.Join(filesDir, "doomed-page.html")
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("pre-create blocking directory %s: %v", blockedPath, err)
	}

	_, err = store.Upsert(ctx, Doc{Slug: "doomed-page", Name: "Doomed Page", HTMLContent: "<p>x</p>"})
	if err == nil {
		t.Fatal("Upsert: expected an error when the html file write is blocked, got nil")
	}
	if !strings.Contains(err.Error(), "write html file") {
		t.Errorf("Upsert error = %q, want it to name the file-write failure", err.Error())
	}

	docs, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("List after a failed Upsert returned %d row(s), want 0", len(docs))
	}
}

// --- search ---

func TestSearch_FTS5(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []Doc{
		{Slug: "alpha-deploy-guide", Name: "Alpha Deploy Guide", Description: "filler one", HTMLContent: "a"},
		{Slug: "beta-filler", Name: "blueprint filler", Description: "rollback plan beta", HTMLContent: "b"},
		{Slug: "gamma-filler", Name: "filler", Description: "blueprint notes", HTMLContent: "c"},
		{Slug: "delta-unrelated", Name: "totally unrelated", Description: "no keywords here", HTMLContent: "d"},
	}
	for i, d := range docs {
		if _, err := store.Upsert(ctx, d); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}

	tests := []struct {
		name      string
		query     string
		wantNames []string
	}{
		{"name match", "deploy", []string{"Alpha Deploy Guide"}},
		{"description match", "rollback", []string{"blueprint filler"}},
		{"multi-field match", "blueprint", []string{"blueprint filler", "filler"}},
		{"partial prefix match", "roll", []string{"blueprint filler"}},
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

func TestSearch_IndexesOnlyNameAndDescription(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Upsert(ctx, Doc{Slug: "a-page", Name: "A Page", Description: "nothing special", HTMLContent: "irrelevant-html-marker"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := store.Search(ctx, "irrelevant-html-marker")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Search matched HTML content, want name/description-only indexing: %+v", got)
	}
}

// --- list ordering ---

func TestList_MostRecentlyChangedFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Upsert(ctx, Doc{Slug: "page-a", Name: "oldest", HTMLContent: "a"}); err != nil {
		t.Fatalf("Upsert page-a: %v", err)
	}
	if _, err := store.Upsert(ctx, Doc{Slug: "page-b", Name: "middle", HTMLContent: "b"}); err != nil {
		t.Fatalf("Upsert page-b: %v", err)
	}
	if _, err := store.Upsert(ctx, Doc{Slug: "page-c", Name: "newest", HTMLContent: "c"}); err != nil {
		t.Fatalf("Upsert page-c: %v", err)
	}

	got, err := store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 || got[0].Slug != "page-c" || got[1].Slug != "page-b" || got[2].Slug != "page-a" {
		t.Fatalf("List order = %v, want [page-c page-b page-a] (newest-changed first)", sluglist(got))
	}

	// Patching the oldest page must move it to the front, ahead of pages
	// created more recently but not patched since.
	if _, err := store.Upsert(ctx, Doc{Slug: "page-a", Name: "oldest, now patched", HTMLContent: "a2"}); err != nil {
		t.Fatalf("patch page-a: %v", err)
	}
	got, err = store.List(ctx, 0)
	if err != nil {
		t.Fatalf("List after patch: %v", err)
	}
	if len(got) != 3 || got[0].Slug != "page-a" {
		t.Fatalf("List order after patching page-a = %v, want page-a first", sluglist(got))
	}
}

func sluglist(docs []Doc) []string {
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = d.Slug
	}
	return out
}

func TestList_LimitCap(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := range 5 {
		slug := "page-" + string(rune('a'+i))
		if _, err := store.Upsert(ctx, Doc{Slug: slug, Name: "doc", HTMLContent: "x"}); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
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

func TestGetBySlug_Missing(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetBySlug(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing slug, got nil")
	}
	// sql.ErrNoRows sentinel must be unwrappable so callers can branch.
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected a sql.ErrNoRows-wrapped error, got: %v", err)
	}
}

func TestGetBySlugContent_MissingSlug(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetBySlugContent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing slug content, got nil")
	}
}
