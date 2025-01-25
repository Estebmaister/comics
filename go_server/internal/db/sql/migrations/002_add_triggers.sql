-- +migrate Up
CREATE OR REPLACE FUNCTION update_last_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_update = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_comics_last_update
    BEFORE UPDATE ON comics
    FOR EACH ROW
    EXECUTE FUNCTION update_last_update();

-- +migrate Down
DROP TRIGGER IF EXISTS update_comics_last_update ON comics;
DROP FUNCTION IF EXISTS update_last_update();
