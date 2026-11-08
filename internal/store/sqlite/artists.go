package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PrepareArtists wipes every artist table so the load that follows fully
// replaces this kind's data (see the Store interface doc).
func (s *SQLiteStore) PrepareArtists(ctx context.Context) error {
	for _, table := range []string{"artists", "artist_urls", "artist_name_variations", "artist_aliases", "artist_members"} {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("sqlite: PrepareArtists wipe %s: %w", table, err)
		}
	}
	return nil
}

// PutArtists inserts a batch of artists and their child rows (urls, name
// variations, aliases, members). Plain inserts only -- PrepareArtists
// already wiped the tables for this load, so there's nothing to upsert
// or delete per record.
func (s *SQLiteStore) PutArtists(ctx context.Context, batch []model.Artist) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists begin: %w", err)
	}
	defer tx.Rollback()

	insArtist, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artists (id, name, real_name, profile, data_quality) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert: %w", err)
	}
	defer insArtist.Close()

	insURL, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_urls (artist_id, url) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert urls: %w", err)
	}
	defer insURL.Close()

	insVariation, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_name_variations (artist_id, name) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert variations: %w", err)
	}
	defer insVariation.Close()

	insAlias, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_aliases (artist_id, alias_id, alias_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert aliases: %w", err)
	}
	defer insAlias.Close()

	insMember, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_members (artist_id, member_id, member_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert members: %w", err)
	}
	defer insMember.Close()

	for _, a := range batch {
		if _, err := insArtist.ExecContext(ctx, a.ID, a.Name, nullIfEmpty(a.RealName), nullIfEmpty(a.Profile), nullIfEmpty(a.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutArtists insert id=%d: %w", a.ID, err)
		}
		for _, u := range a.URLs {
			if _, err := insURL.ExecContext(ctx, a.ID, u); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert url id=%d: %w", a.ID, err)
			}
		}
		for _, nv := range a.NameVariations {
			if _, err := insVariation.ExecContext(ctx, a.ID, nv); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert variation id=%d: %w", a.ID, err)
			}
		}
		for _, al := range a.Aliases {
			if _, err := insAlias.ExecContext(ctx, a.ID, al.ID, al.Name); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert alias id=%d: %w", a.ID, err)
			}
		}
		for _, m := range a.Members {
			if _, err := insMember.ExecContext(ctx, a.ID, m.ID, m.Name); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert member id=%d: %w", a.ID, err)
			}
		}
	}

	return tx.Commit()
}

// nullIfEmpty converts an empty string to a SQL NULL so optional text
// columns stay NULL rather than storing "".
func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
