// Command discogs downloads and loads Discogs' monthly XML data dumps.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "fetch":
		err = runFetch(os.Args[2:])
	case "load":
		err = runLoad(os.Args[2:])
	case "index":
		err = runIndex(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "discogs:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `discogs downloads and loads Discogs' monthly XML data dumps.

Usage:
  discogs fetch [--dir datadump] [--year YYYY] [--date YYYYMMDD]
  discogs load  [--dir datadump] --date YYYYMMDD [--db sqlite:discogs.db] [--kinds artists,labels,masters,releases]
  discogs index [--db sqlite:discogs.db]

Run "discogs index" once after all desired "load" invocations are done --
it builds every index/view in a single pass, rather than maintaining them
incrementally during each load.

Run "discogs fetch -h", "discogs load -h", or "discogs index -h" for flag details.`)
}
