package domain

import (
	"context"
)

type SignUpRequest struct {
	Name     string `form:"name" binding:"required"`
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required"`
}

type SignUpResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type SignUpUsecase interface {
	Create(c context.Context, user *User) error
	// TODO: RestorePassword(c context.Context, user *User) error
	LoginUsecase
}
