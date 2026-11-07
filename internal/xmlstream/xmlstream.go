// Package xmlstream provides a generic helper for streaming large XML
// documents without buffering the whole document in memory.
package xmlstream

import (
	"encoding/xml"
	"fmt"
	"io"
)

// StreamDecode walks r token by token, and for every start element named
// elem decodes it into a fresh T and invokes fn. It never buffers the full
// document, making it suitable for multi-gigabyte XML streams.
//
// If fn returns an error, decoding stops immediately and that error is
// returned.
func StreamDecode[T any](r io.Reader, elem string, fn func(T) error) error {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("xmlstream: read token: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != elem {
			continue
		}
		var v T
		if err := dec.DecodeElement(&v, &se); err != nil {
			return fmt.Errorf("xmlstream: decode %s: %w", elem, err)
		}
		if err := fn(v); err != nil {
			return err
		}
	}
}
