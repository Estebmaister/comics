package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"comics/bootstrap"
	"comics/domain"
	"comics/internal/repo"
	"comics/internal/service"
	"comics/internal/tokenutil"
)

// AuthControl
type AuthControl struct {
	userService domain.UserService
	env         *bootstrap.Env
}

// NewAuthControl creates a new AuthControl
func NewAuthControl(userService domain.UserService, env *bootstrap.Env) *AuthControl {
	return &AuthControl{userService: userService, env: env}
}

// GetAccessTokenExpirySeconds returns the expiry time in seconds
func (ac *AuthControl) GetAccessTokenExpirySeconds() int {
	return int(ac.env.JWT.AccessTokenExpiryHour.Seconds())
}

// GetRefreshTokenExpirySeconds returns the expiry time in seconds
func (ac *AuthControl) GetRefreshTokenExpirySeconds() int {
	return int(ac.env.JWT.RefreshTokenExpiryHour.Seconds())
}

// GetUserByJWT returns a user from a JWT
func (ac *AuthControl) GetUserByJWT(ctx context.Context, accessToken string) (
	*domain.User, error) {
	// Load secret key from environment variable
	secretKey, _, err := ac.getSecretKeys()
	if err != nil {
		return nil, err
	}

	token := strings.TrimPrefix(accessToken, "Bearer ")

	// Validate the token
	claims, err := tokenutil.VerifyToken(token, secretKey)
	if err != nil {
		return nil, err
	}
	return ac.userService.GetByID(ctx, claims.UserID)
}

// Login handles user login
func (ac *AuthControl) Login(ctx context.Context, accessToken string, user domain.LoginRequest) (
	*domain.AuthResponse, error) {
	// Load secret key from environment variable
	secretKey, refreshSecretKey, err := ac.getSecretKeys()
	if err != nil {
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	if accessToken != "" {
		token := strings.TrimPrefix(accessToken, "Bearer ")

		// Validate the token
		claims, err := tokenutil.VerifyToken(token, secretKey)
		if err == nil { // success
			return &domain.AuthResponse{
				Status:  http.StatusOK,
				Message: "Authenticated with token",
				Data: &domain.AuthData{
					UserID:      claims.UserID.String(),
					AccessToken: token,
				},
			}, nil
		}
		// If token is invalid, fallback to password-based login
	}

	dbUser, err := ac.userService.Login(ctx, user)
	if errors.Is(err, repo.ErrNotFound) {
		err := fmt.Errorf("invalid credentials") // avoids exposing specific error
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	} else if err != nil {
		log.Printf("error login user: %s, %s", user.Email, err)
		err := fmt.Errorf("invalid credentials") // avoids exposing specific error
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}

	// Generate a JWT
	accessToken, errAccess := tokenutil.GenerateTokenWithRole(
		dbUser.ID, secretKey, ac.env.JWT.AccessTokenExpiryHour, dbUser.Role)
	refreshToken, errRefresh := tokenutil.GenerateTokenWithRole(
		dbUser.ID, refreshSecretKey, ac.env.JWT.RefreshTokenExpiryHour, dbUser.Role)
	if errAccess != nil && errRefresh != nil {
		err := fmt.Errorf("error generating token")
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	// Return tokens in response
	return &domain.AuthResponse{
		Status:  http.StatusOK,
		Message: "Login successful",
		Data: &domain.AuthData{
			UserID:       dbUser.ID.String(),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

// Register handles user registration
func (ac *AuthControl) Register(ctx context.Context, user domain.SignUpRequest) (
	*domain.AuthResponse, error) {
	dbUser, err := ac.userService.Register(ctx, user)
	if errors.Is(err, service.ErrCredsAlreadyExist) {
		return &domain.AuthResponse{Status: http.StatusConflict, Message: err.Error()}, err
	} else if err != nil {
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	return &domain.AuthResponse{
		Status: http.StatusCreated,
		// Message: "User registered successfully",
		Message: fmt.Sprintf("%v", dbUser), // Debug response
		Data:    &domain.AuthData{UserID: dbUser.ID.String()},
	}, nil
}

func (ac *AuthControl) RefreshToken(ctx context.Context, refreshToken, role string) (
	*domain.AuthResponse, error) {
	// Load secret key from environment variable
	secretKey, refreshSecretKey, err := ac.getSecretKeys()
	if err != nil {
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	if refreshToken == "" {
		err := fmt.Errorf("no access token provided")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}
	token := strings.TrimPrefix(refreshToken, "Bearer ")

	// Validate the token
	claims, err := tokenutil.VerifyToken(token, refreshSecretKey)
	if err != nil {
		err := fmt.Errorf("invalid refresh token")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}
	if claims.Subject != role {
		err := fmt.Errorf("invalid role")
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: err.Error()}, err
	}

	// Generate a JWT
	accessToken, err := tokenutil.GenerateToken(
		claims.UserID, secretKey, ac.env.JWT.AccessTokenExpiryHour)
	if err != nil {
		log.Printf("error generating token: %s", err)
		err := fmt.Errorf("error generating token")
		return &domain.AuthResponse{Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	return &domain.AuthResponse{
		Status: http.StatusOK,
		Data: &domain.AuthData{
			UserID:       claims.UserID.String(),
			AccessToken:  accessToken,
			RefreshToken: token,
		},
		Message: "Authenticated with token",
	}, nil
}

// getSecretKeys returns the secret keys from the environment variables
func (ac *AuthControl) getSecretKeys() (secretKey, refreshSecretKey []byte, err error) {
	// Load secret key from environment variable
	secretKey = []byte(ac.env.JWT.AccessTokenSecret)
	refreshSecretKey = []byte(ac.env.JWT.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		err = fmt.Errorf("JWT secret keys not set")
	}
	return
}
