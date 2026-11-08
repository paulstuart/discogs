package sqlite

import (
	"os"
	"testing"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

func loadArtistsFixture(t *testing.T) []model.Artist {
	t.Helper()
	f, err := os.Open("testdata/artists_fixture.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var artists []model.Artist
	if err := xmlstream.StreamDecode(f, "artist", func(a model.Artist) error {
		artists = append(artists, a)
		return nil
	}); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return artists
}

func TestPutArtists(t *testing.T) {
	artists := loadArtistsFixture(t)
	if len(artists) != 3 {
		t.Fatalf("got %d artists from fixture, want 3", len(artists))
	}

	s := openTestStore(t)
	putArtistsFull(t, s, artists)

	count := scalar[int](t, s, "SELECT count(*) FROM artists")
	if count != 3 {
		t.Errorf("artists row count = %d, want 3", count)
	}

	name := scalar[string](t, s, "SELECT name FROM artists WHERE id = 1")
	if name != "The Persuader" {
		t.Errorf("artist 1 name = %q, want The Persuader", name)
	}

	aliasCount := scalar[int](t, s, "SELECT count(*) FROM artist_aliases WHERE artist_id = 1")
	if aliasCount != 7 {
		t.Errorf("artist 1 alias count = %d, want 7", aliasCount)
	}

	memberCount := scalar[int](t, s, "SELECT count(*) FROM artist_members WHERE artist_id = 2")
	if memberCount != 2 {
		t.Errorf("artist 2 member count = %d, want 2 (it's a group)", memberCount)
	}

	// A second load (Prepare wipes, then reload) should fully replace,
	// not duplicate, the data.
	putArtistsFull(t, s, artists)
	count = scalar[int](t, s, "SELECT count(*) FROM artists")
	if count != 3 {
		t.Errorf("artists row count after rerun = %d, want 3", count)
	}
}
