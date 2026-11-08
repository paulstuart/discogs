package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadArtists(ctx context.Context, st store.Store, r io.Reader) error {
	if err := st.PrepareArtists(ctx); err != nil {
		return err
	}
	return batchDecode(r, "artist", func(batch []model.Artist) error {
		return st.PutArtists(ctx, batch)
	})
}
