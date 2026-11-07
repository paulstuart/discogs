-- Schema for the Discogs dump SQLite store. No FOREIGN KEY constraints are
-- enforced: dump files are loaded independently and can contain forward
-- references (e.g. a label's sublabel id that hasn't been loaded yet).
-- FK-shaped columns exist for joins/indexes only.

CREATE TABLE IF NOT EXISTS artists (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    real_name TEXT,
    profile TEXT,
    data_quality TEXT
);
CREATE TABLE IF NOT EXISTS artist_urls (
    artist_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    PRIMARY KEY (artist_id, url)
);
CREATE TABLE IF NOT EXISTS artist_name_variations (
    artist_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (artist_id, name)
);
CREATE TABLE IF NOT EXISTS artist_aliases (
    artist_id INTEGER NOT NULL,
    alias_id INTEGER NOT NULL,
    alias_name TEXT,
    PRIMARY KEY (artist_id, alias_id)
);
CREATE TABLE IF NOT EXISTS artist_members (
    artist_id INTEGER NOT NULL,
    member_id INTEGER NOT NULL,
    member_name TEXT,
    PRIMARY KEY (artist_id, member_id)
);

CREATE TABLE IF NOT EXISTS labels (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    contact_info TEXT,
    profile TEXT,
    data_quality TEXT,
    parent_label_id INTEGER,
    parent_label_name TEXT
);
CREATE TABLE IF NOT EXISTS label_urls (
    label_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    PRIMARY KEY (label_id, url)
);
CREATE TABLE IF NOT EXISTS label_sublabels (
    label_id INTEGER NOT NULL,
    sublabel_id INTEGER NOT NULL,
    sublabel_name TEXT,
    PRIMARY KEY (label_id, sublabel_id)
);

CREATE TABLE IF NOT EXISTS masters (
    id INTEGER PRIMARY KEY,
    main_release INTEGER,
    year INTEGER,
    title TEXT NOT NULL,
    data_quality TEXT
);
CREATE TABLE IF NOT EXISTS master_genres (
    master_id INTEGER NOT NULL,
    genre TEXT NOT NULL,
    PRIMARY KEY (master_id, genre)
);
CREATE TABLE IF NOT EXISTS master_styles (
    master_id INTEGER NOT NULL,
    style TEXT NOT NULL,
    PRIMARY KEY (master_id, style)
);
CREATE TABLE IF NOT EXISTS master_artists (
    master_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    artist_id INTEGER,
    name TEXT,
    anv TEXT,
    join_str TEXT,
    role TEXT,
    PRIMARY KEY (master_id, seq)
);
CREATE TABLE IF NOT EXISTS master_videos (
    master_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    src TEXT,
    duration INTEGER,
    embed INTEGER,
    title TEXT,
    description TEXT,
    PRIMARY KEY (master_id, seq)
);

CREATE TABLE IF NOT EXISTS releases (
    id INTEGER PRIMARY KEY,
    status TEXT,
    title TEXT NOT NULL,
    country TEXT,
    released TEXT,
    notes TEXT,
    master_id INTEGER,
    is_main_release INTEGER,
    data_quality TEXT
);
CREATE TABLE IF NOT EXISTS release_artists (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    artist_id INTEGER,
    name TEXT,
    anv TEXT,
    join_str TEXT,
    role TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_extraartists (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    artist_id INTEGER,
    name TEXT,
    anv TEXT,
    role TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_labels (
    release_id INTEGER NOT NULL,
    label_id INTEGER,
    label_name TEXT,
    catno TEXT,
    PRIMARY KEY (release_id, label_id, catno)
);
CREATE TABLE IF NOT EXISTS release_formats (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    name TEXT,
    qty TEXT,
    text TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_format_descriptions (
    release_id INTEGER NOT NULL,
    format_seq INTEGER NOT NULL,
    description TEXT NOT NULL,
    PRIMARY KEY (release_id, format_seq, description)
);
CREATE TABLE IF NOT EXISTS release_genres (
    release_id INTEGER NOT NULL,
    genre TEXT NOT NULL,
    PRIMARY KEY (release_id, genre)
);
CREATE TABLE IF NOT EXISTS release_styles (
    release_id INTEGER NOT NULL,
    style TEXT NOT NULL,
    PRIMARY KEY (release_id, style)
);
CREATE TABLE IF NOT EXISTS release_tracks (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    position TEXT,
    title TEXT,
    duration TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_track_artists (
    release_id INTEGER NOT NULL,
    track_seq INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    artist_id INTEGER,
    name TEXT,
    anv TEXT,
    join_str TEXT,
    role TEXT,
    PRIMARY KEY (release_id, track_seq, seq)
);
CREATE TABLE IF NOT EXISTS release_identifiers (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    type TEXT,
    description TEXT,
    value TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_videos (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    src TEXT,
    duration INTEGER,
    embed INTEGER,
    title TEXT,
    description TEXT,
    PRIMARY KEY (release_id, seq)
);
CREATE TABLE IF NOT EXISTS release_companies (
    release_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,
    company_id INTEGER,
    name TEXT,
    entity_type TEXT,
    entity_type_name TEXT,
    PRIMARY KEY (release_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_releases_master_id ON releases(master_id);
CREATE INDEX IF NOT EXISTS idx_master_artists_artist_id ON master_artists(artist_id);
CREATE INDEX IF NOT EXISTS idx_release_artists_artist_id ON release_artists(artist_id);
CREATE INDEX IF NOT EXISTS idx_release_labels_label_id ON release_labels(label_id);

-- Name indexes, so lookups/searches by artist name (either the canonical
-- artists row or the as-credited name on a release/master) can use an
-- index instead of a full scan. LIKE 'prefix%' can use a NOCASE index;
-- a leading-wildcard search ('%name%') still can't.
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_release_artists_name ON release_artists(name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_master_artists_name ON master_artists(name COLLATE NOCASE);

-- release_search/master_search flatten the id-based artist credit tables
-- into one row per (release, credited artist) / (master, credited artist),
-- resolving artist_name to the canonical artists.name when the credit is
-- linked by id and falling back to the as-credited name otherwise. This is
-- the shape most "find X by artist name" queries want, e.g.:
--   SELECT DISTINCT release_id, release_title FROM release_search
--   WHERE artist_name LIKE 'Miles Davis%';
CREATE VIEW IF NOT EXISTS release_search AS
SELECT
    r.id                      AS release_id,
    r.title                   AS release_title,
    r.released,
    r.country,
    r.data_quality,
    ra.seq,
    ra.artist_id,
    coalesce(a.name, ra.name) AS artist_name,
    ra.role
FROM releases r
JOIN release_artists ra ON ra.release_id = r.id
LEFT JOIN artists a ON a.id = ra.artist_id;

CREATE VIEW IF NOT EXISTS master_search AS
SELECT
    m.id                      AS master_id,
    m.title                   AS master_title,
    m.year,
    m.data_quality,
    ma.seq,
    ma.artist_id,
    coalesce(a.name, ma.name) AS artist_name,
    ma.role
FROM masters m
JOIN master_artists ma ON ma.master_id = m.id
LEFT JOIN artists a ON a.id = ma.artist_id;
