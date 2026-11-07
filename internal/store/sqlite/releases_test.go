package sqlite

import (
	"context"
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
	ctx := context.Background()
	if err := s.PutReleases(ctx, releases); err != nil {
		t.Fatalf("PutReleases: %v", err)
	}

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

	// Re-run idempotency: PutReleases twice should not duplicate child rows.
	if err := s.PutReleases(ctx, releases); err != nil {
		t.Fatalf("PutReleases (rerun): %v", err)
	}
	got := scalar[int](t, s, "SELECT count(*) FROM release_tracks WHERE release_id = 1")
	if got != 6 {
		t.Errorf("release_tracks row count after rerun = %d, want 6 (no duplicates)", got)
	}
}
