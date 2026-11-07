// Package fetch discovers and downloads Discogs monthly data dump batches
// from https://data.discogs.com/.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"

	"github.com/paulstuart/discogs/internal/config"
	"golang.org/x/net/html"
)

// DefaultBaseURL is the production Discogs data dump host.
const DefaultBaseURL = "https://data.discogs.com"

// Discoverer lists available dump years and batches by scraping the HTML
// index pages served at BaseURL. BaseURL and Client are overridable for
// testing against an httptest.Server.
type Discoverer struct {
	BaseURL string
	Client  *http.Client
}

// NewDiscoverer returns a Discoverer pointed at the real Discogs host.
func NewDiscoverer() *Discoverer {
	return &Discoverer{BaseURL: DefaultBaseURL, Client: http.DefaultClient}
}

var filenameRE = regexp.MustCompile(`^discogs_(\d{8})_(artists|labels|masters|releases|CHECKSUM)\.(?:xml\.gz|txt)$`)

// fetchAnchorParam GETs path on the Discoverer's host, parses the HTML
// response, and returns the decoded value of query parameter param for
// every <a href="..."> found.
func (d *Discoverer) fetchAnchorParam(ctx context.Context, path, param string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch: build request for %s: %w", path, err)
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("fetch: GET %s: status %s", path, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch: parse HTML from %s: %w", path, err)
	}

	var values []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				u, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				if v := u.Query().Get(param); v != "" {
					values = append(values, v)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return values, nil
}

// ListYears returns every year directory advertised at the site root, in
// ascending order.
func (d *Discoverer) ListYears(ctx context.Context) ([]int, error) {
	prefixes, err := d.fetchAnchorParam(ctx, "/", "prefix")
	if err != nil {
		return nil, err
	}
	yearRE := regexp.MustCompile(`^data/(\d{4})/$`)
	var years []int
	for _, p := range prefixes {
		m := yearRE.FindStringSubmatch(p)
		if m == nil {
			continue
		}
		var y int
		fmt.Sscanf(m[1], "%d", &y)
		years = append(years, y)
	}
	sort.Ints(years)
	return years, nil
}

// ListBatches returns every date batch (keyed by YYYYMMDD) found in a
// year's directory, mapping to the bare filenames present for that date
// (the four *.xml.gz kinds plus CHECKSUM.txt, whichever are present).
func (d *Discoverer) ListBatches(ctx context.Context, year int) (map[string][]string, error) {
	downloads, err := d.fetchAnchorParam(ctx, fmt.Sprintf("/?prefix=data/%04d/", year), "download")
	if err != nil {
		return nil, err
	}
	batches := map[string][]string{}
	for _, dl := range downloads {
		// dl looks like "data/2026/discogs_20261101_artists.xml.gz"; take
		// the filename after the last slash.
		name := dl
		if i := lastSlash(dl); i >= 0 {
			name = dl[i+1:]
		}
		m := filenameRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		date := m[1]
		batches[date] = append(batches[date], name)
	}
	return batches, nil
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// isComplete reports whether filenames contains all four dump kinds plus
// the CHECKSUM file for the given date.
func isComplete(date string, filenames []string) bool {
	want := map[string]bool{
		config.ChecksumFileName(date): false,
	}
	for _, k := range config.AllKinds {
		want[config.DumpFileName(date, k)] = false
	}
	for _, f := range filenames {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for _, present := range want {
		if !present {
			return false
		}
	}
	return true
}

// LatestBatch finds the most recent complete date batch, searching years
// from newest to oldest. An empty year directory (no files at all) falls
// through to the previous year; a year whose latest date batch is present
// but missing a kind or its CHECKSUM is treated as an error rather than
// silently skipped, since that's a genuine anomaly worth surfacing.
func (d *Discoverer) LatestBatch(ctx context.Context) (year int, date string, filenames []string, err error) {
	years, err := d.ListYears(ctx)
	if err != nil {
		return 0, "", nil, err
	}
	if len(years) == 0 {
		return 0, "", nil, fmt.Errorf("fetch: no year directories found at %s", d.BaseURL)
	}
	for i := len(years) - 1; i >= 0; i-- {
		y := years[i]
		date, files, err := d.LatestInYear(ctx, y)
		if err != nil {
			return 0, "", nil, err
		}
		if date == "" {
			continue // empty year directory, fall back to the previous year
		}
		return y, date, files, nil
	}
	return 0, "", nil, fmt.Errorf("fetch: no non-empty year directories found at %s", d.BaseURL)
}

// LatestInYear returns the most recent date batch within a single year
// directory. If the year directory has no files at all, it returns an
// empty date and a nil error (the caller decides whether that's an error
// or a reason to look at an earlier year). If the year has files but the
// latest date batch is missing a kind or its CHECKSUM, that's returned as
// an error.
func (d *Discoverer) LatestInYear(ctx context.Context, year int) (date string, filenames []string, err error) {
	batches, err := d.ListBatches(ctx, year)
	if err != nil {
		return "", nil, err
	}
	if len(batches) == 0 {
		return "", nil, nil
	}
	var dates []string
	for date := range batches {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	latestDate := dates[len(dates)-1]
	files := batches[latestDate]
	if !isComplete(latestDate, files) {
		return "", nil, fmt.Errorf("fetch: latest batch %d/%s at %s is incomplete: found %v", year, latestDate, d.BaseURL, files)
	}
	return latestDate, files, nil
}

// Batch resolves an explicit year/date to its filenames, validating that
// the batch exists and is complete.
func (d *Discoverer) Batch(ctx context.Context, year int, date string) (filenames []string, err error) {
	batches, err := d.ListBatches(ctx, year)
	if err != nil {
		return nil, err
	}
	files, ok := batches[date]
	if !ok {
		return nil, fmt.Errorf("fetch: no batch found for %d/%s at %s", year, date, d.BaseURL)
	}
	if !isComplete(date, files) {
		return nil, fmt.Errorf("fetch: batch %d/%s at %s is incomplete: found %v", year, date, d.BaseURL, files)
	}
	return files, nil
}
