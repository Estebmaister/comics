package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"comics/domain"
	"comics/internal/identity"
)

type memoryComicRepo struct {
	comics map[int]domain.Comic
	nextID int
}

func newMemoryComicRepo() *memoryComicRepo {
	return &memoryComicRepo{comics: map[int]domain.Comic{}, nextID: 1}
}

func (r *memoryComicRepo) Ping(context.Context) error { return nil }
func (r *memoryComicRepo) Close() error               { return nil }

func (r *memoryComicRepo) List(_ context.Context, query domain.ComicListQuery) (domain.ComicListResult, error) {
	items := []domain.Comic{}
	for _, comic := range r.comics {
		if comic.Deleted {
			continue
		}
		if query.OnlyTracked && !comic.Track {
			continue
		}
		if query.OnlyUnchecked && (!comic.Track || comic.CurrentChap == comic.ViewedChap) {
			continue
		}
		items = append(items, comic)
	}
	total := len(items)
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	return domain.ComicListResult{
		Comics:      items,
		Total:       total,
		TotalPages:  1,
		CurrentPage: 1,
	}, nil
}

func (r *memoryComicRepo) Get(_ context.Context, id int) (domain.Comic, error) {
	comic, ok := r.comics[id]
	if !ok {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	return comic, nil
}

func (r *memoryComicRepo) GetByIdentityKey(_ context.Context, identityKey string) (domain.Comic, error) {
	for _, comic := range r.comics {
		if comic.Deleted {
			continue
		}
		key := identity.BuildIdentityKeyFromTitles(comic.Titles, comic.ComType)
		if key == identityKey {
			return comic, nil
		}
	}
	return domain.Comic{}, domain.ErrComicNotFound
}

func (r *memoryComicRepo) Create(_ context.Context, comic domain.Comic) (domain.Comic, error) {
	comic.ID = r.nextID
	r.nextID++
	r.comics[comic.ID] = comic
	return comic, nil
}

func (r *memoryComicRepo) Update(_ context.Context, comic domain.Comic) (domain.Comic, error) {
	if _, ok := r.comics[comic.ID]; !ok {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	r.comics[comic.ID] = comic
	return comic, nil
}

func (r *memoryComicRepo) Delete(_ context.Context, id int) error {
	if _, ok := r.comics[id]; !ok {
		return domain.ErrComicNotFound
	}
	delete(r.comics, id)
	return nil
}

func (r *memoryComicRepo) SetCoverVisibility(_ context.Context, id int, visible bool) (domain.Comic, error) {
	comic, ok := r.comics[id]
	if !ok {
		return domain.Comic{}, domain.ErrComicNotFound
	}
	comic.CoverVisible = visible
	r.comics[id] = comic
	return comic, nil
}

func (r *memoryComicRepo) WithTx(ctx context.Context, fn func(domain.ComicRepository) error) error {
	return fn(r)
}

func TestComicServiceUpdateRespectsExplicitZeroAndEmptyValues(t *testing.T) {
	ctx := context.Background()
	svc := NewComicService(newMemoryComicRepo())
	created, err := svc.Create(ctx, domain.Comic{
		Titles:      []string{"Original"},
		CurrentChap: 10,
		ComType:     3,
		Status:      2,
		PublishedIn: []int{1, 2},
		Genres:      []int{4, 5},
		Description: "description",
		Author:      "author",
		Track:       true,
		ViewedChap:  8,
		Rating:      4,
	})
	if err != nil {
		t.Fatal(err)
	}

	zero := 0
	empty := ""
	emptyInts := []int{}
	track := false
	updated, err := svc.Update(ctx, created.ID, domain.ComicPatch{
		CurrentChap: &zero,
		ViewedChap:  &zero,
		ComType:     &zero,
		Status:      &zero,
		Rating:      &zero,
		Author:      &empty,
		Description: &empty,
		PublishedIn: &emptyInts,
		Genres:      &emptyInts,
		Track:       &track,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentChap != 0 || updated.ViewedChap != 0 || updated.ComType != 0 ||
		updated.Status != 0 || updated.Rating != 0 || updated.Track {
		t.Fatalf("expected explicit zero/false fields to apply, got %#v", updated)
	}
	if updated.Author != "" || updated.Description != "" || len(updated.PublishedIn) != 0 || len(updated.Genres) != 0 {
		t.Fatalf("expected empty optional fields to clear, got %#v", updated)
	}
}

func TestComicServiceCoverVisibilityPrecedence(t *testing.T) {
	ctx := context.Background()
	svc := NewComicService(newMemoryComicRepo())
	created, err := svc.Create(ctx, domain.Comic{
		Titles:       []string{"Cover precedence"},
		Cover:        "old.webp",
		CoverVisible: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	nextCover := "new.webp"
	updated, err := svc.Update(ctx, created.ID, domain.ComicPatch{Cover: &nextCover})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Cover != nextCover || !updated.CoverVisible {
		t.Fatalf("expected changed cover to become visible, got %#v", updated)
	}

	anotherCover := "newer.webp"
	visible := false
	updated, err = svc.Update(ctx, created.ID, domain.ComicPatch{
		Cover:               &anotherCover,
		CoverVisible:        &visible,
		CoverVisiblePresent: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Cover != anotherCover || updated.CoverVisible {
		t.Fatalf("expected explicit cover_visible to win, got %#v", updated)
	}
}

func TestComicServiceMergeRules(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo)
	base, err := svc.Create(ctx, domain.Comic{
		Titles:       []string{"Base"},
		CurrentChap:  10,
		Cover:        "base.webp",
		CoverVisible: false,
		ComType:      3,
	})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := svc.Create(ctx, domain.Comic{
		Titles:       []string{"Duplicate"},
		CurrentChap:  12,
		Cover:        "duplicate.webp",
		CoverVisible: true,
		ComType:      3,
		Track:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	merged, err := svc.Merge(ctx, base.ID, duplicate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Cover != duplicate.Cover || !merged.CoverVisible || merged.CurrentChap != 12 || !merged.Track {
		t.Fatalf("expected merged values to prefer richer duplicate data, got %#v", merged)
	}
	if _, err := repo.Get(ctx, duplicate.ID); !errors.Is(err, domain.ErrComicNotFound) {
		t.Fatalf("expected duplicate to be deleted, got %v", err)
	}
}

func TestComicServiceCreateRejectsDuplicate(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo)
	if _, err := svc.Create(ctx, domain.Comic{Titles: []string{"Solo leveling"}, ComType: 3}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(ctx, domain.Comic{Titles: []string{"Solo Leveling"}, ComType: 3})
	if !errors.Is(err, domain.ErrDuplicateComic) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestComicServiceNormalizesPagination(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryComicRepo()
	svc := NewComicService(repo)
	for i := 0; i < 105; i++ {
		if _, err := svc.Create(ctx, domain.Comic{Titles: []string{fmt.Sprintf("title-%d", i)}}); err != nil {
			t.Fatal(err)
		}
	}

	result, err := svc.List(ctx, domain.ComicListQuery{Offset: -20, Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Comics) != maxComicPageLimit {
		t.Fatalf("expected limit to clamp to %d, got %d", maxComicPageLimit, len(result.Comics))
	}
}
