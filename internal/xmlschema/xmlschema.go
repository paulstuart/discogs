// Package xmlschema sniffs the structure of an XML document by streaming
// through it token by token and recording every distinct element path and
// attribute name encountered, without buffering the document or requiring a
// predefined schema. It exists so that discogs dump formats (artists.xml,
// labels.xml, masters.xml, releases.xml) can be inspected directly instead
// of by repeated ad hoc grepping.
package xmlschema

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// PathInfo describes everything observed at one element path.
type PathInfo struct {
	// Count is how many times this path occurred.
	Count int
	// Attrs maps attribute local name to how many times it appeared on
	// this path.
	Attrs map[string]int
	// HasText is true if this path was ever seen with non-whitespace
	// character data directly inside it.
	HasText bool
}

// Report is the set of element paths observed in a document, keyed by a
// "/"-joined path of local element names from the document root.
type Report struct {
	Paths map[string]*PathInfo
}

// Sniff streams r and returns the set of element paths, attributes, and
// text-content occurrences it contains. It never buffers the whole document,
// so it is safe to run against multi-gigabyte dump files.
func Sniff(r io.Reader) (*Report, error) {
	dec := xml.NewDecoder(r)
	rep := &Report{Paths: map[string]*PathInfo{}}
	var stack []string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return rep, nil
		}
		if err != nil {
			return nil, fmt.Errorf("xmlschema: read token: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			path := strings.Join(stack, "/")
			pi, ok := rep.Paths[path]
			if !ok {
				pi = &PathInfo{Attrs: map[string]int{}}
				rep.Paths[path] = pi
			}
			pi.Count++
			for _, a := range t.Attr {
				pi.Attrs[a.Name.Local]++
			}
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 || len(strings.TrimSpace(string(t))) == 0 {
				continue
			}
			if pi, ok := rep.Paths[strings.Join(stack, "/")]; ok {
				pi.HasText = true
			}
		}
	}
}

// SortedPaths returns every observed path in alphabetical order.
func (r *Report) SortedPaths() []string {
	paths := make([]string, 0, len(r.Paths))
	for p := range r.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// WriteReport renders the report as one line per path: the path, how many
// times it occurred, whether it ever carried text content, and which
// attributes were seen on it with their own counts.
func (r *Report) WriteReport(w io.Writer) error {
	for _, path := range r.SortedPaths() {
		pi := r.Paths[path]

		line := fmt.Sprintf("%s (%d)", path, pi.Count)
		if pi.HasText {
			line += " text"
		}
		if len(pi.Attrs) > 0 {
			names := make([]string, 0, len(pi.Attrs))
			for a := range pi.Attrs {
				names = append(names, a)
			}
			sort.Strings(names)
			attrs := make([]string, len(names))
			for i, a := range names {
				attrs[i] = fmt.Sprintf("%s(%d)", a, pi.Attrs[a])
			}
			line += " attrs=[" + strings.Join(attrs, ",") + "]"
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
