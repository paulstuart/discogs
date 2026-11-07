package fetch

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/paulstuart/discogs/internal/config"
)

// Checksums maps a bare filename to its expected sha256 hex digest, as
// published in a batch's CHECKSUM.txt.
type Checksums map[string]string

// ParseChecksums parses lines of the form "<64-hex-char-sha256>  <filename>".
func ParseChecksums(r io.Reader) (Checksums, error) {
	sums := Checksums{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("fetch: malformed checksum line %q", line)
		}
		sums[fields[1]] = fields[0]
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("fetch: read checksums: %w", err)
	}
	return sums, nil
}

// sha256File computes the hex sha256 digest of the file at path.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// downloadURL builds the direct-download URL for a dump file. The site
// streams file content directly for this endpoint (HTTP 200, no
// redirect/signing), confirmed against the live host.
func downloadURL(baseURL string, year int, filename string) string {
	key := fmt.Sprintf("data/%04d/%s", year, filename)
	return baseURL + "/?download=" + url.QueryEscape(key)
}

// downloadToFile streams srcURL to destPath via a .part sibling file,
// renaming to destPath only once the full body has been written, so a
// crash or interrupted download never leaves a file that looks complete.
func downloadToFile(ctx context.Context, client *http.Client, srcURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return fmt.Errorf("fetch: build request for %s: %w", srcURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: GET %s: %w", srcURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("fetch: GET %s: status %s", srcURL, resp.Status)
	}

	partPath := destPath + ".part"
	out, err := os.Create(partPath)
	if err != nil {
		return fmt.Errorf("fetch: create %s: %w", partPath, err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(partPath)
		return fmt.Errorf("fetch: write %s: %w", partPath, err)
	}
	if err := out.Close(); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("fetch: close %s: %w", partPath, err)
	}
	if err := os.Rename(partPath, destPath); err != nil {
		return fmt.Errorf("fetch: rename %s to %s: %w", partPath, destPath, err)
	}
	return nil
}

// DownloadBatch downloads every file in a date batch into
// <dataDir>/archive/<year>/, skipping any file that already exists and
// matches its published checksum, and verifying every freshly downloaded
// file against the same checksum. filenames must include the batch's
// CHECKSUM.txt alongside the four *.xml.gz files (as returned by
// Discoverer.LatestBatch/Batch).
func DownloadBatch(ctx context.Context, d *Discoverer, dataDir string, year int, date string, filenames []string) error {
	archiveDir := config.ArchiveDir(dataDir, year)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("fetch: create archive dir %s: %w", archiveDir, err)
	}

	checksumName := config.ChecksumFileName(date)
	checksumPath := filepath.Join(archiveDir, checksumName)
	// The checksum file itself is tiny and re-fetched unconditionally so
	// it's always current for verifying the larger files below.
	if err := downloadToFile(ctx, d.Client, downloadURL(d.BaseURL, year, checksumName), checksumPath); err != nil {
		return err
	}
	cf, err := os.Open(checksumPath)
	if err != nil {
		return fmt.Errorf("fetch: open %s: %w", checksumPath, err)
	}
	sums, err := ParseChecksums(cf)
	cf.Close()
	if err != nil {
		return err
	}

	for _, name := range filenames {
		if name == checksumName {
			continue
		}
		want, ok := sums[name]
		if !ok {
			return fmt.Errorf("fetch: no checksum entry for %s in %s", name, checksumName)
		}
		destPath := filepath.Join(archiveDir, name)

		if existing, err := os.Stat(destPath); err == nil && !existing.IsDir() {
			got, err := sha256File(destPath)
			if err != nil {
				return fmt.Errorf("fetch: hash existing %s: %w", destPath, err)
			}
			if got == want {
				log.Printf("fetch: %s already present and verified, skipping", name)
				continue
			}
			log.Printf("fetch: %s present but checksum mismatch, re-downloading", name)
		}

		log.Printf("fetch: downloading %s", name)
		if err := downloadToFile(ctx, d.Client, downloadURL(d.BaseURL, year, name), destPath); err != nil {
			return err
		}
		got, err := sha256File(destPath)
		if err != nil {
			return fmt.Errorf("fetch: hash %s: %w", destPath, err)
		}
		if got != want {
			os.Remove(destPath)
			return fmt.Errorf("fetch: checksum mismatch for %s: got %s, want %s", name, got, want)
		}
		log.Printf("fetch: %s verified", name)
	}
	return nil
}
