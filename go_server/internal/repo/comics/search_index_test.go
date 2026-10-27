package comics

import (
	"context"
	"errors"
	"testing"

	"comics/domain"
)

func TestTrigramSearchMatchesLegacyScan(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	seed := []domain.Comic{
		{Titles: []string{"The hero returns"}, CurrentChap: 3},
		{Titles: []string{"Alt title|Second name"}, CurrentChap: 1},
		{Titles: []string{"Smart “quotes”"}, CurrentChap: 2},
	}
	for _, comic := range seed {
		if _, err := repo.Create(ctx, comic); err != nil {
			t.Fatal(err)
		}
	}
	if err := RebuildCatalogSearchIndex(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	terms := []string{"hero", "sec", "quo", "zz", "a%b", "仙"}
	for _, term := range terms {
		scan := listWithSearchMode(t, ctx, repo, term, searchModeScan)
		auto := listWithSearchMode(t, ctx, repo, term, searchModeAuto)
		if !sameComicIDs(scan, auto) {
			t.Fatalf("term %q: scan %#v != auto %#v", term, comicIDs(scan), comicIDs(auto))
		}
	}
}

func listWithSearchMode(t *testing.T, ctx context.Context, repo *SQLComicRepository, term string, mode string) []domain.Comic {
	t.Setenv("COMICS_SEARCH_MODE", mode)
	result, err := repo.List(ctx, domain.ComicListQuery{SearchTitle: term, Limit: 50})
	if err != nil {
		t.Fatalf("list %q mode %s: %v", term, mode, err)
	}
	return result.Comics
}

func comicIDs(comics []domain.Comic) []int {
	ids := make([]int, 0, len(comics))
	for _, comic := range comics {
		ids = append(ids, comic.ID)
	}
	return ids
}

func sameComicIDs(a, b []domain.Comic) bool {
	return equalIntSlices(comicIDs(a), comicIDs(b))
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[int]int{}
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
		if seen[v] < 0 {
			return false
		}
	}
	return true
}

func TestCatalogSearchTriggersStayInSyncOnRollback(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	if err := RebuildCatalogSearchIndex(ctx, repo.db); err != nil {
		t.Fatal(err)
	}
	err := repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		_, err := txRepo.Create(ctx, domain.Comic{Titles: []string{"Tx comic"}, CurrentChap: 1})
		if err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	result, err := repo.List(ctx, domain.ComicListQuery{SearchTitle: "Tx comic", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Comics) != 0 {
		t.Fatalf("expected rolled-back comic absent from search, got %d", len(result.Comics))
	}
}

func TestTrigramFTSCapable(t *testing.T) {
	repo := newTestSQLiteRepo(t)
	if !trigramFTSCapable(context.Background(), repo.db) {
		t.Fatal("expected modernc sqlite FTS5 trigram support")
	}
}
