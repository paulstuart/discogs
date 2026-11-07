# AGENTS.md

Guidance for AI agents (and humans) developing this repo. This project downloads Discogs'
monthly XML data dumps and loads them into a database. Read this before making structural
changes — it captures decisions and real-data gotchas that aren't obvious from the code alone.

## What this is

```
discogs fetch [--dir datadump] [--year YYYY] [--date YYYYMMDD]
discogs load  [--dir datadump] --date YYYYMMDD [--db sqlite:discogs.db] [--kinds artists,labels,masters,releases]
```

`fetch` and `load` are deliberately separate invocations. `fetch` downloads large files from
`https://data.discogs.com/` into `<dir>/archive/<YYYY>/`; `load` streams an already-archived
batch into a database. `load` never triggers a `fetch`, and `fetch` is never automatic —
these files are large (releases.xml.gz alone is ~11GB compressed, ~100GB+ decompressed) and
downloads should stay deliberate and infrequent.

## Build / test

```
go build ./...
go vet ./...
go test ./...
gofmt -l .        # should print nothing
```

No network access is required for `go test ./...` — `internal/fetch`'s tests use
`httptest.Server` with fixed HTML fixtures, and `internal/store/sqlite`/`internal/load`'s
tests use small XML fixtures trimmed from real dump samples (`internal/store/sqlite/testdata/`,
3-5 records each, real IDs/names preserved) rather than synthetic data. This matters because
the real XML has structural quirks (see below) that synthetic test data tends not to
reproduce.

## Architecture

```
cmd/discogs/            thin CLI: flag parsing + dispatch only, no logic
cmd/xmlschema/           standalone tool: sniffs any XML file's element/attribute shape
internal/model/          canonical structs (xml + json tags), used by both decoder and store
internal/xmlstream/      generic StreamDecode[T](r, elem, fn) — one streaming decoder, reused
                         by all four entity types. Never buffers the document.
internal/fetch/          discovery (HTML scraping via golang.org/x/net/html) + checksum-
                         verified download
internal/load/           gzip -> xmlstream -> batch (2000 records) -> store.Put*
internal/store/          Store interface + DSN dispatch (store.Open("sqlite:path.db"))
internal/store/sqlite/   the only implementation today; schema.sql (go:embed'd), one file
                         per entity (artists.go, labels.go, masters.go, releases.go)
```

`internal/model` is the single struct set for all four entities — don't create parallel
"XML struct" vs "DB struct" pairs. If a new output format is needed (e.g. real JSON export),
add it as a serialization of these structs, not a new model package.

`store.Store` is the extensibility seam for other databases. Adding a backend means a new
`internal/store/<name>` package implementing the five-method interface, plus one `case` in
`store.Open`'s DSN dispatch — no changes to `internal/load`.

## Schema design (why it looks the way it does)

- **No FOREIGN KEY constraints.** The four dump files are loaded independently and contain
  forward references (e.g. a label's sublabel ID that hasn't been loaded yet, or a release's
  `master_id` pointing at a master you may never load). FK-shaped columns exist for joins and
  are indexed, just not enforced.
- **Upsert, not insert.** Parent tables (`artists`, `labels`, `masters`, `releases`) use
  `INSERT ... ON CONFLICT(id) DO UPDATE`. Every child/join table uses `DELETE FROM child WHERE
  parent_id = ?` immediately followed by fresh inserts, all in the same transaction as the
  parent upsert. This is required, not just tidy: a re-loaded record's child list can *shrink*
  (fewer URLs, fewer tracks), which `ON CONFLICT DO UPDATE` can't express for a list but
  delete-then-reinsert handles naturally.
- **Child-table inserts are `INSERT OR IGNORE`, not `INSERT`.** Real Discogs data is not fully
  deduplicated — e.g. a label's `<urls>` list can contain the same URL twice. This was caught
  live (not by the fixture tests) during the first real load: label id 14 had a duplicate URL
  and hit the `label_urls (label_id, url)` composite primary key. If you add a new child table,
  default to `INSERT OR IGNORE` unless you've specifically confirmed the source data can't
  repeat that key.
- **`master_id` on a release decodes as attribute + chardata.** `<master_id
  is_main_release="true">1660109</master_id>` maps to `model.MasterRef{IsMainRelease bool
  `xml:"is_main_release,attr"`; ID int64 `xml:",chardata"`}`. Confirmed working via
  `internal/model/model_test.go` — don't "simplify" this to a plain int field.
- **`released` stays a raw string**, e.g. `"1999-03-00"` (zero day is valid in Discogs data).
  Never parse it into a SQL DATE/time.Time.
- **Batch size is 2000** top-level records per transaction (`internal/load/load.go`). This
  balances transaction overhead against memory; re-tune only with a measured reason.
- **SQLite tuning**: WAL journal mode, `synchronous=NORMAL`, `SetMaxOpenConns(1)` (SQLite is
  single-writer; serializing avoids fighting `SQLITE_BUSY` entirely rather than retrying).
  `release_search` / `master_search` views and name indexes (`idx_*_name ... COLLATE NOCASE`)
  exist in `schema.sql` for "find records by artist name" queries without needing to join
  through `release_artists`/`master_artists` by hand.

## Known real-data field shapes (confirmed against live dumps, not assumed)

- `id` is an XML **attribute** on `<master>` and `<release>`, but a child **element** on
  `<artist>` and `<label>`. Easy to get backwards when adding a new field.
- `labels.xml`: a `parentLabel` element is documented by Discogs for sublabel entries but
  wasn't observed in the samples pulled so far — `model.Label.ParentLabel` is modeled as
  `*LabelRef` defensively. If you're touching label parsing, verify against a larger sample
  rather than trusting its absence.
- `images` on `<release>` is intentionally unmodeled — `encoding/xml` silently ignores
  unmapped elements, so this is a no-op omission, not a bug.
- `data.discogs.com` ignores HTTP `Range` requests (confirmed: a `curl -r 0-300000` against a
  multi-GB file returned the full 200 OK body, not a 206 partial). Don't build a resume-via-
  Range feature; `internal/fetch/download.go`'s approach (download fully to `.part`, verify
  checksum, rename) is the right model — re-download from scratch on any mismatch.
- The site's "directory listing" is a custom CloudFront-style proxy
  (`?prefix=data/<year>/` to list, `?download=data/<year>/<file>` to stream), not a raw S3
  bucket listing — `internal/fetch/discover.go` scrapes this HTML, it doesn't call S3 APIs.

## Scale, for context when changing the hot path

Loading the 2026-11-01 batch produced: 10.3M artists, 2.4M labels, 2.6M masters, 19.6M
releases, and roughly 470M total child rows (tracklists, credits, identifiers, etc.) in a
~40GB SQLite file. `internal/load`/`internal/store` sit on the hot path for all of that —
profile before "optimizing," and re-run a real `fetch`+`load` (not just `go test`) to validate
any performance change, since the fixture tests are deliberately tiny.

## Working conventions observed in this repo

- **No comments explaining *what* code does; comments only for non-obvious *why*** (a hidden
  constraint, a workaround, a surprising real-data shape). Several comments in this codebase
  exist specifically to flag the gotchas above — follow that pattern, don't strip them as
  "unnecessary."
- **Validate real schema/behavior claims against live data before committing to them in code
  or docs.** Several design choices here (checksum format, HTML proxy behavior, Range support,
  exact XML field shapes) were confirmed by `curl`/live fetch rather than assumed from memory
  of the Discogs API docs, and that caught real discrepancies (the duplicate-URL case above).
  When extending parsing for a field not yet covered, pull a live sample rather than guessing.
- **Prefer a strong recommended default over an open-ended question** when proposing
  implementation choices — this repo's early decisions (stdlib `flag` over cobra,
  `mattn/go-sqlite3` over a pure-Go driver, fully normalized schema over JSON blobs) were all
  made by presenting one clearly-recommended option plus an alternative, not a blank "what do
  you want" question.
- **`datadump/` and `*.db*` are gitignored.** Never try to commit downloaded dumps or the
  loaded database. Don't assume disk usage is free either — a full load is currently ~40GB+
  the ~12GB of archived `.xml.gz` source files.
- **Multiple sessions may be working on this repo concurrently.** `cmd/xmlschema`, the
  `release_search`/`master_search` views, and some of the name indexes in `schema.sql` were
  added by a parallel session during this project's initial build, not sequentially. If you
  find code you don't recognize, it's more likely a concurrent collaborator than a mistake —
  check git history/timestamps before assuming it needs fixing, and don't silently overwrite
  it.
- **Git**: never commit unless explicitly asked, even after a large/successful change. This
  repo's history so far reflects the human committing their own work directly.
