package sqlite

import (
	"context"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PrepareLabels wipes every label table so the load that follows fully
// replaces this kind's data (see the Store interface doc).
func (s *SQLiteStore) PrepareLabels(ctx context.Context) error {
	for _, table := range []string{"labels", "label_urls", "label_sublabels"} {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("sqlite: PrepareLabels wipe %s: %w", table, err)
		}
	}
	return nil
}

// PutLabels inserts a batch of labels and their child rows (urls,
// sublabels). Plain inserts only -- PrepareLabels already wiped the
// tables for this load.
func (s *SQLiteStore) PutLabels(ctx context.Context, batch []model.Label) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels begin: %w", err)
	}
	defer tx.Rollback()

	insLabel, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO labels (id, name, contact_info, profile, data_quality, parent_label_id, parent_label_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare insert: %w", err)
	}
	defer insLabel.Close()

	insURL, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO label_urls (label_id, url) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare insert urls: %w", err)
	}
	defer insURL.Close()

	insSub, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO label_sublabels (label_id, sublabel_id, sublabel_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare insert sublabels: %w", err)
	}
	defer insSub.Close()

	for _, l := range batch {
		var parentID, parentName any
		if l.ParentLabel != nil {
			parentID, parentName = l.ParentLabel.ID, l.ParentLabel.Name
		}
		if _, err := insLabel.ExecContext(ctx, l.ID, l.Name, nullIfEmpty(l.ContactInfo), nullIfEmpty(l.Profile), nullIfEmpty(l.DataQuality), parentID, parentName); err != nil {
			return fmt.Errorf("sqlite: PutLabels insert id=%d: %w", l.ID, err)
		}
		for _, u := range l.URLs {
			if _, err := insURL.ExecContext(ctx, l.ID, u); err != nil {
				return fmt.Errorf("sqlite: PutLabels insert url id=%d: %w", l.ID, err)
			}
		}
		for _, sl := range l.SubLabels {
			if _, err := insSub.ExecContext(ctx, l.ID, sl.ID, sl.Name); err != nil {
				return fmt.Errorf("sqlite: PutLabels insert sublabel id=%d: %w", l.ID, err)
			}
		}
	}

	return tx.Commit()
}
