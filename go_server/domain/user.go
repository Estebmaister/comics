package domain

import (
	"context"

	"github.com/google/uuid"
)

const (
	USERS = "users" // collection or table name
)

type User struct {
	ID       uuid.UUID `bson:"_id"`
	Username string    `bson:"username"`
	Email    string    `bson:"email"`
	Password string    `bson:"password"`
	Role     string    `bson:"role"`
}

type UserRepository interface {
	Create(c context.Context, user *User) error
	Fetch(c context.Context) ([]User, error)
	GetByEmail(c context.Context, email string) (User, error)
	GetByID(c context.Context, id string) (User, error)
}
