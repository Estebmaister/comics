-- +migrate Up
ALTER TABLE comics
ADD COLUMN IF NOT EXISTS cover_visible BOOLEAN NOT NULL DEFAULT true;

-- +migrate Down
ALTER TABLE comics
DROP COLUMN IF EXISTS cover_visible;
