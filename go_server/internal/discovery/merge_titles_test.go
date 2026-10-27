package discovery_test

import (
	"context"
	"testing"

	"comics/domain"
	"comics/internal/discovery"
	"comics/internal/identity"
)

type mergeTitleRepo struct {
	owners map[string]int
}

func (r *mergeTitleRepo) Ping(context.Context) error { return nil }
func (r *mergeTitleRepo) Close() error               { return nil }
func (r *mergeTitleRepo) List(context.Context, domain.ComicListQuery) (domain.ComicListResult, error) {
	return domain.ComicListResult{}, nil
}
func (r *mergeTitleRepo) Get(context.Context, int) (domain.Comic, error) {
	return domain.Comic{}, domain.ErrComicNotFound
}
func (r *mergeTitleRepo) Create(context.Context, domain.Comic) (domain.Comic, error) {
	return domain.Comic{}, nil
}
func (r *mergeTitleRepo) Update(context.Context, domain.Comic) (domain.Comic, error) {
	return domain.Comic{}, nil
}
func (r *mergeTitleRepo) Delete(context.Context, int) error { return nil }
func (r *mergeTitleRepo) SetCoverVisibility(context.Context, int, bool) (domain.Comic, error) {
	return domain.Comic{}, nil
}
func (r *mergeTitleRepo) SyncDiscoveryKeys(context.Context, int, int, []string) error { return nil }
func (r *mergeTitleRepo) DeleteDiscoveryKeysForComic(context.Context, int) error { return nil }
func (r *mergeTitleRepo) WithTx(ctx context.Context, fn func(domain.ComicRepository) error) error {
	return fn(r)
}

func (r *mergeTitleRepo) FindComicsByDiscoveryTitleKeys(
	_ context.Context,
	scope string,
	titleKeys []string,
) ([]domain.Comic, error) {
	out := []domain.Comic{}
	for _, key := range titleKeys {
		id := r.owners[scope+":"+key]
		if id == 0 {
			continue
		}
		out = append(out, domain.Comic{ID: id, ComType: 3})
	}
	return out, nil
}

func (r *mergeTitleRepo) FindComicsByDiscoveryKeyToken(context.Context, string, string, int) ([]domain.Comic, error) {
	return nil, nil
}

func TestFilterTitlesForMergeDropsThirdPartyDiscoveryKey(t *testing.T) {
	ctx := context.Background()
	repo := &mergeTitleRepo{owners: map[string]int{
		"series:" + identity.DiscoveryTitleKey("I became the first prince: legend of sword's song"): 35359,
	}}
	titles := []string{
		"I became the rogue first prince",
		"I became the first prince: legend of sword's song",
	}
	filtered, err := discovery.FilterTitlesForMerge(ctx, repo, 35351, 35360, 3, titles)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0] != titles[0] {
		t.Fatalf("expected only rogue title, got %#v", filtered)
	}
}
