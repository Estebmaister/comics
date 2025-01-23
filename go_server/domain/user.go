package domain

import (
	"context"

	"github.com/google/uuid"
)

const (
	// collection or table name
	USERS = "users"
)

// User model
type User struct {
	ID        uuid.UUID `bson:"_id"`
	Username  string    `bson:"username"`
	Email     string    `bson:"email"`
	Password  string    `bson:"password"`
	Role      string    `bson:"role"`
	CreatedAt int64     `bson:"created_at"`
	UpdatedAt int64     `bson:"updated_at"`
}

// User repository operations
type UserStore interface {
	Fetch(c context.Context) ([]User, error)
	UserReader
	UserWriter
}

// User read operations
type UserReader interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// User write operations
type UserWriter interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}
