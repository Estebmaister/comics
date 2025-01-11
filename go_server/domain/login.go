package domain

import (
	"context"
)

type LoginRequest struct {
	// binding:"required,email" is a stronger validator for the field
	Email    string `form:"email" binding:"required" example:"test@example.com"`
	Password string `form:"password" binding:"required" example:"password123"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type LoginUsecase interface {
	GetUserByEmail(c context.Context, email string) (User, error)
	CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
	CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
}
