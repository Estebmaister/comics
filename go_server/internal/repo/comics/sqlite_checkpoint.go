package comics

import (
	"context"
	"database/sql"
	"fmt"
)

// CheckpointSQLiteWAL flushes the write-ahead log into the main database file.
// TRUNCATE resets the WAL file after a successful checkpoint so git and backups
// see updates in comics.db.
func CheckpointSQLiteWAL(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	var busy, walFrames, checkpointed int
	err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &walFrames, &checkpointed)
	if err != nil {
		return fmt.Errorf("wal_checkpoint: %w", err)
	}
	if busy != 0 {
		return fmt.Errorf("wal_checkpoint: database busy (busy=%d)", busy)
	}
	return nil
}

func (r *SQLComicRepository) CheckpointWAL(ctx context.Context) error {
	if r.dialect != dialectSQLite || !r.ownsDB || r.db == nil {
		return nil
	}
	return CheckpointSQLiteWAL(ctx, r.db)
}
