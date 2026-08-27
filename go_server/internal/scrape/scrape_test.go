package scrape

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"comics/domain"
	comicrepo "comics/internal/repo/comics"

	_ "modernc.org/sqlite"
)

type staticFetcher map[string]string

func (f staticFetcher) FetchHTML(_ context.Context, url string) (string, error) {
	return f[url], nil
}

func TestNormalizeScrapedComicParsesChapter(t *testing.T) {
	normalized, ok := normalizeScrapedComic(ScrapedComic{
		Chapter:  "Chapter 42",
		Title:    "Sample Hero",
		CoverURL: "https://example.com/cover.webp",
		ComType:  "manhwa",
		Status:   "ongoing",
	}, PublisherAsura)
	if !ok {
		t.Fatal("expected normalized comic")
	}
	if normalized.CurrentChap != 42 {
		t.Fatalf("expected chapter 42, got %d", normalized.CurrentChap)
	}
	if normalized.IdentityKey != "series:sample hero" {
		t.Fatalf("unexpected identity key %q", normalized.IdentityKey)
	}
}

func TestAstroDecodeNestedObject(t *testing.T) {
	decoded := astroDecode([]any{0, map[string]any{
		"comic_name": []any{0, "Hero"},
		"number":     []any{0, 12},
	}})
	obj, ok := decoded.(map[string]any)
	if !ok || obj["comic_name"] != "Hero" {
		t.Fatalf("unexpected decode result %#v", decoded)
	}
}

func TestExtractAsuraComicsFromProps(t *testing.T) {
	comics := extractAsuraComicsFromProps(map[string]any{
		"chapters": []any{
			map[string]any{
				"comic_name": "Hero",
				"number":     12,
				"comic_cover": "https://example.com/a.webp",
				"type":       "manhwa",
				"status":     "ongoing",
			},
		},
	})
	if len(comics) != 1 || comics[0].Title != "Hero" {
		t.Fatalf("unexpected comics %#v", comics)
	}
}

func extractAsuraComicsFromProps(props map[string]any) []ScrapedComic {
	comics := make([]ScrapedComic, 0)
	for _, key := range []string{"chapters", "items"} {
		collection, ok := props[key].([]any)
		if !ok {
			continue
		}
		for _, entry := range collection {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if comic, ok := comicFromAsuraEntry(item); ok {
				comics = append(comics, comic)
			}
		}
	}
	return comics
}

func TestExtractAsuraInitialSeries(t *testing.T) {
	raw := `{"initialSeries":[1,[[0,{"title":[0,"Browse Hero"],"chapter_count":[0,42],"cover":[0,"https://example.com/c.webp"],"type":[0,"manhwa"],"status":[0,"ongoing"]}]]]}`
	comics := extractAsuraComics(raw)
	if len(comics) != 1 {
		t.Fatalf("expected 1 comic, got %d", len(comics))
	}
	if comics[0].Title != "Browse Hero" || comics[0].Chapter != "42" {
		t.Fatalf("unexpected comic %#v", comics[0])
	}
}

func TestParseFlameCoverURL(t *testing.T) {
	cardHTML := `flamecomics.xyz%2Fuploads%2Fimages%2Fseries%2F51%2Fthumbnail.jpg%3F1758724372`
	got := parseFlameCoverURL(cardHTML)
	want := "https://flamecomics.xyz/uploads/images/series/51/thumbnail.jpg?1758724372"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestScrapeFlameFromFixture(t *testing.T) {
	ctx := context.Background()
	repo := newTestScrapeRepo(t)
	html := `<div class="SeriesCard-module__chapterCardContainer"><a class="SeriesCard-module__chapterImageLink" title="Flame Hero" href="/series/51"><img alt="Flame Hero"/></a><div>Chapter 3</div>flamecomics.xyz%2Fuploads%2Fimages%2Fseries%2F51%2Fthumb.jpg%3F1</div>`
	registered := 0
	err := scrapeFlamePage(ctx, "fixture://flame", PublisherFlameScans, staticFetcher{
		"fixture://flame": html,
	}, func(raw ScrapedComic, publisherID int) error {
		registered++
		return NewRegistrar(repo, NewRunState()).Register(ctx, raw, publisherID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if registered != 1 {
		t.Fatalf("expected one registration, got %d", registered)
	}
}

func newTestScrapeRepo(t *testing.T) domain.ComicRepository {
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
			cover_visible BOOLEAN NOT NULL DEFAULT 1,
			identity_key TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	repo, err := comicrepo.NewSQLiteComicRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}
