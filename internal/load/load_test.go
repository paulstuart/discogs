package load

import (
	"compress/gzip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/paulstuart/discogs/internal/config"
)

// gzipFixture compresses a small fixture XML file into the expected
// archive location for a given date/kind.
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

func TestRunEndToEnd(t *testing.T) {
	dataDir := t.TempDir()
	const date = "20261101"
	gzipFixture(t, dataDir, 2026, date, config.KindArtists, "../store/sqlite/testdata/artists_fixture.xml")
	gzipFixture(t, dataDir, 2026, date, config.KindLabels, "../store/sqlite/testdata/labels_fixture.xml")

	dbPath := filepath.Join(t.TempDir(), "discogs.db")
	opts := Options{
		DataDir: dataDir,
		Date:    date,
		DB:      "sqlite:" + dbPath,
		Kinds:   []config.Kind{config.KindArtists, config.KindLabels},
	}

	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertCount := func(table string, want int) {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		defer db.Close()
		var got int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Errorf("%s row count = %d, want %d", table, got, want)
		}
	}
	assertCount("artists", 3)
	assertCount("labels", 3)

	// Re-running over the same files should upsert, not duplicate.
	if err := Run(context.Background(), opts); err != nil {
		t.Fatalf("Run (rerun): %v", err)
	}
	assertCount("artists", 3)
	assertCount("labels", 3)
}

func TestRunMissingFileError(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "discogs.db")
	opts := Options{
		DataDir: dataDir,
		Date:    "20261101",
		DB:      "sqlite:" + dbPath,
		Kinds:   []config.Kind{config.KindArtists},
	}
	if err := Run(context.Background(), opts); err == nil {
		t.Fatal("Run: want error for missing archive file, got nil")
	}
}
