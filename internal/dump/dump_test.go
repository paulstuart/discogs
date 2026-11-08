package dump

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/paulstuart/discogs/internal/config"
	"github.com/paulstuart/discogs/internal/model"
)

// gzipFixture compresses a small fixture XML file into the expected
// archive location for a given date/kind, mirroring internal/load's test
// helper.
func gzipFixture(t *testing.T, dataDir string, year int, date string, kind config.Kind, fixturePath string) {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	archiveDir := config.ArchiveDir(dataDir, year)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", archiveDir, err)
	}
	dest := config.ArchivePath(dataDir, year, date, kind)
	f, err := os.Create(dest)
	if err != nil {
		t.Fatalf("create %s: %v", dest, err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", dest, err)
	}
}

func TestRunArtistsJSONLines(t *testing.T) {
	dataDir := t.TempDir()
	const date = "20261101"
	gzipFixture(t, dataDir, 2026, date, config.KindArtists, "../store/sqlite/testdata/artists_fixture.xml")

	var buf bytes.Buffer
	opts := Options{
		DataDir: dataDir,
		Date:    date,
		Kind:    config.KindArtists,
		Out:     &buf,
	}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(lines), buf.String())
	}
	var first model.Artist
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	if first.ID == 0 || first.Name == "" {
		t.Errorf("decoded artist looks empty: %+v", first)
	}
}

func TestRunReleasesJSONLines(t *testing.T) {
	dataDir := t.TempDir()
	const date = "20261101"
	gzipFixture(t, dataDir, 2026, date, config.KindReleases, "../store/sqlite/testdata/releases_fixture.xml")

	var buf bytes.Buffer
	opts := Options{
		DataDir: dataDir,
		Date:    date,
		Kind:    config.KindReleases,
		Out:     &buf,
	}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	dec := json.NewDecoder(&buf)
	var got []model.Release
	for dec.More() {
		var r model.Release
		if err := dec.Decode(&r); err != nil {
			t.Fatalf("decode release: %v", err)
		}
		got = append(got, r)
	}
	if len(got) == 0 {
		t.Fatal("decoded 0 releases, want at least 1")
	}
	for _, r := range got {
		if r.ID == 0 || r.Title == "" {
			t.Errorf("decoded release looks empty: %+v", r)
		}
	}
}

func TestRunMissingFileError(t *testing.T) {
	dataDir := t.TempDir()
	opts := Options{
		DataDir: dataDir,
		Date:    "20261101",
		Kind:    config.KindArtists,
		Out:     &bytes.Buffer{},
	}
	if err := Run(context.Background(), opts); err == nil {
		t.Fatal("Run: want error for missing archive file, got nil")
	}
}

func TestRunUnknownKindError(t *testing.T) {
	opts := Options{
		DataDir: t.TempDir(),
		Date:    "20261101",
		Kind:    config.Kind("bogus"),
		Out:     &bytes.Buffer{},
	}
	if err := Run(context.Background(), opts); err == nil {
		t.Fatal("Run: want error for unknown kind, got nil")
	}
}
