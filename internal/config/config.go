// Package config holds shared path layout used by both the fetch and load subcommands.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// DefaultDataDir is used when the --dir flag is not given.
const DefaultDataDir = "datadump"

// Kind identifies one of the four Discogs dump file types.
type Kind string

const (
	KindArtists  Kind = "artists"
	KindLabels   Kind = "labels"
	KindMasters  Kind = "masters"
	KindReleases Kind = "releases"
)

// AllKinds lists every supported dump kind, in a stable order.
var AllKinds = []Kind{KindArtists, KindLabels, KindMasters, KindReleases}

// ArchiveDir returns the directory under dataDir that holds downloaded dump files for year.
func ArchiveDir(dataDir string, year int) string {
	return filepath.Join(dataDir, "archive", fmt.Sprintf("%04d", year))
}

// DumpFileName returns the expected filename for a given date batch (YYYYMMDD) and kind.
func DumpFileName(date string, kind Kind) string {
	return fmt.Sprintf("discogs_%s_%s.xml.gz", date, kind)
}

// ChecksumFileName returns the expected CHECKSUM filename for a given date batch (YYYYMMDD).
func ChecksumFileName(date string) string {
	return fmt.Sprintf("discogs_%s_CHECKSUM.txt", date)
}

// ArchivePath joins ArchiveDir and DumpFileName for a given year/date/kind.
func ArchivePath(dataDir string, year int, date string, kind Kind) string {
	return filepath.Join(ArchiveDir(dataDir, year), DumpFileName(date, kind))
}

// ChecksumPath joins ArchiveDir and ChecksumFileName for a given year/date.
func ChecksumPath(dataDir string, year int, date string) string {
	return filepath.Join(ArchiveDir(dataDir, year), ChecksumFileName(date))
}

var checksumNameRE = regexp.MustCompile(`^discogs_(\d{8})_CHECKSUM\.txt$`)

// ListArchivedDates scans <dataDir>/archive/*/ for CHECKSUM files and
// returns every date batch (YYYYMMDD) found, across all years, sorted.
// Used to produce a helpful error when a requested --date isn't on disk.
func ListArchivedDates(dataDir string) ([]string, error) {
	archiveRoot := filepath.Join(dataDir, "archive")
	yearDirs, err := os.ReadDir(archiveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", archiveRoot, err)
	}

	var dates []string
	for _, yd := range yearDirs {
		if !yd.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(archiveRoot, yd.Name()))
		if err != nil {
			return nil, fmt.Errorf("config: read %s: %w", filepath.Join(archiveRoot, yd.Name()), err)
		}
		for _, e := range entries {
			if m := checksumNameRE.FindStringSubmatch(e.Name()); m != nil {
				dates = append(dates, m[1])
			}
		}
	}
	sort.Strings(dates)
	return dates, nil
}
