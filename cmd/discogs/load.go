package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/load"
)

func runLoad(args []string) error {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	dir := fs.String("dir", config.DefaultDataDir, "data directory (reads archived files from <dir>/archive/)")
	date := fs.String("date", "", "YYYYMMDD batch to load (required)")
	db := fs.String("db", "sqlite:discogs.db", "database DSN, scheme:path")
	kindsFlag := fs.String("kinds", "artists,labels,masters,releases", "comma-separated kinds to load")
	fs.Parse(args)

	if *date == "" {
		return missingDateError(*dir)
	}

	kinds, err := parseKinds(*kindsFlag)
	if err != nil {
		return err
	}

	opts := load.Options{
		DataDir: *dir,
		Date:    *date,
		DB:      *db,
		Kinds:   kinds,
	}
	return load.Run(context.Background(), opts)
}

func parseKinds(s string) ([]config.Kind, error) {
	valid := map[config.Kind]bool{}
	for _, k := range config.AllKinds {
		valid[k] = true
	}
	var kinds []config.Kind
	for _, part := range strings.Split(s, ",") {
		k := config.Kind(strings.TrimSpace(part))
		if !valid[k] {
			return nil, fmt.Errorf("invalid --kinds entry %q (want one of artists,labels,masters,releases)", part)
		}
		kinds = append(kinds, k)
	}
	return kinds, nil
}

func missingDateError(dir string) error {
	dates, err := config.ListArchivedDates(dir)
	if err != nil {
		return fmt.Errorf("--date is required: %w", err)
	}
	if len(dates) == 0 {
		return fmt.Errorf("--date is required, and no archived batches were found under %s/archive (run \"discogs fetch\" first)", dir)
	}
	return fmt.Errorf("--date is required; available batches under %s/archive: %s", dir, strings.Join(dates, ", "))
}
