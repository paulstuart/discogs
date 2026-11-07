// Package load streams previously-downloaded Discogs dump files
// (gzip-compressed XML) into a store.Store, batching records into
// transactions as they're decoded, without ever writing decompressed XML
// to disk.
package load

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/store"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

// batchSize is the number of top-level records accumulated before each
// flush to the store, balancing transaction overhead against memory use.
const batchSize = 2000

// Options configures a load run.
type Options struct {
	DataDir string
	Date    string // YYYYMMDD, required
	DB      string // store DSN, e.g. "sqlite:discogs.db"
	Kinds   []config.Kind
}

// Run opens the store named by opts.DB and loads each requested kind from
// the archived file for opts.Date.
func Run(ctx context.Context, opts Options) error {
	year, err := yearFromDate(opts.Date)
	if err != nil {
		return err
	}

	st, err := store.Open(opts.DB)
	if err != nil {
		return err
	}
	defer st.Close()

	for _, kind := range opts.Kinds {
		path := config.ArchivePath(opts.DataDir, year, opts.Date, kind)
		if err := loadFile(ctx, st, path, kind); err != nil {
			return fmt.Errorf("load: %s: %w", kind, err)
		}
	}
	return nil
}

func yearFromDate(date string) (int, error) {
	if len(date) != 8 {
		return 0, fmt.Errorf("load: invalid --date %q, want YYYYMMDD", date)
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0, fmt.Errorf("load: invalid --date %q, want YYYYMMDD", date)
	}
	return year, nil
}

func loadFile(ctx context.Context, st store.Store, path string, kind config.Kind) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found at %s (run \"discogs fetch\" first, or check --date/--dir): %w", path, err)
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip %s: %w", path, err)
	}
	defer gz.Close()

	switch kind {
	case config.KindArtists:
		return loadArtists(ctx, st, gz)
	case config.KindLabels:
		return loadLabels(ctx, st, gz)
	case config.KindMasters:
		return loadMasters(ctx, st, gz)
	case config.KindReleases:
		return loadReleases(ctx, st, gz)
	default:
		return fmt.Errorf("unknown kind %q", kind)
	}
}

// batchDecode streams elem records from r and flushes them to flush every
// batchSize records (and once more for any final partial batch).
func batchDecode[T any](r io.Reader, elem string, flush func([]T) error) error {
	var batch []T
	err := xmlstream.StreamDecode(r, elem, func(v T) error {
		batch = append(batch, v)
		if len(batch) >= batchSize {
			if err := flush(batch); err != nil {
				return err
			}
			batch = batch[:0:0] // release the backing array; flush may retain it
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(batch) > 0 {
		return flush(batch)
	}
	return nil
}
