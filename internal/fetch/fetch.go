package fetch

import (
	"context"
	"fmt"
)

// Options configures a fetch run.
type Options struct {
	DataDir string
	// Year overrides auto-discovery of the latest year (0 = auto).
	Year int
	// Date overrides auto-discovery of the latest date batch within Year
	// (requires Year to be set).
	Date string
}

// Run discovers (or validates an explicit override of) the target batch
// and downloads it into <DataDir>/archive/<year>/.
func Run(ctx context.Context, opts Options) error {
	if opts.Date != "" && opts.Year == 0 {
		return fmt.Errorf("fetch: --date requires --year")
	}

	d := NewDiscoverer()

	var year int
	var date string
	var files []string
	var err error

	switch {
	case opts.Year != 0 && opts.Date != "":
		year, date = opts.Year, opts.Date
		files, err = d.Batch(ctx, year, date)
	case opts.Year != 0:
		year = opts.Year
		date, files, err = d.LatestInYear(ctx, year)
		if err == nil && date == "" {
			err = fmt.Errorf("fetch: no batches found for year %d", year)
		}
	default:
		year, date, files, err = d.LatestBatch(ctx)
	}
	if err != nil {
		return err
	}

	return DownloadBatch(ctx, d, opts.DataDir, year, date, files)
}
