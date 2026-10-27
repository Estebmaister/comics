package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	comicrepo "comics/internal/repo/comics"
	"comics/internal/usecase"
)

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func TestCatalogHTTPLatencyReport(t *testing.T) {
	if os.Getenv("COMICS_PERF") != "1" {
		t.Skip("set COMICS_PERF=1 to run catalog HTTP latency report")
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
	_ = repo.RebuildSearchIndex(context.Background())

	comics := usecase.NewComicService(repo, nil)
	router := newComicTestRouter(comics)

	scenarios := []struct {
		name string
		path string
	}{
		{name: "list_first", path: "/comics?from=0&limit=12"},
		{name: "list_deep", path: "/comics?from=240&limit=12"},
		{name: "search_common", path: "/comics/search/the?from=0&limit=12"},
	}

	const iterations = 40
	for _, scenario := range scenarios {
		samples := make([]time.Duration, 0, iterations)
		for i := 0; i < iterations; i++ {
			start := time.Now()
			req := httptest.NewRequest(http.MethodGet, scenario.path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d: %s", scenario.name, res.Code, res.Body.String())
			}
			samples = append(samples, time.Since(start))
		}
		p50 := percentile(samples, 0.50)
		p95 := percentile(samples, 0.95)
		p99 := percentile(samples, 0.99)
		t.Logf("catalog_http %s p50=%s p95=%s p99=%s", scenario.name, p50, p95, p99)
		if os.Getenv("COMICS_PERF_STRICT") == "1" {
			budgets, err := comicrepo.LoadPerfBudgets(comicrepo.PerfBudgetsPath())
			if err != nil {
				t.Fatal(err)
			}
			switch scenario.name {
			case "search_common":
				maxP95 := time.Duration(comicrepo.PerfBudgetMS(budgets, "search_trigram_p95", 10)) * time.Millisecond
				maxP99 := time.Duration(comicrepo.PerfBudgetMS(budgets, "search_trigram_p99", 25)) * time.Millisecond
				if p95 > maxP95 {
					t.Fatalf("%s p95 %s exceeds %s", scenario.name, p95, maxP95)
				}
				if p99 > maxP99 {
					t.Fatalf("%s p99 %s exceeds %s", scenario.name, p99, maxP99)
				}
			default:
				maxP95 := time.Duration(comicrepo.PerfBudgetMS(budgets, "list_http_p95", 100)) * time.Millisecond
				if p95 > maxP95 {
					t.Fatalf("%s p95 %s exceeds %s", scenario.name, p95, maxP95)
				}
			}
		}
	}
}
