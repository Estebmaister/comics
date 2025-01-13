package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"comics/bootstrap"
	"comics/domain"
	"comics/internal/tokenutil"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	USER = "user-uuid-gen-01" // UUID namespace for generating user IDs, 16 bytes
)

func Login(c context.Context, env *bootstrap.Env, accessToken string, user domain.LoginRequest) (
	*domain.AuthResponse, int, error) {
	// Load secret key from environment variable
	secretKey := []byte(env.AccessTokenSecret)
	refreshSecretKey := []byte(env.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("JWT secret keys not set")
	}

	// Check for Authorization header
	if accessToken != "" {
		token := strings.TrimPrefix(accessToken, "Bearer ")

		// Validate the token
		claims, err := tokenutil.VerifyToken(token, secretKey)
		if err == nil {
			return &domain.AuthResponse{
				UserID:      claims.UserID.String(),
				AccessToken: token,
				Message:     "Authenticated with token",
			}, http.StatusOK, nil
		}
		// If token is invalid, fallback to password-based login
	}

	// Fallback to standard password-based login
	// TODO: Simulate fetching user from database ( replace with real DB call)
	dbUser, err := getUserByEmail(user.Email)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid user credentials")
	}

	// Validate password (assuming bcrypt hash comparison)
	if err := bcrypt.CompareHashAndPassword(
		[]byte(dbUser.Password), []byte(user.Password)); err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
	}

	// Generate a JWT
	accessToken, errAccess := tokenutil.GenerateToken(
		dbUser.ID, secretKey, env.AccessTokenExpiryHour)
	refreshToken, errRefresh := tokenutil.GenerateToken(
		dbUser.ID, refreshSecretKey, env.RefreshTokenExpiryHour)
	if errAccess != nil && errRefresh != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error generating token")
	}

	// Return tokens in response
	return &domain.AuthResponse{
		UserID:       dbUser.ID.String(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Message:      "Login successful",
	}, http.StatusOK, nil
}

func Register(c context.Context, user domain.SignUpRequest) (
	*domain.AuthResponse, int, error) {
	dbUser := &domain.User{
		Email:    user.Email,
		Username: user.Username,
	}

	// Simulate fetching user from database (replace with real DB call)
	_, err := getUserByEmail(user.Email)
	if err == nil {
		return nil, http.StatusConflict,
			fmt.Errorf("email already exists")
	}

	// Simulate fetching user from database (replace with real DB call)
	_, err = getUserByUsername(user.Username)
	if err == nil {
		return nil, http.StatusConflict,
			fmt.Errorf("username already exists")
	}

	newID, err := uuid.NewV7FromReader(bytes.NewReader([]byte(USER)))
	if err != nil {
		println("Error generating UUID:", err.Error())
		newID = uuid.New() // Fallback to a random UUID if generation fails
	}
	dbUser.ID = newID

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, http.StatusInternalServerError,
			fmt.Errorf("error hashing password: %w", err)
	}
	dbUser.Password = string(hashedPassword)

	// Users can only be registered as "users" not "admin"
	dbUser.Role = tokenutil.ROLE_USER

	// Store the user in the database (replace this logic with real database operations)
	// ...
	// err = db.CreateUser(user)
	// if err !=nil ...

	return &domain.AuthResponse{
		UserID: dbUser.ID.String(),
		// Message: "User registered successfully",
		Message: fmt.Sprintf("%v", dbUser),
	}, http.StatusCreated, nil
}

func RefreshToken(c context.Context, env *bootstrap.Env, refreshToken, role string) (
	*domain.AuthResponse, int, error) {
	// Load secret key from environment variable
	secretKey := []byte(env.AccessTokenSecret)
	refreshSecretKey := []byte(env.RefreshTokenSecret)
	if len(secretKey) == 0 || len(refreshSecretKey) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("JWT secret keys not set")
	}

	// Check for Authorization header
	if refreshToken == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("no access token provided")
	}
	token := strings.TrimPrefix(refreshToken, "Bearer ")

	// Validate the token
	claims, err := tokenutil.VerifyToken(token, refreshSecretKey)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid refresh token")
	}
	if claims.Subject != role {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid role")
	}

	// Generate a JWT
	accessToken, err := tokenutil.GenerateToken(
		claims.UserID, secretKey, env.AccessTokenExpiryHour)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error generating token")
	}

	return &domain.AuthResponse{
		UserID:      claims.UserID.String(),
		AccessToken: accessToken,
		Message:     "Authenticated with token",
	}, http.StatusOK, nil
}

// Mocked user data for demonstration
var mockUser = domain.User{
	ID: func() uuid.UUID {
		uuid, _ := uuid.Parse("473d0a1d-f23b-42cf-b8ee-7e4ad29e42bf")
		return uuid
	}(),
	Username: "testuser",
	Email:    "test@example.com",
	Password: "$2a$10$WXvnJNuriTzpJThUJ6BJWOm6cxZNhQUsITXlXJTf0VGQYKATc9QMu", // bcrypt hash for "password123"
	Role:     "user",
}

// getUserByEmail simulates a database fetch (replace with real DB interaction)
func getUserByEmail(email string) (*domain.User, error) {
	if email != mockUser.Email {
		return nil, fmt.Errorf("user not found")
	}
	return &mockUser, nil
}

// getUserByUsername simulates a database fetch (replace with real DB interaction)
func getUserByUsername(username string) (*domain.User, error) {
	if username != mockUser.Username {
		return nil, fmt.Errorf("user not found")
	}
	return &mockUser, nil
}
