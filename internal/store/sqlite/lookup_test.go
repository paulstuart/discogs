package sqlite

import (
	"context"
	"testing"
)

func TestLookupCache(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer tx.Rollback()

	c := newLookupCache("genres")

	id1, err := c.id(ctx, tx, "Electronic")
	if err != nil {
		t.Fatalf("id(Electronic): %v", err)
	}
	id1Again, err := c.id(ctx, tx, "Electronic")
	if err != nil {
		t.Fatalf("id(Electronic) again: %v", err)
	}
	if id1 != id1Again {
		t.Errorf("id(Electronic) = %d then %d, want same id (cache hit)", id1, id1Again)
	}

	id2, err := c.id(ctx, tx, "Rock")
	if err != nil {
		t.Fatalf("id(Rock): %v", err)
	}
	if id2 == id1 {
		t.Errorf("id(Rock) = %d, want different id from id(Electronic) = %d", id2, id1)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	count := scalar[int](t, s, "SELECT count(*) FROM genres")
	if count != 2 {
		t.Errorf("genres row count = %d, want 2 (one insert per distinct name)", count)
	}
}
