package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/paulstuart/discogs/internal/model"
)

// openTestStore opens a fresh SQLite store backed by a temp file (not
// in-memory, so the real file-based pragma/WAL path is exercised), and
// builds indexes/views immediately -- unlike a real bulk load, test
// fixtures are a handful of rows, so there's no mid-insert cost to avoid
// by deferring BuildIndexes, and tests want views available right away.
func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.BuildIndexes(context.Background()); err != nil {
		t.Fatalf("BuildIndexes: %v", err)
	}
	return s
}

// putArtistsFull drives the same Prepare+Put sequence a real `load`
// invocation does, so tests exercise the full-replace wipe rather than
// calling PutArtists against whatever tables already happened to hold.
func putArtistsFull(t *testing.T, s *SQLiteStore, batch []model.Artist) {
	t.Helper()
	ctx := context.Background()
	if err := s.PrepareArtists(ctx); err != nil {
		t.Fatalf("PrepareArtists: %v", err)
	}
	if err := s.PutArtists(ctx, batch); err != nil {
		t.Fatalf("PutArtists: %v", err)
	}
}

func putLabelsFull(t *testing.T, s *SQLiteStore, batch []model.Label) {
	t.Helper()
	ctx := context.Background()
	if err := s.PrepareLabels(ctx); err != nil {
		t.Fatalf("PrepareLabels: %v", err)
	}
	if err := s.PutLabels(ctx, batch); err != nil {
		t.Fatalf("PutLabels: %v", err)
	}
}

func putMastersFull(t *testing.T, s *SQLiteStore, batch []model.Master) {
	t.Helper()
	ctx := context.Background()
	if err := s.PrepareMasters(ctx); err != nil {
		t.Fatalf("PrepareMasters: %v", err)
	}
	if err := s.PutMasters(ctx, batch); err != nil {
		t.Fatalf("PutMasters: %v", err)
	}
	if err := s.FinishMasters(ctx); err != nil {
		t.Fatalf("FinishMasters: %v", err)
	}
}

func putReleasesFull(t *testing.T, s *SQLiteStore, batch []model.Release) {
	t.Helper()
	ctx := context.Background()
	if err := s.PrepareReleases(ctx); err != nil {
		t.Fatalf("PrepareReleases: %v", err)
	}
	if err := s.PutReleases(ctx, batch); err != nil {
		t.Fatalf("PutReleases: %v", err)
	}
	if err := s.FinishReleases(ctx); err != nil {
		t.Fatalf("FinishReleases: %v", err)
	}
}

func scalar[T any](t *testing.T, s *SQLiteStore, query string, args ...any) T {
	t.Helper()
	var v T
	if err := s.db.QueryRowContext(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return v
}
