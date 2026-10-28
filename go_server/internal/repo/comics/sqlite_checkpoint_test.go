package comics

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCheckpointSQLiteWAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY); INSERT INTO t VALUES (1);`); err != nil {
		t.Fatal(err)
	}
	if err := CheckpointSQLiteWAL(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}
