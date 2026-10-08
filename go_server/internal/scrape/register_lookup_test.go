package scrape

import (
	"context"
	"testing"

	"comics/domain"
)

type prefixLookupRepo struct {
	domain.ComicRepository
	listResult domain.ComicListResult
}

func (p *prefixLookupRepo) List(ctx context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	return p.listResult, nil
}

func TestFindComicByTitlePrefixResolvesTruncatedTitle(t *testing.T) {
	full := "The strongest assassin gets transferred to another world with his whole class"
	short := "The strongest assassin gets transferr"
	repo := &prefixLookupRepo{
		listResult: domain.ComicListResult{
			Comics: []domain.Comic{{
				ID:      9,
				Titles:  []string{full},
				ComType: ComTypeManhwa,
			}},
		},
	}
	normalized := normalizedComic{
		Titles:      []string{short},
		ComType:     ComTypeManhwa,
		CurrentChap: 134,
	}
	got, ok, err := findComicByTitlePrefix(context.Background(), repo, normalized)
	if err != nil || !ok || got.ID != 9 {
		t.Fatalf("expected prefix match, got ok=%v id=%d err=%v", ok, got.ID, err)
	}
}
