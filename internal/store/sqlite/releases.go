package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PrepareReleases wipes every release table, so the load that follows
// fully replaces this kind's data (see the Store interface doc), and
// creates the raw staging tables PutReleases/FinishReleases use to
// normalize country/genre/style/identifier-type text in bulk after the
// load instead of resolving it per row during the hot insert path.
// company_roles is not wiped: its id is Discogs' own stable entity_type
// code, not a surrogate tied to this load.
func (s *SQLiteStore) PrepareReleases(ctx context.Context) error {
	for _, table := range []string{
		"releases", "release_artists", "release_extraartists", "release_labels",
		"release_formats", "release_format_descriptions", "release_genres", "release_styles",
		"release_tracks", "release_track_artists", "release_identifiers", "release_videos",
		"release_companies",
	} {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("sqlite: PrepareReleases wipe %s: %w", table, err)
		}
	}
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS releases_raw",
		"DROP TABLE IF EXISTS release_genres_raw",
		"DROP TABLE IF EXISTS release_styles_raw",
		"DROP TABLE IF EXISTS release_identifiers_raw",
		`CREATE TABLE releases_raw (
			id INTEGER NOT NULL, status TEXT, title TEXT NOT NULL, country TEXT,
			released TEXT, notes TEXT, master_id INTEGER, is_main_release INTEGER, data_quality TEXT)`,
		"CREATE TABLE release_genres_raw (release_id INTEGER NOT NULL, genre TEXT NOT NULL)",
		"CREATE TABLE release_styles_raw (release_id INTEGER NOT NULL, style TEXT NOT NULL)",
		`CREATE TABLE release_identifiers_raw (
			release_id INTEGER NOT NULL, seq INTEGER NOT NULL, type TEXT, description TEXT, value TEXT)`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: PrepareReleases %q: %w", stmt, err)
		}
	}
	return nil
}

// releaseStmts groups the prepared statements used by PutReleases so the
// per-record loop stays readable.
type releaseStmts struct {
	insReleaseRaw *sql.Stmt

	insArtist      *sql.Stmt
	insExtraArtist *sql.Stmt
	insLabel       *sql.Stmt
	insFormat      *sql.Stmt
	insFormatDesc  *sql.Stmt
	insGenreRaw    *sql.Stmt
	insStyleRaw    *sql.Stmt
	insTrack       *sql.Stmt
	insTrackArtist *sql.Stmt
	insIdentRaw    *sql.Stmt
	insVideo       *sql.Stmt
	insCompany     *sql.Stmt
	insCompanyRole *sql.Stmt
}

func prepareReleaseStmts(ctx context.Context, tx *sql.Tx) (*releaseStmts, error) {
	prep := func(query string) (*sql.Stmt, error) {
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("sqlite: PutReleases prepare %q: %w", query, err)
		}
		return stmt, nil
	}

	st := &releaseStmts{}
	var err error
	if st.insReleaseRaw, err = prep(`
		INSERT INTO releases_raw (id, status, title, country, released, notes, master_id, is_main_release, data_quality)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`); err != nil {
		return nil, err
	}
	if st.insArtist, err = prep("INSERT OR IGNORE INTO release_artists (release_id, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insExtraArtist, err = prep("INSERT OR IGNORE INTO release_extraartists (release_id, seq, artist_id, name, anv, role) VALUES (?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insLabel, err = prep("INSERT OR IGNORE INTO release_labels (release_id, label_id, label_name, catno) VALUES (?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insFormat, err = prep("INSERT OR IGNORE INTO release_formats (release_id, seq, name, qty, text) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insFormatDesc, err = prep("INSERT OR IGNORE INTO release_format_descriptions (release_id, format_seq, description) VALUES (?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insGenreRaw, err = prep("INSERT INTO release_genres_raw (release_id, genre) VALUES (?, ?)"); err != nil {
		return nil, err
	}
	if st.insStyleRaw, err = prep("INSERT INTO release_styles_raw (release_id, style) VALUES (?, ?)"); err != nil {
		return nil, err
	}
	if st.insTrack, err = prep("INSERT OR IGNORE INTO release_tracks (release_id, seq, position, title, duration) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insTrackArtist, err = prep("INSERT OR IGNORE INTO release_track_artists (release_id, track_seq, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insIdentRaw, err = prep("INSERT INTO release_identifiers_raw (release_id, seq, type, description, value) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insVideo, err = prep("INSERT OR IGNORE INTO release_videos (release_id, seq, src, duration, embed, title, description) VALUES (?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insCompany, err = prep("INSERT OR IGNORE INTO release_companies (release_id, seq, company_id, name, entity_type) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.insCompanyRole, err = prep("INSERT OR IGNORE INTO company_roles (id, name) VALUES (?, ?)"); err != nil {
		return nil, err
	}
	return st, nil
}

func (st *releaseStmts) closeAll() {
	for _, s := range []*sql.Stmt{
		st.insReleaseRaw,
		st.insArtist, st.insExtraArtist, st.insLabel,
		st.insFormat, st.insFormatDesc,
		st.insGenreRaw, st.insStyleRaw,
		st.insTrack, st.insTrackArtist,
		st.insIdentRaw, st.insVideo,
		st.insCompany, st.insCompanyRole,
	} {
		if s != nil {
			s.Close()
		}
	}
}

// PutReleases inserts a batch of releases: non-vocabulary child rows
// (artist credits, labels, formats, tracklist + per-track artist
// credits, videos, companies) go straight into their final tables;
// country/genre/style/identifier-type text goes into the raw staging
// tables for FinishReleases to resolve in bulk once the whole file has
// been loaded.
func (s *SQLiteStore) PutReleases(ctx context.Context, batch []model.Release) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutReleases begin: %w", err)
	}
	defer tx.Rollback()

	st, err := prepareReleaseStmts(ctx, tx)
	if err != nil {
		return err
	}
	defer st.closeAll()

	for _, r := range batch {
		var masterID, isMainRelease any
		if r.MasterID != nil {
			masterID, isMainRelease = r.MasterID.ID, r.MasterID.IsMainRelease
		}
		if _, err := st.insReleaseRaw.ExecContext(ctx, r.ID, nullIfEmpty(r.Status), r.Title, nullIfEmpty(r.Country), nullIfEmpty(r.Released), nullIfEmpty(r.Notes), masterID, isMainRelease, nullIfEmpty(r.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutReleases insert raw id=%d: %w", r.ID, err)
		}

		for i, ac := range r.Artists {
			if _, err := st.insArtist.ExecContext(ctx, r.ID, i, ac.ID, ac.Name, nullIfEmpty(ac.ANV), nullIfEmpty(ac.Join), nullIfEmpty(ac.Role)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert artist id=%d: %w", r.ID, err)
			}
		}

		for i, ea := range r.ExtraArtists {
			if _, err := st.insExtraArtist.ExecContext(ctx, r.ID, i, ea.ID, ea.Name, nullIfEmpty(ea.ANV), nullIfEmpty(ea.Role)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert extraartist id=%d: %w", r.ID, err)
			}
		}

		for _, rl := range r.Labels {
			if _, err := st.insLabel.ExecContext(ctx, r.ID, rl.ID, rl.Name, nullIfEmpty(rl.Catno)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert label id=%d: %w", r.ID, err)
			}
		}

		for i, f := range r.Formats {
			if _, err := st.insFormat.ExecContext(ctx, r.ID, i, f.Name, nullIfEmpty(f.Qty), nullIfEmpty(f.Text)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert format id=%d: %w", r.ID, err)
			}
			for _, d := range f.Descriptions {
				if _, err := st.insFormatDesc.ExecContext(ctx, r.ID, i, d); err != nil {
					return fmt.Errorf("sqlite: PutReleases insert format description id=%d: %w", r.ID, err)
				}
			}
		}

		for _, g := range r.Genres {
			if _, err := st.insGenreRaw.ExecContext(ctx, r.ID, g); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert genre raw id=%d: %w", r.ID, err)
			}
		}

		for _, sty := range r.Styles {
			if _, err := st.insStyleRaw.ExecContext(ctx, r.ID, sty); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert style raw id=%d: %w", r.ID, err)
			}
		}

		for ti, t := range r.Tracklist {
			if _, err := st.insTrack.ExecContext(ctx, r.ID, ti, t.Position, t.Title, nullIfEmpty(t.Duration)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert track id=%d: %w", r.ID, err)
			}
			for ai, ac := range t.Artists {
				if _, err := st.insTrackArtist.ExecContext(ctx, r.ID, ti, ai, ac.ID, ac.Name, nullIfEmpty(ac.ANV), nullIfEmpty(ac.Join), nullIfEmpty(ac.Role)); err != nil {
					return fmt.Errorf("sqlite: PutReleases insert track artist id=%d: %w", r.ID, err)
				}
			}
		}

		for i, id := range r.Identifiers {
			if _, err := st.insIdentRaw.ExecContext(ctx, r.ID, i, nullIfEmpty(id.Type), nullIfEmpty(id.Description), id.Value); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert identifier raw id=%d: %w", r.ID, err)
			}
		}

		for i, v := range r.Videos {
			if _, err := st.insVideo.ExecContext(ctx, r.ID, i, v.Src, v.Duration, v.Embed, nullIfEmpty(v.Title), nullIfEmpty(v.Description)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert video id=%d: %w", r.ID, err)
			}
		}

		for i, c := range r.Companies {
			var entityType any
			if c.EntityType != 0 {
				entityType = c.EntityType
				if _, err := st.insCompanyRole.ExecContext(ctx, c.EntityType, c.EntityTypeName); err != nil {
					return fmt.Errorf("sqlite: PutReleases insert company role id=%d: %w", r.ID, err)
				}
			}
			if _, err := st.insCompany.ExecContext(ctx, r.ID, i, c.ID, c.Name, entityType); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert company id=%d: %w", r.ID, err)
			}
		}
	}

	return tx.Commit()
}

// FinishReleases bulk-resolves the country/genre/style/identifier-type
// text accumulated in the raw staging tables into their lookup-table
// ids and the final releases/release_genres/release_styles/
// release_identifiers rows, then drops the staging tables. Runs once at
// the end of a releases load, after every batch has been through
// PutReleases.
func (s *SQLiteStore) FinishReleases(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: FinishReleases begin: %w", err)
	}
	defer tx.Rollback()

	stmts := []string{
		`INSERT INTO countries (name) SELECT DISTINCT country FROM releases_raw WHERE country IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO releases (id, status, title, country_id, released, notes, master_id, is_main_release, data_quality)
			SELECT r.id, r.status, r.title, c.id, r.released, r.notes, r.master_id, r.is_main_release, r.data_quality
			FROM releases_raw r LEFT JOIN countries c ON c.name = r.country`,

		`INSERT INTO genres (name) SELECT DISTINCT genre FROM release_genres_raw WHERE genre IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO release_genres (release_id, genre_id)
			SELECT rg.release_id, g.id FROM release_genres_raw rg JOIN genres g ON g.name = rg.genre`,

		`INSERT INTO styles (name) SELECT DISTINCT style FROM release_styles_raw WHERE style IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO release_styles (release_id, style_id)
			SELECT rs.release_id, s.id FROM release_styles_raw rs JOIN styles s ON s.name = rs.style`,

		`INSERT INTO identifier_types (name) SELECT DISTINCT type FROM release_identifiers_raw WHERE type IS NOT NULL
			ON CONFLICT(name) DO NOTHING`,
		`INSERT OR IGNORE INTO release_identifiers (release_id, seq, type_id, description, value)
			SELECT ri.release_id, ri.seq, it.id, ri.description, ri.value
			FROM release_identifiers_raw ri LEFT JOIN identifier_types it ON it.name = ri.type`,

		`DROP TABLE releases_raw`,
		`DROP TABLE release_genres_raw`,
		`DROP TABLE release_styles_raw`,
		`DROP TABLE release_identifiers_raw`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: FinishReleases %q: %w", stmt, err)
		}
	}

	return tx.Commit()
}
