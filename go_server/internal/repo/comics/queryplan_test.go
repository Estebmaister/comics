package comics

import (
	"context"
	"strings"
	"testing"

	"comics/domain"
)

func explainQueryPlan(t *testing.T, repo *SQLComicRepository, sql string, args ...any) string {
	t.Helper()
	rows, err := repo.db.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	lines := []string{}
	for rows.Next() {
		var id, parent, unused, detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

func TestDefaultPaginationUsesActiveUpdateIndex(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	where, args := repo.comicFilters(domain.ComicListQuery{})
	sql := baseComicSelect() + " " + where + " " + repo.orderByClause(domain.ComicListQuery{}) + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), 12, 0)

	plan := explainQueryPlan(t, repo, sql, listArgs...)
	if !strings.Contains(plan, "idx_comics_active_update") {
		t.Fatalf("expected idx_comics_active_update in plan, got:\n%s", plan)
	}
}

func TestTrackedPaginationUsesTrackIndex(t *testing.T) {
	repo := newTestSQLiteRepo(t)

	where, args := repo.comicFilters(domain.ComicListQuery{OnlyTracked: true})
	sql := baseComicSelect() + " " + where + " " + repo.orderByClause(domain.ComicListQuery{OnlyTracked: true}) + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), 12, 0)

	plan := explainQueryPlan(t, repo, sql, listArgs...)
	if !strings.Contains(plan, "idx_comics_active_track_update") {
		t.Fatalf("expected idx_comics_active_track_update in plan, got:\n%s", plan)
	}
}

func TestTrigramEligibleTitleSearchUsesFTS(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	if _, err := repo.Create(ctx, domain.Comic{Titles: []string{"Hero returns"}, CurrentChap: 1}); err != nil {
		t.Fatal(err)
	}
	if err := RebuildCatalogSearchIndex(ctx, repo.db); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMICS_SEARCH_MODE", searchModeTrigram)
	search := repo.buildTitleSearchFilter(ctx, domain.ComicListQuery{SearchTitle: "her"})
	if search.path != "trigram" {
		t.Fatalf("expected trigram path, got %s", search.path)
	}
	fromClause := "comics" + search.joinSQL
	sql := baseComicSelectFrom(fromClause) + " " + search.where + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, search.args...), 12, 0)
	plan := explainQueryPlan(t, repo, sql, listArgs...)
	if !strings.Contains(plan, "VIRTUAL TABLE") || !strings.Contains(strings.ToLower(plan), "fts") {
		t.Fatalf("expected FTS virtual table in plan, got:\n%s", plan)
	}
}

func TestSubstringTitleSearchPlanDocumentsScan(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	if _, err := repo.Create(ctx, domain.Comic{Titles: []string{"Hero returns"}, CurrentChap: 1}); err != nil {
		t.Fatal(err)
	}

	search := repo.buildTitleSearchFilter(ctx, domain.ComicListQuery{SearchTitle: "he"})
	fromClause := "comics" + search.joinSQL
	sql := baseComicSelectFrom(fromClause) + " " + search.where + " " + repo.orderByClause(domain.ComicListQuery{}) + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, search.args...), 12, 0)
	plan := explainQueryPlan(t, repo, sql, listArgs...)

	if strings.TrimSpace(plan) == "" {
		t.Fatal("expected non-empty query plan for substring search")
	}
	// Leading-wildcard LIKE is not title-index-backed; planner output varies by SQLite version/cardinality.
	t.Logf("substring search plan (LIKE remains a filter, not asserted as index-only):\n%s", plan)
}
