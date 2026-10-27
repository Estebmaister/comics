package comics

import (
	"context"
	"os"
	"testing"

	"comics/domain"
)

func openPerfCatalogRepo(b *testing.B) *SQLComicRepository {
	return openPerfCatalogRepoTB(b)
}

func openPerfCatalogRepoTB(tb testing.TB) *SQLComicRepository {
	path := PerfCatalogDBPath()
	if path == "" {
		tb.Skip("set COMICS_PERF_DB to a verified comics.db snapshot (or place db/comics.db)")
	}
	repo, err := NewSQLiteComicRepositoryReadOnly(path)
	if err != nil {
		tb.Fatalf("open perf db: %v", err)
	}
	return repo
}

func warmupList(ctx context.Context, repo *SQLComicRepository, query domain.ComicListQuery) {
	_, _ = repo.List(ctx, query)
}

func BenchmarkCatalogListFirstPage(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, Offset: 0}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogListDeepPage(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, Offset: 240}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogListTracked(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, OnlyTracked: true}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogListRatingSort(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, SortBy: "rating", SortDir: "desc"}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogSearchCommon(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, SearchTitle: "the"}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogSearchRare(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	query := domain.ComicListQuery{Limit: 12, SearchTitle: "zzznomatch"}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCatalogSearchCJK(b *testing.B) {
	repo := openPerfCatalogRepo(b)
	ctx := context.Background()
	term := os.Getenv("COMICS_PERF_CJK")
	if term == "" {
		term = "仙"
	}
	query := domain.ComicListQuery{Limit: 12, SearchTitle: term}
	warmupList(ctx, repo, query)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.List(ctx, query); err != nil {
			b.Fatal(err)
		}
	}
}
