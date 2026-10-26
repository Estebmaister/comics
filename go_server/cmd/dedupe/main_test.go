package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	comicrepo "comics/internal/repo/comics"
	"comics/internal/identity"

	_ "modernc.org/sqlite"
)

func TestDedupeIntegrityAuditFixture(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dedupe-audit.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := seedCollisionRow(db); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	repo, err := comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := repo.AuditDiscoveryIntegrity(context.Background())
	_ = repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	if audit.Summary.TotalComics != 2 || audit.Summary.KeyRows != 1 {
		t.Fatalf("unexpected audit: %+v", audit.Summary)
	}

	repo, err = comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	collisions, err := repo.ListDiscoveryKeyCollisions(context.Background())
	_ = repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlan(collisions)
	if plan.GroupCount != 0 {
		t.Fatalf("expected no collisions in keyless-only fixture, got %d", plan.GroupCount)
	}
}

func seedCollisionRow(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE comics (
			id INTEGER NOT NULL PRIMARY KEY,
			titles TEXT NOT NULL,
			current_chap INTEGER NOT NULL DEFAULT 0,
			cover TEXT NOT NULL DEFAULT '',
			last_update INTEGER NOT NULL DEFAULT 0,
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
			cover_visible BOOLEAN NOT NULL DEFAULT 1
		);
		CREATE TABLE comic_title_keys (
			comic_id INTEGER NOT NULL,
			scope TEXT NOT NULL,
			title_key TEXT NOT NULL,
			PRIMARY KEY (scope, title_key)
		);
		INSERT INTO comics (id, titles, last_update) VALUES (1, 'Alpha', 0), (2, 'Beta', 0);
	`)
	if err != nil {
		return err
	}
	keys := identity.DiscoveryTitleKeyPairsForTitles([]string{"Alpha"}, 3)
	if len(keys) != 1 {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO comic_title_keys (comic_id, scope, title_key) VALUES (1, ?, ?)`,
		keys[0].Scope, keys[0].TitleKey,
	)
	return err
}
