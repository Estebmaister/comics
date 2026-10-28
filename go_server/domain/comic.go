package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// COMICS name of the collection/table on the DB
	COMICS = "comics"
)

var (
	ErrComicNotFound         = errors.New("comic not found")
	ErrDuplicateComic        = errors.New("comic is already in the database")
	ErrInvalidComicPayload   = errors.New("invalid comic payload")
	ErrInvalidComicMerge     = errors.New("invalid comic merge")
	ErrDiscoveryKeyConflict  = errors.New("discovery key conflict")
	ErrAmbiguousComicMatch   = errors.New("ambiguous comic match")
)

type DiscoveryKeyConflictError struct {
	ComicID      int
	OwnerComicID int
	Scope        string
	TitleKey     string
}

func (e *DiscoveryKeyConflictError) Error() string {
	return fmt.Sprintf(
		"%s: %s:%s belongs to comic %d, cannot assign to comic %d",
		ErrDiscoveryKeyConflict,
		e.Scope,
		e.TitleKey,
		e.OwnerComicID,
		e.ComicID,
	)
}

func (e *DiscoveryKeyConflictError) Is(target error) bool {
	return target == ErrDiscoveryKeyConflict || target == ErrDuplicateComic
}

// DuplicateComicMatchError is returned when a fuzzy discovery match finds another comic.
type DuplicateComicMatchError struct {
	ComicID      int
	OwnerComicID int
	MatchKind    string
}

func (e *DuplicateComicMatchError) Error() string {
	return fmt.Sprintf(
		"%s: comic %d matches existing comic %d (%s)",
		ErrDuplicateComic,
		e.ComicID,
		e.OwnerComicID,
		e.MatchKind,
	)
}

func (e *DuplicateComicMatchError) Is(target error) bool {
	return target == ErrDuplicateComic
}

type AmbiguousComicMatchError struct {
	IncomingTitle string
	Scope         string
	MatchKind     string
	CandidateIDs  []int
	BestScore     int
	RunnerUpScore int
}

func (e *AmbiguousComicMatchError) Error() string {
	return fmt.Sprintf(
		"%s: %q in %s scope matched comics %v with scores %d and %d",
		ErrAmbiguousComicMatch,
		e.IncomingTitle,
		e.Scope,
		e.CandidateIDs,
		e.BestScore,
		e.RunnerUpScore,
	)
}

func (e *AmbiguousComicMatchError) Is(target error) bool {
	return target == ErrAmbiguousComicMatch
}

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
	FindComicsByDiscoveryTitleKeys(ctx context.Context, scope string, titleKeys []string) ([]Comic, error)
	FindComicsByDiscoveryKeyToken(ctx context.Context, scope string, token string, limit int) ([]Comic, error)
	SyncDiscoveryKeys(ctx context.Context, comicID int, comType int, titles []string) error
	DeleteDiscoveryKeysForComic(ctx context.Context, comicID int) error
	Create(ctx context.Context, comic Comic) (Comic, error)
	Update(ctx context.Context, comic Comic) (Comic, error)
	Delete(ctx context.Context, id int) error
	SetCoverVisibility(ctx context.Context, id int, visible bool) (Comic, error)
	WithTx(ctx context.Context, fn func(ComicRepository) error) error
}

type ComicUseCase interface {
	Ping(ctx context.Context) error
	Close() error
	// CheckpointWAL persists SQLite WAL pages into the main DB file (no-op on Postgres).
	CheckpointWAL(ctx context.Context) error
	List(ctx context.Context, query ComicListQuery) (ComicListResult, error)
	Search(ctx context.Context, query ComicListQuery) (ComicListResult, error)
	Get(ctx context.Context, id int) (Comic, error)
	Create(ctx context.Context, comic Comic) (Comic, error)
	Update(ctx context.Context, id int, patch ComicPatch) (Comic, error)
	Delete(ctx context.Context, id int) error
	UpdateCoverVisibility(ctx context.Context, id int, cover string, visible bool) (Comic, error)
	Merge(ctx context.Context, baseID int, mergingID int) (Comic, error)
	Scrape(ctx context.Context) error
	ScrapeStart(ctx context.Context) error
	ScrapeStatus(ctx context.Context) (ScrapeStatus, error)
}

// ScrapeStatus reports background/manual scrape activity for the UI.
type ScrapeStatus struct {
	Running             bool
	Source              string
	LastCompletedAt     *time.Time
	LastCompletedSource string
}
