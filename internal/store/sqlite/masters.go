package sqlite

import (
	"context"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PrepareMasters wipes every master table, so the load that follows
// fully replaces this kind's data (see the Store interface doc), and
// creates the raw staging tables PutMasters/FinishMasters use to
// normalize genre/style text in bulk after the load instead of
// resolving it per row during the hot insert path.
func (s *SQLiteStore) PrepareMasters(ctx context.Context) error {
	for _, table := range []string{"masters", "master_genres", "master_styles", "master_artists", "master_videos"} {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("sqlite: PrepareMasters wipe %s: %w", table, err)
		}
	}
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS master_genres_raw",
		"DROP TABLE IF EXISTS master_styles_raw",
		"CREATE TABLE master_genres_raw (master_id INTEGER NOT NULL, genre TEXT NOT NULL)",
		"CREATE TABLE master_styles_raw (master_id INTEGER NOT NULL, style TEXT NOT NULL)",
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: PrepareMasters %q: %w", stmt, err)
		}
	}
	return nil
}

// PutMasters inserts a batch of masters and their child rows (artists,
// videos) directly into their final tables, and genre/style text into
// the master_genres_raw/master_styles_raw staging tables for
// FinishMasters to resolve in bulk.
func (s *SQLiteStore) PutMasters(ctx context.Context, batch []model.Master) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters begin: %w", err)
	}
	defer tx.Rollback()

	insMaster, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO masters (id, main_release, year, title, data_quality)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert: %w", err)
	}
	defer insMaster.Close()

	insGenreRaw, err := tx.PrepareContext(ctx, "INSERT INTO master_genres_raw (master_id, genre) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert genres raw: %w", err)
	}
	defer insGenreRaw.Close()

	insStyleRaw, err := tx.PrepareContext(ctx, "INSERT INTO master_styles_raw (master_id, style) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert styles raw: %w", err)
	}
	defer insStyleRaw.Close()

	insArtist, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_artists (master_id, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert artists: %w", err)
	}
	defer insArtist.Close()

	insVideo, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_videos (master_id, seq, src, duration, embed, title, description) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert videos: %w", err)
	}
	defer insVideo.Close()

	for _, m := range batch {
		if _, err := insMaster.ExecContext(ctx, m.ID, m.MainRelease, m.Year, m.Title, nullIfEmpty(m.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutMasters insert id=%d: %w", m.ID, err)
		}
		for _, g := range m.Genres {
			if _, err := insGenreRaw.ExecContext(ctx, m.ID, g); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert genre raw id=%d: %w", m.ID, err)
			}
		}
		for _, st := range m.Styles {
			if _, err := insStyleRaw.ExecContext(ctx, m.ID, st); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert style raw id=%d: %w", m.ID, err)
			}
		}
		for i, ac := range m.Artists {
			if _, err := insArtist.ExecContext(ctx, m.ID, i, ac.ID, ac.Name, nullIfEmpty(ac.ANV), nullIfEmpty(ac.Join), nullIfEmpty(ac.Role)); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert artist id=%d: %w", m.ID, err)
			}
		}
		for i, v := range m.Videos {
			if _, err := insVideo.ExecContext(ctx, m.ID, i, v.Src, v.Duration, v.Embed, nullIfEmpty(v.Title), nullIfEmpty(v.Description)); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert video id=%d: %w", m.ID, err)
			}
		}
	}

	return tx.Commit()
}

// FinishMasters bulk-resolves the genre/style text accumulated in the
// raw staging tables into their lookup-table ids and the final
// master_genres/master_styles join tables, then drops the staging
// tables. Runs once at the end of a masters load, after every batch has
// been through PutMasters.
func (s *SQLiteStore) FinishMasters(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: FinishMasters begin: %w", err)
	}
	defer tx.Rollback()

	stmts := []string{
		`INSERT INTO genres (name) SELECT DISTINCT genre FROM master_genres_raw WHERE genre IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO master_genres (master_id, genre_id)
			SELECT mg.master_id, g.id FROM master_genres_raw mg JOIN genres g ON g.name = mg.genre`,
		`INSERT INTO styles (name) SELECT DISTINCT style FROM master_styles_raw WHERE style IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO master_styles (master_id, style_id)
			SELECT ms.master_id, s.id FROM master_styles_raw ms JOIN styles s ON s.name = ms.style`,
		`DROP TABLE master_genres_raw`,
		`DROP TABLE master_styles_raw`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: FinishMasters %q: %w", stmt, err)
		}
	}

	return tx.Commit()
}
