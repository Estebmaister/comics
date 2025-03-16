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
	"comics/internal/service"
	"comics/internal/tokenutil"
)

const (
	invalidCredentials = "invalid credentials" // #nosec G101
)

// AuthControl is a controller to handle authentication
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
	return int(ac.env.JWTConfig.AccessTokenExpiryHour.Seconds())
}

// GetRefreshTokenExpirySeconds returns the expiry time in seconds
func (ac *AuthControl) GetRefreshTokenExpirySeconds() int {
	return int(ac.env.JWTConfig.RefreshTokenExpiryHour.Seconds())
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
	if err != nil {
		err := fmt.Errorf("%s: %w", invalidCredentials, err)
		return &domain.AuthResponse{Status: http.StatusUnauthorized, Message: invalidCredentials}, err
	}

	// Generate a JWT
	accessToken, errAccess := tokenutil.GenerateTokenWithRole(
		dbUser.ID, secretKey,
		ac.env.JWTConfig.AccessTokenExpiryHour, dbUser.Role)
	refreshToken, errRefresh := tokenutil.GenerateTokenWithRole(
		dbUser.ID, refreshSecretKey,
		ac.env.JWTConfig.RefreshTokenExpiryHour, dbUser.Role)
	if errAccess != nil && errRefresh != nil {
		err := fmt.Errorf("error generating token: %w %w", errAccess, errRefresh)
		return &domain.AuthResponse{
			Status: http.StatusInternalServerError, Message: "error generating token",
		}, err
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

// Register creates a new user or returns an error if the user already exists
func (ac *AuthControl) Register(ctx context.Context, user domain.SignUpRequest) (
	*domain.AuthResponse, error) {
	dbUser, err := ac.userService.Register(ctx, user)
	if errors.Is(err, service.ErrCredsAlreadyExist) {
		return &domain.AuthResponse{
			Status: http.StatusConflict, Message: err.Error()}, err
	} else if err != nil {
		return &domain.AuthResponse{
			Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	return &domain.AuthResponse{
		Status: http.StatusCreated,
		// Message: "User registered successfully",
		Message: fmt.Sprintf("%v", dbUser), // Debug response
		Data:    &domain.AuthData{UserID: dbUser.ID.String()},
	}, nil
}

// RefreshToken receives a refresh token and returns a new access token
func (ac *AuthControl) RefreshToken(_ context.Context, refreshToken, role string) (
	*domain.AuthResponse, error) {
	// Load secret key from environment variable
	secretKey, refreshSecretKey, err := ac.getSecretKeys()
	if err != nil {
		return &domain.AuthResponse{
			Status: http.StatusInternalServerError, Message: err.Error()}, err
	}

	if refreshToken == "" {
		err := fmt.Errorf("no access token provided")
		return &domain.AuthResponse{
			Status: http.StatusUnauthorized, Message: err.Error()}, err
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
		return &domain.AuthResponse{
			Status: http.StatusUnauthorized, Message: err.Error()}, err
	}

	// Generate a JWT
	accessToken, err := tokenutil.GenerateToken(
		claims.UserID, secretKey, ac.env.JWTConfig.AccessTokenExpiryHour)
	if err != nil {
		log.Printf("error generating token: %s", err)
		err := fmt.Errorf("error generating token")
		return &domain.AuthResponse{
			Status: http.StatusInternalServerError, Message: err.Error()}, err
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
	secretKey = []byte(ac.env.JWTConfig.AccessTokenSecret)
	refreshSecretKey = []byte(ac.env.JWTConfig.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		err = fmt.Errorf("JWT secret keys not set")
	}
	return
}
