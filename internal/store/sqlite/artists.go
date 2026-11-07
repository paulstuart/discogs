package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PutArtists upserts a batch of artists and fully replaces each artist's
// child rows (urls, name variations, aliases, members) within one
// transaction.
func (s *SQLiteStore) PutArtists(ctx context.Context, batch []model.Artist) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists begin: %w", err)
	}
	defer tx.Rollback()

	upsert, err := tx.PrepareContext(ctx, `
		INSERT INTO artists (id, name, real_name, profile, data_quality)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, real_name=excluded.real_name,
			profile=excluded.profile, data_quality=excluded.data_quality`)
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare upsert: %w", err)
	}
	defer upsert.Close()

	delURLs, err := tx.PrepareContext(ctx, "DELETE FROM artist_urls WHERE artist_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare delete urls: %w", err)
	}
	defer delURLs.Close()
	insURL, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_urls (artist_id, url) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert urls: %w", err)
	}
	defer insURL.Close()

	delVariations, err := tx.PrepareContext(ctx, "DELETE FROM artist_name_variations WHERE artist_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare delete variations: %w", err)
	}
	defer delVariations.Close()
	insVariation, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_name_variations (artist_id, name) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert variations: %w", err)
	}
	defer insVariation.Close()

	delAliases, err := tx.PrepareContext(ctx, "DELETE FROM artist_aliases WHERE artist_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare delete aliases: %w", err)
	}
	defer delAliases.Close()
	insAlias, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_aliases (artist_id, alias_id, alias_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert aliases: %w", err)
	}
	defer insAlias.Close()

	delMembers, err := tx.PrepareContext(ctx, "DELETE FROM artist_members WHERE artist_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare delete members: %w", err)
	}
	defer delMembers.Close()
	insMember, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO artist_members (artist_id, member_id, member_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutArtists prepare insert members: %w", err)
	}
	defer insMember.Close()

	for _, a := range batch {
		if _, err := upsert.ExecContext(ctx, a.ID, a.Name, nullIfEmpty(a.RealName), nullIfEmpty(a.Profile), nullIfEmpty(a.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutArtists upsert id=%d: %w", a.ID, err)
		}

		if _, err := delURLs.ExecContext(ctx, a.ID); err != nil {
			return fmt.Errorf("sqlite: PutArtists delete urls id=%d: %w", a.ID, err)
		}
		for _, u := range a.URLs {
			if _, err := insURL.ExecContext(ctx, a.ID, u); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert url id=%d: %w", a.ID, err)
			}
		}

		if _, err := delVariations.ExecContext(ctx, a.ID); err != nil {
			return fmt.Errorf("sqlite: PutArtists delete variations id=%d: %w", a.ID, err)
		}
		for _, nv := range a.NameVariations {
			if _, err := insVariation.ExecContext(ctx, a.ID, nv); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert variation id=%d: %w", a.ID, err)
			}
		}

		if _, err := delAliases.ExecContext(ctx, a.ID); err != nil {
			return fmt.Errorf("sqlite: PutArtists delete aliases id=%d: %w", a.ID, err)
		}
		for _, al := range a.Aliases {
			if _, err := insAlias.ExecContext(ctx, a.ID, al.ID, al.Name); err != nil {
				return fmt.Errorf("sqlite: PutArtists insert alias id=%d: %w", a.ID, err)
			}
		}

		if _, err := delMembers.ExecContext(ctx, a.ID); err != nil {
			return fmt.Errorf("sqlite: PutArtists delete members id=%d: %w", a.ID, err)
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
