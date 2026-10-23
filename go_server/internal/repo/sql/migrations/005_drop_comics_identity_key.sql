-- +migrate Up
DROP INDEX IF EXISTS uq_comics_identity_key;
DROP INDEX IF EXISTS idx_comics_identity_key;
ALTER TABLE comics DROP COLUMN IF EXISTS identity_key;

-- +migrate Down
ALTER TABLE comics ADD COLUMN IF NOT EXISTS identity_key TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_comics_identity_key ON comics(identity_key);
CREATE UNIQUE INDEX IF NOT EXISTS uq_comics_identity_key
    ON comics(identity_key)
    WHERE deleted = false AND identity_key <> '';
