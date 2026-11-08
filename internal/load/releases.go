package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadReleases(ctx context.Context, st store.Store, r io.Reader) error {
	if err := st.PrepareReleases(ctx); err != nil {
		return err
	}
	if err := batchDecode(r, "release", func(batch []model.Release) error {
		return st.PutReleases(ctx, batch)
	}); err != nil {
		return err
	}
	return st.FinishReleases(ctx)
}
