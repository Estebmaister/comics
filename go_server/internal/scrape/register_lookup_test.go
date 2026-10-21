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

func (p *prefixLookupRepo) List(_ context.Context, _ domain.ComicListQuery) (domain.ComicListResult, error) {
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

func TestFindComicByExactTitleKeyMatchesAcrossNonNovelTypes(t *testing.T) {
	title := "Catastrophic necromancer"
	repo := &prefixLookupRepo{
		listResult: domain.ComicListResult{
			Comics: []domain.Comic{{
				ID:      3546,
				Titles:  []string{title},
				ComType: ComTypeManhua,
			}},
		},
	}
	normalized := normalizedComic{
		Titles:      []string{title},
		ComType:     ComTypeManhwa,
		CurrentChap: 315,
	}
	got, ok, err := findComicByExactTitleKey(context.Background(), repo, normalized)
	if err != nil || !ok || got.ID != 3546 {
		t.Fatalf("expected exact cross-type match, got ok=%v id=%d err=%v", ok, got.ID, err)
	}
}

func TestFindComicByTitlePrefixMatchesAcrossNonNovelTypes(t *testing.T) {
	full := "Catastrophic necromancer"
	short := "Catastrophic necroman"
	repo := &prefixLookupRepo{
		listResult: domain.ComicListResult{
			Comics: []domain.Comic{{
				ID:      3546,
				Titles:  []string{full},
				ComType: ComTypeManhua,
			}},
		},
	}
	normalized := normalizedComic{
		Titles:  []string{short},
		ComType: ComTypeManhwa,
	}
	got, ok, err := findComicByTitlePrefix(context.Background(), repo, normalized)
	if err != nil || !ok || got.ID != 3546 {
		t.Fatalf("expected prefix cross-type match, got ok=%v id=%d err=%v", ok, got.ID, err)
	}
}
