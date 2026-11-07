package sqlite

import (
	"context"
	"os"
	"testing"

	"github.com/paulstuart/discogs/internal/model"
	"github.com/paulstuart/discogs/internal/xmlstream"
)

func loadMastersFixture(t *testing.T) []model.Master {
	t.Helper()
	f, err := os.Open("testdata/masters_fixture.xml")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var masters []model.Master
	if err := xmlstream.StreamDecode(f, "master", func(m model.Master) error {
		masters = append(masters, m)
		return nil
	}); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return masters
}

func TestPutMasters(t *testing.T) {
	masters := loadMastersFixture(t)
	if len(masters) != 2 {
		t.Fatalf("got %d masters from fixture, want 2", len(masters))
	}

	s := openTestStore(t)
	ctx := context.Background()
	if err := s.PutMasters(ctx, masters); err != nil {
		t.Fatalf("PutMasters: %v", err)
	}

	count := scalar[int](t, s, "SELECT count(*) FROM masters")
	if count != 2 {
		t.Errorf("masters row count = %d, want 2", count)
	}

	title := scalar[string](t, s, "SELECT title FROM masters WHERE id = 113")
	if title != "Moments In Time" {
		t.Errorf("master 113 title = %q, want Moments In Time", title)
	}

	videoCount := scalar[int](t, s, "SELECT count(*) FROM master_videos WHERE master_id = 113")
	if videoCount != 7 {
		t.Errorf("master 113 video count = %d, want 7", videoCount)
	}

	styleCount := scalar[int](t, s, "SELECT count(*) FROM master_styles WHERE master_id = 113")
	if styleCount != 2 {
		t.Errorf("master 113 style count = %d, want 2", styleCount)
	}

	artistID := scalar[int64](t, s, "SELECT artist_id FROM master_artists WHERE master_id = 113 AND seq = 0")
	if artistID != 3225 {
		t.Errorf("master 113 artist[0].id = %d, want 3225", artistID)
	}

	// Styles are normalized into a lookup table; master_styles_named
	// resolves style_id back to the text.
	hasTechno := scalar[int](t, s, `
		SELECT count(*) FROM master_styles_named
		WHERE master_id = 113 AND style = 'Techno'`)
	if hasTechno != 1 {
		t.Errorf("master 113 master_styles_named has %d 'Techno' rows, want 1", hasTechno)
	}

	// Master 114 in the fixture also has a "Techno" style; it should
	// resolve to the same styles.id as master 113's, not a duplicate row.
	styleRowCount := scalar[int](t, s, "SELECT count(*) FROM styles WHERE name = 'Techno'")
	if styleRowCount != 1 {
		t.Errorf("styles table has %d 'Techno' rows, want exactly 1 (shared lookup)", styleRowCount)
	}
}
