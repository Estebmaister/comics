package comics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const sqliteReadPoolSize = 4

func configureSQLiteWriter(db *sql.DB) error {
	// SQLite allows one writer; a single connection avoids "database is locked" under scrape + API load.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return ensureSQLiteHealthy(db)
}

func configureSQLiteReader(db *sql.DB) error {
	db.SetMaxOpenConns(sqliteReadPoolSize)
	db.SetMaxIdleConns(sqliteReadPoolSize)
	db.SetConnMaxLifetime(0)
	return nil
}

func configureSQLiteDB(db *sql.DB) error {
	return configureSQLiteWriter(db)
}

func ensureSQLiteHealthy(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ok, detail := sqliteIntegrityOK(ctx, db)
	if ok {
		return nil
	}

	for _, statement := range []string{
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"REINDEX",
		"VACUUM",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf(
				"sqlite integrity check failed (%s); repair %q failed: %w (try: make db-restore)",
				detail,
				statement,
				err,
			)
		}
	}

	ok, detail = sqliteIntegrityOK(ctx, db)
	if ok {
		return nil
	}
	return fmt.Errorf(
		"sqlite database is still unhealthy after REINDEX/VACUUM: %s (try: make db-restore)",
		detail,
	)
}

func sqliteIntegrityOK(ctx context.Context, db *sql.DB) (bool, string) {
	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return false, err.Error()
	}
	defer rows.Close()

	lines := []string{}
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return false, err.Error()
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return false, err.Error()
	}
	if len(lines) == 1 && lines[0] == "ok" {
		return true, "ok"
	}
	return false, strings.Join(lines, "; ")
}
