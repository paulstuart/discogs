package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadReleases(ctx context.Context, st store.Store, r io.Reader) error {
	return batchDecode(r, "release", func(batch []model.Release) error {
		return st.PutReleases(ctx, batch)
	})
}
