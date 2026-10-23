package main

import (
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
	flag.Parse()

	if !*dropIdentity {
		fmt.Fprintln(os.Stderr, "usage: db_maint --drop-legacy-identity-key")
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
