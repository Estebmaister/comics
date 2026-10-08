package scrape

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"comics/domain"
)

type RunState struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewRunState() *RunState {
	return &RunState{seen: map[string]struct{}{}}
}

type Registrar struct {
	repo domain.ComicRepository
	run  *RunState
}

func NewRegistrar(repo domain.ComicRepository, run *RunState) *Registrar {
	if run == nil {
		run = NewRunState()
	}
	return &Registrar{repo: repo, run: run}
}

func (r *Registrar) Register(ctx context.Context, raw ScrapedComic, publisherID int) error {
	normalized, ok := normalizeScrapedComic(raw, publisherID)
	if !ok {
		return nil
	}
	key := fmt.Sprintf("%d|%s", publisherID, normalized.IdentityKey)
	r.run.mu.Lock()
	if _, exists := r.run.seen[key]; exists {
		r.run.mu.Unlock()
		return nil
	}
	r.run.seen[key] = struct{}{}
	r.run.mu.Unlock()

	return r.repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
		existing, err := txRepo.GetByIdentityKey(ctx, normalized.IdentityKey)
		if err != nil && !errors.Is(err, domain.ErrComicNotFound) {
			return err
		}
		if errors.Is(err, domain.ErrComicNotFound) {
			prefixMatch, ok, lookupErr := findComicByTitlePrefix(ctx, txRepo, normalized)
			if lookupErr != nil {
				return lookupErr
			}
			if ok {
				return updateExistingComic(ctx, txRepo, prefixMatch, normalized, publisherID)
			}
			_, err = txRepo.Create(ctx, domain.Comic{
				Titles:      normalized.Titles,
				CurrentChap: normalized.CurrentChap,
				Cover:       normalized.Cover,
				CoverVisible: normalized.Cover != "",
				ComType:     normalized.ComType,
				Status:      normalized.Status,
				Author:      normalized.Author,
				PublishedIn: []int{publisherID},
			})
			return err
		}
		return updateExistingComic(ctx, txRepo, existing, normalized, publisherID)
	})
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
	if normalized.ComType != ComTypeUnknown {
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
	if normalized.ComType != ComTypeUnknown && existing.ComType == ComTypeUnknown {
		updated.ComType = normalized.ComType
	}
	if normalized.Status != StatusUnknown {
		updated.Status = normalized.Status
	}
	if shouldUpdateCover(existing, normalized.Cover, publisherID) {
		updated.Cover = normalized.Cover
		updated.CoverVisible = true
	}
	_, err := repo.Update(ctx, updated)
	return err
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
