package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadArtists(ctx context.Context, st store.Store, r io.Reader) error {
	return batchDecode(r, "artist", func(batch []model.Artist) error {
		return st.PutArtists(ctx, batch)
	})
}
