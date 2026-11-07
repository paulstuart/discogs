// Package sqlite implements store.Store on top of SQLite via
// github.com/mattn/go-sqlite3, tuned for large bulk-insert workloads.
package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// SQLiteStore implements store.Store.
type SQLiteStore struct {
	db *sql.DB
}

// Open opens (creating if necessary) a SQLite database at path, applies
// bulk-load pragmas, and ensures the schema exists.
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

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
