package main

import (
	"context"
	"flag"

	"github.com/paulstuart/discogs/internal/store"
)

func runIndex(args []string) error {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	db := fs.String("db", "sqlite:discogs.db", "database DSN, scheme:path")
	fs.Parse(args)

	st, err := store.Open(*db)
	if err != nil {
		return err
	}
	defer st.Close()

	return st.BuildIndexes(context.Background())
}
