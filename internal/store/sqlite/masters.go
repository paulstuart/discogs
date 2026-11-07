package sqlite

import (
	"context"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PutMasters upserts a batch of masters and fully replaces each master's
// child rows (genres, styles, artist credits, videos) within one
// transaction.
func (s *SQLiteStore) PutMasters(ctx context.Context, batch []model.Master) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters begin: %w", err)
	}
	defer tx.Rollback()

	upsert, err := tx.PrepareContext(ctx, `
		INSERT INTO masters (id, main_release, year, title, data_quality)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			main_release=excluded.main_release, year=excluded.year,
			title=excluded.title, data_quality=excluded.data_quality`)
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare upsert: %w", err)
	}
	defer upsert.Close()

	delGenres, err := tx.PrepareContext(ctx, "DELETE FROM master_genres WHERE master_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare delete genres: %w", err)
	}
	defer delGenres.Close()
	insGenre, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_genres (master_id, genre_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert genres: %w", err)
	}
	defer insGenre.Close()

	delStyles, err := tx.PrepareContext(ctx, "DELETE FROM master_styles WHERE master_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare delete styles: %w", err)
	}
	defer delStyles.Close()
	insStyle, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_styles (master_id, style_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert styles: %w", err)
	}
	defer insStyle.Close()

	delArtists, err := tx.PrepareContext(ctx, "DELETE FROM master_artists WHERE master_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare delete artists: %w", err)
	}
	defer delArtists.Close()
	insArtist, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_artists (master_id, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert artists: %w", err)
	}
	defer insArtist.Close()

	delVideos, err := tx.PrepareContext(ctx, "DELETE FROM master_videos WHERE master_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare delete videos: %w", err)
	}
	defer delVideos.Close()
	insVideo, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO master_videos (master_id, seq, src, duration, embed, title, description) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutMasters prepare insert videos: %w", err)
	}
	defer insVideo.Close()

	for _, m := range batch {
		if _, err := upsert.ExecContext(ctx, m.ID, m.MainRelease, m.Year, m.Title, nullIfEmpty(m.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutMasters upsert id=%d: %w", m.ID, err)
		}

		if _, err := delGenres.ExecContext(ctx, m.ID); err != nil {
			return fmt.Errorf("sqlite: PutMasters delete genres id=%d: %w", m.ID, err)
		}
		for _, g := range m.Genres {
			genreID, err := s.genres.id(ctx, tx, g)
			if err != nil {
				return err
			}
			if _, err := insGenre.ExecContext(ctx, m.ID, genreID); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert genre id=%d: %w", m.ID, err)
			}
		}

		if _, err := delStyles.ExecContext(ctx, m.ID); err != nil {
			return fmt.Errorf("sqlite: PutMasters delete styles id=%d: %w", m.ID, err)
		}
		for _, st := range m.Styles {
			styleID, err := s.styles.id(ctx, tx, st)
			if err != nil {
				return err
			}
			if _, err := insStyle.ExecContext(ctx, m.ID, styleID); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert style id=%d: %w", m.ID, err)
			}
		}

		if _, err := delArtists.ExecContext(ctx, m.ID); err != nil {
			return fmt.Errorf("sqlite: PutMasters delete artists id=%d: %w", m.ID, err)
		}
		for i, ac := range m.Artists {
			if _, err := insArtist.ExecContext(ctx, m.ID, i, ac.ID, ac.Name, nullIfEmpty(ac.ANV), nullIfEmpty(ac.Join), nullIfEmpty(ac.Role)); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert artist id=%d: %w", m.ID, err)
			}
		}

		if _, err := delVideos.ExecContext(ctx, m.ID); err != nil {
			return fmt.Errorf("sqlite: PutMasters delete videos id=%d: %w", m.ID, err)
		}
		for i, v := range m.Videos {
			if _, err := insVideo.ExecContext(ctx, m.ID, i, v.Src, v.Duration, v.Embed, nullIfEmpty(v.Title), nullIfEmpty(v.Description)); err != nil {
				return fmt.Errorf("sqlite: PutMasters insert video id=%d: %w", m.ID, err)
			}
		}
	}

	return tx.Commit()
}
