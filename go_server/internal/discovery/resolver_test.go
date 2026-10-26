package discovery_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"comics/domain"
	"comics/internal/discovery"
	"comics/internal/identity"
	"comics/internal/scrape"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

func TestResolverPolicy(t *testing.T) {
	tests := []struct {
		name      string
		incoming  string
		byKeys    []domain.Comic
		byToken   []domain.Comic
		excludeID int
		wantID    int
		wantKind  discovery.MatchKind
		ambiguous bool
	}{
		{
			name:     "exact stored text beats article variant",
			incoming: "Executioner",
			byKeys: []domain.Comic{
				{ID: 1071, Titles: []string{"The executioner"}, ComType: 3},
				{ID: 34458, Titles: []string{"Executioner"}, ComType: 3},
			},
			wantID:   34458,
			wantKind: discovery.MatchDiscoveryKey,
		},
		{
			name:     "equal exact relations are ambiguous",
			incoming: "Executioner",
			byKeys: []domain.Comic{
				{ID: 20, Titles: []string{"The executioner"}, ComType: 3},
				{ID: 10, Titles: []string{"The executioner"}, ComType: 3, Track: true, CurrentChap: 999},
			},
			ambiguous: true,
		},
		{
			name:      "update excludes itself",
			incoming:  "Executioner",
			byKeys:    []domain.Comic{{ID: 10, Titles: []string{"Executioner"}, ComType: 3}},
			excludeID: 10,
		},
		{
			name:     "prefix beats no relation",
			incoming: "The strongest assassin gets transferr",
			byToken: []domain.Comic{
				{ID: 9, Titles: []string{"The strongest assassin gets transferred to another world"}, ComType: 3},
			},
			wantID:   9,
			wantKind: discovery.MatchTitlePrefix,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := discovery.NewResolver(discovery.NoopObserver{})
			repo := &stubRepo{byKeys: tc.byKeys, byToken: tc.byToken}
			comic, kind, found, err := resolver.Resolve(
				context.Background(),
				repo,
				tc.incoming,
				3,
				discovery.ResolveOptions{ExcludeComicID: tc.excludeID},
			)
			if tc.ambiguous {
				var ambiguous *domain.AmbiguousComicMatchError
				if !errors.As(err, &ambiguous) {
					t.Fatalf("expected ambiguity, got %v", err)
				}
				return
			}
			if err != nil || comic.ID != tc.wantID || kind != tc.wantKind || found != (tc.wantID != 0) {
				t.Fatalf("got id=%d kind=%s found=%v err=%v", comic.ID, kind, found, err)
			}
		})
	}
}

func TestResolverRecordsBoundedAmbiguityMetric(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := discovery.NewPrometheusObserver(registry)
	resolver := discovery.NewResolver(observer)
	repo := &stubRepo{byKeys: []domain.Comic{
		{ID: 1, Titles: []string{"The executioner"}, ComType: 3},
		{ID: 2, Titles: []string{"The executioner"}, ComType: 3},
	}}
	_, _, _, _ = resolver.Resolve(
		context.Background(),
		repo,
		"Executioner",
		3,
		discovery.ResolveOptions{},
	)
	err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(`# HELP comics_discovery_resolution_total Discovery resolution outcomes.
# TYPE comics_discovery_resolution_total counter
comics_discovery_resolution_total{match_kind="discovery_key",outcome="ambiguous"} 1
comics_discovery_resolution_total{match_kind="none",outcome="not_found"} 0
`),
		"comics_discovery_resolution_total",
	)
	if err != nil {
		t.Fatal(err)
	}
}
