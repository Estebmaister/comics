-- +migrate Up
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL,
    applied_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (version)
);

CREATE TABLE IF NOT EXISTS comics (
    id SERIAL PRIMARY KEY,
    titles TEXT[] NOT NULL,
    author TEXT,
    cover TEXT,
    description TEXT,

    type INTEGER NOT NULL,
    status INTEGER NOT NULL,
    rating INTEGER NOT NULL,
    publishers INTEGER[],
    genres INTEGER[],

    current_chap INTEGER NOT NULL DEFAULT 0,
    viewed_chap INTEGER NOT NULL DEFAULT 0,

    track BOOLEAN NOT NULL DEFAULT false,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comics_track ON comics(track) WHERE NOT deleted;
CREATE INDEX IF NOT EXISTS idx_comics_titles ON comics USING gin(titles);
CREATE INDEX IF NOT EXISTS idx_comics_updated_at ON comics(updated_at DESC);

-- +migrate Down
DROP TABLE IF EXISTS comics;
DROP TABLE IF EXISTS schema_migrations;
