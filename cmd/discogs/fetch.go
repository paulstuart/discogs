package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/fetch"
)

func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	dir := fs.String("dir", config.DefaultDataDir, "data directory (downloads are archived under <dir>/archive/)")
	year := fs.Int("year", 0, "override: year to fetch (0 = auto-discover the latest)")
	date := fs.String("date", "", "override: YYYYMMDD batch date within --year (requires --year)")
	fs.Parse(args)

	opts := fetch.Options{
		DataDir: *dir,
		Year:    *year,
		Date:    *date,
	}
	if opts.Date != "" && opts.Year == 0 {
		return fmt.Errorf("--date requires --year")
	}
	fmt.Printf("fetch: dir=%s year=%d date=%q\n", opts.DataDir, opts.Year, opts.Date)
	return fetch.Run(context.Background(), opts)
}
