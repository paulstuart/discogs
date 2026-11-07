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
