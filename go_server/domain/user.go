package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	// USERS name of the collection/table in the DB
	USERS = "users"
)

// User model
type User struct {
	ID        uuid.UUID `bson:"_id"`
	Username  string    `bson:"username"`
	Email     string    `bson:"email"`
	Password  string    `bson:"password" json:"-"`
	Role      string    `bson:"role"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
	Active    bool      `bson:"active"`
}

// UserStore interface abstracts user repository operations
type UserStore interface {
	UserReader
	UserWriter

	Tx(ctx context.Context, fn func(context.Context) error) error
	Ping(ctx context.Context) error
	GetStats() map[string]string
}

// UserReader interface abstracts user read operations
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)

	List(ctx context.Context, page, pageSize int) ([]*User, int64, error)
	FindActiveUsersByRole(ctx context.Context, role string) ([]*User, error)
}

// UserWriter interface abstracts user write operations
type UserWriter interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
