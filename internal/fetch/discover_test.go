package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const rootIndexHTML = `<html><body><pre>Download Discogs Data</pre><h2>Discogs Data</h2>
<pre>Last Modified                   Size           Name
-----------------------------------------------------------------------------
                                -              <a href="?prefix=data%2F2024%2F">2024/</a>
                                -              <a href="?prefix=data%2F2025%2F">2025/</a>
                                -              <a href="?prefix=data%2F2026%2F">2026/</a>
</pre></body></html>`

const year2026HTML = `<html><body><pre>
<a href="?download=data%2F2026%2Fdiscogs_20261001_CHECKSUM.txt">discogs_20261001_CHECKSUM.txt</a>
<a href="?download=data%2F2026%2Fdiscogs_20261001_artists.xml.gz">discogs_20261001_artists.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261001_labels.xml.gz">discogs_20261001_labels.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261001_masters.xml.gz">discogs_20261001_masters.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261001_releases.xml.gz">discogs_20261001_releases.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261101_CHECKSUM.txt">discogs_20261101_CHECKSUM.txt</a>
<a href="?download=data%2F2026%2Fdiscogs_20261101_artists.xml.gz">discogs_20261101_artists.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261101_labels.xml.gz">discogs_20261101_labels.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261101_masters.xml.gz">discogs_20261101_masters.xml.gz</a>
<a href="?download=data%2F2026%2Fdiscogs_20261101_releases.xml.gz">discogs_20261101_releases.xml.gz</a>
</pre></body></html>`

// year2025HTML has only one batch, and it's missing the releases file --
// an incomplete-latest-batch edge case.
const year2025HTML = `<html><body><pre>
<a href="?download=data%2F2025%2Fdiscogs_20251201_CHECKSUM.txt">discogs_20251201_CHECKSUM.txt</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_artists.xml.gz">discogs_20251201_artists.xml.gz</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_labels.xml.gz">discogs_20251201_labels.xml.gz</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_masters.xml.gz">discogs_20251201_masters.xml.gz</a>
</pre></body></html>`

const emptyYearHTML = `<html><body><pre>no files yet</pre></body></html>`

func newTestServer(t *testing.T, byQuery map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := byQuery[r.URL.RawQuery]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListYears(t *testing.T) {
	srv := newTestServer(t, map[string]string{"": rootIndexHTML})
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}
	years, err := d.ListYears(context.Background())
	if err != nil {
		t.Fatalf("ListYears: %v", err)
	}
	want := []int{2024, 2025, 2026}
	if len(years) != len(want) {
		t.Fatalf("years = %v, want %v", years, want)
	}
	for i := range want {
		if years[i] != want[i] {
			t.Errorf("years[%d] = %d, want %d", i, years[i], want[i])
		}
	}
}

func TestListBatches(t *testing.T) {
	srv := newTestServer(t, map[string]string{"prefix=data/2026/": year2026HTML})
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}
	batches, err := d.ListBatches(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2 (20261001, 20261101)", len(batches))
	}
	if len(batches["20261101"]) != 5 {
		t.Errorf("20261101 batch has %d files, want 5", len(batches["20261101"]))
	}
}

func TestLatestBatch(t *testing.T) {
	srv := newTestServer(t, map[string]string{
		"":                  rootIndexHTML,
		"prefix=data/2026/": year2026HTML,
	})
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}
	year, date, files, err := d.LatestBatch(context.Background())
	if err != nil {
		t.Fatalf("LatestBatch: %v", err)
	}
	if year != 2026 || date != "20261101" {
		t.Errorf("LatestBatch = %d/%s, want 2026/20261101", year, date)
	}
	if len(files) != 5 {
		t.Errorf("LatestBatch files = %v, want 5 entries", files)
	}
}

// TestLatestBatchFallsBackPastEmptyYear confirms that when the newest year
// directory has no files at all, discovery falls back to the previous
// year rather than erroring.
func TestLatestBatchFallsBackPastEmptyYear(t *testing.T) {
	rootWithEmptyLatestYear := `<html><body><pre>
<a href="?prefix=data%2F2025%2F">2025/</a>
<a href="?prefix=data%2F2027%2F">2027/</a>
</pre></body></html>`
	srv := newTestServer(t, map[string]string{
		"":                  rootWithEmptyLatestYear,
		"prefix=data/2027/": emptyYearHTML,
		"prefix=data/2025/": year2025Complete,
	})
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}
	year, date, _, err := d.LatestBatch(context.Background())
	if err != nil {
		t.Fatalf("LatestBatch: %v", err)
	}
	if year != 2025 || date != "20251201" {
		t.Errorf("LatestBatch = %d/%s, want fallback to 2025/20251201", year, date)
	}
}

const year2025Complete = `<html><body><pre>
<a href="?download=data%2F2025%2Fdiscogs_20251201_CHECKSUM.txt">discogs_20251201_CHECKSUM.txt</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_artists.xml.gz">discogs_20251201_artists.xml.gz</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_labels.xml.gz">discogs_20251201_labels.xml.gz</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_masters.xml.gz">discogs_20251201_masters.xml.gz</a>
<a href="?download=data%2F2025%2Fdiscogs_20251201_releases.xml.gz">discogs_20251201_releases.xml.gz</a>
</pre></body></html>`

func TestLatestBatchErrorsOnIncompleteBatch(t *testing.T) {
	srv := newTestServer(t, map[string]string{
		"":                  rootIndexHTML,
		"prefix=data/2026/": emptyYearHTML,
		"prefix=data/2025/": year2025HTML, // present but missing releases.xml.gz
	})
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}
	_, _, _, err := d.LatestBatch(context.Background())
	if err == nil {
		t.Fatal("LatestBatch: want error for incomplete batch, got nil")
	}
}
