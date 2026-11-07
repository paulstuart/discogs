package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// releaseStmts groups the prepared statements used by PutReleases so the
// per-record loop stays readable.
type releaseStmts struct {
	upsert *sql.Stmt

	delArtists, insArtist           *sql.Stmt
	delExtraArtists, insExtraArtist *sql.Stmt
	delLabels, insLabel             *sql.Stmt
	delFormats, insFormat           *sql.Stmt
	delFormatDesc, insFormatDesc    *sql.Stmt
	delGenres, insGenre             *sql.Stmt
	delStyles, insStyle             *sql.Stmt
	delTracks, insTrack             *sql.Stmt
	delTrackArtists, insTrackArtist *sql.Stmt
	delIdentifiers, insIdentifier   *sql.Stmt
	delVideos, insVideo             *sql.Stmt
	delCompanies, insCompany        *sql.Stmt
	insCompanyRole                  *sql.Stmt
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
	if st.upsert, err = prep(`
		INSERT INTO releases (id, status, title, country_id, released, notes, master_id, is_main_release, data_quality)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status, title=excluded.title, country_id=excluded.country_id,
			released=excluded.released, notes=excluded.notes, master_id=excluded.master_id,
			is_main_release=excluded.is_main_release, data_quality=excluded.data_quality`); err != nil {
		return nil, err
	}
	if st.delArtists, err = prep("DELETE FROM release_artists WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insArtist, err = prep("INSERT OR IGNORE INTO release_artists (release_id, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delExtraArtists, err = prep("DELETE FROM release_extraartists WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insExtraArtist, err = prep("INSERT OR IGNORE INTO release_extraartists (release_id, seq, artist_id, name, anv, role) VALUES (?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delLabels, err = prep("DELETE FROM release_labels WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insLabel, err = prep("INSERT OR IGNORE INTO release_labels (release_id, label_id, label_name, catno) VALUES (?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delFormats, err = prep("DELETE FROM release_formats WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insFormat, err = prep("INSERT OR IGNORE INTO release_formats (release_id, seq, name, qty, text) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delFormatDesc, err = prep("DELETE FROM release_format_descriptions WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insFormatDesc, err = prep("INSERT OR IGNORE INTO release_format_descriptions (release_id, format_seq, description) VALUES (?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delGenres, err = prep("DELETE FROM release_genres WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insGenre, err = prep("INSERT OR IGNORE INTO release_genres (release_id, genre_id) VALUES (?, ?)"); err != nil {
		return nil, err
	}
	if st.delStyles, err = prep("DELETE FROM release_styles WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insStyle, err = prep("INSERT OR IGNORE INTO release_styles (release_id, style_id) VALUES (?, ?)"); err != nil {
		return nil, err
	}
	if st.delTracks, err = prep("DELETE FROM release_tracks WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insTrack, err = prep("INSERT OR IGNORE INTO release_tracks (release_id, seq, position, title, duration) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delTrackArtists, err = prep("DELETE FROM release_track_artists WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insTrackArtist, err = prep("INSERT OR IGNORE INTO release_track_artists (release_id, track_seq, seq, artist_id, name, anv, join_str, role) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delIdentifiers, err = prep("DELETE FROM release_identifiers WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insIdentifier, err = prep("INSERT OR IGNORE INTO release_identifiers (release_id, seq, type_id, description, value) VALUES (?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delVideos, err = prep("DELETE FROM release_videos WHERE release_id = ?"); err != nil {
		return nil, err
	}
	if st.insVideo, err = prep("INSERT OR IGNORE INTO release_videos (release_id, seq, src, duration, embed, title, description) VALUES (?, ?, ?, ?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if st.delCompanies, err = prep("DELETE FROM release_companies WHERE release_id = ?"); err != nil {
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
		st.upsert,
		st.delArtists, st.insArtist,
		st.delExtraArtists, st.insExtraArtist,
		st.delLabels, st.insLabel,
		st.delFormats, st.insFormat, st.delFormatDesc, st.insFormatDesc,
		st.delGenres, st.insGenre,
		st.delStyles, st.insStyle,
		st.delTracks, st.insTrack, st.delTrackArtists, st.insTrackArtist,
		st.delIdentifiers, st.insIdentifier,
		st.delVideos, st.insVideo,
		st.delCompanies, st.insCompany, st.insCompanyRole,
	} {
		if s != nil {
			s.Close()
		}
	}
}

// PutReleases upserts a batch of releases and fully replaces every nested
// child structure (artist credits, extra artists, labels, formats,
// genres, styles, tracklist + per-track artist credits, identifiers,
// videos, companies) within one transaction.
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
		var countryID any
		if r.Country != "" {
			id, err := s.countries.id(ctx, tx, r.Country)
			if err != nil {
				return err
			}
			countryID = id
		}
		if _, err := st.upsert.ExecContext(ctx, r.ID, nullIfEmpty(r.Status), r.Title, countryID, nullIfEmpty(r.Released), nullIfEmpty(r.Notes), masterID, isMainRelease, nullIfEmpty(r.DataQuality)); err != nil {
			return fmt.Errorf("sqlite: PutReleases upsert id=%d: %w", r.ID, err)
		}

		if _, err := st.delArtists.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete artists id=%d: %w", r.ID, err)
		}
		for i, ac := range r.Artists {
			if _, err := st.insArtist.ExecContext(ctx, r.ID, i, ac.ID, ac.Name, nullIfEmpty(ac.ANV), nullIfEmpty(ac.Join), nullIfEmpty(ac.Role)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert artist id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delExtraArtists.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete extraartists id=%d: %w", r.ID, err)
		}
		for i, ea := range r.ExtraArtists {
			if _, err := st.insExtraArtist.ExecContext(ctx, r.ID, i, ea.ID, ea.Name, nullIfEmpty(ea.ANV), nullIfEmpty(ea.Role)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert extraartist id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delLabels.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete labels id=%d: %w", r.ID, err)
		}
		for _, rl := range r.Labels {
			if _, err := st.insLabel.ExecContext(ctx, r.ID, rl.ID, rl.Name, nullIfEmpty(rl.Catno)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert label id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delFormats.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete formats id=%d: %w", r.ID, err)
		}
		if _, err := st.delFormatDesc.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete format descriptions id=%d: %w", r.ID, err)
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

		if _, err := st.delGenres.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete genres id=%d: %w", r.ID, err)
		}
		for _, g := range r.Genres {
			genreID, err := s.genres.id(ctx, tx, g)
			if err != nil {
				return err
			}
			if _, err := st.insGenre.ExecContext(ctx, r.ID, genreID); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert genre id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delStyles.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete styles id=%d: %w", r.ID, err)
		}
		for _, sty := range r.Styles {
			styleID, err := s.styles.id(ctx, tx, sty)
			if err != nil {
				return err
			}
			if _, err := st.insStyle.ExecContext(ctx, r.ID, styleID); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert style id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delTracks.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete tracks id=%d: %w", r.ID, err)
		}
		if _, err := st.delTrackArtists.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete track artists id=%d: %w", r.ID, err)
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

		if _, err := st.delIdentifiers.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete identifiers id=%d: %w", r.ID, err)
		}
		for i, id := range r.Identifiers {
			var typeID any
			if id.Type != "" {
				tid, err := s.identifierTypes.id(ctx, tx, id.Type)
				if err != nil {
					return err
				}
				typeID = tid
			}
			if _, err := st.insIdentifier.ExecContext(ctx, r.ID, i, typeID, nullIfEmpty(id.Description), id.Value); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert identifier id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delVideos.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete videos id=%d: %w", r.ID, err)
		}
		for i, v := range r.Videos {
			if _, err := st.insVideo.ExecContext(ctx, r.ID, i, v.Src, v.Duration, v.Embed, nullIfEmpty(v.Title), nullIfEmpty(v.Description)); err != nil {
				return fmt.Errorf("sqlite: PutReleases insert video id=%d: %w", r.ID, err)
			}
		}

		if _, err := st.delCompanies.ExecContext(ctx, r.ID); err != nil {
			return fmt.Errorf("sqlite: PutReleases delete companies id=%d: %w", r.ID, err)
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
