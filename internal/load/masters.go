package load

import (
	"context"
	"io"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/store"
)

func loadMasters(ctx context.Context, st store.Store, r io.Reader) error {
	return batchDecode(r, "master", func(batch []model.Master) error {
		return st.PutMasters(ctx, batch)
	})
}
