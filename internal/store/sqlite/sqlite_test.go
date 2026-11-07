package sqlite

import (
	"context"
	"path/filepath"
	"testing"
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

func scalar[T any](t *testing.T, s *SQLiteStore, query string, args ...any) T {
	t.Helper()
	var v T
	if err := s.db.QueryRowContext(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return v
}
