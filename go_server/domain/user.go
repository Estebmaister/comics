package domain

import (
	"context"

	"github.com/google/uuid"
)

const (
	CollectionUser = "users"
)

type User struct {
	ID       uuid.UUID `json:"_id" bson:"_id"`
	Username string    `json:"username" bson:"username"`
	Email    string    `json:"email" bson:"email"`
	Password string    `json:"-" bson:"password"`
	Role     string    `json:"role" bson:"role"`
}

type UserRepository interface {
	Create(c context.Context, user *User) error
	Fetch(c context.Context) ([]User, error)
	GetByEmail(c context.Context, email string) (User, error)
	GetByID(c context.Context, id string) (User, error)
}

type UserRegister struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
