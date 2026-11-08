package sqlite

import (
	"testing"
)

// TestReleaseSearchView exercises the release_search view end to end:
// load the releases fixture (artist "The Persuader", id 1) and find it by
// a name-prefix search, the way a "find albums by artist" query would.
func TestReleaseSearchView(t *testing.T) {
	releases := loadReleasesFixture(t)
	s := openTestStore(t)
	putReleasesFull(t, s, releases)

	title := scalar[string](t, s, `
		SELECT release_title FROM release_search
		WHERE artist_name LIKE 'The Persuader%'`)
	if title != "Stockholm" {
		t.Errorf("release_search title = %q, want Stockholm", title)
	}
}

// TestMasterSearchView mirrors TestReleaseSearchView for master_search,
// against the masters fixture.
func TestMasterSearchView(t *testing.T) {
	masters := loadMastersFixture(t)
	s := openTestStore(t)
	putMastersFull(t, s, masters)

	title := scalar[string](t, s, `
		SELECT master_title FROM master_search
		WHERE artist_name LIKE 'Vince Watson%'`)
	if title != "Moments In Time" {
		t.Errorf("master_search title = %q, want Moments In Time", title)
	}
}
