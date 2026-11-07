package sqlite

import (
	"context"
	"fmt"

	"github.com/paulstuart/discogs/internal/model"
)

// PutLabels upserts a batch of labels and fully replaces each label's
// child rows (urls, sublabels) within one transaction.
func (s *SQLiteStore) PutLabels(ctx context.Context, batch []model.Label) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels begin: %w", err)
	}
	defer tx.Rollback()

	upsert, err := tx.PrepareContext(ctx, `
		INSERT INTO labels (id, name, contact_info, profile, data_quality, parent_label_id, parent_label_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, contact_info=excluded.contact_info,
			profile=excluded.profile, data_quality=excluded.data_quality,
			parent_label_id=excluded.parent_label_id, parent_label_name=excluded.parent_label_name`)
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare upsert: %w", err)
	}
	defer upsert.Close()

	delURLs, err := tx.PrepareContext(ctx, "DELETE FROM label_urls WHERE label_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare delete urls: %w", err)
	}
	defer delURLs.Close()
	insURL, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO label_urls (label_id, url) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare insert urls: %w", err)
	}
	defer insURL.Close()

	delSub, err := tx.PrepareContext(ctx, "DELETE FROM label_sublabels WHERE label_id = ?")
	if err != nil {
		return fmt.Errorf("sqlite: PutLabels prepare delete sublabels: %w", err)
	}
	defer delSub.Close()
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
		if _, err := upsert.ExecContext(ctx, l.ID, l.Name, nullIfEmpty(l.ContactInfo), nullIfEmpty(l.Profile), nullIfEmpty(l.DataQuality), parentID, parentName); err != nil {
			return fmt.Errorf("sqlite: PutLabels upsert id=%d: %w", l.ID, err)
		}

		if _, err := delURLs.ExecContext(ctx, l.ID); err != nil {
			return fmt.Errorf("sqlite: PutLabels delete urls id=%d: %w", l.ID, err)
		}
		for _, u := range l.URLs {
			if _, err := insURL.ExecContext(ctx, l.ID, u); err != nil {
				return fmt.Errorf("sqlite: PutLabels insert url id=%d: %w", l.ID, err)
			}
		}

		if _, err := delSub.ExecContext(ctx, l.ID); err != nil {
			return fmt.Errorf("sqlite: PutLabels delete sublabels id=%d: %w", l.ID, err)
		}
		for _, sl := range l.SubLabels {
			if _, err := insSub.ExecContext(ctx, l.ID, sl.ID, sl.Name); err != nil {
				return fmt.Errorf("sqlite: PutLabels insert sublabel id=%d: %w", l.ID, err)
			}
		}
	}

	return tx.Commit()
}
