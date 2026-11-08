// Package sqlite implements store.Store on top of SQLite via
// github.com/mattn/go-sqlite3, tuned for large bulk-insert workloads.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

//go:embed indexes.sql
var indexesSQL string

// SQLiteStore implements store.Store.
type SQLiteStore struct {
	db *sql.DB
}

// Open opens (creating if necessary) a SQLite database at path, applies
// bulk-load pragmas, and ensures the table schema exists. Indexes and
// views are not created here -- see BuildIndexes -- so a fresh load never
// pays index-maintenance cost mid-insert.
func Open(path string) (*SQLiteStore, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=off", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %s: %w", path, err)
	}
	// SQLite is single-writer; serialize all writes through one connection
	// rather than fighting SQLITE_BUSY with retries/timeouts.
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA temp_store = MEMORY",
		"PRAGMA cache_size = -262144", // 256MB page cache
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite: %s: %w", pragma, err)
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: apply schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// BuildIndexes creates every index and view defined in indexes.sql. Run
// this once, explicitly (the `discogs index` subcommand), after all
// desired `load --kinds` invocations are done -- not automatically at the
// end of every load, since that would reintroduce incremental
// index-maintenance cost on any load that runs after the first one builds
// the indexes.
func (s *SQLiteStore) BuildIndexes(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, indexesSQL); err != nil {
		return fmt.Errorf("sqlite: build indexes: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
