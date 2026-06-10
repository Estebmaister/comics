package controller

import (
	"context"

	"comics/domain"
)

// AuthControl adapts HTTP routes to the auth use case.
type AuthControl struct {
	auth domain.AuthUseCase
}

func NewAuthControl(auth domain.AuthUseCase) *AuthControl {
	return &AuthControl{auth: auth}
}

func (ac *AuthControl) GetAccessTokenExpirySeconds() int {
	return ac.auth.GetAccessTokenExpirySeconds()
}

func (ac *AuthControl) GetRefreshTokenExpirySeconds() int {
	return ac.auth.GetRefreshTokenExpirySeconds()
}

func (ac *AuthControl) GetUserByJWT(ctx context.Context, accessToken string) (*domain.User, error) {
	return ac.auth.GetUserByJWT(ctx, accessToken)
}

func (ac *AuthControl) Login(
	ctx context.Context,
	accessToken string,
	user domain.LoginRequest,
) (*domain.AuthResponse, error) {
	return ac.auth.Login(ctx, accessToken, user)
}

func (ac *AuthControl) LoginByOAuthEmail(ctx context.Context, email string) (*domain.AuthResponse, error) {
	return ac.auth.LoginByOAuthEmail(ctx, email)
}

func (ac *AuthControl) UpdateProfile(
	ctx context.Context,
	accessToken string,
	user domain.UpdateRequest,
) (*domain.AuthResponse, error) {
	return ac.auth.UpdateProfile(ctx, accessToken, user)
}

func (ac *AuthControl) Register(ctx context.Context, user domain.SignUpRequest) (*domain.AuthResponse, error) {
	return ac.auth.Register(ctx, user)
}

func (ac *AuthControl) RefreshToken(
	ctx context.Context,
	refreshToken string,
	role string,
) (*domain.AuthResponse, error) {
	return ac.auth.RefreshToken(ctx, refreshToken, role)
}
