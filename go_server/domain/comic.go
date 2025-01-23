package domain

import (
	"context"
	"time"
)

type Comic struct {
	ID          int
	Titles      []string
	Author      string
	Description string
	Cover       string
	ComType     int
	Status      int
	Publishers  []int
	Genres      []int
	Rating      int
	CurrentChap int
	ViewedChap  int
	Track       bool
	LastUpdate  time.Time
	Deleted     bool
}

type ComicRepository interface {
	GetByID(c context.Context, id int) (*Comic, error)
	List(c context.Context, page, pageSize int) ([]Comic, error)
	SearchByTitle(c context.Context, title string, page, pageSize int) ([]Comic, error)
	Create(c context.Context, user *Comic) error
	Update(c context.Context, user *Comic) error
	Delete(c context.Context, id int) error
}
