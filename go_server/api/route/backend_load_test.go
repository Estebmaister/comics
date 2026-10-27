package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"comics/domain"
	comicrepo "comics/internal/repo/comics"
	"comics/internal/usecase"
)

func TestCatalogSearchUnderScrapeLikeWrites(t *testing.T) {
	if os.Getenv("COMICS_PERF") != "1" {
		t.Skip("set COMICS_PERF=1 to run concurrent catalog load test")
	}
	path := os.Getenv("COMICS_PERF_DB")
	if path == "" {
		path = comicrepo.PerfCatalogDBPath()
	}
	if path == "" {
		t.Skip("set COMICS_PERF_DB to a verified comics.db snapshot")
	}

	repo, err := comicrepo.NewSQLiteComicRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()
	if err := repo.RebuildSearchIndex(ctx); err != nil {
		t.Fatal(err)
	}

	list, err := repo.List(ctx, domain.ComicListQuery{Limit: 1})
	if err != nil || len(list.Comics) == 0 {
		t.Skip("perf snapshot has no comics for write churn")
	}
	targetID := list.Comics[0].ID

	comics := usecase.NewComicService(repo, nil)
	router := newComicTestRouter(comics)

	loadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		ticker := time.NewTicker(3 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-loadCtx.Done():
				return
			case <-ticker.C:
				comic, err := repo.Get(loadCtx, targetID)
				if err != nil {
					continue
				}
				comic.ViewedChap++
				_, _ = repo.Update(loadCtx, comic)
			}
		}
	}()

	samples := make([]time.Duration, 0, 40)
	for i := 0; i < 40; i++ {
		start := time.Now()
		req := httptest.NewRequest(http.MethodGet, "/comics/search/the?from=0&limit=12", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.Code)
		}
		samples = append(samples, time.Since(start))
	}
	cancel()

	p95 := percentile(samples, 0.95)
	p99 := percentile(samples, 0.99)
	t.Logf("search_under_write p95=%s p99=%s", p95, p99)
	if os.Getenv("COMICS_PERF_STRICT") == "1" {
		budgets, err := comicrepo.LoadPerfBudgets(comicrepo.PerfBudgetsPath())
		if err != nil {
			t.Fatal(err)
		}
		maxP95 := time.Duration(comicrepo.PerfBudgetMS(budgets, "search_under_scrape_p95", 25)) * time.Millisecond
		maxP99 := time.Duration(comicrepo.PerfBudgetMS(budgets, "search_under_scrape_p99", 75)) * time.Millisecond
		if p95 > maxP95 {
			t.Fatalf("p95 %s exceeds budget %s", p95, maxP95)
		}
		if p99 > maxP99 {
			t.Fatalf("p99 %s exceeds budget %s", p99, maxP99)
		}
	}
}
