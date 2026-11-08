package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadLabels(ctx context.Context, st store.Store, r io.Reader) error {
	if err := st.PrepareLabels(ctx); err != nil {
		return err
	}
	return batchDecode(r, "label", func(batch []model.Label) error {
		return st.PutLabels(ctx, batch)
	})
}
