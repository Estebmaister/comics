package main

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"comics/internal/discovery/repair"
	"comics/internal/identity"
	comicrepo "comics/internal/repo/comics"

	_ "modernc.org/sqlite"
)

func TestApplyDiscoveryRepairRestoresKeyless(t *testing.T) {
	dbPath, backupPath := seedRepairDB(t)
	ctx := context.Background()

	readRepo, err := comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := readRepo.AuditDiscoveryIntegrity(ctx)
	_ = readRepo.Close()
	if err != nil {
		t.Fatal(err)
	}
	plan := repair.BuildPlan(audit, time.Now().UTC())
	plan.Groups = filterGroup(plan.Groups, "keyless:3")
	plan.Groups[0].Review = repair.Review{
		Approved:   true,
		Decision:   repair.DecisionRestoreKeys,
		Reviewer:   "tester",
		ReviewedAt: time.Now().UTC().Format(time.RFC3339),
	}
	planPath := filepath.Join(t.TempDir(), "plan.json")
	if err := repair.WritePlanFile(planPath, plan); err != nil {
		t.Fatal(err)
	}

	before, after, err := applyDiscoveryRepair(ctx, discoveryApplyOptions{
		DBPath:          dbPath,
		PlanPath:        planPath,
		BackupPath:      backupPath,
		GroupID:         "keyless:3",
		ConfirmReviewed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if before.KeylessActive != 3 || after.KeylessActive != 2 {
		t.Fatalf("expected keyless 3 -> 2, got %d -> %d", before.KeylessActive, after.KeylessActive)
	}

	restoreDB(t, dbPath, backupPath)
	readRepo, err = comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := readRepo.AuditDiscoveryIntegrity(ctx)
	_ = readRepo.Close()
	if err != nil {
		t.Fatal(err)
	}
	if restored.DatabaseFingerprint != audit.DatabaseFingerprint {
		t.Fatalf("fingerprint changed after restore")
	}
}

func TestApplyDiscoveryRepairRejectsStalePlan(t *testing.T) {
	dbPath, backupPath := seedRepairDB(t)
	ctx := context.Background()
	readRepo, err := comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := readRepo.AuditDiscoveryIntegrity(ctx)
	_ = readRepo.Close()
	if err != nil {
		t.Fatal(err)
	}
	plan := repair.BuildPlan(audit, time.Now().UTC())
	plan.DatabaseFingerprint = "stale"
	planPath := filepath.Join(t.TempDir(), "plan.json")
	if err := repair.WritePlanFile(planPath, plan); err != nil {
		t.Fatal(err)
	}
	_, _, err = applyDiscoveryRepair(ctx, discoveryApplyOptions{
		DBPath:          dbPath,
		PlanPath:        planPath,
		BackupPath:      backupPath,
		GroupID:         "keyless:3",
		ConfirmReviewed: true,
	})
	if err == nil {
		t.Fatal("expected stale plan failure")
	}
}

func TestApplyDiscoveryRepairRollsBackBatchFailure(t *testing.T) {
	dbPath, backupPath := seedRepairDB(t)
	ctx := context.Background()

	readRepo, err := comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeAudit, err := readRepo.AuditDiscoveryIntegrity(ctx)
	_ = readRepo.Close()
	if err != nil {
		t.Fatal(err)
	}

	plan := repair.BuildPlan(beforeAudit, time.Now().UTC())
	reviewedAt := time.Now().UTC().Format(time.RFC3339)
	group3 := filterGroup(plan.Groups, "keyless:3")
	group4 := filterGroup(plan.Groups, "keyless:4")
	if len(group3) != 1 || len(group4) != 1 {
		t.Fatal("expected keyless groups 3 and 4")
	}
	group3[0].Review = repair.Review{
		Approved: true, Decision: repair.DecisionRestoreKeys,
		Reviewer: "tester", ReviewedAt: reviewedAt,
	}
	group4[0].Review = repair.Review{
		Approved: true, Decision: repair.DecisionRestoreKeys,
		Reviewer: "tester", ReviewedAt: reviewedAt,
	}
	plan.Groups = []repair.Group{group3[0], group4[0]}
	planPath := filepath.Join(t.TempDir(), "plan.json")
	if err := repair.WritePlanFile(planPath, plan); err != nil {
		t.Fatal(err)
	}

	_, _, err = applyDiscoveryRepair(ctx, discoveryApplyOptions{
		DBPath:          dbPath,
		PlanPath:        planPath,
		BackupPath:      backupPath,
		AllApproved:     true,
		ConfirmReviewed: true,
	})
	if err == nil {
		t.Fatal("expected batch apply to fail on blocked keyless restore")
	}

	readRepo, err = comicrepo.NewSQLiteComicRepositoryReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	afterAudit, err := readRepo.AuditDiscoveryIntegrity(ctx)
	_ = readRepo.Close()
	if err != nil {
		t.Fatal(err)
	}
	if afterAudit.Summary.KeylessActive != beforeAudit.Summary.KeylessActive {
		t.Fatalf("expected rollback to preserve keyless=%d, got %d",
			beforeAudit.Summary.KeylessActive, afterAudit.Summary.KeylessActive)
	}
	if afterAudit.DatabaseFingerprint != beforeAudit.DatabaseFingerprint {
		t.Fatalf("batch failure mutated database fingerprint")
	}
}

func TestApplyDiscoveryRepairRequiresConfirm(t *testing.T) {
	_, _, err := applyDiscoveryRepair(context.Background(), discoveryApplyOptions{
		ConfirmReviewed: false,
	})
	if err == nil {
		t.Fatal("expected confirm-reviewed requirement")
	}
}

func seedRepairDB(t *testing.T) (string, string) {
	t.Helper()
	t.Setenv("COMICS_SEARCH_MODE", "scan")
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "comics.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRepairFixture(db); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	repo, err := comicrepo.NewSQLiteComicRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.Close()
	if err := checkpointSQLite(dbPath); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(dir, "backup.db")
	if err := vacuumInto(dbPath, backupPath); err != nil {
		t.Fatal(err)
	}
	return dbPath, backupPath
}

func writeRepairFixture(db *sql.DB) error {
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
	ownerKey := identity.DiscoveryTitleKeyPairsForTitles([]string{"Executioner"}, 3)[0]
	_, err = db.Exec(`
		INSERT INTO comics (
			id, titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap, rating, deleted, cover_visible
		) VALUES
			(1, 'Executioner', 10, '', 0, 3, 0, '', '', '', '', 1, 0, 0, 0, 1),
			(2, 'Executioner', 5, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1),
			(3, '英雄', 1, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1),
			(4, '|', 0, '', 0, 3, 0, '', '', '', '', 0, 0, 0, 0, 1);
		INSERT INTO comic_title_keys (comic_id, scope, title_key) VALUES (1, ?, ?);
	`, ownerKey.Scope, ownerKey.TitleKey)
	return err
}

func filterGroup(groups []repair.Group, id string) []repair.Group {
	out := make([]repair.Group, 0, 1)
	for _, group := range groups {
		if group.GroupID == id {
			out = append(out, group)
		}
	}
	return out
}

func vacuumInto(srcPath, dstPath string) error {
	db, err := sql.Open("sqlite", srcPath)
	if err != nil {
		return err
	}
	_, err = db.Exec(`VACUUM INTO '` + strings.ReplaceAll(dstPath, "'", "''") + `'`)
	if closeErr := db.Close(); err == nil {
		err = closeErr
	}
	return err
}

func checkpointSQLite(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if closeErr := db.Close(); err == nil {
		err = closeErr
	}
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func restoreDB(t *testing.T, dbPath, backupPath string) {
	t.Helper()
	if err := copyFile(backupPath, dbPath); err != nil {
		t.Fatal(err)
	}
}
