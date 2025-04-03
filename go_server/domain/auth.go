package domain

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// UserServicer defines methods for authentication and user management
type UserServicer interface {
	Login(ctx context.Context, user LoginRequest) (*User, error)
	Register(ctx context.Context, user SignUpRequest) (*User, error)
	Update(ctx context.Context, dbUser *User, updateUser UpdateRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// APIResponse define the base generic API response
type APIResponse[T any] struct {
	Status  int    `json:"status"`
	Data    *T     `json:"data,omitempty"`
	Message string `json:"message"`
}

// AuthData is the response body for authentication
type AuthData struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// AuthResponse is the response body for authentication
type AuthResponse = APIResponse[AuthData]

// NoDataResponse is the response body for errors
type NoDataResponse = APIResponse[bool]

// LoginRequest is the request body for login
type LoginRequest struct {
	Email    string `form:"email" binding:"required,email" example:"test@example.com"`
	Password string `form:"password" binding:"required" example:"password123"`
}

// SignUpRequest is the request body for signUp
type SignUpRequest struct {
	Email    string `form:"email" binding:"required,email" example:"test@example.com"`
	Username string `form:"username" binding:"required" example:"testuser"`
	Password string `form:"password" binding:"required" example:"password123"`
}

// UpdateRequest is the request body for updating a user
type UpdateRequest struct {
	Email    string `form:"email" binding:"omitempty,email" example:"test@example.com"`
	Username string `form:"username" binding:"omitempty" example:"testuser"`
	Password string `form:"password" binding:"omitempty" example:"password123"`
}

// Validate checks that at least one field is provided
func (u UpdateRequest) Validate() error {
	if strings.TrimSpace(u.Email) == "" &&
		strings.TrimSpace(u.Username) == "" &&
		strings.TrimSpace(u.Password) == "" {
		return errors.New("at least one field (email, username, or password) must be provided")
	}
	return nil
}
