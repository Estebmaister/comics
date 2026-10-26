package scrape

import (
	"context"
	"errors"
	"testing"

	"comics/domain"
)

type registerStubRepo struct {
	byKeys  []domain.Comic
	created int
	updated int
}

func (s *registerStubRepo) Ping(context.Context) error { return nil }
func (s *registerStubRepo) Close() error               { return nil }
func (s *registerStubRepo) List(context.Context, domain.ComicListQuery) (domain.ComicListResult, error) {
	return domain.ComicListResult{}, nil
}
func (s *registerStubRepo) Get(context.Context, int) (domain.Comic, error) {
	return domain.Comic{}, domain.ErrComicNotFound
}
func (s *registerStubRepo) FindComicsByDiscoveryTitleKeys(context.Context, string, []string) ([]domain.Comic, error) {
	return s.byKeys, nil
}
func (s *registerStubRepo) FindComicsByDiscoveryKeyToken(context.Context, string, string, int) ([]domain.Comic, error) {
	return nil, nil
}
func (s *registerStubRepo) SyncDiscoveryKeys(context.Context, int, int, []string) error { return nil }
func (s *registerStubRepo) DeleteDiscoveryKeysForComic(context.Context, int) error      { return nil }
func (s *registerStubRepo) Create(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	s.created++
	comic.ID = 99
	return comic, nil
}
func (s *registerStubRepo) Update(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	s.updated++
	return comic, nil
}
func (s *registerStubRepo) Delete(context.Context, int) error { return nil }
func (s *registerStubRepo) SetCoverVisibility(context.Context, int, bool) (domain.Comic, error) {
	return domain.Comic{}, nil
}
func (s *registerStubRepo) WithTx(ctx context.Context, fn func(domain.ComicRepository) error) error {
	return fn(s)
}

func TestRegisterAmbiguousMatchStopsWithoutWrite(t *testing.T) {
	stub := &registerStubRepo{
		byKeys: []domain.Comic{
			{ID: 1, Titles: []string{"The executioner"}, ComType: ComTypeManhwa},
			{ID: 2, Titles: []string{"The executioner"}, ComType: ComTypeManhwa},
		},
	}
	registrar := NewRegistrar(stub, NewRunState())
	err := registrar.Register(context.Background(), ScrapedComic{
		Title:   "The executioner",
		Chapter: "10",
		ComType: "manhwa",
	}, PublisherAsura)
	if !errors.Is(err, domain.ErrAmbiguousComicMatch) {
		t.Fatalf("expected ambiguous scrape to stop, got %v", err)
	}
	if stub.created != 0 || stub.updated != 0 {
		t.Fatalf("ambiguous scrape wrote data: created=%d updated=%d", stub.created, stub.updated)
	}
}
