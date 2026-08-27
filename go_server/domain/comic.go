package domain

import (
	"context"
	"errors"
	"time"
)

const (
	// COMICS name of the collection/table on the DB
	COMICS = "comics"
)

var (
	ErrComicNotFound       = errors.New("comic not found")
	ErrDuplicateComic      = errors.New("comic is already in the database")
	ErrInvalidComicPayload = errors.New("invalid comic payload")
	ErrInvalidComicMerge   = errors.New("invalid comic merge")
)

// Comic model
type Comic struct {
	ID           int
	Titles       []string
	CurrentChap  int
	Cover        string
	CoverVisible bool
	LastUpdate   time.Time
	ComType      int
	Status       int
	PublishedIn  []int
	Genres       []int
	Description  string
	Author       string
	Track        bool
	ViewedChap   int
	Rating       int
	Deleted      bool
}

type ComicPatch struct {
	Titles              *[]string
	CurrentChap         *int
	Cover               *string
	CoverVisible        *bool
	ComType             *int
	Status              *int
	PublishedIn         *[]int
	Genres              *[]int
	Description         *string
	Author              *string
	Track               *bool
	ViewedChap          *int
	Rating              *int
	Deleted             *bool
	CoverVisiblePresent bool
}

type ComicListQuery struct {
	Offset        int
	Limit         int
	SearchTitle   string
	OnlyTracked   bool
	OnlyUnchecked bool
	Full          bool
	RatingMin     *int
	RatingMax     *int
	SortBy        string
	SortDir       string
}

type ComicListResult struct {
	Comics      []Comic
	Total       int
	TotalPages  int
	CurrentPage int
}

type ComicRepository interface {
	Ping(ctx context.Context) error
	Close() error
	List(ctx context.Context, query ComicListQuery) (ComicListResult, error)
	Get(ctx context.Context, id int) (Comic, error)
	GetByIdentityKey(ctx context.Context, identityKey string) (Comic, error)
	Create(ctx context.Context, comic Comic) (Comic, error)
	Update(ctx context.Context, comic Comic) (Comic, error)
	Delete(ctx context.Context, id int) error
	SetCoverVisibility(ctx context.Context, id int, visible bool) (Comic, error)
	WithTx(ctx context.Context, fn func(ComicRepository) error) error
}

type ComicUseCase interface {
	Ping(ctx context.Context) error
	Close() error
	List(ctx context.Context, query ComicListQuery) (ComicListResult, error)
	Search(ctx context.Context, query ComicListQuery) (ComicListResult, error)
	Get(ctx context.Context, id int) (Comic, error)
	Create(ctx context.Context, comic Comic) (Comic, error)
	Update(ctx context.Context, id int, patch ComicPatch) (Comic, error)
	Delete(ctx context.Context, id int) error
	UpdateCoverVisibility(ctx context.Context, id int, cover string, visible bool) (Comic, error)
	Merge(ctx context.Context, baseID int, mergingID int) (Comic, error)
	Scrape(ctx context.Context) error
}
