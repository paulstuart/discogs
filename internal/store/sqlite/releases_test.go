package sqlite

import (
	"os"
	"testing"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

func loadReleasesFixture(t *testing.T) []model.Release {
	t.Helper()
	f, err := os.Open("testdata/releases_fixture.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var releases []model.Release
	if err := xmlstream.StreamDecode(f, "release", func(r model.Release) error {
		releases = append(releases, r)
		return nil
	}); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return releases
}

func TestPutReleases(t *testing.T) {
	releases := loadReleasesFixture(t)
	if len(releases) != 1 {
		t.Fatalf("got %d releases from fixture, want 1", len(releases))
	}
	r := releases[0]
	if r.MasterID == nil || r.MasterID.ID != 1660109 || !r.MasterID.IsMainRelease {
		t.Fatalf("release 1 MasterID = %+v, want {1660109 true}", r.MasterID)
	}

	s := openTestStore(t)
	putReleasesFull(t, s, releases)

	count := scalar[int](t, s, "SELECT count(*) FROM releases")
	if count != 1 {
		t.Errorf("releases row count = %d, want 1", count)
	}

	title := scalar[string](t, s, "SELECT title FROM releases WHERE id = 1")
	if title != "Stockholm" {
		t.Errorf("release 1 title = %q, want Stockholm", title)
	}
	masterID := scalar[int64](t, s, "SELECT master_id FROM releases WHERE id = 1")
	if masterID != 1660109 {
		t.Errorf("release 1 master_id column = %d, want 1660109", masterID)
	}

	checks := []struct {
		table string
		want  int
	}{
		{"release_tracks", 6},
		{"release_identifiers", 4},
		{"release_videos", 7},
		{"release_companies", 3},
		{"release_formats", 1},
		{"release_format_descriptions", 2},
		{"release_genres", 1},
		{"release_styles", 1},
		{"release_artists", 1},
	}
	for _, c := range checks {
		got := scalar[int](t, s, "SELECT count(*) FROM "+c.table+" WHERE release_id = 1")
		if got != c.want {
			t.Errorf("%s row count = %d, want %d", c.table, got, c.want)
		}
	}

	// Normalized vocabulary (genre/style/country/identifier type/company
	// role) resolves back to its text via the *_named views / joins.
	genre := scalar[string](t, s, "SELECT genre FROM release_genres_named WHERE release_id = 1")
	if genre != "Electronic" {
		t.Errorf("release 1 genre = %q, want Electronic", genre)
	}
	style := scalar[string](t, s, "SELECT style FROM release_styles_named WHERE release_id = 1")
	if style != "Deep House" {
		t.Errorf("release 1 style = %q, want Deep House", style)
	}
	country := scalar[string](t, s, `
		SELECT co.name FROM releases r JOIN countries co ON co.id = r.country_id
		WHERE r.id = 1`)
	if country != "Sweden" {
		t.Errorf("release 1 country = %q, want Sweden", country)
	}
	idType := scalar[string](t, s, "SELECT type FROM release_identifiers_named WHERE release_id = 1 AND seq = 0")
	if idType != "Matrix / Runout" {
		t.Errorf("release 1 identifier[0] type = %q, want 'Matrix / Runout'", idType)
	}
	// All 4 identifiers share the same type text; confirm that resolved
	// to one shared identifier_types row, not four.
	idTypeRows := scalar[int](t, s, "SELECT count(*) FROM identifier_types WHERE name = 'Matrix / Runout'")
	if idTypeRows != 1 {
		t.Errorf("identifier_types has %d 'Matrix / Runout' rows, want exactly 1 (shared lookup)", idTypeRows)
	}
	companyRole := scalar[string](t, s, "SELECT role FROM release_companies_named WHERE release_id = 1 AND company_id = 56025")
	if companyRole != "Pressed By" {
		t.Errorf("release 1 company 56025 role = %q, want 'Pressed By'", companyRole)
	}

	// A second load (Prepare wipes, then reload) should fully replace,
	// not duplicate, the data.
	putReleasesFull(t, s, releases)
	got := scalar[int](t, s, "SELECT count(*) FROM release_tracks WHERE release_id = 1")
	if got != 6 {
		t.Errorf("release_tracks row count after rerun = %d, want 6 (no duplicates)", got)
	}
}
