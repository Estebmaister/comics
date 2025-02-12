-- +migrate Up
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL,
    applied_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (version)
);

CREATE TABLE IF NOT EXISTS comics (
    id            SERIAL          PRIMARY KEY,
    titles        VARCHAR(255)[]  NOT NULL,
    author        VARCHAR(255)    NULL,
    description   VARCHAR(2083)   NULL,
    cover         VARCHAR(2083)   NULL,
    published_in  INTEGER[]       NOT NULL,
    genres        INTEGER[]       NOT NULL,
    com_type      INTEGER         NOT NULL DEFAULT 0,
    status        INTEGER         NOT NULL DEFAULT 0,
    rating        INTEGER         NOT NULL DEFAULT 0,
    current_chap  INTEGER         NOT NULL DEFAULT 0,
    viewed_chap   INTEGER         NOT NULL DEFAULT 0,
    track         BOOLEAN         NOT NULL DEFAULT false,
    deleted       BOOLEAN         NOT NULL DEFAULT false,
    last_update   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comics_track ON comics(track) WHERE NOT deleted;
CREATE INDEX IF NOT EXISTS idx_comics_titles ON comics USING gin(titles);
CREATE INDEX IF NOT EXISTS idx_comics_last_update ON comics(last_update DESC);

-- +migrate Down
DROP TABLE IF EXISTS comics;
DROP TABLE IF EXISTS schema_migrations;
