package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadMasters(ctx context.Context, st store.Store, r io.Reader) error {
	if err := st.PrepareMasters(ctx); err != nil {
		return err
	}
	if err := batchDecode(r, "master", func(batch []model.Master) error {
		return st.PutMasters(ctx, batch)
	}); err != nil {
		return err
	}
	return st.FinishMasters(ctx)
}
