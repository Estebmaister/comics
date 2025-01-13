-- +migrate Up
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_comics_updated_at
    BEFORE UPDATE ON comics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- +migrate Down
DROP TRIGGER IF EXISTS update_comics_updated_at ON comics;
DROP FUNCTION IF EXISTS update_updated_at();
