package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"comics/domain"
	"comics/internal/identity"
	"comics/internal/scrape"
)

const (
	defaultComicPageLimit = 20
	maxComicPageLimit     = 100
)

type ComicService struct {
	repo        domain.ComicRepository
	scrapeCoord *scrape.Coordinator
}

func NewComicService(repo domain.ComicRepository, scrapeCoord *scrape.Coordinator) *ComicService {
	if scrapeCoord == nil {
		scrapeCoord = scrape.NewCoordinator(repo)
	}
	return &ComicService{repo: repo, scrapeCoord: scrapeCoord}
}

func (s *ComicService) ScrapeCoordinator() *scrape.Coordinator {
	return s.scrapeCoord
}

func (s *ComicService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *ComicService) Close() error {
	return s.repo.Close()
}

func (s *ComicService) List(ctx context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	query.SearchTitle = ""
	query = normalizeComicListQuery(query)
	return s.repo.List(ctx, query)
}

func (s *ComicService) Search(ctx context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	query.SearchTitle = strings.TrimSpace(query.SearchTitle)
	if query.SearchTitle == "" {
		return domain.ComicListResult{}, fmt.Errorf("%w: title cannot be empty", domain.ErrInvalidComicPayload)
	}
	query = normalizeComicListQuery(query)
	return s.repo.List(ctx, query)
}

func (s *ComicService) Get(ctx context.Context, id int) (domain.Comic, error) {
	return s.repo.Get(ctx, id)
}

func (s *ComicService) Create(ctx context.Context, comic domain.Comic) (domain.Comic, error) {
	if !hasNonEmptyTitle(comic.Titles) {
		return domain.Comic{}, fmt.Errorf("%w: titles should be a non-empty list of strings", domain.ErrInvalidComicPayload)
	}
	comic.Titles = identity.NormalizeTitleVariants(comic.Titles, comic.ComType)
	if !hasNonEmptyTitle(comic.Titles) {
		return domain.Comic{}, fmt.Errorf("%w: titles should be a non-empty list of strings", domain.ErrInvalidComicPayload)
	}
	identityKey := identity.BuildIdentityKeyFromTitles(comic.Titles, comic.ComType)
	if identityKey != "" {
		existing, err := s.repo.GetByIdentityKey(ctx, identityKey)
		if err != nil && !errors.Is(err, domain.ErrComicNotFound) {
			return domain.Comic{}, err
		}
		if err == nil && existing.ID != 0 {
			return domain.Comic{}, domain.ErrDuplicateComic
		}
	}
	if comic.Cover == "" {
		comic.CoverVisible = true
	}
	return s.repo.Create(ctx, comic)
}

func (s *ComicService) Update(ctx context.Context, id int, patch domain.ComicPatch) (domain.Comic, error) {
	if patch.Titles != nil && !hasNonEmptyTitle(*patch.Titles) {
		return domain.Comic{}, fmt.Errorf("%w: titles should be a non-empty list of strings", domain.ErrInvalidComicPayload)
	}
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Comic{}, err
	}
	applyComicPatch(&current, patch)
	return s.repo.Update(ctx, current)
}

func (s *ComicService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *ComicService) Scrape(ctx context.Context) error {
	return s.scrapeCoord.Run(ctx, scrape.SourceManual)
}

func (s *ComicService) UpdateCoverVisibility(
	ctx context.Context,
	id int,
	cover string,
	visible bool,
) (domain.Comic, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Comic{}, err
	}
	if current.Cover != cover {
		return current, nil
	}
	return s.repo.SetCoverVisibility(ctx, id, visible)
}

func (s *ComicService) Merge(ctx context.Context, baseID int, mergingID int) (domain.Comic, error) {
	if baseID == mergingID {
		return domain.Comic{}, fmt.Errorf("%w: comics cannot merge with themselves", domain.ErrInvalidComicMerge)
	}

	var merged domain.Comic
	err := s.repo.WithTx(ctx, func(repo domain.ComicRepository) error {
		base, err := repo.Get(ctx, baseID)
		if err != nil {
			return err
		}
		duplicate, err := repo.Get(ctx, mergingID)
		if err != nil {
			return err
		}
		if duplicate.ComType != 0 && base.ComType != duplicate.ComType {
			return fmt.Errorf("%w: comics to merge should be of the same type", domain.ErrInvalidComicMerge)
		}

		merged = mergeComicValues(base, duplicate)
		if _, err = repo.Update(ctx, merged); err != nil {
			return err
		}
		return repo.Delete(ctx, mergingID)
	})
	if err != nil {
		return domain.Comic{}, err
	}
	return s.repo.Get(ctx, merged.ID)
}

func normalizeComicListQuery(query domain.ComicListQuery) domain.ComicListQuery {
	if query.Full {
		query.Offset = 0
		query.Limit = 0
		return query
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if query.Limit <= 0 {
		query.Limit = defaultComicPageLimit
	} else if query.Limit > maxComicPageLimit {
		query.Limit = maxComicPageLimit
	}

	if query.RatingMin != nil {
		value := *query.RatingMin
		if value < 0 {
			value = 0
		} else if value > 5 {
			value = 5
		}
		query.RatingMin = &value
	}
	if query.RatingMax != nil {
		value := *query.RatingMax
		if value < 0 {
			value = 0
		} else if value > 5 {
			value = 5
		}
		query.RatingMax = &value
	}
	if query.RatingMin != nil && query.RatingMax != nil && *query.RatingMax < *query.RatingMin {
		value := *query.RatingMin
		query.RatingMax = &value
	}

	query.SortBy = strings.ToLower(strings.TrimSpace(query.SortBy))
	query.SortDir = strings.ToLower(strings.TrimSpace(query.SortDir))
	switch query.SortBy {
	case "", "last_update", "rating", "id":
	default:
		query.SortBy = "last_update"
	}
	switch query.SortDir {
	case "", "desc", "asc":
	default:
		query.SortDir = "desc"
	}
	if query.SortBy == "" {
		query.SortBy = "last_update"
	}
	if query.SortDir == "" {
		query.SortDir = "desc"
	}
	return query
}

func applyComicPatch(current *domain.Comic, patch domain.ComicPatch) {
	if patch.Titles != nil {
		current.Titles = *patch.Titles
	}
	if patch.Cover != nil && *patch.Cover != current.Cover {
		current.Cover = *patch.Cover
		if !patch.CoverVisiblePresent {
			current.CoverVisible = true
		}
	}
	if patch.CoverVisible != nil {
		current.CoverVisible = *patch.CoverVisible
	}
	if patch.CurrentChap != nil {
		current.CurrentChap = *patch.CurrentChap
	}
	if patch.ComType != nil {
		current.ComType = *patch.ComType
	}
	if patch.Status != nil {
		current.Status = *patch.Status
	}
	if patch.ViewedChap != nil {
		current.ViewedChap = *patch.ViewedChap
	}
	if patch.Rating != nil {
		current.Rating = *patch.Rating
	}
	if patch.PublishedIn != nil {
		current.PublishedIn = *patch.PublishedIn
	}
	if patch.Genres != nil {
		current.Genres = *patch.Genres
	}
	if patch.Description != nil {
		current.Description = *patch.Description
	}
	if patch.Author != nil {
		current.Author = *patch.Author
	}
	if patch.Track != nil {
		current.Track = *patch.Track
	}
	if patch.Deleted != nil {
		current.Deleted = *patch.Deleted
	}
}

func hasNonEmptyTitle(titles []string) bool {
	for _, title := range titles {
		if strings.TrimSpace(title) != "" {
			return true
		}
	}
	return false
}

func mergeComicValues(base domain.Comic, duplicate domain.Comic) domain.Comic {
	base.Titles = mergeStrings(base.Titles, duplicate.Titles)
	base.PublishedIn = mergeInts(base.PublishedIn, duplicate.PublishedIn)
	base.Genres = mergeInts(base.Genres, duplicate.Genres)
	base.CurrentChap = max(base.CurrentChap, duplicate.CurrentChap)
	base.ViewedChap = max(base.ViewedChap, duplicate.ViewedChap)
	base.Track = base.Track || duplicate.Track
	base.Deleted = false
	if duplicate.Author != "" && base.Author == "" {
		base.Author = duplicate.Author
	}
	if duplicate.Description != "" && base.Description == "" {
		base.Description = duplicate.Description
	}
	if duplicate.Cover != "" && (!base.CoverVisible || base.Cover == "") {
		base.Cover = duplicate.Cover
		base.CoverVisible = duplicate.CoverVisible
	}
	if duplicate.Rating > base.Rating {
		base.Rating = duplicate.Rating
	}
	if duplicate.Status != 0 {
		base.Status = duplicate.Status
	}
	return base
}

func mergeStrings(a []string, b []string) []string {
	seen := map[string]bool{}
	merged := []string{}
	for _, value := range append(a, b...) {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, value)
	}
	return merged
}

func mergeInts(a []int, b []int) []int {
	seen := map[int]bool{}
	merged := []int{}
	for _, value := range append(a, b...) {
		if seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	return merged
}
