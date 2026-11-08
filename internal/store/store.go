// Package store defines the database-agnostic interface used by the loader,
// plus a DSN-based constructor that dispatches to a concrete backend.
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store/sqlite"
)

// Store is implemented by every supported database backend. Loading one
// kind is a three-phase sequence: PrepareX (wipe that kind's tables, so
// the load is a full replace rather than a merge -- Discogs dumps are
// complete monthly snapshots, not deltas, so there's no case where two
// different dumps for the same kind should be merged), then PutX called
// once per decoded batch with plain inserts, then FinishX (masters/
// releases only) to bulk-resolve free-text vocabulary into its lookup
// tables via SQL and drop the intermediate raw tables. Replacing the
// whole kind's data on every load also means a shrunk child list (fewer
// tracks, fewer urls) is reflected for free -- the old table is gone,
// not merged into.
type Store interface {
	PrepareArtists(ctx context.Context) error
	PutArtists(ctx context.Context, batch []model.Artist) error

	PrepareLabels(ctx context.Context) error
	PutLabels(ctx context.Context, batch []model.Label) error

	PrepareMasters(ctx context.Context) error
	PutMasters(ctx context.Context, batch []model.Master) error
	FinishMasters(ctx context.Context) error

	PrepareReleases(ctx context.Context) error
	PutReleases(ctx context.Context, batch []model.Release) error
	FinishReleases(ctx context.Context) error

	// BuildIndexes creates every index/view used for querying. Call this
	// once, explicitly, after all desired loads are done -- not
	// automatically after every load -- so bulk inserts never pay
	// index-maintenance cost mid-load.
	BuildIndexes(ctx context.Context) error
	Close() error
}

// Open opens a Store from a DSN of the form "scheme:path", e.g.
// "sqlite:discogs.db". Only the sqlite scheme is implemented today; the
// interface above is the seam a future backend (e.g. postgres) would
// implement without any change to callers.
func Open(dsn string) (Store, error) {
	scheme, path, ok := strings.Cut(dsn, ":")
	if !ok {
		return nil, fmt.Errorf("store: invalid DSN %q, want scheme:path", dsn)
	}
	switch scheme {
	case "sqlite":
		return sqlite.Open(path)
	default:
		return nil, fmt.Errorf("store: unsupported scheme %q", scheme)
	}
}
