// Command xmlschema reports every distinct element path, attribute, and
// text-content occurrence found in one or more XML files, so the discogs
// dump formats can be inspected directly instead of by ad hoc grepping.
//
// Usage:
//
//	xmlschema file1.xml file2.xml.gz ...
//
// Gzipped input is detected automatically (by ".gz" suffix or gzip magic
// bytes), so dump archives can be pointed at directly.
package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/paulstuart/discogs/internal/xmlschema"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "xmlschema:", err)
		os.Exit(1)
	}
}

func run(paths []string) error {
	if len(paths) == 0 {
		return sniff("stdin", os.Stdin)
	}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		err = sniffFile(path, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func sniffFile(path string, f *os.File) error {
	r, err := maybeGunzip(path, f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := sniff(path, r); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func sniff(label string, r io.Reader) error {
	rep, err := xmlschema.Sniff(r)
	if err != nil {
		return err
	}
	fmt.Printf("== %s ==\n", label)
	return rep.WriteReport(os.Stdout)
}

// maybeGunzip wraps f in a gzip reader if path ends in ".gz" or the stream
// starts with the gzip magic bytes, regardless of extension.
func maybeGunzip(path string, f *os.File) (io.Reader, error) {
	if strings.HasSuffix(path, ".gz") {
		return gzip.NewReader(f)
	}

	br := bufio.NewReader(f)
	magic, err := br.Peek(2)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		return gzip.NewReader(br)
	}
	return br, nil
}
