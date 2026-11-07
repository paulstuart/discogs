package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/paulstuart/discogs/internal/config"
)

func hexSum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// downloadServer serves fixed byte payloads for each dump kind plus a
// matching CHECKSUM.txt, and counts requests per path so tests can assert
// skip-if-valid behavior.
func newDownloadServer(t *testing.T, date string, bodies map[config.Kind][]byte) (*httptest.Server, map[string]int) {
	t.Helper()
	counts := map[string]int{}
	checksum := ""
	for _, k := range config.AllKinds {
		name := config.DumpFileName(date, k)
		checksum += hexSum(bodies[k]) + "  " + name + "\n"
	}
	checksumName := config.ChecksumFileName(date)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("download")
		name := filepath.Base(key)
		counts[name]++
		if name == checksumName {
			w.Write([]byte(checksum))
			return
		}
		for _, k := range config.AllKinds {
			if name == config.DumpFileName(date, k) {
				w.Write(bodies[k])
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, counts
}

func TestDownloadBatchFreshDownload(t *testing.T) {
	const date = "20261101"
	bodies := map[config.Kind][]byte{
		config.KindArtists:  []byte("artists-data"),
		config.KindLabels:   []byte("labels-data"),
		config.KindMasters:  []byte("masters-data"),
		config.KindReleases: []byte("releases-data"),
	}
	srv, counts := newDownloadServer(t, date, bodies)
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}

	dataDir := t.TempDir()
	var filenames []string
	for _, k := range config.AllKinds {
		filenames = append(filenames, config.DumpFileName(date, k))
	}
	filenames = append(filenames, config.ChecksumFileName(date))

	if err := DownloadBatch(context.Background(), d, dataDir, 2026, date, filenames); err != nil {
		t.Fatalf("DownloadBatch: %v", err)
	}

	for _, k := range config.AllKinds {
		path := config.ArchivePath(dataDir, 2026, date, k)
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != string(bodies[k]) {
			t.Errorf("%s content = %q, want %q", path, got, bodies[k])
		}
		if counts[config.DumpFileName(date, k)] != 1 {
			t.Errorf("%s requested %d times, want 1", k, counts[config.DumpFileName(date, k)])
		}
	}
}

func TestDownloadBatchSkipsValidExisting(t *testing.T) {
	const date = "20261101"
	bodies := map[config.Kind][]byte{
		config.KindArtists:  []byte("artists-data"),
		config.KindLabels:   []byte("labels-data"),
		config.KindMasters:  []byte("masters-data"),
		config.KindReleases: []byte("releases-data"),
	}
	srv, counts := newDownloadServer(t, date, bodies)
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}

	dataDir := t.TempDir()
	archiveDir := config.ArchiveDir(dataDir, 2026)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-place a correct artists file; it should be skipped (no GET).
	artistsPath := config.ArchivePath(dataDir, 2026, date, config.KindArtists)
	if err := os.WriteFile(artistsPath, bodies[config.KindArtists], 0o644); err != nil {
		t.Fatal(err)
	}

	var filenames []string
	for _, k := range config.AllKinds {
		filenames = append(filenames, config.DumpFileName(date, k))
	}
	filenames = append(filenames, config.ChecksumFileName(date))

	if err := DownloadBatch(context.Background(), d, dataDir, 2026, date, filenames); err != nil {
		t.Fatalf("DownloadBatch: %v", err)
	}

	if n := counts[config.DumpFileName(date, config.KindArtists)]; n != 0 {
		t.Errorf("artists file requested %d times, want 0 (should be skipped)", n)
	}
	if n := counts[config.DumpFileName(date, config.KindLabels)]; n != 1 {
		t.Errorf("labels file requested %d times, want 1", n)
	}
}

func TestDownloadBatchRedownloadsCorrupt(t *testing.T) {
	const date = "20261101"
	bodies := map[config.Kind][]byte{
		config.KindArtists:  []byte("artists-data"),
		config.KindLabels:   []byte("labels-data"),
		config.KindMasters:  []byte("masters-data"),
		config.KindReleases: []byte("releases-data"),
	}
	srv, counts := newDownloadServer(t, date, bodies)
	d := &Discoverer{BaseURL: srv.URL, Client: srv.Client()}

	dataDir := t.TempDir()
	archiveDir := config.ArchiveDir(dataDir, 2026)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artistsPath := config.ArchivePath(dataDir, 2026, date, config.KindArtists)
	if err := os.WriteFile(artistsPath, []byte("corrupted-partial-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var filenames []string
	for _, k := range config.AllKinds {
		filenames = append(filenames, config.DumpFileName(date, k))
	}
	filenames = append(filenames, config.ChecksumFileName(date))

	if err := DownloadBatch(context.Background(), d, dataDir, 2026, date, filenames); err != nil {
		t.Fatalf("DownloadBatch: %v", err)
	}

	if n := counts[config.DumpFileName(date, config.KindArtists)]; n != 1 {
		t.Errorf("artists file requested %d times, want 1 (corrupt copy should be redownloaded)", n)
	}
	got, err := os.ReadFile(artistsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(bodies[config.KindArtists]) {
		t.Errorf("artists content after redownload = %q, want %q", got, bodies[config.KindArtists])
	}
}
