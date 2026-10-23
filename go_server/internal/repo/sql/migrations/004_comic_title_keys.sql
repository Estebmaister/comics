-- +migrate Up
CREATE TABLE IF NOT EXISTS comic_title_keys (
    comic_id  INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    scope     TEXT    NOT NULL,
    title_key TEXT    NOT NULL,
    PRIMARY KEY (scope, title_key)
);

CREATE INDEX IF NOT EXISTS idx_comic_title_keys_comic_id ON comic_title_keys(comic_id);

-- +migrate Down
DROP TABLE IF EXISTS comic_title_keys;
