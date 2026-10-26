package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	comicrepo "comics/internal/repo/comics"

	_ "modernc.org/sqlite"
)

func main() {
	dropIdentity := flag.Bool("drop-legacy-identity-key", false, "drop comics.identity_key after comic_title_keys is populated")
	applyPlan := flag.String("apply-discovery-plan", "", "apply a reviewed discovery repair plan")
	backupPath := flag.String("backup", "", "verified pre-repair SQLite backup")
	groupID := flag.String("group-id", "", "apply one approved group")
	allApproved := flag.Bool("all-approved", false, "apply every explicitly approved group")
	confirmReviewed := flag.Bool("confirm-reviewed", false, "confirm the plan received human review")
	flag.Parse()

	if *applyPlan != "" {
		before, after, err := applyDiscoveryRepair(context.Background(), discoveryApplyOptions{
			DBPath:          firstNonEmpty(os.Getenv("COMICS_SQLITE_PATH"), "../db/comics.db"),
			PlanPath:        *applyPlan,
			BackupPath:      *backupPath,
			GroupID:         *groupID,
			AllApproved:     *allApproved,
			ConfirmReviewed: *confirmReviewed,
		})
		if err != nil {
			fatal(err)
		}
		fmt.Printf(
			"discovery repair applied: keyless %d -> %d conflicts %d -> %d restorable %d -> %d\n",
			before.KeylessActive,
			after.KeylessActive,
			before.ActiveConflicts,
			after.ActiveConflicts,
			before.Restorable,
			after.Restorable,
		)
		return
	}

	if !*dropIdentity {
		fmt.Fprintln(os.Stderr, "usage: db_maint --drop-legacy-identity-key | discovery repair flags")
		os.Exit(2)
	}

	path := firstNonEmpty(os.Getenv("COMICS_SQLITE_PATH"), "../db/comics.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	var keyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comic_title_keys`).Scan(&keyCount); err != nil {
		fatal(err)
	}
	if keyCount == 0 {
		fatal(fmt.Errorf("%s: comic_title_keys is empty; run the API once or backfill before dropping identity_key", path))
	}

	start := time.Now()
	if err := comicrepo.DropLegacyIdentityKeyColumn(db); err != nil {
		fatal(err)
	}
	fmt.Printf("dropped legacy identity_key on %s (%d discovery keys, %s)\n",
		path, keyCount, time.Since(start).Round(time.Millisecond))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
