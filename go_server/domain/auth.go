package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserService defines methods for authentication and user management
type UserService interface {
	Login(ctx context.Context, user LoginRequest) (*User, error)
	Register(ctx context.Context, user SignUpRequest) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
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
