package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	comicrepo "comics/internal/repo/comics"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

type sourceComic struct {
	ID           int
	Titles       string
	CurrentChap  int
	Cover        string
	LastUpdate   int64
	ComType      int
	Status       int
	PublishedIn  string
	Genres       string
	Description  string
	Author       string
	Track        bool
	ViewedChap   int
	Rating       int
	Deleted      bool
	IdentityKey  string
	CoverVisible bool
}

type verifyCounts struct {
	Total          int
	Active         int
	Tracked        int
	CoverInvisible int
}

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	sqlitePath := firstNonEmpty(os.Getenv("COMICS_SQLITE_PATH"), "../src/db/comics.db")
	postgresURL := os.Getenv("COMICS_POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = os.Getenv("DATABASE_URL")
	}
	if postgresURL == "" {
		fatal(errors.New("COMICS_POSTGRES_URL or DATABASE_URL is required"))
	}
	postgresURL = requireSSLMode(postgresURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := importComics(ctx, sqlitePath, postgresURL); err != nil {
		fatal(err)
	}
}

func importComics(ctx context.Context, sqlitePath string, postgresURL string) error {
	source, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := sql.Open("pgx", postgresURL)
	if err != nil {
		return err
	}
	defer target.Close()

	if err := source.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite source unavailable: %w", err)
	}
	if err := target.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres target unavailable: %w", err)
	}
	if err := comicrepo.EnsurePostgresComicSchema(target); err != nil {
		return fmt.Errorf("ensuring postgres schema: %w", err)
	}

	comics, sourceCounts, err := readSourceComics(ctx, source)
	if err != nil {
		return err
	}
	if err := replaceTargetComics(ctx, target, comics); err != nil {
		return err
	}
	targetCounts, err := countTargetComics(ctx, target)
	if err != nil {
		return err
	}
	if sourceCounts != targetCounts {
		return fmt.Errorf("verification mismatch: source=%+v target=%+v", sourceCounts, targetCounts)
	}

	fmt.Printf("Imported %d comics into Postgres\n", targetCounts.Total)
	fmt.Printf("Verified active=%d tracked=%d cover_invisible=%d\n",
		targetCounts.Active,
		targetCounts.Tracked,
		targetCounts.CoverInvisible,
	)
	return nil
}

func readSourceComics(ctx context.Context, db *sql.DB) ([]sourceComic, verifyCounts, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap, rating,
			deleted, identity_key, cover_visible
		FROM comics
		ORDER BY id`)
	if err != nil {
		return nil, verifyCounts{}, err
	}
	defer rows.Close()

	comics := []sourceComic{}
	counts := verifyCounts{}
	for rows.Next() {
		var comic sourceComic
		if err := rows.Scan(
			&comic.ID,
			&comic.Titles,
			&comic.CurrentChap,
			&comic.Cover,
			&comic.LastUpdate,
			&comic.ComType,
			&comic.Status,
			&comic.PublishedIn,
			&comic.Genres,
			&comic.Description,
			&comic.Author,
			&comic.Track,
			&comic.ViewedChap,
			&comic.Rating,
			&comic.Deleted,
			&comic.IdentityKey,
			&comic.CoverVisible,
		); err != nil {
			return nil, verifyCounts{}, err
		}
		comics = append(comics, comic)
		counts.Total++
		if !comic.Deleted {
			counts.Active++
		}
		if comic.Track {
			counts.Tracked++
		}
		if !comic.CoverVisible {
			counts.CoverInvisible++
		}
	}
	return comics, counts, rows.Err()
}

func replaceTargetComics(ctx context.Context, db *sql.DB, comics []sourceComic) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint:errcheck

	var existing int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM comics").Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		backupName := "comics_backup_" + time.Now().UTC().Format("20060102_150405")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s AS TABLE comics", backupName)); err != nil {
			return fmt.Errorf("creating backup table: %w", err)
		}
		fmt.Printf("Backed up %d existing Postgres rows to %s\n", existing, backupName)
	}
	if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE comics RESTART IDENTITY"); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO comics (
			id, titles, current_chap, cover, last_update, com_type, status,
			published_in, genres, description, author, track, viewed_chap, rating,
			deleted, identity_key, cover_visible
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, comic := range comics {
		if _, err := stmt.ExecContext(
			ctx,
			comic.ID,
			comic.Titles,
			comic.CurrentChap,
			comic.Cover,
			comic.LastUpdate,
			comic.ComType,
			comic.Status,
			comic.PublishedIn,
			comic.Genres,
			comic.Description,
			comic.Author,
			comic.Track,
			comic.ViewedChap,
			comic.Rating,
			comic.Deleted,
			comic.IdentityKey,
			comic.CoverVisible,
		); err != nil {
			return fmt.Errorf("importing comic %d: %w", comic.ID, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "SELECT setval('comic_id_seq', COALESCE((SELECT MAX(id) FROM comics), 1), true)"); err != nil {
		return err
	}
	return tx.Commit()
}

func countTargetComics(ctx context.Context, db *sql.DB) (verifyCounts, error) {
	var counts verifyCounts
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE deleted = false),
			COUNT(*) FILTER (WHERE track = true),
			COUNT(*) FILTER (WHERE cover_visible = false)
		FROM comics`).Scan(
		&counts.Total,
		&counts.Active,
		&counts.Tracked,
		&counts.CoverInvisible,
	)
	return counts, err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func requireSSLMode(url string) string {
	if strings.Contains(url, "sslmode=") {
		return url
	}
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + "sslmode=require"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
