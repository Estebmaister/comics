package comics

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"comics/domain"

	_ "modernc.org/sqlite"
)

func newTestSQLiteRepo(t *testing.T) *SQLComicRepository {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "comics.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE comics (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			titles TEXT NOT NULL,
			current_chap INTEGER NOT NULL DEFAULT 0,
			cover TEXT NOT NULL DEFAULT '',
			last_update INTEGER NOT NULL,
			com_type INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 0,
			published_in TEXT NOT NULL DEFAULT '0',
			genres TEXT NOT NULL DEFAULT '0',
			description TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT '',
			track BOOLEAN NOT NULL DEFAULT 0,
			viewed_chap INTEGER NOT NULL DEFAULT 0,
			rating INTEGER NOT NULL DEFAULT 0,
			deleted BOOLEAN NOT NULL DEFAULT 0,
			cover_visible BOOLEAN NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	repo, err := NewSQLiteComicRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestSQLComicRepositoryListSearchAndMapping(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	created, err := repo.Create(ctx, domain.Comic{
		Titles:       []string{"The sample hero"},
		CurrentChap:  12,
		Cover:        "https://example.com/cover.webp",
		CoverVisible: true,
		ComType:      3,
		Status:       2,
		PublishedIn:  []int{1},
		Genres:       []int{6},
		Track:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.LastUpdate.IsZero() {
		t.Fatalf("expected id and last_update to round trip, got %#v", created)
	}

	result, err := repo.List(ctx, domain.ComicListQuery{SearchTitle: "SAMPLE", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Comics) != 1 {
		t.Fatalf("expected one result, got total=%d len=%d", result.Total, len(result.Comics))
	}
	if strings.Join(result.Comics[0].Titles, "|") != "The sample hero" {
		t.Fatalf("expected title mapping, got %#v", result.Comics[0].Titles)
	}
}

func TestSQLComicRepositorySetCoverVisibilityAndTransaction(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	created, err := repo.Create(ctx, domain.Comic{
		Titles:       []string{"Cover"},
		Cover:        "cover.webp",
		CoverVisible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		updated, err := txRepo.SetCoverVisibility(ctx, created.ID, false)
		if err != nil {
			return err
		}
		if updated.CoverVisible {
			t.Fatal("expected cover to be invisible in transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CoverVisible {
		t.Fatal("expected cover visibility update to commit")
	}
}

func TestSQLComicRepositoryRatingFilterAndSort(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)

	_, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"Low"},
		Rating:      1,
		CurrentChap: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	high, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"High"},
		Rating:      5,
		CurrentChap: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	min := 4
	result, err := repo.List(ctx, domain.ComicListQuery{
		Limit:     20,
		RatingMin: &min,
		SortBy:    "rating",
		SortDir:   "desc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Comics) != 1 {
		t.Fatalf("expected one result, got total=%d len=%d", result.Total, len(result.Comics))
	}
	if result.Comics[0].ID != high.ID {
		t.Fatalf("expected highest rated comic first, got %#v", result.Comics[0])
	}
}
