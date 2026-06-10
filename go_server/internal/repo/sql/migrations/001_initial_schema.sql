-- +migrate Up
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL,
    applied_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (version)
);

CREATE TABLE IF NOT EXISTS comics (
    id            SERIAL  PRIMARY KEY,
    titles        TEXT    NOT NULL,
    author        TEXT    NOT NULL DEFAULT '',
    description   TEXT    NOT NULL DEFAULT '',
    cover         TEXT    NOT NULL DEFAULT '',
    cover_visible BOOLEAN NOT NULL DEFAULT true,
    published_in  TEXT    NOT NULL DEFAULT '',
    genres        TEXT    NOT NULL DEFAULT '',
    identity_key  TEXT    NOT NULL DEFAULT '',
    com_type      INTEGER NOT NULL DEFAULT 0,
    status        INTEGER NOT NULL DEFAULT 0,
    rating        INTEGER NOT NULL DEFAULT 0,
    current_chap  INTEGER NOT NULL DEFAULT 0,
    viewed_chap   INTEGER NOT NULL DEFAULT 0,
    track         BOOLEAN NOT NULL DEFAULT false,
    deleted       BOOLEAN NOT NULL DEFAULT false,
    last_update   BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM now())::BIGINT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_comics_identity_key
    ON comics(identity_key)
    WHERE deleted = false AND identity_key <> '';
CREATE INDEX IF NOT EXISTS idx_comics_identity_key ON comics(identity_key);
CREATE INDEX IF NOT EXISTS idx_comics_active_update ON comics(deleted, last_update DESC, id);
CREATE INDEX IF NOT EXISTS idx_comics_active_track_update ON comics(deleted, track, last_update DESC, id);

-- +migrate Down
DROP TABLE IF EXISTS comics;
DROP TABLE IF EXISTS schema_migrations;
