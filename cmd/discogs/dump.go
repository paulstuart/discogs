package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/dump"
)

func runDump(args []string) error {
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	dir := fs.String("dir", config.DefaultDataDir, "data directory (reads archived files from <dir>/archive/)")
	date := fs.String("date", "", "YYYYMMDD batch to dump (required)")
	kindFlag := fs.String("kind", "", "dump kind: one of artists,labels,masters,releases (required)")
	out := fs.String("out", "", "output file for newline-delimited JSON (default: stdout)")
	fs.Parse(args)

	if *date == "" {
		return missingDateError(*dir)
	}

	kind, err := parseKind(*kindFlag)
	if err != nil {
		return err
	}

	opts := dump.Options{
		DataDir: *dir,
		Date:    *date,
		Kind:    kind,
	}

	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		opts.Out = f
	}

	return dump.Run(context.Background(), opts)
}

func parseKind(s string) (config.Kind, error) {
	kind := config.Kind(strings.TrimSpace(s))
	for _, k := range config.AllKinds {
		if kind == k {
			return kind, nil
		}
	}
	return "", fmt.Errorf("--kind is required and must be one of artists,labels,masters,releases, got %q", s)
}
