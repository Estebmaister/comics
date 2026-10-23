package discovery_test

import (
	"context"
	"testing"

	"comics/domain"
	"comics/internal/discovery"
	"comics/internal/identity"
	"comics/internal/scrape"
)

type stubRepo struct {
	domain.ComicRepository
	byKeys  []domain.Comic
	byToken []domain.Comic
}

func (s *stubRepo) FindComicsByDiscoveryTitleKeys(context.Context, string, []string) ([]domain.Comic, error) {
	return s.byKeys, nil
}

func (s *stubRepo) FindComicsByDiscoveryKeyToken(context.Context, string, string, int) ([]domain.Comic, error) {
	return s.byToken, nil
}

func TestResolveComicExecutionerAmbiguity(t *testing.T) {
	repo := &stubRepo{
		byKeys: []domain.Comic{
			{ID: 1071, Titles: []string{"The executioner (lee jehwan)", "The executioner"}, ComType: scrape.ComTypeManhwa},
			{ID: 34458, Titles: []string{"The executioner", "Executioner"}, ComType: scrape.ComTypeManhwa, Track: true},
		},
	}
	comic, kind, ok, err := discovery.ResolveComic(context.Background(), repo, "Executioner", scrape.ComTypeManhwa)
	if err != nil || !ok || kind != discovery.MatchDiscoveryKey || comic.ID != 34458 {
		t.Fatalf("got ok=%v kind=%s id=%d err=%v", ok, kind, comic.ID, err)
	}
}

func TestResolveComicFrozenPlayerReordered(t *testing.T) {
	repo := &stubRepo{
		byKeys: nil,
		byToken: []domain.Comic{
			{
				ID:      93,
				Titles:  []string{"The frozen player returns", "Return of the frozen player"},
				ComType: scrape.ComTypeManhwa,
				Track:   true,
			},
		},
	}
	comic, _, ok, err := discovery.ResolveComic(context.Background(), repo, "Return of the frozen player", scrape.ComTypeManhwa)
	if err != nil || !ok || comic.ID != 93 {
		t.Fatalf("got ok=%v id=%d err=%v", ok, comic.ID, err)
	}
}

func TestDiscoveryKeyUnifiesExecutionerTitles(t *testing.T) {
	a := identity.DiscoveryKey("The executioner", 3)
	b := identity.DiscoveryKey("Executioner", 3)
	if a != b || a != "series:executioner" {
		t.Fatalf("unexpected keys %q %q", a, b)
	}
}
