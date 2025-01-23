package domain

import (
	"context"
	"time"
)

const (
	// collection or table name
	COMICS = "comics"
)

// Comic model
type Comic struct {
	ID          int       `json:"id"`
	Titles      []string  `json:"titles"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Cover       string    `json:"cover"`
	ComType     int       `json:"com_type"`
	Status      int       `json:"status"`
	Publishers  []int     `json:"published_in"`
	Genres      []int     `json:"genres"`
	Rating      int       `json:"rating"`
	CurrentChap int       `json:"current_chap"`
	ViewedChap  int       `json:"viewed_chap"`
	Track       bool      `json:"track"`
	LastUpdate  time.Time `json:"last_update"`
	Deleted     bool      `json:"deleted"`
}

// Comic repository operations
type ComicStore interface {
	ComicReader
	ComicWriter
}

// Comic read operations
type ComicReader interface {
	GetByID(ctx context.Context, id int) (*Comic, error)
	List(ctx context.Context, page, pageSize int) ([]Comic, error)
	SearchByTitle(ctx context.Context, title string, page, pageSize int) ([]Comic, error)
}

// Comic write operations
type ComicWriter interface {
	Create(ctx context.Context, comic *Comic) error
	Update(ctx context.Context, comic *Comic) error
	Delete(ctx context.Context, id int) error
}
