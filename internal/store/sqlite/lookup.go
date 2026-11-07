package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// lookupCache resolves a free-text vocabulary value (genre, style,
// country, identifier type) to its surrogate id in a lookup table, with
// an in-process cache. The vocabulary for these is small (dozens to a
// few hundred distinct values) and stable within a load run, so after
// warmup this is a pure in-memory lookup rather than a DB round trip per
// row. The mutex is defensive -- current callers are single-writer,
// matching SetMaxOpenConns(1) in sqlite.go -- but costs nothing at this
// call volume.
type lookupCache struct {
	table string
	mu    sync.Mutex
	cache map[string]int64
}

func newLookupCache(table string) *lookupCache {
	return &lookupCache{table: table, cache: map[string]int64{}}
}

// id returns the surrogate id for name, inserting a new row (via the
// given tx, so it's part of the caller's transaction) if name hasn't
// been seen before.
func (c *lookupCache) id(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if id, ok := c.cache[name]; ok {
		return id, nil
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (name) VALUES (?)
		ON CONFLICT(name) DO UPDATE SET name = excluded.name
		RETURNING id`, c.table)
	var id int64
	if err := tx.QueryRowContext(ctx, query, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("sqlite: lookup %s %q: %w", c.table, name, err)
	}
	c.cache[name] = id
	return id, nil
}
