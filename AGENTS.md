# AGENTS.md

Guidance for AI agents (and humans) developing this repo. This project downloads Discogs'
monthly XML data dumps and loads them into a database. Read this before making structural
changes — it captures decisions and real-data gotchas that aren't obvious from the code alone.

## What this is

```
discogs fetch [--dir datadump] [--year YYYY] [--date YYYYMMDD]
discogs load  [--dir datadump] --date YYYYMMDD [--db sqlite:discogs.db] [--kinds artists,labels,masters,releases]
discogs index [--db sqlite:discogs.db]
discogs dump  [--dir datadump] --date YYYYMMDD --kind artists|labels|masters|releases [--out file.jsonl]
```

`fetch`, `load`, and `index` are deliberately separate invocations. `fetch` downloads large
files from `https://data.discogs.com/` into `<dir>/archive/<YYYY>/`; `load` streams an
already-archived batch into a database; `index` builds every index/view in one pass. None of
these trigger each other automatically — these files are large (releases.xml.gz alone is
~11GB compressed, ~100GB+ decompressed) and downloads should stay deliberate and infrequent,
and index/view creation should happen exactly once, after all desired `load` calls are done
(see the schema section below for why).

`dump` is a fourth, independent path: it streams one kind's archived XML straight to
newline-delimited JSON (one `model.<Kind>` record per line), bypassing `store.Store` entirely.
It exists for ad hoc exploration (`discogs dump --date ... --kind releases | jq ...`, or
loading the JSONL into duckdb/sqlite3 directly) without committing to a full `load`. It's
JSON-only, not TSV/CSV — `Master`/`Release` have deeply nested repeating fields (tracklist
with per-track artist credits, identifiers, companies, formats...), so a flat TSV would mean
splitting into many child files per kind, basically re-deriving the SQL schema as flat files.
JSON needs no flattening since the structs already carry `json` tags for exactly this.

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
internal/dump/           gzip -> xmlstream -> json.Encoder, straight to stdout/file -- no
                         store.Store involved, used by the `discogs dump` subcommand
internal/store/          Store interface + DSN dispatch (store.Open("sqlite:path.db"))
internal/store/sqlite/   the only implementation today; schema.sql + indexes.sql (both
                         go:embed'd, see below), one file per entity (artists.go, labels.go,
                         masters.go, releases.go), each exposing PrepareX/PutX(/FinishX)
```

`internal/model` is the single struct set for all four entities — don't create parallel
"XML struct" vs "DB struct" pairs. If a new output format is needed (e.g. real JSON export),
add it as a serialization of these structs, not a new model package.

`store.Store` is the extensibility seam for other databases. Adding a backend means a new
`internal/store/<name>` package implementing the interface (a `Prepare`/`Put`/(`Finish`) triplet
per entity, plus `BuildIndexes`/`Close`), plus one `case` in `store.Open`'s DSN dispatch — no
changes to `internal/load`.

## Schema design (why it looks the way it does)

- **No FOREIGN KEY constraints.** The four dump files are loaded independently and contain
  forward references (e.g. a label's sublabel ID that hasn't been loaded yet, or a release's
  `master_id` pointing at a master you may never load). FK-shaped columns exist for joins and
  are indexed, just not enforced.
- **Full replace, not merge.** Loading a kind is `PrepareX` (wipe every table for that kind),
  then `PutX` once per decoded batch doing plain inserts, then `FinishX` (masters/releases
  only -- see the vocabulary bullet below). `internal/load`'s `loadArtists`/`loadLabels`/
  `loadMasters`/`loadReleases` call these in sequence; `store.Store` callers shouldn't call
  `PutX` without a preceding `PrepareX` in the same load. This is deliberate, not an
  oversight: Discogs dumps are complete monthly snapshots, not deltas, so there's never a case
  where you want to merge two different dumps' data for the same kind into one DB -- you just
  want the latest one. Replacing the whole kind's tables on every load also means a shrunk
  child list (fewer URLs, fewer tracks) is reflected for free, since the old rows are gone
  rather than merged into -- no per-record `DELETE` needed. (An earlier version of this schema
  used `INSERT ... ON CONFLICT(id) DO UPDATE` on parent tables plus a per-record `DELETE FROM
  child WHERE parent_id = ?` before every child insert, to handle exactly this shrinking-list
  case under a merge model. That's gone now that loads fully replace instead of merging.)
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
- **`schema.sql` is tables only; `indexes.sql` is indexes + views, applied separately.**
  `sqlite.Open()` applies only `schema.sql`. `SQLiteStore.BuildIndexes` (the `discogs index`
  subcommand) applies `indexes.sql`. This split exists so bulk inserts during `load` never pay
  index-maintenance cost — run `index` once, explicitly, after all desired `--kinds` loads
  across however many invocations are done. Don't make `load` call `BuildIndexes`
  automatically: if loads happen in separate invocations (as they did here — labels+artists,
  then masters, then releases, as three separate commands), auto-building indexes after the
  first one would reintroduce incremental index-maintenance cost on every load after that.
- **Free-text vocabulary is normalized, not repeated, via a raw-table + bulk-SQL pass, not a
  per-row Go lookup.** `genre`, `style`, `country`, and identifier `type` used to be stored as
  raw text in every row that referenced them (tens of millions of rows for the big ones).
  They're now lookup tables (`genres`, `styles`, `countries`, `identifier_types`: `(id INTEGER
  PRIMARY KEY, name TEXT NOT NULL UNIQUE)`). `PutMasters`/`PutReleases` write the raw text
  into staging tables (`master_genres_raw`, `release_genres_raw`, `releases_raw.country`,
  `release_identifiers_raw.type`, etc. — created by `PrepareMasters`/`PrepareReleases`,
  dropped by `FinishMasters`/`FinishReleases`) instead of resolving an id per row during the
  hot insert path. `FinishMasters`/`FinishReleases` then do the resolution once, in bulk, with
  plain SQL: `INSERT INTO genres (name) SELECT DISTINCT genre FROM ... ON CONFLICT(name) DO
  NOTHING`, then `INSERT ... SELECT ... JOIN` to populate the final join table from the
  staging table. Note the `WHERE` clause on those `SELECT DISTINCT` statements isn't optional
  filtering — SQLite's parser treats a bare `SELECT ... FROM table ON CONFLICT ...` as
  ambiguous with join syntax and rejects it; a `WHERE` clause (even a no-op `col IS NOT NULL`
  on a `NOT NULL` column) disambiguates it. Confirmed by hitting this live against SQLite
  3.54.1, not assumed from docs. `company_roles` is the one exception: Discogs' own
  `<entity_type>` numeric code (e.g. `23` → "Recorded At") is already a stable key, so
  `company_roles.id` is that code directly, not a synthesized surrogate — no staging table
  needed there, just `INSERT OR IGNORE INTO company_roles (id, name) VALUES (?, ?)` straight
  from `PutReleases`. If you normalize another free-text column, check whether the source XML
  already carries a natural numeric id (use the `company_roles` pattern) before reaching for
  the raw-table-plus-bulk-SQL pattern.
- **`..._named` views resolve ids back to text for querying.** `release_genres_named`,
  `release_styles_named`, `master_genres_named`, `master_styles_named`,
  `release_identifiers_named`, `release_companies_named` all live in `indexes.sql` and give
  back the same shape the old denormalized tables had. `release_search` (added by a concurrent
  session, see below) is updated to join `countries` so it still exposes a plain `country` text
  column despite `releases` storing `country_id` underneath — if you touch `releases.country_id`
  again, check `release_search` doesn't need a matching update.

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
releases, and roughly 470M total child rows (tracklists, credits, identifiers, etc.).
`internal/load`/`internal/store` sit on the hot path for all of that — profile before
"optimizing," and re-run a real `fetch`+`load`+`index` (not just `go test`) to validate any
performance change, since the fixture tests are deliberately tiny.

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
  loaded database. Don't assume disk usage is free either — a full load is tens of GB on top
  of the ~12GB of archived `.xml.gz` source files.
- **Multiple sessions may be working on this repo concurrently.** `cmd/xmlschema`, the
  `release_search`/`master_search` views, and some of the name indexes in `schema.sql` were
  added by a parallel session during this project's initial build, not sequentially. If you
  find code you don't recognize, it's more likely a concurrent collaborator than a mistake —
  check git history/timestamps before assuming it needs fixing, and don't silently overwrite
  it.
- **Git**: never commit unless explicitly asked, even after a large/successful change. This
  repo's history so far reflects the human committing their own work directly.
