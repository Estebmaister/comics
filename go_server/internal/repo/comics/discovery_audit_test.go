package comics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"

	"comics/internal/discovery/repair"
	"comics/internal/identity"

	_ "modernc.org/sqlite"
)

func TestDiscoveryAuditIsReadOnly(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := createDiscoveryAuditFixture(db); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	before, err := sqliteSchemaAndDataHash(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	repo, err := NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := repo.AuditDiscoveryIntegrity(context.Background())
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}

	after, err := sqliteSchemaAndDataHash(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("audit mutated database: before=%s after=%s", before, after)
	}

	if audit.Summary.TotalComics != 5 ||
		audit.Summary.ActiveComics != 4 ||
		audit.Summary.KeyRows != 3 ||
		audit.Summary.DeletedKeyRows != 1 ||
		audit.Summary.OrphanKeyRows != 1 ||
		audit.Summary.KeylessActive != 3 {
		t.Fatalf("unexpected summary: %+v", audit.Summary)
	}
	if audit.Summary.ActiveConflicts != 1 ||
		audit.Summary.Restorable != 1 ||
		audit.Summary.BlockedNoKey != 1 {
		t.Fatalf("unexpected classifications: %+v", audit.Summary)
	}
	if audit.DatabaseFingerprint == "" {
		t.Fatal("expected fingerprint")
	}
}

func createDiscoveryAuditFixture(db *sql.DB) error {
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
	`)
	if err != nil {
		return err
	}

	ownerKeys := identity.DiscoveryTitleKeyPairsForTitles([]string{"Executioner"}, 3)
	if len(ownerKeys) != 1 {
		return fmt.Errorf("unexpected keys: %#v", ownerKeys)
	}
	ownerKey := ownerKeys[0]
	deletedKeys := identity.DiscoveryTitleKeyPairsForTitles([]string{"Deleted hero"}, 3)
	if len(deletedKeys) != 1 {
		return fmt.Errorf("unexpected deleted keys: %#v", deletedKeys)
	}
	deletedKey := deletedKeys[0]

	_, err = db.Exec(`
		INSERT INTO comics (
			id, titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap, rating, deleted, cover_visible
		) VALUES
			(1, 'Executioner', 10, '', 0, 3, 0, '', '', '', '', 1, 0, 0, 0, 1),
			(2, 'Executioner', 5, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1),
			(3, '英雄', 1, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1),
			(4, '|', 0, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1),
			(5, 'Deleted hero', 1, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 1, 1);
		INSERT INTO comic_title_keys (comic_id, scope, title_key) VALUES
			(1, ?, ?),
			(5, ?, ?),
			(999, 'series', 'orphan');
	`, ownerKey.Scope, ownerKey.TitleKey, deletedKey.Scope, deletedKey.TitleKey)
	return err
}

func sqliteSchemaAndDataHash(dbPath string) (string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT type, name, tbl_name, sql FROM sqlite_master
		UNION ALL
		SELECT 'comic', CAST(id AS TEXT), titles, CAST(deleted AS TEXT) FROM comics
		UNION ALL
		SELECT 'key', CAST(comic_id AS TEXT), scope, title_key FROM comic_title_keys
		ORDER BY 1, 2, 3, 4`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	hash := sha256.New()
	for rows.Next() {
		var a, b, c string
		var d sql.NullString
		if err := rows.Scan(&a, &b, &c, &d); err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%q\x00%q\x00%q\x00%q\n", a, b, c, d.String)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func TestKeylessRecordClassifications(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "classify.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := createDiscoveryAuditFixture(db); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	repo, err := NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := repo.AuditDiscoveryIntegrity(context.Background())
	_ = repo.Close()
	if err != nil {
		t.Fatal(err)
	}

	byID := map[int]repair.KeylessRecord{}
	for _, record := range audit.Records {
		byID[record.Comic.ID] = record
	}
	if byID[2].Classification != repair.ClassificationActiveConflict {
		t.Fatalf("comic 2: %#v", byID[2])
	}
	if byID[3].Classification != repair.ClassificationRestorable {
		t.Fatalf("comic 3: %#v", byID[3])
	}
	if byID[4].Classification != repair.ClassificationBlockedNoKey {
		t.Fatalf("comic 4: %#v", byID[4])
	}
}
