-- Indexes and views. Applied only by SQLiteStore.BuildIndexes (the
-- `discogs index` subcommand), run once after all desired `load --kinds`
-- invocations are done -- not at Open()/load time, so bulk inserts never
-- pay index-maintenance cost mid-load.

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
-- country is resolved through countries so the view's shape stays a plain
-- human-readable column despite releases storing country_id underneath.
CREATE VIEW IF NOT EXISTS release_search AS
SELECT
    r.id                      AS release_id,
    r.title                   AS release_title,
    r.released,
    co.name                   AS country,
    r.data_quality,
    ra.seq,
    ra.artist_id,
    coalesce(a.name, ra.name) AS artist_name,
    ra.role
FROM releases r
JOIN release_artists ra ON ra.release_id = r.id
LEFT JOIN artists a ON a.id = ra.artist_id
LEFT JOIN countries co ON co.id = r.country_id;

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

-- Flesh-out views: resolve the normalized lookup tables back to
-- human-readable names, one row per (parent, value) pair -- the same
-- shape the old denormalized tables had, just computed via a join instead
-- of stored redundantly.
CREATE VIEW IF NOT EXISTS release_genres_named AS
SELECT rg.release_id, g.name AS genre
FROM release_genres rg JOIN genres g ON g.id = rg.genre_id;

CREATE VIEW IF NOT EXISTS release_styles_named AS
SELECT rs.release_id, s.name AS style
FROM release_styles rs JOIN styles s ON s.id = rs.style_id;

CREATE VIEW IF NOT EXISTS master_genres_named AS
SELECT mg.master_id, g.name AS genre
FROM master_genres mg JOIN genres g ON g.id = mg.genre_id;

CREATE VIEW IF NOT EXISTS master_styles_named AS
SELECT ms.master_id, s.name AS style
FROM master_styles ms JOIN styles s ON s.id = ms.style_id;

CREATE VIEW IF NOT EXISTS release_identifiers_named AS
SELECT ri.release_id, ri.seq, it.name AS type, ri.description, ri.value
FROM release_identifiers ri LEFT JOIN identifier_types it ON it.id = ri.type_id;

CREATE VIEW IF NOT EXISTS release_companies_named AS
SELECT rc.release_id, rc.seq, rc.company_id, rc.name, cr.name AS role
FROM release_companies rc LEFT JOIN company_roles cr ON cr.id = rc.entity_type;
