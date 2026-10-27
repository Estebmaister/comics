package comics

import (
	"context"
	"sort"
	"testing"
	"time"

	"comics/domain"
)

func TestStrictCatalogSearchBudget(t *testing.T) {
	if !PerfStrictEnabled() {
		t.Skip("set COMICS_PERF_STRICT=1 for absolute search timing gates")
	}
	repo := openPerfCatalogRepoTB(t)
	ctx := context.Background()
	if err := RebuildCatalogSearchIndex(ctx, repo.db); err != nil {
		t.Fatal(err)
	}
	budgets, err := LoadPerfBudgets(perfBudgetsPath())
	if err != nil {
		t.Fatal(err)
	}
	trigramP95 := time.Duration(perfBudgetMS(budgets, "search_trigram_p95", 10)) * time.Millisecond
	trigramP99 := time.Duration(perfBudgetMS(budgets, "search_trigram_p99", 25)) * time.Millisecond
	shortP95 := time.Duration(perfBudgetMS(budgets, "search_short_p95", 50)) * time.Millisecond

	cases := []struct {
		term   string
		p95Max time.Duration
		p99Max time.Duration
	}{
		{term: "the", p95Max: trigramP95, p99Max: trigramP99},
		{term: "he", p95Max: shortP95, p99Max: shortP95},
	}
	for _, tc := range cases {
		samples := sampleListLatency(ctx, repo, tc.term, 60)
		p95 := durationPercentile(samples, 0.95)
		p99 := durationPercentile(samples, 0.99)
		t.Logf("strict_search term=%q p95=%s p99=%s", tc.term, p95, p99)
		if p95 > tc.p95Max {
			t.Fatalf("term %q p95 %s exceeds budget %s", tc.term, p95, tc.p95Max)
		}
		if p99 > tc.p99Max {
			t.Fatalf("term %q p99 %s exceeds budget %s", tc.term, p99, tc.p99Max)
		}
	}
}

func sampleListLatency(ctx context.Context, repo *SQLComicRepository, term string, iterations int) []time.Duration {
	query := domain.ComicListQuery{Limit: 12, SearchTitle: term}
	warmupList(ctx, repo, query)
	samples := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		if _, err := repo.List(ctx, query); err != nil {
			panic(err)
		}
		samples = append(samples, time.Since(start))
	}
	return samples
}

func durationPercentile(samples []time.Duration, p float64) time.Duration {
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
