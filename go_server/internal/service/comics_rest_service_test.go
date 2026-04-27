package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestComicService(t *testing.T) *SQLiteComicService {
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
	db.Close()

	service, err := NewSQLiteComicService(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() }) // nolint:errcheck
	return service
}

func TestSQLiteComicServiceListAndSearch(t *testing.T) {
	service := newTestComicService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, ComicJSON{
		Titles:       []string{"The sample hero"},
		CurrentChap:  12,
		Cover:        "https://example.com/cover.webp",
		CoverVisible: true,
		LastUpdate:   time.Now().UTC().Format(time.RFC3339),
		ComType:      3,
		Status:       2,
		PublishedIn:  []int{1},
		Genres:       []int{6},
		Track:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("expected created comic id")
	}

	result, err := service.Search(ctx, "sample", 0, 20, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Comics) != 1 {
		t.Fatalf("expected one result, got total=%d len=%d", result.Total, len(result.Comics))
	}
	if result.Comics[0].CoverVisible != true {
		t.Fatal("expected cover_visible to round trip")
	}
}

func TestSQLiteComicServiceCoverVisibilityUsesStaleCoverGuard(t *testing.T) {
	service := newTestComicService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, ComicJSON{
		Titles:       []string{"Cover guard"},
		Cover:        "https://example.com/current.webp",
		CoverVisible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	unchanged, err := service.UpdateCoverVisibility(
		ctx,
		created.ID,
		"https://example.com/stale.webp",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.CoverVisible {
		t.Fatal("stale cover report should not mark the current cover invisible")
	}

	updated, err := service.UpdateCoverVisibility(ctx, created.ID, created.Cover, false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CoverVisible {
		t.Fatal("expected current cover to be marked invisible")
	}
}

func TestSQLiteComicServiceMergePrefersVisibleCover(t *testing.T) {
	service := newTestComicService(t)
	ctx := context.Background()

	base, err := service.Create(ctx, ComicJSON{
		Titles:       []string{"Base title"},
		CurrentChap:  10,
		Cover:        "https://example.com/base.webp",
		CoverVisible: false,
		ComType:      3,
		PublishedIn:  []int{1},
		Genres:       []int{2},
	})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.Create(ctx, ComicJSON{
		Titles:       []string{"Duplicate title"},
		CurrentChap:  12,
		Cover:        "https://example.com/duplicate.webp",
		CoverVisible: true,
		ComType:      3,
		PublishedIn:  []int{3},
		Genres:       []int{4},
	})
	if err != nil {
		t.Fatal(err)
	}

	merged, err := service.Merge(ctx, base.ID, duplicate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Cover != duplicate.Cover || !merged.CoverVisible {
		t.Fatalf("expected visible duplicate cover, got %s visible=%v", merged.Cover, merged.CoverVisible)
	}
	if merged.CurrentChap != 12 {
		t.Fatalf("expected newest chapter, got %d", merged.CurrentChap)
	}
	if _, err := service.Get(ctx, duplicate.ID); err != ErrComicNotFound {
		t.Fatal("expected duplicate record to be deleted")
	}
}
