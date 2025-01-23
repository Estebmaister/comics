package repository

import (
	"comics/domain"
	"context"
)

// Comic operations
type ComicRepo interface {
	ComicReader
	ComicWriter
}

// Comic read operations
type ComicReader interface {
	GetByID(ctx context.Context, id int) (*domain.Comic, error)
	List(ctx context.Context, page, pageSize int) ([]domain.Comic, error)
	SearchByTitle(ctx context.Context, title string, page, pageSize int) ([]domain.Comic, error)
}

// Comic write operations
type ComicWriter interface {
	Create(ctx context.Context, comic *domain.Comic) error
	Update(ctx context.Context, comic *domain.Comic) error
	Delete(ctx context.Context, id int) error
}

// ---=====---

// User operations
type UserRepo interface {
	UserReader
	UserWriter
}

// User read operations
type UserReader interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

// User write operations
type UserWriter interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}
