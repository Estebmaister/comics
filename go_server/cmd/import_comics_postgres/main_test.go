package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadSourceComicsCountsSQLiteTruth(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comics.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE comics (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			titles TEXT NOT NULL,
			current_chap INTEGER NOT NULL DEFAULT 0,
			cover TEXT NOT NULL DEFAULT '',
			last_update INTEGER NOT NULL,
			com_type INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 0,
			published_in TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT '',
			track BOOLEAN NOT NULL DEFAULT 0,
			viewed_chap INTEGER NOT NULL DEFAULT 0,
			rating INTEGER NOT NULL DEFAULT 0,
			deleted BOOLEAN NOT NULL DEFAULT 0,
			identity_key TEXT NOT NULL DEFAULT '',
			cover_visible BOOLEAN NOT NULL DEFAULT 1
		);
		INSERT INTO comics (
			id, titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap,
			rating, deleted, identity_key, cover_visible
		) VALUES
			(10, 'Alpha', 3, '', 1, 1, 2, '1', '2', '', '', 1, 2, 0, 0, '1:alpha', 1),
			(11, 'Beta', 1, '', 2, 1, 2, '1', '2', '', '', 0, 1, 0, 1, '1:beta', 0);
	`)
	if err != nil {
		t.Fatal(err)
	}

	comics, counts, err := readSourceComics(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(comics) != 2 || counts.Total != 2 || counts.Active != 1 ||
		counts.Tracked != 1 || counts.CoverInvisible != 1 {
		t.Fatalf("unexpected import counts: len=%d counts=%+v", len(comics), counts)
	}
	if comics[0].ID != 10 || comics[1].ID != 11 {
		t.Fatalf("expected source IDs to be preserved, got %#v", comics)
	}
}

func TestRequireSSLModeAppendsCorrectSeparator(t *testing.T) {
	tests := map[string]string{
		"postgres://host/db":                 "postgres://host/db?sslmode=require",
		"postgres://host/db?connect=fast":    "postgres://host/db?connect=fast&sslmode=require",
		"postgres://host/db?sslmode=require": "postgres://host/db?sslmode=require",
	}
	for input, want := range tests {
		if got := requireSSLMode(input); got != want {
			t.Fatalf("requireSSLMode(%q) = %q, want %q", input, got, want)
		}
	}
}
