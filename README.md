# discogs

A CLI that downloads Discogs' monthly XML data dumps and loads them into a database.

Discogs publishes four dump types monthly at [data.discogs.com](https://data.discogs.com/): artists, labels, masters, and releases. `discogs fetch` downloads the latest (or a specific) batch into a local archive directory; `discogs load` separately streams an archived batch into a database, without ever writing decompressed XML to disk.

## Usage

```
discogs fetch [--dir datadump] [--year YYYY] [--date YYYYMMDD]
discogs load  [--dir datadump] --date YYYYMMDD [--db sqlite:discogs.db] [--kinds artists,labels,masters,releases]
```

`--dir` (default `datadump`) is the data directory used by both subcommands; downloaded files are archived under `<dir>/archive/<YYYY>/`.

### fetch

Downloads the latest complete monthly batch (all four `*.xml.gz` files plus their `CHECKSUM.txt`), auto-discovering the newest year and date from the site index. Already-downloaded files are verified against their published sha256 checksum and skipped if they match, so re-running `fetch` is cheap and safe. `--year`/`--date` override auto-discovery to fetch a specific (e.g. older) batch.

These files are large (artists ~470MB, labels ~85MB, masters ~600MB, releases ~10.5GB+ as of late 2026) — `fetch` is meant to be run deliberately and infrequently, never automatically from `load`.

### load

Streams a previously-archived batch's `*.xml.gz` files into a database, gzip-decompressing and decoding XML token-by-token so the full file is never buffered or written to disk. Records are upserted (keyed on Discogs' own numeric IDs), so re-running `load` for the same or a newer date batch is safe and won't duplicate data.

Only `sqlite:<path>` is implemented today; the `store.Store` interface (`internal/store/store.go`) is the seam for adding another backend later.

## Development

```
go build ./...
go test ./...
```

Store/decoder tests run against small fixtures trimmed from real dump samples (`internal/store/sqlite/testdata/`), so they exercise the real schema shape without needing network access or multi-gigabyte files.
