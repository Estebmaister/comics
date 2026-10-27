package scrape

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"comics/domain"
	"comics/internal/discovery"
	"comics/internal/identity"
	"comics/internal/metrics"
)

type RunState struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewRunState() *RunState {
	return &RunState{seen: map[string]struct{}{}}
}

type Registrar struct {
	repo     domain.ComicRepository
	run      *RunState
	resolver *discovery.Resolver
	observer discovery.Observer
}

func NewRegistrar(repo domain.ComicRepository, run *RunState) *Registrar {
	if run == nil {
		run = NewRunState()
	}
	observer := discovery.DefaultObserver()
	return &Registrar{
		repo:     repo,
		run:      run,
		resolver: discovery.NewResolver(observer),
		observer: observer,
	}
}

func (r *Registrar) Register(ctx context.Context, raw ScrapedComic, publisherID int) error {
	normalized, ok := normalizeScrapedComic(raw, publisherID)
	if !ok {
		return nil
	}
	discoveryKey := identity.DiscoveryKey(normalized.Titles[0], normalized.ComType)
	runKey := fmt.Sprintf("%d|%s", publisherID, discoveryKey)
	r.run.mu.Lock()
	if _, exists := r.run.seen[runKey]; exists {
		r.run.mu.Unlock()
		return nil
	}
	r.run.seen[runKey] = struct{}{}
	r.run.mu.Unlock()

	return r.repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		existing, _, found, err := r.resolver.Resolve(
			ctx,
			txRepo,
			normalized.Titles[0],
			normalized.ComType,
			discovery.ResolveOptions{},
		)
		if err != nil {
			if errors.Is(err, domain.ErrAmbiguousComicMatch) {
				r.observer.ObserveScrapeRegistration("ambiguous")
			}
			return err
		}
		if found {
			return r.updateExistingComic(ctx, txRepo, existing, normalized, publisherID)
		}
		_, createErr := txRepo.Create(ctx, domain.Comic{
			Titles:       normalized.Titles,
			CurrentChap:  normalized.CurrentChap,
			Cover:        normalized.Cover,
			CoverVisible: normalized.Cover != "",
			ComType:      normalized.ComType,
			Status:       normalized.Status,
			Author:       normalized.Author,
			PublishedIn:  []int{publisherID},
		})
		if errors.Is(createErr, domain.ErrDiscoveryKeyConflict) {
			r.observer.ObserveConflict("scrape")
		}
		return createErr
	})
}

func (r *Registrar) updateExistingComic(
	ctx context.Context,
	repo domain.ComicRepository,
	existing domain.Comic,
	normalized normalizedComic,
	publisherID int,
) error {
	err := updateExistingComic(ctx, repo, existing, normalized, publisherID)
	if errors.Is(err, domain.ErrDiscoveryKeyConflict) {
		r.observer.ObserveConflict("scrape")
	}
	return err
}

func updateExistingComic(
	ctx context.Context,
	repo domain.ComicRepository,
	existing domain.Comic,
	normalized normalizedComic,
	publisherID int,
) error {
	updated := existing
	comType := existing.ComType
	if comType == ComTypeUnknown && normalized.ComType != ComTypeUnknown {
		comType = normalized.ComType
	}
	updated.Titles = mergeTitleVariants(existing.Titles, normalized.Titles, comType)
	updated.ComType = comType
	updated.PublishedIn = mergeUniqueInts(existing.PublishedIn, []int{publisherID})
	if normalized.CurrentChap > existing.CurrentChap {
		updated.CurrentChap = normalized.CurrentChap
		updated.LastUpdate = time.Now().UTC()
	}
	if normalized.Author != "" && existing.Author == "" {
		updated.Author = normalized.Author
	}
	if normalized.Status != StatusUnknown {
		updated.Status = normalized.Status
	}
	if shouldUpdateCover(existing, normalized.Cover, publisherID) {
		updated.Cover = normalized.Cover
		updated.CoverVisible = true
	}
	if scrapeExistingUnchanged(existing, updated) {
		metrics.IncScrapeNoopSkipped()
		return nil
	}
	_, err := repo.Update(ctx, updated)
	return err
}

func scrapeExistingUnchanged(before, after domain.Comic) bool {
	return before.CurrentChap == after.CurrentChap &&
		before.ComType == after.ComType &&
		before.Status == after.Status &&
		before.Author == after.Author &&
		before.Cover == after.Cover &&
		before.CoverVisible == after.CoverVisible &&
		intSlicesEqual(before.PublishedIn, after.PublishedIn) &&
		stringSlicesEqual(before.Titles, after.Titles)
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mergeUniqueInts(base []int, extra []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(base)+len(extra))
	for _, value := range append(base, extra...) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func shouldUpdateCover(existing domain.Comic, cover string, publisherID int) bool {
	if cover == "" || existing.Cover == cover {
		return false
	}
	if existing.Cover == "" {
		return true
	}
	if !existing.CoverVisible {
		return true
	}
	if publisherID == PublisherDemonicScans {
		return true
	}
	publishers := map[int]struct{}{}
	for _, pub := range existing.PublishedIn {
		publishers[pub] = struct{}{}
	}
	if _, ok := publishers[PublisherDemonicScans]; ok {
		return false
	}
	if _, ok := lowPriorityCoverPublishers[publisherID]; ok {
		for pub := range publishers {
			if _, low := lowPriorityCoverPublishers[pub]; !low && pub != 0 {
				return false
			}
		}
		return true
	}
	for pub := range publishers {
		if _, ok := lowPriorityCoverPublishers[pub]; ok {
			return true
		}
	}
	if _, restricted := restrictedCoverPublishers[publisherID]; restricted {
		return false
	}
	_, allowed := coverUpdatePublishers[publisherID]
	return allowed
}
