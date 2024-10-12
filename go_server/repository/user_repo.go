package repository

import (
	"comics/domain"
	"context"

	"github.com/google/uuid"
)

type UserDB interface {
	FindAll(ctx context.Context) ([]domain.User, error)
	FindById(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
