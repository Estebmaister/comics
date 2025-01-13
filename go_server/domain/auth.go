package domain

import (
	"context"
)

type LoginRequest struct {
	Email    string `form:"email" binding:"required,email" example:"test@example.com"`
	Password string `form:"password" binding:"required" example:"password123"`
}

type SignUpRequest struct {
	Email    string `form:"email" binding:"required,email" example:"test@example.com"`
	Username string `form:"username" binding:"required" example:"testuser"`
	Password string `form:"password" binding:"required" example:"password123"`
}

type AuthResponse struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message"`
}

type LoginUsecase interface {
	GetUserByEmail(c context.Context, email string) (User, error)
	CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
	CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
}

type SignUpUsecase interface {
	Register(c context.Context, user *SignUpRequest) error
	// TODO: RestorePassword(c context.Context, user *User) error
	LoginUsecase
}
