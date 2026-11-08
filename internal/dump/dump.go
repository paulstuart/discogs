// Package dump streams a single previously-downloaded Discogs dump file
// (gzip-compressed XML) to newline-delimited JSON, for ad hoc exploration
// on the command line (jq, duckdb, sqlite3 .import, etc) without going
// through a store.Store. It reads the same archived files load.Run does,
// but never touches a database.
package dump

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

// Options configures a dump run.
type Options struct {
	DataDir string
	Date    string // YYYYMMDD, required
	Kind    config.Kind
	Out     io.Writer // defaults to os.Stdout if nil
}

// elemByKind maps each dump kind to its XML element name, as used by
// xmlstream.StreamDecode.
var elemByKind = map[config.Kind]string{
	config.KindArtists:  "artist",
	config.KindLabels:   "label",
	config.KindMasters:  "master",
	config.KindReleases: "release",
}

// Run streams opts.Kind's archived file for opts.Date as newline-delimited
// JSON to opts.Out, one model.<Kind> record per line.
func Run(ctx context.Context, opts Options) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	elem, ok := elemByKind[opts.Kind]
	if !ok {
		return fmt.Errorf("dump: unknown kind %q", opts.Kind)
	}

	year, err := config.Year(opts.Date)
	if err != nil {
		return err
	}

	path := config.ArchivePath(opts.DataDir, year, opts.Date, opts.Kind)
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

	enc := json.NewEncoder(out)
	switch opts.Kind {
	case config.KindArtists:
		return xmlstream.StreamDecode(gz, elem, encodeFn[model.Artist](enc))
	case config.KindLabels:
		return xmlstream.StreamDecode(gz, elem, encodeFn[model.Label](enc))
	case config.KindMasters:
		return xmlstream.StreamDecode(gz, elem, encodeFn[model.Master](enc))
	case config.KindReleases:
		return xmlstream.StreamDecode(gz, elem, encodeFn[model.Release](enc))
	default:
		return fmt.Errorf("dump: unknown kind %q", opts.Kind)
	}
}

// encodeFn adapts a *json.Encoder into the func(T) error shape
// xmlstream.StreamDecode expects.
func encodeFn[T any](enc *json.Encoder) func(T) error {
	return func(v T) error { return enc.Encode(v) }
}
